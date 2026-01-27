# walk.io

**walk.io** is a lightweight platform for running untrusted or isolated workloads on a single Linux server using **Firecracker microVMs**.

It aims to bridge the gap between:

* container-style developer experience (OCI images, simple builds), and
* VM-level isolation (real kernel, hardware virtualization),

without the complexity of Kubernetes or a full multi-node cloud.

You can think of it as a **single-node, opinionated, simpler Fly.io**.

---

## Goals

* Run arbitrary OCI images inside Firecracker microVMs
* Strong isolation via microVMs instead of containers
* Fast startup suitable for FaaS-like workloads
* Simple, inspectable architecture
* Single-server first (no cluster assumptions)

Non-goals (for now):

* Multi-node scheduling
* Full Kubernetes compatibility
* General-purpose container runtime replacement

---

## High-level architecture

```
OCI Image
   ↓
Image Builder
(OCI → ext4 rootfs)
   ↓
Firecracker Runtime
(microVM per workload)
   ↓
Network / API / Activator
(request-driven lifecycle)
```

Key ideas:

* OCI images are **materialized into ext4 root disks**
* Each microVM boots with:
  * a minimal kernel
  * an immutable root filesystem + app filesystem
  * a statefull disk
* Runtime configuration (`argv`, `env`, `workdir`) is injected explicitly
* MicroVM lifecycle is controlled by a small orchestrator

---

## Image model

walk.io does **not** run containers inside VMs.

Instead:

1. An OCI image is pulled
2. Its filesystem layers are unpacked
3. A new ext4 disk image is created:

   ```
   <digest>-<id>-app.ext4
   ```

4. Runtime metadata is injected into the filesystem:

   ```
   /walk/argv
   /walk/env
   WORKDIR (as env)
   ```

5. Firecracker boots directly into the workload

---

## Why Firecracker?

* Hardware virtualization (KVM)
* Very small device model
* Fast boot times
* Strong isolation boundaries

Compared to containers:

* No shared kernel attack surface
* Better fit for running untrusted or user-submitted code

---

## Current status

🚧 **Early development / research phase**

What exists:

* Building of ext4 block device from OCI Image (pulling from registry)

What’s coming next:

* vm controller

---

## Requirements

Host system:

* Linux
* KVM enabled
* Firecracker
* `mkfs.ext4` (e2fsprogs)
* root or `CAP_SYS_ADMIN`

Go:

* Go 1.22+

---

## Inspiration

* Fly.io
* Firecracker-containerd
* Kata Containers
* AWS Lambda (execution model, not architecture)
