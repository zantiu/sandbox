# Workflows

## How this project maps to the Margo spec

- **Workload Supplier/Application:** Provides application packages to the WFM.
- **Fleet Manager Supplier/Workload Fleet Manager:** Orchestrates and provisions workloads on devices.
- **Device Supplier/Device:** End devices where workloads are deployed.

## Typical Usage

1. **Onboard an Application Package**
   - Use the API client to submit a new app package (from Git or other sources).
2. **List Application Packages**
   - Query the WFM for available packages.
3. **Delete Application Package**
   - Remove a package from the WFM.
4. **Authentication**
   - Use provided plugins for basic, no-auth, or custom authentication.

## Extending Workflows
- NOTE: Not considered for now 
- Add new package sources by implementing the `PackageSource` interface.
- Support new authentication flows as needed.

---
# Symphony CLI and Margo SDK integration workflow

```mermaid
graph TB
    subgraph "Margo SDK"
        A[WFM API Spec Definition]
        B[WFM API Client Code]
        C[Common Utilities]
        C1[Git Repository Utils]
        C2[Package Parser]
        C3[Package Validator]
        
        C --> C1
        C --> C2
        C --> C3
    end
    
    subgraph "Symphony CLI Tool"
        G[CLI Commands]
        H[SDK Client Integration]
    end
    
    subgraph "External Sources"
        J[Git Repositories]
        K[Package Files]
    end
    
    %% CLI uses SDK client code
    B -->|used by| H
    
    %% CLI workflow
    G -->|triggers| H
    
    %% External source interactions through utilities
    C1 -->|pulls from| J
    C2 & C3 -->|processes| K
    
    %% Styling
    classDef sdk fill:#e3f2fd,stroke:#1976d2,stroke-width:2px
    classDef cli fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px
    classDef external fill:#e8f5e8,stroke:#388e3c,stroke-width:2px
    
    class A,B,C,C1,C2,C3 sdk
    class G,H cli
    class J,K external
```

# Symphony Server and Margo SDK integration workflow

```mermaid
graph TB
    subgraph "Margo SDK"
        A[WFM API Spec Definition]
        B[WFM API Client Code]
        C[Common Utilities]
        C1[Git Repository Utils]
        C2[Package Parser]
        C3[Package Validator]
        
        C --> C1
        C --> C2
        C --> C3
    end
    
    subgraph "Symphony Server"
        D[WFM API Implementation]
        E[Server Business Logic]
        F[Package Processing Engine]
    end
    
    subgraph "External Sources"
        J[Git Repositories]
        K[Package Files]
    end
    
    %% SDK provides spec definition
    A -.->|defines contract for| D
    
    %% Server uses common utilities
    C -->|shared utilities| F
    
    %% Server workflow
    D -->|processes requests| E
    E -->|handles packages| F
    
    %% External source interactions through utilities
    C1 -->|pulls from| J
    C2 & C3 -->|processes| K
    
    %% Server operations with external sources
    F -->|uses utilities for| J
    F -->|uses utilities for| K
    
    %% Styling
    classDef sdk fill:#e3f2fd,stroke:#1976d2,stroke-width:2px
    classDef server fill:#fff3e0,stroke:#f57c00,stroke-width:2px
    classDef external fill:#e8f5e8,stroke:#388e3c,stroke-width:2px
    
    class A,C,C1,C2,C3 sdk
    class D,E,F server
    class J,K external
```