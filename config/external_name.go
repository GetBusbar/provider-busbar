package config

import (
	"github.com/crossplane/upjet/v2/pkg/config"
)

// ExternalNameConfigs contains all external name configurations for this
// provider.
//
// busbar identity model:
//   - busbar_virtual_key: the server assigns an opaque id (vk_...) at create.
//     The plaintext secret is only returned once, so the id is the stable
//     external name -> IdentifierFromProvider.
//   - busbar_config: the document is versioned by the server; the "id" is
//     provider-assigned -> IdentifierFromProvider.
//   - busbar_hook: identified by its "name" (there is no separate id
//     attribute; ImportState passes through name) -> NameAsIdentifier.
var ExternalNameConfigs = map[string]config.ExternalName{
	"busbar_virtual_key": config.IdentifierFromProvider,
	"busbar_config":      config.IdentifierFromProvider,
	"busbar_hook":        config.NameAsIdentifier,
}

// ExternalNameConfigurations applies all external name configs listed in the
// table ExternalNameConfigs and sets the version of those resources to v1beta1
// assuming they will be tested.
func ExternalNameConfigurations() config.ResourceOption {
	return func(r *config.Resource) {
		if e, ok := ExternalNameConfigs[r.Name]; ok {
			r.ExternalName = e
		}
	}
}

// ExternalNameConfigured returns the list of all resources whose external name
// is configured manually.
func ExternalNameConfigured() []string {
	l := make([]string, len(ExternalNameConfigs))
	i := 0
	for name := range ExternalNameConfigs {
		// $ is added to match the exact string since the format is regex.
		l[i] = name + "$"
		i++
	}
	return l
}
