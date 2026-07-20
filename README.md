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

```bash
kubectl crossplane install provider ghcr.io/getbusbar/provider-busbar:v0.1.0
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

Regenerating the schema (when the upstream provider changes) requires the
`getbusbar/busbar` provider to be resolvable by `terraform providers schema`.
Because the provider is not yet on a public registry, `config/schema.json` in
this repo was produced from a local build of `terraform-provider-busbar` v0.1.0
via a filesystem mirror.

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
