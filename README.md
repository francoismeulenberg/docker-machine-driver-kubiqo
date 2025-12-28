# docker-machine-driver-kubiqo

A customized Exoscale driver for Docker Machine / Rancher Machine, forked from [rancher/machine/drivers/exoscale](https://github.com/rancher/machine/blob/master/drivers/exoscale/exoscale.go).

## Changes from Original Rancher Exoscale Driver

### 1. Driver Naming
- **Package:** Renamed from `package exoscale` to `package kubiqo`
- **Driver Name:** Changed `DriverName()` to return `"kubiqo"` instead of `"exoscale"`
- **Binary:** Built as `docker-machine-driver-kubiqo` instead of `docker-machine-driver-exoscale`

### 2. Import Path Standardization
- **Before:** Mixed imports from `github.com/rancher/machine/libmachine/...` and `github.com/docker/machine/libmachine/...`
- **After:** All imports use `github.com/docker/machine/libmachine/...` for type consistency
- **Reason:** Prevents interface type mismatches (e.g., `GetCreateFlags` signature differences)
- **Implementation:** `go.mod` uses `replace` directive to map to Rancher fork:
  ```go
  replace github.com/docker/machine => github.com/rancher/machine v0.16.2
  ```

### 3. Private Network Support
- **Added:** `PrivateNetworks []string` field to Driver struct
- **Added:** `--exoscale-private-network` flag (StringSlice) for specifying private networks by name
- **Implementation:** 
  - Lookup private networks by name and resolve to IDs during instance creation
  - Attach networks **after** instance creation using `client.AttachInstanceToPrivateNetwork()`
  - Exoscale v3 API doesn't support private networks during `CreateInstance`, requires separate attachment calls
- **Code Location:** Lines 800-875 in [driver/exoscale-kubiqo.go](driver/exoscale-kubiqo.go)

### 4. Template Visibility Support
- **Added:** `ImageVisibility string` field to Driver struct
- **Added:** `--exoscale-image-visibility` flag with values `public` (default) or `private`
- **Added:** `TemplateVisibility` type with constants `TemplateVisibilityPublic` and `TemplateVisibilityPrivate`
- **Implementation:**
  - Filter templates by visibility when searching by name (lines 622-638)
  - **Limitation:** Exoscale v3 `ListTemplates()` only returns public templates
  - **Workaround:** Use template UUID directly for private/custom templates

### 5. Direct Template UUID Lookup
- **Added:** UUID-based template retrieval for private/custom templates
- **Implementation:** 
  - Detects if `--exoscale-image` is a UUID format (regex: `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
  - If UUID detected, calls `client.GetTemplate(ctx, v3.UUID(image))` directly
  - If not UUID, falls back to name-based search through `ListTemplates()`
- **Reason:** Private/custom templates aren't returned by `ListTemplates()`, must be fetched by ID
- **Code Location:** Lines 607-617 in [driver/exoscale-kubiqo.go](driver/exoscale-kubiqo.go)

### 6. RPC Driver Compatibility Fix
- **Problem:** Original code used `rpcdriver.GetDriverOpts()` which doesn't exist in rancher/machine fork
- **Solution:** Removed RPC-specific credential reloading from `UnmarshalJSON()`
- **New Approach:** Simplified to only check environment variables:
  ```go
  if d.APIKey == "" {
      if apiKey := os.Getenv("EXOSCALE_API_KEY"); apiKey != "" {
          d.APIKey = apiKey
      }
  }
  ```
- **Effect:** Works with both direct CLI usage and Rancher's credential management

### 7. Removed Disk Size Filter
- **Original:** Template search filtered to only 10GB images: `if tpl.Size>>30 != 10 { continue }`
- **Changed:** Filter remains in name-based search but bypassed when using UUID lookup
- **Reason:** Custom templates (like SLE Micro 6.1 with 32GB disk) would be excluded by this filter

### 8. Enhanced Debug Logging
- **Added:** Visibility breakdown logging to troubleshoot template discovery
- **Added:** UUID detection logging
- **Added:** Private network attachment status logging
- **Code:** Lines 619-625, 608, 851-854 in [driver/exoscale-kubiqo.go](driver/exoscale-kubiqo.go)

## Build and Install

Build from the module directory:

```sh
go build -o dist/docker-machine-driver-kubiqo .
```

Install the driver:

```sh
sudo cp dist/docker-machine-driver-kubiqo /usr/local/bin/
chmod +x /usr/local/bin/docker-machine-driver-kubiqo
```

Verify installation:

```sh
docker-machine create --driver kubiqo --help
```

## Usage

### With Public Templates (by name)
```sh
docker-machine create -d kubiqo \
  --exoscale-api-key <key> \
  --exoscale-api-secret-key <secret> \
  --exoscale-availability-zone "de-fra-1" \
  --exoscale-image "linux-ubuntu-24.04-lts-64-bit" \
  my-machine
```

### With Private Templates (by UUID)
```sh
docker-machine create -d kubiqo \
  --exoscale-api-key <key> \
  --exoscale-api-secret-key <secret> \
  --exoscale-availability-zone "de-fra-1" \
  --exoscale-image "d0d80630-3d41-443b-b954-b2c0c022dc8a" \
  my-machine
```

### With Private Networks
```sh
docker-machine create -d kubiqo \
  --exoscale-api-key <key> \
  --exoscale-api-secret-key <secret> \
  --exoscale-availability-zone "de-fra-1" \
  --exoscale-private-network "my-private-network" \
  --exoscale-image "d0d80630-3d41-443b-b954-b2c0c022dc8a" \
  my-machine
```

### For SLE Micro 6.x Images

SLE Micro uses transactional updates and may require custom Docker installation. Options:

1. **Pre-installed Docker image** (recommended):
   ```sh
   docker-machine create -d kubiqo \
     --exoscale-image "<sle-micro-template-uuid>" \
     --engine-install-url "none" \
     ...
   ```

2. **Custom cloud-init** (`userdata.yaml`):
   ```yaml
   #cloud-config
   packages:
     - docker
   runcmd:
     - systemctl enable --now docker
     - usermod -aG docker root
   ```
   ```sh
   docker-machine create -d kubiqo \
     --exoscale-userdata /path/to/userdata.yaml \
     ...
   ```

## Dependencies

```go
require (
    github.com/docker/machine v0.16.2
    github.com/exoscale/egoscale/v3 v3.1.31
)

replace (
    github.com/docker/machine => github.com/rancher/machine v0.16.2
    github.com/docker/docker => github.com/docker/docker v20.10.7+incompatible
)
```

## Notes

- All changes are isolated to this module
- Compatible with Rancher's Docker Machine fork (v0.16.2)
- Uses Exoscale egoscale/v3 SDK (v3.1.31)
- Private templates must be specified by UUID, not by name
