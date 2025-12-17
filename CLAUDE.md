# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is the Margo Code-First Sandbox - an open-source implementation of the Margo specification for interoperable fleet management of edge applications and devices. The project implements interfaces between a Workload Fleet Manager (WFM) and edge device agents using Eclipse Symphony as the WFM component.

## Build Commands

```bash
# Install dependencies
go mod download

# Build device agent
cd poc/device/agent && go build -o agent .

# Run device agent locally
./agent

# Run with custom config
go run ./poc/device/agent -config custom-config.yaml

# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for specific packages
go test ./shared-lib/...
go test ./poc/device/agent/...
```

## Code Generation

API client/model code is generated from OpenAPI specs using oapi-codegen:

```bash
# Generate standard (Margo spec) SBI client/models
cd standard && ./generate.sh

# Generate non-standard NBI client/models
cd non-standard && ./generate.sh
```

Generated code lives in:
- `standard/generatedCode/wfm/sbi/` - Margo-compliant SBI models and client
- `non-standard/generatedCode/wfm/nbi/` - Non-standard NBI models and client

## Architecture

### Repository Structure

- **`shared-lib/`** - Reusable Go packages imported across the codebase:
  - `git/` - Git operations (clone, pull)
  - `file/` - HTTP file downloads
  - `crypto/` - TLS, payload signing with certificates
  - `http/` - HTTP client with auth utilities
  - `oci/` - Container image operations
  - `workloads/` - Helm and Docker Compose clients
  - `archive/` - tar.gz packing/unpacking
  - `cache/` - Bundle and deployment caching

- **`standard/`** - Margo specification API definitions and generated Go code

- **`non-standard/`** - Enabling APIs not defined by Margo but required for end-to-end workflows (e.g., repository metadata APIs)

- **`poc/device/agent/`** - Device agent implementation:
  - `main.go` - Bootstrap and lifecycle management
  - `database/database.go` - In-memory DB with persistence and event hooks
  - `onboarding.go` - Device registration and credential management
  - `stateSync.go` - Desired vs actual state reconciliation with WFM
  - `deployment.go` - Workload deploy/update/remove via runtime clients
  - `monitor.go` - Runtime state polling and status updates
  - `status.go` - Status reporting API calls to WFM
  - `device/capabilities.go` - Capability discovery and reporting

- **`poc/wfm/cli/`** - CLI wrapper for WFM API interactions

- **`pipeline/`** - Automation scripts for build, deployment, and operations

- **`deployments/`** - GitOps-like declarative deployment state (desired-state.yaml)

- **`scripts/`** - Utility scripts including GitOps sync tooling

- **`docker-compose/`** - Device agent Docker Compose deployment files

- **`helmchart/`** - Device agent Kubernetes/Helm deployment

### Device Agent Architecture

The device agent is event-driven: database writes emit events consumed by components to perform actions. Components use contexts for cancellation and support graceful shutdown.

**Supported Runtimes:**
- Kubernetes (Helm v3) - full lifecycle management for Helm charts
- Docker (Compose) - deploy/remove Compose projects

**Adding New Runtime Support:**
1. Implement workload interfaces in `shared-lib/workloads/`
2. Add runtime configuration to `poc/device/agent/types/config.go`
3. Update deployment manager to handle new runtime type
4. Add monitoring support in `monitor.go`
5. Register runtime in agent initialization

### Key Patterns

- **MVP Pattern** (Manager-Vendor-Provider): Eclipse Symphony architecture pattern where Managers implement business logic, Vendors expose APIs, and Providers bridge to external systems
- **Structured logging** with zap
- **Table-driven tests** with mocks
- **Structured errors** with context

## Environment Setup

### Pipeline Scripts

WFM setup (run on WFM VM):
```bash
source pipeline/wfm.env && sudo -E bash pipeline/wfm.sh
```

Device agent setup (run on device VM):
```bash
# For K3s device
source pipeline/device-agent_k3s.env && sudo -E bash pipeline/device-agent.sh

# For Docker device
source pipeline/device-agent_docker.env && sudo -E bash pipeline/device-agent.sh
```

Easy CLI for WFM operations:
```bash
source pipeline/wfm.env && sudo -E bash pipeline/wfm-cli.sh
```

### GitOps-like Deployments

Git-triggered deployment sync (works airgapped):
```bash
# One-time setup: install git hook
./scripts/install-hooks.sh

# Edit desired state
vim deployments/desired-state.yaml

# Commit triggers auto-sync to WFM
git add deployments/ && git commit -m "Deploy app"

# Or run manually
./scripts/sync-to-wfm.sh --dry-run  # Preview
./scripts/sync-to-wfm.sh            # Apply
```

### Required Environment Variables

```bash
export GITHUB_USER=<github-username>
export GITHUB_TOKEN=<github-personal-access-token>
export EXPOSED_HARBOR_IP=<wfm-machine-ip>
export EXPOSED_SYMPHONY_IP=<wfm-machine-ip>
export WFM_IP=<wfm-machine-ip>
```

## Service Ports

| Service | Port |
|---------|------|
| Harbor Registry | 8081 |
| Symphony API | 8082 |
| Prometheus | 30900 |
| Grafana | 32000 |
| Jaeger | 32500 |
| K3s API | 6443 |
| OTEL Collector | 30999 |

## Dependencies

- Go 1.24.4+
- Docker and Docker Compose V2
- Helm 3.15.1
- K3s (for Kubernetes deployments)
- Harbor (container registry)
