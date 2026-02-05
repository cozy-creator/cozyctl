#!/usr/bin/env python3
"""
Create a new user via AuthKit API.

Usage:
    python create_user.py --email user@example.com --username myuser --password mypassword
    python create_user.py --phone +14155551234 --username myuser --password mypassword

    # With custom hub URL
    python create_user.py --hub-url http://localhost:8080 --email user@example.com --username myuser --password mypassword
"""

import argparse
import json
import re
import sys
import requests


DEFAULT_HUB_URL = "https://api.cozy.art"


def validate_username(username: str) -> None:
    """Validate username according to AuthKit rules."""
    username = username.strip()

    if len(username) < 4:
        raise ValueError("username must be at least 4 characters")
    if len(username) > 30:
        raise ValueError("username must be at most 30 characters")

    if not username[0].isalpha():
        raise ValueError("username must start with a letter")

    if not re.match(r'^[a-zA-Z0-9_]+$', username):
        raise ValueError("username can only contain letters, numbers, and underscores")

    if username.lower() in ("admin", "moderator"):
        raise ValueError("username is reserved")


def validate_password(password: str) -> None:
    """Validate password according to AuthKit rules."""
    if len(password) < 8:
        raise ValueError("password must be at least 8 characters")


def validate_email(email: str) -> None:
    """Basic email validation."""
    if "@" not in email or len(email) < 5:
        raise ValueError("invalid email format")


def validate_phone(phone: str) -> None:
    """Validate phone number (E.164 format)."""
    if not phone.startswith("+"):
        raise ValueError("phone number must start with + (E.164 format)")
    if len(phone) < 10:
        raise ValueError("phone number too short")
    if not re.match(r'^\+[0-9]+$', phone):
        raise ValueError("phone number can only contain digits after +")


def create_user(hub_url: str, identifier: str, username: str, password: str) -> dict:
    """
    Create a new user via AuthKit registration API.

    Args:
        hub_url: The Cozy Hub API URL
        identifier: Email address or phone number
        username: Desired username
        password: User password

    Returns:
        API response as dict
    """
    url = f"{hub_url.rstrip('/')}/api/v1/auth/register"

    payload = {
        "identifier": identifier,
        "username": username,
        "password": password,
    }

    headers = {
        "Content-Type": "application/json",
    }

    response = requests.post(url, json=payload, headers=headers, timeout=30)

    try:
        data = response.json()
    except json.JSONDecodeError:
        data = {"raw_response": response.text}

    if response.status_code not in (200, 201, 202):
        error_msg = data.get("error", "")
        if not error_msg:
            error_msg = data.get("message", "")
        if not error_msg:
            error_msg = f"HTTP {response.status_code}"
        # Include full response for debugging
        print(f"Response status: {response.status_code}", file=sys.stderr)
        print(f"Response body: {json.dumps(data, indent=2)}", file=sys.stderr)
        raise Exception(f"Registration failed: {error_msg}")

    return data


def main():
    parser = argparse.ArgumentParser(
        description="Create a new user via AuthKit API",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Register with email
  python create_user.py --email user@example.com --username myuser --password mypassword

  # Register with phone
  python create_user.py --phone +14155551234 --username myuser --password mypassword

  # Custom hub URL
  python create_user.py --hub-url http://localhost:8080 --email test@test.com --username testuser --password testpass123
        """
    )

    parser.add_argument(
        "--hub-url",
        default=DEFAULT_HUB_URL,
        help=f"Cozy Hub API URL (default: {DEFAULT_HUB_URL})"
    )
    parser.add_argument(
        "--email",
        help="Email address for registration"
    )
    parser.add_argument(
        "--phone",
        help="Phone number for registration (E.164 format, e.g., +14155551234)"
    )
    parser.add_argument(
        "--username",
        required=True,
        help="Username (4-30 chars, starts with letter, alphanumeric + underscore)"
    )
    parser.add_argument(
        "--password",
        required=True,
        help="Password (minimum 8 characters)"
    )

    args = parser.parse_args()

    # Determine identifier (email or phone)
    if args.email and args.phone:
        print("Error: Provide either --email or --phone, not both", file=sys.stderr)
        sys.exit(1)

    if not args.email and not args.phone:
        print("Error: Must provide either --email or --phone", file=sys.stderr)
        sys.exit(1)

    identifier = args.email or args.phone

    # Validate inputs
    try:
        validate_username(args.username)
        validate_password(args.password)

        if args.email:
            validate_email(args.email)
        else:
            validate_phone(args.phone)

    except ValueError as e:
        print(f"Validation error: {e}", file=sys.stderr)
        sys.exit(1)

    # Create user
    print(f"Creating user...")
    print(f"  Hub URL: {args.hub_url}")
    print(f"  Identifier: {identifier}")
    print(f"  Username: {args.username}")
    print()

    try:
        result = create_user(
            hub_url=args.hub_url,
            identifier=identifier,
            username=args.username,
            password=args.password,
        )

        print("Registration successful!")
        print(json.dumps(result, indent=2))

        if result.get("message"):
            print(f"\nNote: {result['message']}")

    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
