---
marp: true
theme: default
paginate: true
---

# Margo Dev Sprint-1 Demo

---

# What We Built

## 🎯 Sprint-1 Achievements

- (Workload Supplier -> WFM API) for AppPkg
- Comprehensive SDK library for reusability
- Symphony Extension to support (Workload Supplier -> WFM) compliance
- Pipeline Infrastructure (CI/CD)

---

# 1. WorkloadSupplier to WFM(Northbound) API

## 📋 OpenAPI Specification

- ✅ Created standardized API contract for app-pkgs
- ✅ This becomes the foundation for standardized payloads
- ✅ And foundation for margo compliant client
- ✅ Generated the client stub using [oapi-codegen](github.com/deepmap/oapi-codegen/cmd/oapi-codegen)

---

# 2. SDK Development

## 🛠️ Core Components

- **Client Wrapper** - A cli wrapper to easily integration with Margo compliance WFM and Devices.
- **Helper Packages** - Git interactions, OCI interactions, and others.

### Philosophy: *"Any code/logic that is of value for other Adopters of Margo, goes into the SDK, except the server stub"*

---

# 3. Symphony Integration

## 🎼 Foundation

- ✅ Forked Eclipse Symphony to Margo [repo](https://github.com/margo/symphony)
- ✅ Added the Margo APIs for app-pkg management

---

# 4. Infrastructure

## 🚀 Development Pipeline

- ✅ Used Github Actions.
- ✅ Deployment of symphony server on Azure via this pipeline.

---

# What's Next - Soon

## 🔮 Sprint-2 Priorities

- ⬇️ (Device -> WFM API) design and implementation. 
- 🔐 Security mechanism (Bearer based auth)
- 🏠 Local Git (Gogs or Gitea) and OCI registry (Harbor) for testing
- 📈 Enhanced SDK

---

# What's Next - Future

## 🚀 Strategic Initiatives - Slide 1

- 🤔 Need clarity on the undefined margo interfaces like Device Onboarding, OCI Image credentials etc.
- 🤝 Move client implementations to SDK, so that any adopter can directly get the cli out-of the sdk box.
- #️⃣ Manifest hash system, for state-seeking.
- 🔑 mTLS?
- 🤖 Gitops Pattern?

---

## 🚀 Strategic Initiatives - Slide 2
- 📖 Symphony extension guides, for other devs to ramp-up with the learning curve.
- ⬇️ WFM to device APIs (probably)
- 🤖 Device simulator (probably)
- 📚 Better documentation

---

# Source Code

- SDK : https://github.com/margo/dev-repo
- Symphony : https://github.com/margo/symphony

---

# Thank You

## Questions & Discussion

---