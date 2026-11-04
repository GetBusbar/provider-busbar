package main

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/upjet/v2/pkg/terraform"
)

// newWorkspaceStore shares controller flags so initProvider and ignore_changes follow the
// management-policy flag enabled during controller setup.
func newWorkspaceStore(log logging.Logger, flags *feature.Flags) *terraform.WorkspaceStore {
	return terraform.NewWorkspaceStore(log, terraform.WithFeatures(flags))
}
