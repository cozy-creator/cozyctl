# Cozyctl

A command-line tool for deploying and managing machine learning functions on the Cozy platform.

cozyctl makes it convenient for developers to interact with:

- **Gen-Builder** service - Create and manage deployments
- **Cozy-Hub** service - Upload and download model weights
- **Gen-Orchestrator** service - Invoke functions and fetch results

## Installation

```bash
# Clone the repository
git clone https://github.com/cozy-creator/cozyctl.git
cd cozyctl

# Build the binary
go build -o cozyctl .

# Run
./cozyctl --help
```

## Quick Start

```bash
# Login with your API key
cozyctl login --api-key YOUR_API_KEY

# Deploy a project
cozyctl deploy ./my-project

# List builds
cozyctl builds list

# View build logs
cozyctl builds logs BUILD_ID
```

## Multi-Profile Support

cozyctl supports managing multiple accounts and environments through a two-level configuration system:

- **Name**: User or account identifier (e.g., `personal`, `work`, `client-name`)
- **Profile**: Environment within that name (e.g., `dev`, `staging`, `prod`)

### Authentication with Profiles

```bash
# Login with default profile
cozyctl login --api-key YOUR_API_KEY

# Login with custom name and profile
cozyctl login --name briheet --profile dev --api-key YOUR_DEV_KEY
cozyctl login --name briheet --profile prod --api-key YOUR_PROD_KEY

# Login to work account
cozyctl login --name work --profile staging --api-key WORK_KEY

# Import existing config file
cozyctl login --name briheet --profile prod --config-file ./prod-config.yaml
```

### Managing Profiles

```bash
# List all profiles (current profile marked with *)
cozyctl profile

# Show current profile
cozyctl profile current

# Switch profiles
cozyctl profile switch --name briheet --profile prod
cozyctl profile switch --profile staging              # Keep current name, switch profile
cozyctl profile switch --name work                    # Switch name, keep current profile

# Delete a profile
cozyctl profile delete --name briheet --profile staging
```

### Using Profiles

```bash
# Commands use the current profile by default
cozyctl deploy .
cozyctl builds list

# Override profile for a single command
cozyctl --name work --profile prod builds list
cozyctl --name briheet --profile dev deploy .
```

### Configuration Structure

Profiles are stored in `~/.cozy/`:

```
~/.cozy/
  ├── default/
  │   └── config.yaml                    # Tracks current name+profile
  ├── briheet/
  │   ├── dev/
  │   │   └── config.yaml                # Config for briheet/dev
  │   ├── staging/
  │   │   └── config.yaml                # Config for briheet/staging
  │   └── prod/
  │       └── config.yaml                # Config for briheet/prod
  └── work/
      └── prod/
          └── config.yaml                # Config for work/prod
```

For detailed documentation, see [docs/login.md](docs/login.md).

See [example.config.yaml](example.config.yaml) for a sample configuration file structure.

## Commands

### 1. Login
Authentication with Cozy platform

```bash
cozyctl login [--name NAME] [--profile PROFILE] [--api-key KEY]
cozyctl login --config-file PATH
```
Authenticate with API key or import config file into a name/profile combination.

### 2. Deploy
Build and register a new deployment with the orchestrator.

```bash
cozyctl deploy ./my-project              # Build + register
cozyctl deploy ./my-project --dry-run    # Preview without executing
cozyctl deploy ./my-project --register=false  # Build only
cozyctl deploy ./my-project --min-workers 2 --max-workers 10
```

### 3. Update
Rebuild and update an existing deployment.

```bash
cozyctl update ./my-project              # Rebuild + update
cozyctl update ./my-project --image-only # Only update image
cozyctl update ./my-project --dry-run    # Preview without executing
```

### 4. Builds
Manage builds

- `list` - List recent builds with status
- `logs` - View build logs (supports streaming with `--follow`)
- `cancel` - Cancel a running build

### 5. Build
Build Docker images locally from projects with `pyproject.toml`

```bash
cozyctl build -l -d ./path/to/project
```

#### Local Build Demo

Test with the included SDXL-Turbo worker:

```bash
# Build the image
./bin/cozyctl build -l -d ./test/config/sdxl-turbo-worker/

# Download model (~7GB, one-time)
huggingface-cli download stabilityai/sdxl-turbo --local-dir ~/models/sdxl-turbo

# Run the container
docker run \
  -v ~/models/sdxl-turbo:/models/sdxl-turbo \
  -v $(pwd)/test/test-output:/output \
  -e MODEL_PATH=/models/sdxl-turbo \
  cozy-build-sdxl-turbo-test-<build-id>:latest

# View result
open test/test-output/output.png
```

Base image auto-selected from `[tool.cozy]` config:
- CPU: `python:3.11-slim`
- PyTorch CPU: `cozycreator/gen-worker:cpu-torch2.9`
- PyTorch + CUDA: `cozycreator/gen-worker:cuda12.6-torch2.9`

### 6. Profiles
Manage configuration profiles

```bash
cozyctl profile                           # List all profiles
cozyctl profile switch --name NAME --profile PROFILE  # Switch profile
cozyctl profile current                            # Show current profile
cozyctl profile delete --name NAME --profile PROFILE
```

### 7. Job
Talks to gen-orchestrator for inference jobs
- `submit`, `get`, `logs`, `cancel`, `list`, `download`

### 8. Models
Talks to cozy-hub for model management
- `download`, `list`, `search`, `get`, `url`
- `upload`, `delete` (admin only)

## Project Configuration

Projects require a `pyproject.toml` with `[tool.cozy]` configuration:

```toml
[project]
name = "my-worker"
dependencies = ["gen-worker", "torch", "diffusers"]

[tool.cozy]
deployment-id = "my-deployment"
python = "3.11"
pytorch = "2.5"
cuda = "12.6"

[tool.cozy.environment]
HF_HOME = "/app/.cache/huggingface"

# Optional: Define functions explicitly (auto-detected if omitted)
[tool.cozy.functions]
generate = { requires_gpu = true }
health = { requires_gpu = false }
```

Functions can be defined three ways (in priority order):
1. `--functions` CLI flag
2. `[tool.cozy.functions]` in pyproject.toml
3. Auto-detection from `@worker_function()` decorators
