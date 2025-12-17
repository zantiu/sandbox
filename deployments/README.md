# GitOps-like Deployments

NOTE: quick GitOps-like concept, set up by Claude Code, not tested!
The aim is just to show that a git repo can be used to declaratively describe the desired state.

This directory enables Git-triggered deployment synchronization to the WFM. Each `.yaml` file represents one deployment. When you commit changes, deployments are automatically synced to the Workload Fleet Manager.

## Directory Structure

```
deployments/
  otel-service-prod.yaml    # One deployment per file
  nextcloud-prod.yaml
  my-new-app.yaml           # Add more as needed
  README.md
```

## Quick Start

### 1. Set up environment

```bash
source pipeline/wfm.env
```

### 2. Install git hook (one-time)

```bash
./scripts/install-hooks.sh
```

### 3. Create a deployment file

Create a new `.yaml` file in this directory:

```yaml
# deployments/my-app.yaml
apiVersion: gitops.margo.org/v1
kind: Deployment

name: my-app-prod
description: "My application"

package:
  name: my-app-package        # Must exist in Harbor/WFM

device:
  id: "device-123"            # Target device ID

profile:
  type: helm.v3               # or "compose"
  components:
    - name: my-app
      properties:
        repository: "oci://${EXPOSED_HARBOR_IP}:8081/library/my-app"
        revision: "1.0.0"
```

### 4. Commit to trigger sync

```bash
git add deployments/my-app.yaml
git commit -m "Deploy my-app"
# Hook automatically syncs to WFM
```

## Deployment File Schema

```yaml
apiVersion: gitops.margo.org/v1
kind: Deployment

name: <deployment-name>          # Unique name (required)
description: "Optional description"

package:
  name: <package-name>           # Package name in WFM (required)

device:
  id: "<device-id>"              # Specific device ID
  # OR use labels to match multiple devices:
  # labels:
  #   runtime: k3s
  #   environment: production

profile:
  type: helm.v3                  # or "compose" (required)
  components:
    - name: <component-name>
      properties:
        # For Helm:
        repository: "oci://..."
        revision: "1.0.0"
        wait: true
        timeout: 5m
        # For Compose:
        # packageLocation: "https://..."

parameters:                      # Optional parameter overrides
  paramName:
    value: "param-value"
    targets:
      - pointer: env.VAR_NAME
        components: ["component-name"]
```

## Manual Sync

You can also run the sync script manually:

```bash
# Preview changes (dry run)
./scripts/sync-to-wfm.sh --dry-run

# Apply changes
./scripts/sync-to-wfm.sh

# Verbose output
./scripts/sync-to-wfm.sh --verbose

# Custom directory
./scripts/sync-to-wfm.sh -D /path/to/deployments
```

## Environment Variables

The sync script requires these environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `EXPOSED_SYMPHONY_IP` | WFM Symphony API IP | (required) |
| `EXPOSED_SYMPHONY_PORT` | WFM Symphony API port | 8082 |
| `EXPOSED_HARBOR_IP` | Harbor registry IP | (for substitution) |
| `EXPOSED_HARBOR_PORT` | Harbor registry port | 8081 |

You can use `${VAR}` syntax in deployment files for environment variable substitution.

## Prerequisites

Before deployments can be synced:

1. **Packages must be uploaded to Harbor** - Use the existing upload workflow
2. **Packages must be onboarded to WFM** - Use `wfm-cli.sh` option 5
3. **Devices must be registered** - Device agents must have onboarded to WFM

## Behavior

- **One file per deployment**: Each `.yaml` file is one deployment
- **Additive only**: Creates new deployments, never deletes existing ones
- **Idempotent**: Safe to run multiple times. Existing deployments are skipped.
- **Works alongside CLI**: Does not replace `wfm-cli.sh`

## Troubleshooting

### Hook not running

1. Verify hook is installed: `ls -la .git/hooks/post-commit`
2. Check environment is set: `echo $EXPOSED_SYMPHONY_IP`
3. View sync logs: `cat logs/gitops-sync.log`

### Package not found

Ensure the package is:
1. Uploaded to Harbor (`oras push ...`)
2. Onboarded to WFM (`wfm-cli.sh` → option 5)

### Device not found

1. Check device is registered: `wfm-cli.sh` → option 2 (List Devices)
2. Verify device ID in your deployment file

### Remove hook

```bash
./scripts/install-hooks.sh --remove
```

## Files

| File | Purpose |
|------|---------|
| `*.yaml` | Individual deployment definitions |
| `../scripts/sync-to-wfm.sh` | Sync script |
| `../scripts/install-hooks.sh` | Hook installer |
| `../logs/gitops-sync.log` | Sync log file |
