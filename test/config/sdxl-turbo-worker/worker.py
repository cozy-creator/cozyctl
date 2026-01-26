"""
SDXL-Turbo Image Generation Worker

Works on:
- Apple Silicon (MPS) - for local testing
- NVIDIA GPU (CUDA) - for RunPod deployment
- CPU - fallback (slow but works)

Environment variables:
- MODEL_PATH: Path to local model (default: downloads from HuggingFace)
"""

import torch
from diffusers import AutoPipelineForText2Image
import os
import base64
from io import BytesIO

# Global pipeline instance
pipe = None

# Model path - can be overridden via environment variable
DEFAULT_MODEL = "stabilityai/sdxl-turbo"
MODEL_PATH = os.environ.get("MODEL_PATH", DEFAULT_MODEL)


def get_device():
    """Detect the best available device."""
    if torch.cuda.is_available():
        return "cuda"
    elif torch.backends.mps.is_available():
        return "mps"
    return "cpu"


def load_model():
    """Load SDXL-Turbo model from local path or HuggingFace."""
    global pipe

    device = get_device()
    dtype = torch.float16 if device in ["cuda", "mps"] else torch.float32

    model_source = MODEL_PATH
    print(f"Loading model from: {model_source}")
    print(f"Device: {device}, Dtype: {dtype}")

    # Check if local path exists
    if os.path.isdir(model_source):
        print(f"Using local model at: {model_source}")
        pipe = AutoPipelineForText2Image.from_pretrained(
            model_source,
            torch_dtype=dtype,
            local_files_only=True,
        )
    else:
        print(f"Downloading from HuggingFace: {model_source}")
        pipe = AutoPipelineForText2Image.from_pretrained(
            model_source,
            torch_dtype=dtype,
            variant="fp16" if dtype == torch.float16 else None,
        )

    pipe.to(device)

    # Disable safety checker for faster inference (optional)
    pipe.safety_checker = None

    print(f"Model loaded successfully on {device}")
    return pipe


def generate(
    prompt: str,
    output_path: str = "output.png",
    num_steps: int = 4,
    width: int = 512,
    height: int = 512,
    seed: int = None,
):
    """
    Generate an image from a text prompt.

    Args:
        prompt: Text description of the image to generate
        output_path: Path to save the generated image
        num_steps: Number of inference steps (1-4 for turbo)
        width: Image width (default 512)
        height: Image height (default 512)
        seed: Random seed for reproducibility

    Returns:
        Path to the saved image
    """
    global pipe

    if pipe is None:
        load_model()

    # Set seed for reproducibility
    generator = None
    if seed is not None:
        generator = torch.Generator(device=get_device()).manual_seed(seed)

    print(f"Generating: '{prompt}'")
    print(f"Settings: steps={num_steps}, size={width}x{height}")

    image = pipe(
        prompt=prompt,
        num_inference_steps=num_steps,
        guidance_scale=0.0,  # SDXL-Turbo doesn't need guidance
        width=width,
        height=height,
        generator=generator,
    ).images[0]

    # Save image
    image.save(output_path)
    print(f"Saved to: {output_path}")

    return output_path


def generate_base64(
    prompt: str,
    num_steps: int = 4,
    width: int = 512,
    height: int = 512,
    seed: int = None,
):
    """
    Generate an image and return as base64 string.
    Useful for API responses.
    """
    global pipe

    if pipe is None:
        load_model()

    generator = None
    if seed is not None:
        generator = torch.Generator(device=get_device()).manual_seed(seed)

    image = pipe(
        prompt=prompt,
        num_inference_steps=num_steps,
        guidance_scale=0.0,
        width=width,
        height=height,
        generator=generator,
    ).images[0]

    # Convert to base64
    buffer = BytesIO()
    image.save(buffer, format="PNG")
    img_base64 = base64.b64encode(buffer.getvalue()).decode("utf-8")

    return img_base64


# RunPod serverless handler (optional)
def handler(event):
    """
    RunPod serverless handler.

    Input format:
    {
        "input": {
            "prompt": "a beautiful sunset",
            "num_steps": 4,
            "width": 512,
            "height": 512,
            "seed": 42
        }
    }
    """
    input_data = event.get("input", {})

    prompt = input_data.get("prompt", "a beautiful landscape")
    num_steps = input_data.get("num_steps", 4)
    width = input_data.get("width", 512)
    height = input_data.get("height", 512)
    seed = input_data.get("seed")

    img_base64 = generate_base64(
        prompt=prompt,
        num_steps=num_steps,
        width=width,
        height=height,
        seed=seed,
    )

    return {
        "image_base64": img_base64,
        "prompt": prompt,
        "settings": {
            "num_steps": num_steps,
            "width": width,
            "height": height,
            "seed": seed,
        }
    }


if __name__ == "__main__":
    # Local test
    import sys

    prompt = sys.argv[1] if len(sys.argv) > 1 else "a cute robot painting a picture"
    output = sys.argv[2] if len(sys.argv) > 2 else "output.png"

    generate(prompt, output)
