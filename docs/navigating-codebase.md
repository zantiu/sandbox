# Navigating the Codebase

A guide to understanding and working with this Go SDK.

## Table of Contents

- [Project Structure Overview](#project-structure-overview)
- [Getting Started](#getting-started)
- [How to Explore](#how-to-explore)
- [Best Practices](#best-practices)

---

## Project Structure Overview

- `docs/*` — Documentation and guides.
- `sdk/api/wfm/*` — OpenAPI spec, generated models, and clients for wfm.
- `sdk/cli/*` — CLI wrappers for API clients.
- `sdk/pkg/packageManager/*` — Package management abstractions.
- `sdk/auth/*` — Authentication helpers.
- `sdk/pkg/models/*` — Core business models.

## Getting Started

1. Read the [README.md](../README.md).
2. Review [design.md](./design.md) for architecture.
3. Explore the OpenAPI spec in `sdk/api/wfm/northbound.yaml`.
4. Check out generated models and clients in `sdk/api/wfm/northbound/`.

## How to Explore

- Start with models and API clients.
- Move to authentication and transport layers.
- Dive into package management abstractions.
- Use CLI helpers for quick testing.

## Best Practices

- Keep models in sync with the OpenAPI spec.
- Write tests for new helpers.
- Document new interfaces and plugins.