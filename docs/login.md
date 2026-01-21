# Authentication and Profile Management

cozyctl uses a two-level configuration system to manage multiple accounts and environments.

## Concepts

### Two-Level Hierarchy

1. **Name**: User or account identifier (e.g., `briheet`, `vikas`, `work-account`)
   - Represents who you are or which account you're using
   - Can be your username, company name, client name, etc.

2. **Profile**: Environment within that name (e.g., `dev`, `staging`, `prod`)
   - Represents which environment you're working with
   - Can be development stages, different projects, etc.

### Current Context

The CLI maintains a "current" name+profile combination. All commands use this context unless overridden with flags.

## File Structure

```
~/.cozy/
  ├── default/
  │   └── config.yaml                    # Pointer: tracks current_name and current_profile
  ├── briheet/
  │   ├── dev/
  │   │   └── config.yaml                # Full config for briheet/dev
  │   ├── staging/
  │   │   └── config.yaml                # Full config for briheet/staging
  │   └── prod/
  │       └── config.yaml                # Full config for briheet/prod
  └── vikas/
      ├── dev/
      │   └── config.yaml                # Full config for vikas/dev
      └── prod/
          └── config.yaml                # Full config for vikas/prod
```

### Default Config (`~/.cozy/default/config.yaml`)

This file acts as a pointer, tracking which name+profile is currently active.

```yaml
current_name: briheet
current_profile: dev
```

### Profile Config (`~/.cozy/{name}/{profile}/config.yaml`)

Each name+profile combination has its own complete configuration.

```yaml
current_name: briheet
current_profile: dev
config:
  hub_url: https://api.cozy.art
  builder_url: https://builder.cozy.art
  tenant_id: tenant-briheet-dev-123
  token: api-key-briheet-dev-xyz
```

## Commands

### Login

#### Login with API Key

```bash
cozyctl login --name briheet --profile dev --api-key YOUR_API_KEY
```

This command:
1. Validates the API key with the Cozy Hub
2. Creates `~/.cozy/briheet/dev/config.yaml` with the credentials
3. Updates `~/.cozy/default/config.yaml` to set briheet/dev as current

**Optional flags:**
- `--hub-url`: Override default Hub URL (default: `https://api.cozy.art`)
- `--builder-url`: Override default Builder URL (default: `https://builder.cozy.art`)
- `--tenant-id`: Explicitly set tenant ID (usually auto-detected)

**Example:**
```bash
cozyctl login \
  --name briheet \
  --profile dev \
  --api-key sk_dev_abc123xyz \
  --hub-url https://api.cozy.art \
  --builder-url https://builder.cozy.art
```

#### Login with Config File

Import an existing config file into a name+profile:

```bash
cozyctl login --name briheet --profile prod --config-file ./prod-config.yaml
```

This command:
1. Reads the config file
2. Creates `~/.cozy/briheet/prod/config.yaml` with the imported config
3. Updates `~/.cozy/default/config.yaml` to set briheet/prod as current

#### Default Name and Profile

If `--name` is not specified, it defaults to `default`.
If `--profile` is not specified, it defaults to `default`.

```bash
# Simple login (creates ~/.cozy/default/default/config.yaml)
cozyctl login --api-key YOUR_API_KEY
```

#### Overwrite Protection

If a profile already exists, the CLI will prompt for confirmation:

```bash
$ cozyctl login --name briheet --profile dev --api-key new-key
Profile 'briheet/dev' already exists. Overwrite? [y/N]:
```

### Switching Contexts

#### Switch Both Name and Profile

```bash
cozyctl use --name vikas --profile prod
```

Updates `~/.cozy/default/config.yaml` to use vikas/prod for all subsequent commands.

#### Switch Only Profile (Keep Current Name)

```bash
cozyctl use --profile staging
```

Updates only the profile in `~/.cozy/default/config.yaml`, keeping the current name.

#### Switch Only Name (Keep Current Profile)

```bash
cozyctl use --name vikas
```

Updates only the name in `~/.cozy/default/config.yaml`, keeping the current profile.

### Listing Profiles

```bash
cozyctl profiles
```

**Output:**
```
NAME       PROFILE    CURRENT
briheet    dev        *
briheet    staging
briheet    prod
vikas      dev
vikas      prod
```

The asterisk (*) indicates the currently active name+profile.

### Show Current Context

```bash
cozyctl current
```

**Output:**
```
briheet/dev
```

### Delete a Profile

```bash
cozyctl delete --name briheet --profile staging
```

This removes the entire `~/.cozy/briheet/staging/` directory.

**Protection:**
- Cannot delete `default/default` profile
- If deleting the currently active profile, automatically switches to `default/default`

### Command-Level Override

Use a different name+profile for a single command without changing the current context:

```bash
# Temporarily use vikas/prod for this command
cozyctl --name vikas --profile prod builds list

# Next command still uses briheet/dev (current context unchanged)
cozyctl deploy .
```

## Usage Examples

### Example 1: Single User, Multiple Environments

```bash
# Setup development environment
cozyctl login --name briheet --profile dev --api-key dev-key-123

# Setup staging environment
cozyctl login --name briheet --profile staging --api-key staging-key-456

# Setup production environment
cozyctl login --name briheet --profile prod --api-key prod-key-789

# Work in dev (current by default after last login)
cozyctl deploy ./my-project
cozyctl builds list

# Switch to staging for testing
cozyctl use --profile staging
cozyctl deploy ./my-project
cozyctl builds list

# Deploy to production
cozyctl use --profile prod
cozyctl deploy ./my-project

# Quick check on dev without switching
cozyctl --profile dev builds list
```

### Example 2: Multiple Users (Team Environment)

```bash
# Login as briheet
cozyctl login --name briheet --profile dev --api-key briheet-key

# Login as vikas (different account)
cozyctl login --name vikas --profile dev --api-key vikas-key

# Work as briheet
cozyctl use --name briheet --profile dev
cozyctl deploy .

# Switch to vikas's account
cozyctl use --name vikas --profile dev
cozyctl deploy .

# Check briheet's builds without switching
cozyctl --name briheet --profile dev builds list
```

### Example 3: Freelancer with Multiple Clients

```bash
# Client 1: Acme Corp
cozyctl login --name acme --profile dev --api-key acme-dev-key
cozyctl login --name acme --profile prod --api-key acme-prod-key

# Client 2: Globex Inc
cozyctl login --name globex --profile dev --api-key globex-dev-key
cozyctl login --name globex --profile prod --api-key globex-prod-key

# Work on Acme's dev environment
cozyctl use --name acme --profile dev
cozyctl deploy ./acme-project

# Deploy to Acme's production
cozyctl use --profile prod
cozyctl deploy ./acme-project

# Switch to Globex
cozyctl use --name globex --profile dev
cozyctl deploy ./globex-project
```

### Example 4: Import Existing Configs

```bash
# Import production config from file
cozyctl login \
  --name briheet \
  --profile prod \
  --config-file ~/.old-cozy-config/prod.yaml

# Import staging config
cozyctl login \
  --name briheet \
  --profile staging \
  --config-file ~/.old-cozy-config/staging.yaml
```

## Environment Variables

All config values can be overridden with environment variables:

```bash
# Override hub URL for one command
export COZY_HUB_URL=https://custom-hub.cozy.art
cozyctl deploy .

# Override token (useful for CI/CD)
export COZY_TOKEN=ci-token-xyz
export COZY_TENANT_ID=ci-tenant-123
cozyctl deploy .
```

Environment variable format: `COZY_<KEY>` (uppercase with underscore separator)

Available variables:
- `COZY_HUB_URL`
- `COZY_BUILDER_URL`
- `COZY_TOKEN`
- `COZY_TENANT_ID`
- `COZY_API_KEY` (only used during login)

## Best Practices

### Naming Conventions

**For Names:**
- Personal use: Use your username (e.g., `briheet`, `vikas`)
- Team use: Use descriptive identifiers (e.g., `personal`, `work`, `client-name`)
- Multi-account: Use account identifiers (e.g., `company-abc`, `freelance-xyz`)

**For Profiles:**
- Environment-based: `dev`, `staging`, `prod`
- Project-based: `project-a`, `project-b`
- Feature-based: `feature-x`, `feature-y`

### Security

1. **File Permissions**: All config files are created with `0600` permissions (owner read/write only)
2. **API Keys**: Never commit config files to version control
3. **CI/CD**: Use environment variables instead of config files
4. **Shared Systems**: Each user should use their own name to avoid conflicts

### Workflow Recommendations

1. **Start Simple**: Begin with default/default profile
2. **Add Environments**: Create dev/staging/prod as needed
3. **Use `current`**: Always check your current context before deploying
4. **Override Carefully**: Use `--name`/`--profile` flags for one-off checks
5. **Clean Up**: Delete unused profiles to keep things organized

## Troubleshooting

### Profile Not Found

```bash
$ cozyctl deploy .
Error: profile 'briheet/dev' not found (run 'cozyctl login --name briheet --profile dev' first)
```

**Solution**: Login to create the profile first.

### Current Context Unknown

If `~/.cozy/default/config.yaml` is missing or corrupted:

```bash
$ cozyctl deploy .
Error: no current profile set (run 'cozyctl login' first)
```

**Solution**: Login to any profile to reset the default config.

### Overwrite Accidentally

If you accidentally overwrote a profile:

**Prevention**: Always pay attention to the overwrite prompt.

**Recovery**: Config files are just YAML - you can manually edit them at `~/.cozy/{name}/{profile}/config.yaml`

### List All Configs

To see all your configs:

```bash
ls -R ~/.cozy/
```

### Manual Cleanup

To remove all configs and start fresh:

```bash
rm -rf ~/.cozy/
```

## Migration from Old Config

If you have an old flat config file, you can import it:

```bash
# Import old config as default/default
cozyctl login --name default --profile default --config-file ~/.cozy.yaml

# Or as a named profile
cozyctl login --name briheet --profile prod --config-file ~/.cozy.yaml
```

## Technical Details

### Config File Schema

#### Default Config Schema

```yaml
current_name: string      # Currently active name
current_profile: string   # Currently active profile
```

#### Profile Config Schema

```yaml
current_name: string      # Name for this config (redundant but useful)
current_profile: string   # Profile for this config
config:
  hub_url: string         # Cozy Hub API URL
  builder_url: string     # Gen-Builder API URL
  tenant_id: string       # Tenant identifier
  token: string           # API authentication token
```

### Directory Permissions

- `~/.cozy/`: `0700` (drwx------)
- `~/.cozy/{name}/`: `0700` (drwx------)
- `~/.cozy/{name}/{profile}/`: `0700` (drwx------)
- `~/.cozy/{name}/{profile}/config.yaml`: `0600` (-rw-------)

### Viper Integration

Each profile config is loaded with its own Viper instance to avoid conflicts. Environment variables are merged at load time.
