@RTK.md

# CLAUDE.md — docker-machine-driver-kubiqo

This file provides scaffolding context and conventions for AI-assisted development in this repository.

---

## Project Overview

Custom Exoscale driver for Rancher Machine (Docker Machine fork), forked from `rancher/machine/drivers/exoscale`. Renamed to driver name `"kubiqo"` and extended with private network attachment, template visibility filtering, and direct UUID-based template lookup. Used by Rancher to provision RKE2 node pool instances on Exoscale.

---

## Repository Structure

```text
docker-machine-driver-kubiqo/
├── driver/
│   └── exoscale-kubiqo.go   # Main driver implementation
├── main.go                  # Entry point
├── go.mod / go.sum          # Go module (replace directive maps to rancher/machine)
├── node-driver-manifest.yaml  # Rancher node driver registration manifest
├── cattle-system-netpol.yaml  # NetworkPolicy for cattle-system
├── fleet-default-netpol.yaml  # NetworkPolicy for fleet-default
└── fd.log                   # Debug log (not committed — git-ignored)
```

---

## Toolchain

- **Go** — driver implementation language
- **Rancher Machine** `v0.16.2` — forked Docker Machine (via `replace` directive in `go.mod`)
- **Exoscale v3 API** — instance provisioning and network attachment

---

## Key Customisations vs. Upstream

| Feature | Detail |
|---|---|
| Driver name | `"kubiqo"` (not `"exoscale"`) |
| Import base | `github.com/docker/machine` → mapped to `github.com/rancher/machine v0.16.2` |
| Private networks | `--exoscale-private-network` flag; attached **after** instance creation (Exoscale v3 API requirement) |
| Template visibility | `--exoscale-image-visibility public\|private`; `ListTemplates()` only returns public — use UUID for private |
| UUID lookup | If `--exoscale-image` matches UUID regex, calls `GetTemplate()` directly instead of name search |
| RPC compatibility | Removed `rpcdriver.GetDriverOpts()` (absent in rancher/machine fork); uses env var fallback only |

---

## Things to Watch Out For

- **Private network attachment is post-create** — Exoscale v3 API does not support attaching private networks during `CreateInstance`; the driver calls `AttachInstanceToPrivateNetwork()` after creation. Do not attempt to move this into the create call
- **`ListTemplates()` only returns public templates** — for private/custom templates (e.g. SLE Micro built by `exoscale-custom-template-builder`), always pass the UUID via `--exoscale-image`, not the name
- **Driver binary name must match** — Rancher expects the binary to be named `docker-machine-driver-kubiqo`; the node driver manifest (`node-driver-manifest.yaml`) references this exact name
- **`fd.log` is a debug artifact** — do not commit it; it is already in `.gitignore`
- **Upstream divergence** — this is a permanent fork; do not attempt to rebase onto upstream `rancher/machine` without verifying all 6 customisations still apply

---

## How to Help Effectively

- All driver logic is in `driver/exoscale-kubiqo.go` — start there for any feature or bug work
- When debugging provisioning failures, correlate with Rancher Machine logs and the Exoscale v3 API response, not just Go errors
- On a branch: write to `CHANGELOG-feat-<branch-purpose>.md`, `TROUBLESHOOTING-feat-<branch-purpose>.md`, and `README-feat-<branch-purpose>.md` — not the main files
- Do not auto-commit — always wait for explicit user instruction
