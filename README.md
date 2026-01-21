# cozyctl

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
cozyctl profiles

# Show current profile
cozyctl current

# Switch profiles
cozyctl use --name briheet --profile prod
cozyctl use --profile staging              # Keep current name, switch profile
cozyctl use --name work                    # Switch name, keep current profile

# Delete a profile
cozyctl delete --name briheet --profile staging
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

Additional auth commands:
- `logout` - ⏳ Coming soon
- `whoami` - ⏳ Coming soon

### 2. Deploy
Deploy projects to Cozy

```bash
cozyctl deploy [PATH]
  --deployment NAME       # Custom deployment name
  --push                  # Push image to registry (default: true)
  --dry-run              # Validate only, don't build
```

Validates projects with `pyproject.toml`, creates tarball, uploads to Gen-Builder, and streams build logs.

### 3. Builds
Manage builds

```bash
cozyctl builds list [--limit N]
cozyctl builds logs BUILD_ID [--follow]
cozyctl builds cancel BUILD_ID
```

- `list` - List recent builds with status
- `logs` - View build logs (supports streaming with `--follow`)
- `cancel` - Cancel a running build

### 4. Profiles
Manage configuration profiles

```bash
cozyctl profiles                           # List all profiles
cozyctl use --name NAME --profile PROFILE  # Switch profile
cozyctl current                            # Show current profile
cozyctl delete --name NAME --profile PROFILE
```

### 5. Job
Talks to gen-orchestrator for inference jobs
- `submit`, `get`, `logs`, `cancel`, `list`, `download`

### 6. Models
Talks to cozy-hub for model management
- `download`, `list`, `search`, `get`, `url`
- `upload`, `delete` (admin only)
