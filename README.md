# provider-busbar

A [Crossplane](https://crossplane.io) provider for **busbar**, the LLM gateway
admin API, generated with [Upjet](https://github.com/crossplane/upjet) from the
[`getbusbar/busbar`](https://github.com/GetBusbar/terraform-provider-busbar)
Terraform provider (`terraform-plugin-framework`, protocol 6).

It manages busbar governance objects as Kubernetes custom resources and
reconciles them through the busbar admin API using the upstream Terraform
provider as the runtime engine.

## Managed resources

| Kind         | API group                     | Terraform resource   | Purpose |
|--------------|-------------------------------|----------------------|---------|
| `VirtualKey` | `busbar.busbar.crossplane.io` | `busbar_virtual_key` | Mintable/revocable governance key with budget + rate caps. |
| `Hook`       | `busbar.busbar.crossplane.io` | `busbar_hook`        | Prompt/webhook hook attached to the gateway pipeline. |
| `Config`     | `busbar.busbar.crossplane.io` | `busbar_config`      | Versioned gateway configuration document. |

Both the cluster-scoped (`busbar.busbar.crossplane.io`) and namespaced
(`busbar.m.crossplane.io`) API groups are generated, following the current
Upjet v2 dual-scope provider layout.

The read-only `busbar_info` Terraform data source is intentionally not exposed:
Crossplane models managed (writable) resources, not data sources.

## Provider engine: Terraform CLI runtime

busbar's Terraform provider is built on the **terraform-plugin-framework**
(protocol 6). Upjet's native Go-schema bridge for framework resources requires
importing the provider's `fwprovider.Provider` object at code-generation time,
but busbar's provider factory lives in an `internal/` package and is not
importable by third parties. This provider therefore uses Upjet's
**CLI / native-runner runtime path**: resources are generated from the
provider's published schema (`config/schema.json`, produced by
`terraform providers schema -json`) and reconciled by driving the actual
`terraform-provider-busbar` binary at runtime. This path is agnostic to whether
the upstream provider uses SDKv2 or the plugin framework.

See `config/provider.go` for the wiring and `config/schema.json` for the schema
this build was generated from.

## Install

The package is published to GHCR. Latest is
`ghcr.io/getbusbar/provider-busbar:v0.1.1`, built against
`terraform-provider-busbar` v0.2.3 (busbar 1.5.3). `:latest` points at the same
image. The package is currently built for **linux/amd64 only**, so it needs an
amd64 node.

Apply it as a Crossplane `Provider`:

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-busbar
spec:
  package: ghcr.io/getbusbar/provider-busbar:v0.1.1
```

Or with the Crossplane CLI:

```bash
crossplane xpkg install provider ghcr.io/getbusbar/provider-busbar:v0.1.1
```

Then create a `ProviderConfig` referencing a `Secret` with your busbar admin
credentials:

```bash
kubectl apply -f examples/cluster/providerconfig/secret.yaml.tmpl   # edit first
kubectl apply -f examples/cluster/providerconfig/providerconfig.yaml
```

The credential JSON accepts: `endpoint` (required), `token` (required),
`client_cert_pem`, `client_key_pem`, `ca_cert_pem`, `insecure`.

## Usage

```bash
kubectl apply -f examples/cluster/busbar/virtualkey.yaml
kubectl apply -f examples/cluster/busbar/hook.yaml
kubectl apply -f examples/cluster/busbar/config.yaml
```

## Developing

The build uses the Crossplane `build` submodule (in `build/`) and Upjet codegen.

```bash
make submodules       # sync the build submodule
make generate         # regenerate CRDs + controllers from config/schema.json
make build            # build the provider binary
make run              # run the provider out-of-cluster
```

Regenerating the schema (when the upstream provider changes) is automatic:
`make generate` downloads the pinned `terraform-provider-busbar` release
(`TERRAFORM_PROVIDER_VERSION` in the `Makefile`) from GitHub into a local Terraform
filesystem mirror and runs `terraform providers schema -json` against it to
refresh `config/schema.json`. A filesystem mirror is used instead of the
public registry because Terraform 1.5.x cannot verify the Ed25519 GPG key the
registry release is signed with. To move to a newer upstream provider, bump
`TERRAFORM_PROVIDER_VERSION` and re-run `make generate`. Note that `make
generate` needs `goimports`, which upjet's generator shells out to by name; the
Makefile installs the go.mod-pinned version into `.cache/tools/<platform>` and
puts it on PATH, so no manual setup is required.

The `release-on-upstream` workflow does this automatically when a new
`terraform-provider-busbar` release appears: it re-pins, regenerates, commits,
and pushes this repo's next `v*` tag, which fires the package publish.

## CI status (honest)

- **CI** (`.github/workflows/ci.yml`): compiles, vets, and unit-tests the
  generated provider on every push/PR. No secrets needed; this is real
  coverage. It does **not** talk to a busbar gateway.
- **Publish Provider Package**
  (`.github/workflows/publish-provider-package.yml`): builds and pushes the
  `.xpkg` to `ghcr.io/getbusbar/provider-busbar` on tag push or manual
  dispatch, using the built-in `GITHUB_TOKEN` (no extra secrets).
- **End to End Testing** (`.github/workflows/e2e.yaml`, triggered by a
  `/test-examples` PR comment): **currently disarmed.** It requires repo
  secrets that are not provisioned, so a `check-secrets` job skips the whole
  pipeline with a loud `SKIPPED: secret ... not provisioned` warning instead
  of failing or pretending coverage. Nobody has run these e2e tests yet.

### End-to-end tests: what's needed to arm them

Provision these repository secrets (Settings → Secrets and variables →
Actions):

- `UPTEST_CLOUD_CREDENTIALS` — the credentials JSON `cluster/test/setup.sh`
  stores in the `provider-secret` Secret: busbar admin credentials of the form
  `{"endpoint": "https://<busbar-admin>:8081", "token": "<admin token>"}`
  (plus optional `client_cert_pem` / `client_key_pem` / `ca_cert_pem` /
  `insecure`). The referenced busbar gateway must be reachable from GitHub
  Actions runners with `governance:` enabled.
- `UPTEST_DATASOURCE` — a YAML file body for uptest's `--data-source`
  injection (see
  [crossplane/uptest](https://github.com/crossplane/uptest#injecting-dynamic-values-and-datasource)).
  May be effectively empty (`{}`) if the examples need no injected values, but
  the secret must exist.

Once both exist, commenting `/test-examples="examples/cluster/busbar"` on a PR
(as a user with write access) runs the pipeline for real.

## Publishing

To publish to the Upbound Marketplace / an OCI registry:

1. `make build` (multi-arch provider images).
2. `make xpkg.build` to build the Crossplane package (`.xpkg`).
3. Push with `up xpkg push` (or `crossplane xpkg push`) to
   `xpkg.upbound.io/getbusbar/provider-busbar:vX.Y.Z` or
   `ghcr.io/getbusbar/provider-busbar:vX.Y.Z`.

Required secrets in CI: a registry token (`XPKG_ACCESS_ID` /
`XPKG_TOKEN` for Upbound, or `GITHUB_TOKEN`/PAT for ghcr.io). The provider
image must also embed the `terraform-provider-busbar` binary in its filesystem
mirror (see `cluster/images/provider-busbar/`).

## License

Apache-2.0. See [LICENSE](LICENSE).
