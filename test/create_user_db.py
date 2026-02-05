#!/usr/bin/env python3
"""
Create a new user directly in the PostgreSQL database (for dev/testing only).
Bypasses email verification - use only for local development.

Usage:
    python create_user_db.py --email user@example.com --username myuser --password mypassword

    # With custom database URL
    python create_user_db.py --db-url "postgresql://user:pass@localhost:5443/cozy" --email user@example.com --username myuser --password mypassword

Requirements:
    pip install psycopg2-binary argon2-cffi
"""

import argparse
import os
import re
import secrets
import sys
import uuid

try:
    import psycopg2
    from argon2 import PasswordHasher
    from argon2.low_level import Type
except ImportError as e:
    print(f"Missing dependency: {e}", file=sys.stderr)
    print("Install with: pip install psycopg2-binary argon2-cffi", file=sys.stderr)
    sys.exit(1)


DEFAULT_DB_URL = "postgresql://cozy:cozy@localhost:5443/cozy"


def hash_password_argon2id(password: str) -> str:
    """
    Hash password using Argon2id with AuthKit-compatible parameters.
    Returns PHC-encoded string.
    """
    # AuthKit defaults: time=1, memory=64*1024 (64MB), parallelism=1, salt_len=16, hash_len=32
    ph = PasswordHasher(
        time_cost=1,
        memory_cost=64 * 1024,
        parallelism=1,
        hash_len=32,
        salt_len=16,
        type=Type.ID,
    )
    return ph.hash(password)


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


def create_user_in_db(db_url: str, email: str, username: str, password: str) -> dict:
    """
    Create a new user directly in the AuthKit PostgreSQL database.

    Args:
        db_url: PostgreSQL connection URL
        email: User email address
        username: Desired username
        password: User password (will be hashed)

    Returns:
        Dict with user_id and username
    """
    user_id = str(uuid.uuid4())
    password_hash = hash_password_argon2id(password)

    conn = psycopg2.connect(db_url)
    try:
        with conn.cursor() as cur:
            # Check if email already exists
            cur.execute(
                "SELECT id FROM profiles.users WHERE email = %s",
                (email,)
            )
            if cur.fetchone():
                raise Exception(f"Email '{email}' already exists")

            # Check if username already exists
            cur.execute(
                "SELECT id FROM profiles.users WHERE username = %s",
                (username,)
            )
            if cur.fetchone():
                raise Exception(f"Username '{username}' already exists")

            # Create user
            cur.execute(
                """
                INSERT INTO profiles.users (id, email, username, email_verified, created_at, updated_at)
                VALUES (%s, %s, %s, %s, NOW(), NOW())
                RETURNING id
                """,
                (user_id, email, username, True)  # email_verified=True for dev
            )

            # Create password entry
            cur.execute(
                """
                INSERT INTO profiles.user_passwords (user_id, password_hash, hash_algo, password_updated_at)
                VALUES (%s, %s, %s, NOW())
                """,
                (user_id, password_hash, "argon2id")
            )

            conn.commit()

    except Exception as e:
        conn.rollback()
        raise e
    finally:
        conn.close()

    return {
        "user_id": user_id,
        "email": email,
        "username": username,
        "email_verified": True,
    }


def main():
    parser = argparse.ArgumentParser(
        description="Create a new user directly in the database (dev only)",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Create user with default local DB
  python create_user_db.py --email user@example.com --username myuser --password mypassword

  # Custom database URL
  python create_user_db.py --db-url "postgresql://user:pass@localhost:5443/mydb" \\
      --email test@test.com --username testuser --password testpass123

NOTE: This script bypasses email verification and should only be used for local development.
        """
    )

    parser.add_argument(
        "--db-url",
        default=os.getenv("DATABASE_URL", DEFAULT_DB_URL),
        help=f"PostgreSQL connection URL (default: {DEFAULT_DB_URL})"
    )
    parser.add_argument(
        "--email",
        required=True,
        help="Email address for the user"
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

    # Validate inputs
    try:
        validate_email(args.email)
        validate_username(args.username)
        validate_password(args.password)
    except ValueError as e:
        print(f"Validation error: {e}", file=sys.stderr)
        sys.exit(1)

    # Create user
    print(f"Creating user in database...")
    print(f"  Database: {args.db_url.split('@')[-1] if '@' in args.db_url else args.db_url}")
    print(f"  Email: {args.email}")
    print(f"  Username: {args.username}")
    print()

    try:
        result = create_user_in_db(
            db_url=args.db_url,
            email=args.email,
            username=args.username,
            password=args.password,
        )

        print("User created successfully!")
        print(f"  User ID: {result['user_id']}")
        print(f"  Email: {result['email']}")
        print(f"  Username: {result['username']}")
        print(f"  Email Verified: {result['email_verified']}")
        print()
        print("You can now login with:")
        print(f"  cozyctl login --email {args.email} --password <password>")

    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
