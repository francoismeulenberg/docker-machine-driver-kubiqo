# docker-machine-driver-kubiqo

- Problem: `go mod tidy` failed because `github.com/docker/docker/pkg/term` was removed from newer docker releases, but `github.com/docker/machine/libmachine/ssh` still imports it.
- Fixes applied:
	- Added replace to pin docker engine module: `replace github.com/docker/docker => github.com/docker/docker v20.10.7+incompatible`.
	- Switched all `libmachine` imports from `github.com/rancher/machine/...` to `github.com/docker/machine/...` to satisfy driver interfaces.
	- Removed dependency on `drivers/rpc` helper; manually reloaded `EXOSCALE_API_KEY` / `EXOSCALE_API_SECRET_KEY` from env vars and `--exoscale-api-key` / `--exoscale-api-secret-key` args inside `UnmarshalJSON`.
	- Dropped `github.com/rancher/machine` from go.mod; kept `github.com/docker/machine v0.16.2`.
	- Added driver entrypoint (`main.go`) that registers the driver with the Machine plugin server.
	- Ran `gofmt` on touched files.

## Build the driver binary

From repo root (`node_driver_dev`):

```sh
go mod tidy
go test ./...
go build -o dist/docker-machine-driver-exoscale .
```

The resulting binary `dist/docker-machine-driver-exoscale` must be on the Rancher host `PATH` (or copied into the directory Rancher scans). Example:

```sh
chmod +x dist/docker-machine-driver-exoscale
sudo cp dist/docker-machine-driver-exoscale /usr/local/bin/
```

Quick sanity:

```sh
dist/docker-machine-driver-exoscale --help
```

## Optional package-level test

```sh
cd driver
go test .
```
