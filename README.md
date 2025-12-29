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
- **Reason:** Custom templates with different disk sizes would be excluded by this filter

### 8. Enhanced Debug Logging

- **Added:** Visibility breakdown logging to troubleshoot template discovery
- **Added:** UUID detection logging
- **Added:** Private network attachment status logging
- **Code:** Lines 619-625, 608, 851-854 in [driver/exoscale-kubiqo.go](driver/exoscale-kubiqo.go)

### 9. Flag Naming for Rancher Integration

- **Problem:** Rancher passes flags with driver name as prefix (e.g., `--kubiqo-availability-zone`)
- **Original:** All flags used `exoscale-` prefix (e.g., `--exoscale-availability-zone`)
- **Changed:** All flag names updated from `exoscale-*` to `kubiqo-*` to match driver name
- **Affected Flags:** `kubiqo-url`, `kubiqo-api-key`, `kubiqo-api-secret-key`, `kubiqo-instance-profile`, `kubiqo-disk-size`, `kubiqo-image`, `kubiqo-image-visibility`, `kubiqo-security-group`, `kubiqo-availability-zone`, `kubiqo-ssh-user`, `kubiqo-ssh-key`, `kubiqo-userdata`, `kubiqo-affinity-group`, `kubiqo-private-network`, `kubiqo-use-instance-pool`, `kubiqo-instance-pool-size`
- **Environment Variables:** Still use `EXOSCALE_*` for CLI compatibility (e.g., `EXOSCALE_API_KEY`)
- **Code Location:** Lines 88-167, 256-272 in [driver/exoscale-kubiqo.go](driver/exoscale-kubiqo.go)

### 10. Rancher Credential Environment Variable Support

- **Problem:** Rancher sets credentials via environment variables, not flags
- **Implementation:** Added environment variable fallback in both `UnmarshalJSON()` and `SetConfigFromFlags()`
- **Credential Variables Checked:**
  1. `EXOSCALE_API_KEY` / `EXOSCALE_API_SECRET_KEY` (CLI/direct usage)
  2. `KUBIQO_API_KEY` / `KUBIQO_API_SECRET_KEY` (Rancher format)
- **Reason:** Rancher injects credentials as `KUBIQO_API_KEY` environment variables based on NodeDriver credential field mappings
- **Code Location:** Lines 229-248 (UnmarshalJSON), Lines 273-290 (SetConfigFromFlags) in [driver/exoscale-kubiqo.go](driver/exoscale-kubiqo.go)

### 11. Binary Size Optimization

- **Problem:** Original build was 17MB with debug symbols, causing download timeouts in Rancher
- **Solution:** Build with stripped symbols using `-ldflags="-s -w"`
- **Result:** Reduced from 17MB to 12MB (30% reduction)
- **Build Command:**

  ```sh
  go build -ldflags="-s -w" -o dist/docker-machine-driver-kubiqo main.go
  ```

- **Impact:** Faster downloads, fewer timeout errors during machine provisioning

### 12. Instance Pool Support

- **Added:** Ability to create and manage instance pools instead of single instances
- **New Fields:** `UseInstancePool`, `InstancePoolID`, `InstancePoolSize` in Driver struct
- **New Flags:**
  - `--kubiqo-use-instance-pool` (BoolFlag): Enable instance pool mode
  - `--kubiqo-instance-pool-size` (IntFlag): Number of instances in the pool (default: 1)
- **Implementation:**
  - When enabled, creates an instance pool with specified size instead of a single instance
  - The first instance in the pool becomes the primary machine for docker-machine operations
  - Pool inherits all configurations: template, instance type, security groups, anti-affinity groups, private networks, SSH keys, user-data
  - Deleting the machine removes the entire instance pool
- **Use Cases:**
  - High availability clusters with multiple identical nodes
  - Simplified scaling for homogeneous workloads
  - Centralized management of multiple instances as a single unit
- **Environment Variables:** `EXOSCALE_USE_INSTANCE_POOL`, `EXOSCALE_INSTANCE_POOL_SIZE`

## Build and Install

### Standard Build (with debug symbols)

Build from the module directory:

```bash
go build -o dist/docker-machine-driver-kubiqo .
```

### Production Build (optimized, stripped symbols)

For deployment in Rancher or bandwidth-constrained environments:

```bash
go build -ldflags="-s -w" -o dist/docker-machine-driver-kubiqo main.go
```

The `-ldflags="-s -w"` flags strip debug symbols, reducing binary size from ~17MB to ~12MB.

Calculate checksum for Rancher NodeDriver manifest:

```bash
sha256sum dist/docker-machine-driver-kubiqo
```

Install the driver:

```bash
sudo cp dist/docker-machine-driver-kubiqo /usr/local/bin/
chmod +x /usr/local/bin/docker-machine-driver-kubiqo
```

Verify installation:

```bash
docker-machine create --driver kubiqo --help
```

## Usage

### With Public Templates (by name)

```bash
docker-machine create -d kubiqo \
  --kubiqo-api-key <key> \
  --kubiqo-api-secret-key <secret> \
  --kubiqo-availability-zone "de-fra-1" \
  --kubiqo-image "Linux Ubuntu 24.04 LTS 64-bit" \
  my-machine
```

### With Private Templates (by UUID)

```bash
docker-machine create -d kubiqo \
  --kubiqo-api-key <key> \
  --kubiqo-api-secret-key <secret> \
  --kubiqo-availability-zone "de-fra-1" \
  --kubiqo-image <template-uuid> \
  my-machine
```

### With Private Networks

```bash
docker-machine create -d kubiqo \
  --kubiqo-api-key <key> \
  --kubiqo-api-secret-key <secret> \
  --kubiqo-availability-zone "de-fra-1" \
  --kubiqo-private-network "my-network-1" \
  --kubiqo-private-network "my-network-2" \
  --kubiqo-image <template-uuid> \
  my-machine
```

### With Instance Pool

Create a pool of identical instances managed as a single unit:

```bash
docker-machine create -d kubiqo \
  --kubiqo-api-key <key> \
  --kubiqo-api-secret-key <secret> \
  --kubiqo-availability-zone "de-fra-1" \
  --kubiqo-use-instance-pool \
  --kubiqo-instance-pool-size 3 \
  --kubiqo-image "Linux Ubuntu 24.04 LTS 64-bit" \
  my-cluster
```

**Notes:**

- The pool name will be the machine name (`my-cluster` in this example)
- The first instance in the pool becomes the primary machine for docker-machine operations
- All instances share the same configuration (template, instance type, networks, security groups)
- Removing the machine with `docker-machine rm my-cluster` deletes the entire pool
- Instance pool size can be set via `--kubiqo-instance-pool-size` flag or `EXOSCALE_INSTANCE_POOL_SIZE` environment variable

2. **Custom cloud-init** (`userdata.yaml`):

   ```yaml
   #cloud-config
   packages:
     - docker
   runcmd:
     - systemctl enable --now docker
     - usermod -aG docker root
   ```

   ```bash
   docker-machine create -d kubiqo \
     --kubiqo-userdata /path/to/userdata.yaml \
     ...
   ```

## Rancher Integration

### NodeDriver Manifest

Create a NodeDriver resource for Rancher:

```yaml
apiVersion: management.cattle.io/v3
kind: NodeDriver
metadata:
  name: kubiqo
  annotations:
    privateCredentialFields: apiSecretKey
    publicCredentialFields: apiKey
spec:
  active: true
  builtin: false
  checksum: <sha256-checksum-of-binary>
  displayName: kubiqo
  url: https://<your-rancher-url>/assets/docker-machine-driver-kubiqo
  whitelistDomains:
  - api.exoscale.ch
```

### Deployment Steps

1. **Build the driver** (with stripped symbols for smaller size):

   ```bash
   go build -ldflags="-s -w" -o dist/docker-machine-driver-kubiqo main.go
   ```

2. **Calculate checksum**:

   ```bash
   sha256sum dist/docker-machine-driver-kubiqo
   ```

3. **Upload to Rancher**:
   - Upload binary to `https://<rancher-url>/assets/docker-machine-driver-kubiqo`
   - Ensure it's accessible from cluster pods

4. **Update manifest** with the checksum and apply:

   ```bash
   kubectl apply -f node-driver-manifest.yaml
   ```

5. **Create Cloud Credentials** in Rancher:
   - Navigate to Cluster Management → Cloud Credentials
   - Create new credential with:
     - `apiKey`: Your Exoscale API key
     - `apiSecretKey`: Your Exoscale API secret

6. **Provision Cluster**: Use the kubiqo driver with your cloud credentials

### Known Issues & Solutions

#### Download Timeouts

**Problem:** Machine provisioning pods timeout downloading the driver binary (curl error 28)

**Cause:**

- Binary size (12MB even when stripped)
- Nginx ingress `proxy-connect-timeout: 30s` may be insufficient for slower connections

**Solutions:**

1. Increase nginx ingress timeout in Rancher ingress annotations:

   ```yaml
   nginx.ingress.kubernetes.io/proxy-connect-timeout: "120"
   ```

2. Use CDN or faster hosting (e.g., GitHub Releases)

3. Further optimize binary size with UPX compression (advanced)

#### Credential Errors

**Problem:** `error setting machine configuration: missing an API key`

**Cause:** Environment variables not being read correctly

**Solution:** Ensure both `UnmarshalJSON()` and `SetConfigFromFlags()` check environment variables. The driver now checks:

- `EXOSCALE_API_KEY` / `EXOSCALE_API_SECRET_KEY` (CLI usage)
- `KUBIQO_API_KEY` / `KUBIQO_API_SECRET_KEY` (Rancher injection)

#### Flag Naming Mismatch

**Problem:** `flag provided but not defined: -kubiqo-availability-zone`

**Cause:** Rancher prepends driver name to flags, but driver expected `--exoscale-*` flags

**Solution:** All flags renamed from `exoscale-*` to `kubiqo-*` prefix

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
- Flag names use `kubiqo-` prefix for Rancher compatibility
- Environment variables still use `EXOSCALE_` prefix for CLI compatibility
- Binary should be built with stripped symbols (`-ldflags="-s -w"`) for production use
- Instance pool support available via `--kubiqo-use-instance-pool` flag for managing multiple identical instances

## Troubleshooting

### Check Driver Download from Pod

```bash
kubectl run test-download --rm -it --image=curlimages/curl -- \
  curl -v -o /tmp/driver https://<rancher-url>/assets/docker-machine-driver-kubiqo
```

### Verify Credentials in Machine Pod

```bash
kubectl get secret <machine-name>-machine-driver-secret -n fleet-default -o yaml
```

Should contain:

- `KUBIQO_API_KEY`
- `KUBIQO_API_SECRET_KEY`

### Check Machine Provisioning Logs

```bash
kubectl logs -n fleet-default <machine-pod-name> -c machine
```

Common errors:

- `Curl failed with error code 28`: Download timeout (increase ingress timeout)
- `flag provided but not defined`: Flag naming mismatch (ensure using kubiqo- prefix)
- `missing an API key`: Credential environment variables not set/read correctly
