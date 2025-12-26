# docker-machine-driver-kubiqo

This module provides the Exoscale driver for Docker Machine-compatible tooling (used by Rancher). Since `docker/machine` is archived, we use the Rancher-maintained fork while keeping imports consistent to avoid type mismatches.

## What was changed
- Problem: `go mod tidy` failed because `github.com/docker/docker/pkg/term` was removed from newer docker releases, but `libmachine/ssh` still imports it. We also had mixed imports between `rancher/machine` and `docker/machine`, causing interface type mismatches (e.g., `GetCreateFlags`).
- Fixes applied:
	- Standardized imports to `github.com/docker/machine/libmachine/...` across the module for a single type universe.
	- Mapped the implementation to Rancher’s fork via `go.mod`:

		```go
		replace github.com/docker/machine => github.com/rancher/machine v0.16.2
		replace github.com/docker/docker => github.com/docker/docker v20.10.7+incompatible

		require (
				github.com/docker/machine v0.16.2
				github.com/exoscale/egoscale/v3 v3.1.31
				github.com/stretchr/testify v1.11.1
		)
		```

	- Kept `egoscale/v3` for Exoscale API calls.
	- Added a driver entrypoint in [infrastructure/automation/docker-machine-driver-kubiqo/main.go](infrastructure/automation/docker-machine-driver-kubiqo/main.go) registering the driver plugin.
	- In `UnmarshalJSON`, we reload `EXOSCALE_API_KEY` / `EXOSCALE_API_SECRET_KEY` and `--exoscale-api-key` / `--exoscale-api-secret-key` from env/args to align with RPC driver behavior.

## Build and Test
Run these from the module directory [infrastructure/automation/docker-machine-driver-kubiqo](infrastructure/automation/docker-machine-driver-kubiqo):

```sh
go mod tidy
go build ./...
go test ./...
```

Build a binary:

```sh
go build -o dist/docker-machine-driver-exoscale .
chmod +x dist/docker-machine-driver-exoscale
```

Install the binary on a Rancher host `PATH`:

```sh
sudo cp dist/docker-machine-driver-exoscale /usr/local/bin/
```

Quick sanity:

```sh
dist/docker-machine-driver-exoscale --help
```

## Scope
All changes are isolated to this module. Other projects in the workspace remain unaffected.
