// Package busbar contains the cluster-scoped resource configurations for the
// busbar Terraform provider (busbar_virtual_key, busbar_hook, busbar_config).
package busbar

import (
	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
)

// Group is the API group for all busbar managed resources.
const Group = "busbar"

// Configure registers the per-resource overrides (kinds, groups) for the
// cluster-scoped provider.
func Configure(p *ujconfig.Provider) {
	p.AddResourceConfigurator("busbar_virtual_key", func(r *ujconfig.Resource) {
		r.ShortGroup = Group
		r.Kind = "VirtualKey"
	})
	p.AddResourceConfigurator("busbar_hook", func(r *ujconfig.Resource) {
		r.ShortGroup = Group
		r.Kind = "Hook"
	})
	p.AddResourceConfigurator("busbar_config", func(r *ujconfig.Resource) {
		r.ShortGroup = Group
		r.Kind = "Config"
	})
}
