# Preface
The code in this repo is directly guided by the spec defined in the Margo Spec which can be found here.

## What is this SDK about?

This SDK provides Go-based building blocks for interacting with and implementing systems compliant with the Margo Workload Fleet Manager (WFM) specification. It offers:
- **Auto-generated API clients and models** for the Northbound interface
- **Package management abstractions** for onboarding and managing application packages
- **Other helpers** Authentication mechanisms, etc...

**Goal:**  
Enable consumers to easily build clients, CLIs, and server implementations for Margo-compliant orchestration platforms and devices.

---

## What does it aim to do?

- **Standardize**: Provide canonical Go models and client code for the Margo Spec.
- **Accelerate**: Help teams quickly build integrations, CLIs, and server stubs.
- **Modularize**: Offer reusable helpers for authentication, transport, and package management.
- **Document**: Serve as a reference for architecture and best practices for Margo-compliant systems.

---

## Prerequisites

- **Go 1.21+** (see go.mod)
- Familiarity with:
  - OpenAPI
  - Go modules and packages
  - Margo concepts, personas etc (please refer the official margo spec repo for this)
- Tools:  
  - [`oapi-codegen`](https://github.com/deepmap/oapi-codegen) for code generation (see generateNorthBound.sh)

---

## Folder Design Philosophy

- **wfm**:  
  - `northbound/`: OpenAPI spec, generated models, and client for the WFM Northbound API.
- **auth**:  
  - Pluggable authentication helpers (basic, bearer, no-auth, custom).
- **cli**:  
  - CLI-oriented wrappers for API clients (e.g., onboarding, listing packages).
- **pkg**:  
  - Core business models (application, package, registry).
  - Package management abstractions (sources, managers).
- **transport**:  
  - Protocol-agnostic transport layer (HTTP1, future HTTP2/3, etc.).
- **docs**:  
  - Architecture, design, and codebase navigation guides.

This structure follows Go best practices:  
- `internal/` for private code (not present yet)
- `pkg/` for reusable libraries
- `cmd/` for entry points (not present yet)

---

## How to Explore the SDK (Recommended Sequence)

1. **Read README.md** for a high-level overview.
2. **Understand the architecture** in design.md.
3. **Review the OpenAPI spec** (northbound.yaml) to see the API surface.
4. **Explore generated models and clients** in sdk/api/wfm/northbound/models and sdk/api/wfm/northbound/client.
5. **Check authentication helpers** in auth.
6. **Look at CLI helpers** in northbound.go.
7. **Dive into core models and package management** in models and packageManager.
8. **Review transport abstractions** in transport.
9. **Consult navigating-codebase.md** for deeper codebase navigation tips.

---

## Planned Improvements

- **Better error handling** and richer response types.
- **Documentation**: Usage guides, code samples, and API docs.
- **Server-side helpers** (middleware, validation) for implementers.
- **Test coverage and examples** for all major components.

---

## Known Shortcomings

- **No server implementation**: Only helpers, not a full server.
- **Some package management code is stubbed or commented out** (see sdk/pkg/packageManager/).
- **Limited test coverage** and examples.
- **Some models may not be fully aligned with evolving spec** (keep northbound.yaml in sync).
- **Authentication plugins** are basic; advanced flows (refresh, OAuth) are not yet implemented.
- **Transport layer** only supports HTTP/1.1 for now.

---

**Summary:**  
This SDK is a foundation for building Margo-compliant clients and servers in Go, with a focus on modularity, extensibility, and alignment with the Margo WFM spec. It is not a server, but a toolkit for implementers.

For more details, see design.md and navigating-codebase.md.