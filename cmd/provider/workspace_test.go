package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
	"github.com/crossplane/upjet/v2/pkg/resource"
	"github.com/crossplane/upjet/v2/pkg/terraform"
	"k8s.io/apimachinery/pkg/util/uuid"

	cluster "github.com/GetBusbar/provider-busbar/apis/cluster/busbar/v1alpha1"
	namespaced "github.com/GetBusbar/provider-busbar/apis/namespaced/busbar/v1alpha1"
	"github.com/GetBusbar/provider-busbar/config"
)

func TestWorkspaceManagementPolicies(t *testing.T) {
	document := `{"config":{"providers":{},"models":{}}}`
	for _, tc := range []struct {
		name     string
		managed  resource.Terraformed
		provider *ujconfig.Provider
	}{
		{"cluster", &cluster.Config{Spec: cluster.ConfigSpec{InitProvider: cluster.ConfigInitParameters{Document: &document}}}, config.GetProvider()},
		{"namespaced", &namespaced.Config{Spec: namespaced.ConfigSpec{InitProvider: namespaced.ConfigInitParameters{Document: &document}}}, config.GetProviderNamespaced()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			flags := &feature.Flags{}
			store := newWorkspaceStore(logging.NewNopLogger(), flags)
			terraform.WithDisableInit(true)(store) // Only produce files; never execute Terraform.
			flags.Enable(feature.EnableBetaManagementPolicies)
			tc.managed.SetName("config")
			tc.managed.SetUID(uuid.NewUUID())
			t.Cleanup(func() {
				if err := store.Remove(tc.managed); err != nil {
					t.Error(err)
				}
			})
			setup := terraform.Setup{Version: "1.5.7", Requirement: terraform.ProviderRequirement{Source: "getbusbar/busbar", Version: "0.1.0"}, Configuration: map[string]any{}}
			if _, err := store.Workspace(context.Background(), nil, tc.managed, setup, tc.provider.Resources["busbar_config"]); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join(os.TempDir(), string(tc.managed.GetUID()), "main.tf.json"))
			if err != nil {
				t.Fatal(err)
			}
			var mainTF struct {
				Resource map[string]map[string]struct {
					Document  string `json:"document"`
					Lifecycle struct {
						IgnoreChanges []string `json:"ignore_changes"`
					} `json:"lifecycle"`
				} `json:"resource"`
			}
			if err := json.Unmarshal(data, &mainTF); err != nil {
				t.Fatal(err)
			}
			got := mainTF.Resource["busbar_config"]["config"]
			if got.Document != document || len(got.Lifecycle.IgnoreChanges) != 1 || got.Lifecycle.IgnoreChanges[0] != "document" {
				t.Fatal("initProvider document must be used on creation and ignored for later updates")
			}
		})
	}
}
