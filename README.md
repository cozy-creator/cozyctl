# cozyctl

This CLI makes it convenient for developers to interact with:

- Gen-Builder service, to create deployments
- Cozy-Hub service, to upload and download model-weights
- Gen-Orchestrator service, to invoke functions and fetch results back

cozyctl will have these functionalities:

### 1. Auth
Authentication management
- `login`, `logout`, `whoami`

### 2. Deploy
Talks to gen-builder
- `create`, `list`, `get`, `logs`, `promote`, `rollback`, `config`, `delete`

### 3. Build
Talks to gen-builder. Will support multiple builds at once via a TOML/YAML file.
- `list`, `get`, `logs`, `watch`, `cancel`

### 4. Job
Talks to gen-orchestrator for inference jobs
- `submit`, `get`, `logs`, `cancel`, `list`, `download`

### 5. Models
Talks to cozy-hub for model management
- `download`, `list`, `search`, `get`, `url`
- `upload`, `delete` (admin only - TBD)
