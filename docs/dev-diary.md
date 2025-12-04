# Developer's Diary: Understanding the Margo Ecosystem

##### `[Back To Main](../README.md)`

## About This Document

**Who should read this?**  
Anyone curious about engineering. Whether you're a new team member, a contributor, or just exploring the codebase, this diary is for you.

**What is this?**  
Think of this as a developer's journal. We've documented our journey through the Margo codebase, capturing confusions, and the clarifications. Our goal is to build a mental model of how everything came into existence and how they fit together.`

**Why does this exist?**  
Just wanted to extend a helping hand to those who just want to be in the shoes of the developer. If you find this helpful, great! If not, feel free to archive it. No hard feelings! : )

**Credits**
The development team. The Margo community. And AI(none specific) which helped us in creativity and rephrased some of our raw diary entries.

---

## Diary - Date: Aug-11:

So, we went through the official documentation and couple of discussions with the Margo members and got a better understanding of what are the expectations and what the project is about. We will try to put all of this into a self-made short story so that can also grasp it.

### Just a short story

Imagine you're a developer at a smart city company. Your city has thousands of traffic cameras, environmental sensors, and edge devices scattered across intersections, parks, and buildings. Each device needs different applications—some need AI models for traffic analysis, others need data aggregation tools, and some need security monitoring software.

Now, here's the challenge:
- **Application developers** (these are known as Workload/App Suppliers, please [refer margo official docs](https://specification.margo.org/personas-and-definitions/personas/)) create these specialized applications
- **Device manufacturers** (Device Suppliers) produce the hardware running at the edge
- **You** (the Workload Fleet Manager or WFM) need to orchestrate everything—deciding which apps run on which devices, managing updates, handling deployments across thousands of edge nodes

This is exactly the problem Margo solves. It's a standardized framework that bridges the gap between application suppliers, device suppliers, and fleet managers in edge computing scenarios.

### Before You Dive In: Prerequisites

To make the most of this journey, you should have a basic understanding of:

1. **Edge Computing Fundamentals**
   - What edge devices are (think IoT devices, edge servers, gateways)
   - Why we need edge computing (latency, bandwidth, privacy, offline operation)
   - The challenges of managing distributed edge infrastructure

2. **Margo's Official Documentation**
   - Start here: `[Margo GitHub](https://github.com/margo)`
   - Read the overview: `[Margo.org](https://margo.org/)`
   - Understand the problem space and Margo's vision
   - Go through its documentation as much as possible, as that's the official source.

3. **Technical Foundations**
   - HTTP/HTTPS and RESTful APIs
   - Container technologies (Docker, OCI)
   - Git and version control
   - YAML and configuration management
   - OpenAPI specifications (how to read and understand them)
   - And obviously any new technology you see in this project : )

**Pro tip**: Don't try to understand everything at once. Skim through the official docs first, then come back to specific sections as you need them.

One more thing to add here is that the [Eclipsey Symphony](https://github.com/eclipse-symphony/symphony) was already chosen as WFM by the committee members and the Margo test code is supposed to build on top of it.

---

## Diary - Date: Aug-29

Eclipse Symphony has its own development patterns. It is very extensible in nature and they call the design pattern as MVP, ie Manager, Vendor and Provider. This is similar to Business Logic, API Controller, and Third-party Interaction Handler. To extend symphony check this [directory](https://github.com/margo/symphony/tree/main/api/pkg/apis/v1alpha1) .


## Diary - Date: Aug-29

### What We Expected vs. What We Found

When we first looked at Margo's system design diagram (`[see here](https://github.com/margo/specification/blob/f31c8ad0879676437fb11185b661cda4ce25977c/system-design/overview/envisioned-system-design.md)`), we made an assumption that seemed logical:

**Our Initial Assumption:**  
"Margo defines the interface between Application Suppliers and Workload Fleet Managers, so it must have a complete API specification for how they communicate."

**The Reality:**  
Margo currently defines the **payload format** (Application Package and Application Description) but **not the transport mechanism**. In other words, Margo tells you *what* to exchange but leaves *how* to exchange it somewhat open-ended.

### The Two Competing Proposals

As of now, there are two main proposals for how Application Packages should be delivered to WFMs:

1. **Git-Based Application Registry**
   - Applications are stored in Git repositories
   - WFM clones/pulls from Git to get application packages
   - Familiar to developers, version control built-in

2. **OCI-Based Application Registry** (`[Proposal Link](https://github.com/margo/specification-enhancements/blob/28f04d64e8cedad8b82dad09840d0918bf6c699a/submitted/sup_app_registry_as_oci.md)`)
   - Applications are packaged as OCI artifacts (like Docker images)
   - WFM pulls from OCI registries
   - Better suited for binary artifacts and container ecosystems

**Our Choice:**  
At this point of time, we are going ahead with the **Git-based approach**. However, we are keeping a bit of extension in the codebase so that other type of approaches can also be introduced later on.

---

## Diary - Date: 11-Sep

### A Real-World Scenario

Let's make this concrete with a story. Before we dive into code, let's understand who does what in the Margo ecosystem.

**Meet Our Characters:**

**Viresh - The WFM Owner**  
Viresh runs a smart city platform called "CityEdge." His platform manages 10,000 edge devices across the city—traffic cameras, air quality sensors, smart streetlights, and parking meters. His job is to:
- Provide a marketplace where city departments can find and deploy applications
- Manage which applications run on which devices
- Handle updates, monitoring, and lifecycle management
- Ensure security and compliance

**Manjinder - The App Supplier**  
Manjinder is a software developer who created "EdgeCache Pro"—a lightweight, distributed caching application perfect for edge devices. It helps reduce latency by caching frequently accessed data locally. Manjinder wants to:
- Package his application in a standard format
- Publish it to WFM marketplaces
- Reach customers without worrying about device-specific details
- Get paid when people use his app

**Nitin - The WFM Platform User**  
Nitin works for the city's transportation department. He needs to deploy real-time traffic analysis on 500 traffic cameras. He wants to:
- Browse available applications on CityEdge marketplace
- Select EdgeCache Pro to improve response times
- Deploy it to his department's devices with a few clicks
- Monitor performance and manage costs

**Sanju - The Device Supplier**  
Sanju's company manufactures ruggedized edge computing devices designed for outdoor deployment. These devices:
- Run in harsh environments (extreme temperatures, weather)
- Have limited compute resources (ARM processors, 4GB RAM)
- Support multiple container runtimes (Docker, containerd)
- Need to be remotely manageable

### The Workflow: How It All Comes Together

Here's how our characters interact in the Margo ecosystem:

1. **Sanju** manufactures edge devices and registers their capabilities (CPU, memory, supported runtimes) with **Viresh's** CityEdge platform

2. **Manjinder** packages EdgeCache Pro as a Margo Application Package:
   - Creates `margo.yaml` (Application Description)
   - Defines deployment profiles (Helm charts, Docker Compose, etc.)
   - Specifies resource requirements and dependencies
   - Publishes to a Git repository or OCI registry (none of them as is standardized at this point of time)

3. **Viresh's** WFM platform:
   - Discovers Manjinder's application
   - Validates it against Margo specifications
   - Lists it in the CityEdge marketplace
   - Matches it with compatible devices from Sanju

4. **Nitin** logs into CityEdge:
   - Browses the marketplace
   - Selects EdgeCache Pro
   - Chooses target devices (his 500 traffic cameras)
   - Clicks "Deploy"

5. **Behind the scenes**, the WFM:
   - Pulls the application package
   - Transforms it into device-specific deployment artifacts
   - Pushes to the selected edge devices
   - Monitors deployment status

### Important Clarifications

See, we just have given an example of what WFM, devices, applications look like. The ownership or the access of these things can vary, and this is completely a business use case. For example, I might have a WFM product for my internal use case, where I am the app developer, the wfm owner, the device owner etc. But may be there could be a cloud based WFM offering that has business use case for different types of users, where an app developer could be somebody different than the app user etc. The WFM can have something like a Marketplace to host the application packages, furthermore and how they create this marketplace is completely a business case(for example, if my WFM is storing things in Harbor, I can call the Harbor as Marketplace :p , or may be I can have a complete different UI/UX and database etc for a marketplace). So, don't take this example in less-flexible way, this is added to make understanding easier for you. 

If we want to say things to the point:

**Ownership Models Can Vary:**
- Devices might be owned by the WFM (Viresh owns the infrastructure)
- Devices might be owned by users (Nitin owns his department's cameras)
- Devices might be provided by third-party device-as-a-service providers
- **Your business model might be completely different—and that's okay!**

**Application Licensing Can Vary:**
- Apps might be free and open-source
- Apps might be paid (subscription, per-device, per-deployment)
- Users might be the app developers themselves (deploying their own apps)
- There might be a Marketplace etc...
- **The example we use is just one scenario—adapt it to your needs!**

**Why This Matters:**  
Throughout this diary, we'll use Viresh, Manjinder, Nitin, and Sanju as reference points. : )

---

## Diary - TODO Add other entries here as well
(learning about device onboarding)
- single client multiple devices
(learning about device capabilities)
(learning about device signature)
(learning about how device agent is designed)
(learning about why symphony wasn't completely used and the problem we faced)
(learning about how others can quickly leverage the codebase to build something of their own)
(learning about repo-structure)
(learning about observability stack)
(learning about easy-cli and why it came into existence)
(learning about nbi, sbi interfaces of wfm)
(learning about the current standardized entities, deployment, application description and package, and how they can be made better)
(learning about local repository vs global repository -- helpful for wfm and device suppliers)