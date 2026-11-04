package config

import (
	// Note: we import this to embed the provider schema document produced by
	// `terraform providers schema -json` for registry.terraform.io/getbusbar/busbar.
	_ "embed"

	ujconfig "github.com/crossplane/upjet/v2/pkg/config"

	busbarCluster "github.com/GetBusbar/provider-busbar/config/cluster/busbar"
	busbarNamespaced "github.com/GetBusbar/provider-busbar/config/namespaced/busbar"
)

const (
	resourcePrefix = "busbar"
	modulePath     = "github.com/GetBusbar/provider-busbar"
)

//go:embed schema.json
var providerSchema string

//go:embed provider-metadata.yaml
var providerMetadata string

// cliResources lists the framework-backed Terraform resources generated from
// the published schema and reconciled with the Terraform CLI runtime.
var cliResources = []string{
	"busbar_virtual_key$",
	"busbar_hook$",
	"busbar_config$",
}

// GetProvider returns the cluster-scoped provider configuration.
func GetProvider() *ujconfig.Provider {
	pc := ujconfig.NewProvider([]byte(providerSchema), resourcePrefix, modulePath, []byte(providerMetadata),
		ujconfig.WithRootGroup("busbar.crossplane.io"),
		// CLI-runtime path: resources are configured by name and generated from
		// schema.json; the actual terraform-provider-busbar binary is driven at
		// runtime via the shared Terraform CLI/native runner.
		ujconfig.WithIncludeList(cliResources),
		ujconfig.WithFeaturesPackage("internal/features"),
		ujconfig.WithDefaultResourceOptions(
			ExternalNameConfigurations(),
		))

	for _, configure := range []func(provider *ujconfig.Provider){
		busbarCluster.Configure,
	} {
		configure(pc)
	}

	pc.ConfigureResources()
	return pc
}

// GetProviderNamespaced returns the namespaced provider configuration.
func GetProviderNamespaced() *ujconfig.Provider {
	pc := ujconfig.NewProvider([]byte(providerSchema), resourcePrefix, modulePath, []byte(providerMetadata),
		ujconfig.WithRootGroup("busbar.m.crossplane.io"),
		// CLI-runtime path: resources are configured by name and generated from
		// schema.json; the actual terraform-provider-busbar binary is driven at
		// runtime via the shared Terraform CLI/native runner.
		ujconfig.WithIncludeList(cliResources),
		ujconfig.WithFeaturesPackage("internal/features"),
		ujconfig.WithDefaultResourceOptions(
			ExternalNameConfigurations(),
		),
		ujconfig.WithExampleManifestConfiguration(ujconfig.ExampleManifestConfiguration{
			ManagedResourceNamespace: "crossplane-system",
		}))

	for _, configure := range []func(provider *ujconfig.Provider){
		busbarNamespaced.Configure,
	} {
		configure(pc)
	}

	pc.ConfigureResources()
	return pc
}
