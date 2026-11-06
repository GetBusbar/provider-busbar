package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
	"github.com/crossplane/upjet/v2/pkg/resource"
	"github.com/crossplane/upjet/v2/pkg/terraform"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cluster "github.com/GetBusbar/provider-busbar/apis/cluster/busbar/v1alpha1"
	namespaced "github.com/GetBusbar/provider-busbar/apis/namespaced/busbar/v1alpha1"
	"github.com/GetBusbar/provider-busbar/config"
)

func TestWorkspaceManagementPolicies(t *testing.T) {
	priority := float64(42)
	kind, module := "gate", "my-installed-ranker"
	for _, tc := range []struct {
		name     string
		managed  resource.Terraformed
		provider *ujconfig.Provider
	}{
		{"cluster", &cluster.Hook{Spec: cluster.HookSpec{ForProvider: cluster.HookParameters{Kind: &kind, Module: &module}, InitProvider: cluster.HookInitParameters{Priority: &priority}}}, config.GetProvider()},
		{"namespaced", &namespaced.Hook{Spec: namespaced.HookSpec{ForProvider: namespaced.HookParameters{Kind: &kind, Module: &module}, InitProvider: namespaced.HookInitParameters{Priority: &priority}}}, config.GetProviderNamespaced()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			flags := &feature.Flags{}
			store := newWorkspaceStore(logging.NewNopLogger(), flags)
			terraform.WithDisableInit(true)(store) // Only produce files; never execute Terraform.
			flags.Enable(feature.EnableBetaManagementPolicies)
			tc.managed.SetName("hook")
			tc.managed.SetUID(uuid.NewUUID())
			t.Cleanup(func() {
				if err := store.Remove(tc.managed); err != nil {
					t.Error(err)
				}
			})
			setup := terraform.Setup{Version: "1.5.7", Requirement: terraform.ProviderRequirement{Source: "getbusbar/busbar", Version: "0.1.0"}, Configuration: map[string]any{}}
			if _, err := store.Workspace(context.Background(), newConfigSecretReader(t, ""), tc.managed, setup, tc.provider.Resources["busbar_hook"]); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join(os.TempDir(), string(tc.managed.GetUID()), "main.tf.json"))
			if err != nil {
				t.Fatal(err)
			}
			var mainTF struct {
				Resource map[string]map[string]struct {
					Priority  float64 `json:"priority"`
					Lifecycle struct {
						IgnoreChanges []string `json:"ignore_changes"`
					} `json:"lifecycle"`
				} `json:"resource"`
			}
			if err := json.Unmarshal(data, &mainTF); err != nil {
				t.Fatal(err)
			}
			got := mainTF.Resource["busbar_hook"]["hook"]
			if got.Priority != priority || len(got.Lifecycle.IgnoreChanges) != 1 || got.Lifecycle.IgnoreChanges[0] != "priority" {
				t.Fatal("initProvider priority must be used on creation and ignored for later updates")
			}
		})
	}
}

// Config secret references are resolved through the same client contract as the
// controller, backed here by an actual fake Kubernetes Secret store.
type configSecretReader struct {
	client   client.Client
	selected []xpv1.SecretKeySelector
}

func newConfigSecretReader(t *testing.T, document string) *configSecretReader {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "gateway-config-document", Namespace: "default"}, Data: map[string][]byte{"document": []byte(document), "override": []byte(`{"config":{"providers":{},"models":{},"pools":{}}}`)}}
	return &configSecretReader{client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()}
}
func (r *configSecretReader) GetSecretData(ctx context.Context, ref *xpv1.SecretReference) (map[string][]byte, error) {
	secret := &corev1.Secret{}
	if err := r.client.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}, secret); err != nil {
		return nil, err
	}
	return secret.Data, nil
}
func (r *configSecretReader) GetSecretValue(ctx context.Context, sel xpv1.SecretKeySelector) ([]byte, error) {
	data, err := r.GetSecretData(ctx, &sel.SecretReference)
	if err != nil {
		return nil, err
	}
	value, ok := data[sel.Key]
	if !ok {
		return nil, fmt.Errorf("missing secret key %q", sel.Key)
	}
	r.selected = append(r.selected, sel)
	return value, nil
}
func TestWorkspaceConfigSecretReferences(t *testing.T) {
	document := `{"config":{"providers":{},"models":{}}}`
	override := `{"config":{"providers":{},"models":{},"pools":{}}}`
	for _, scope := range []string{"cluster", "namespaced"} {
		for _, mode := range []string{"for-provider", "init-only", "override"} {
			t.Run(scope+"/"+mode, func(t *testing.T) {
				clusterRef := xpv1.SecretKeySelector{SecretReference: xpv1.SecretReference{Name: "gateway-config-document", Namespace: "default"}, Key: "document"}
				localRef := xpv1.LocalSecretKeySelector{LocalSecretReference: xpv1.LocalSecretReference{Name: "gateway-config-document"}, Key: "document"}
				clusterOverride, localOverride := clusterRef, localRef
				clusterOverride.Key = "override"
				localOverride.Key = "override"
				var managed resource.Terraformed
				provider := config.GetProvider()
				want, keys := document, []string{"document"}
				ignored := []string(nil)
				if scope == "cluster" {
					object := &cluster.Config{}
					if mode != "for-provider" {
						object.Spec.InitProvider.DocumentSecretRef = &clusterRef
					}
					if mode == "for-provider" {
						object.Spec.ForProvider.DocumentSecretRef = &clusterRef
					}
					if mode == "override" {
						object.Spec.ForProvider.DocumentSecretRef = &clusterOverride
					}
					managed = object
				} else {
					object := &namespaced.Config{ObjectMeta: metav1.ObjectMeta{Namespace: "default"}}
					if mode != "for-provider" {
						object.Spec.InitProvider.DocumentSecretRef = &localRef
					}
					if mode == "for-provider" {
						object.Spec.ForProvider.DocumentSecretRef = &localRef
					}
					if mode == "override" {
						object.Spec.ForProvider.DocumentSecretRef = &localOverride
					}
					managed = object
					provider = config.GetProviderNamespaced()
				}
				if mode == "init-only" {
					ignored = []string{"document"}
				}
				if mode == "override" {
					want = override
					keys = []string{"document", "override"}
				}
				flags := &feature.Flags{}
				flags.Enable(feature.EnableBetaManagementPolicies)
				store := newWorkspaceStore(logging.NewNopLogger(), flags)
				terraform.WithDisableInit(true)(store)
				managed.SetName("config")
				managed.SetUID(uuid.NewUUID())
				t.Cleanup(func() {
					if err := store.Remove(managed); err != nil {
						t.Error(err)
					}
				})
				reader := newConfigSecretReader(t, document)
				setup := terraform.Setup{Version: "1.5.7", Requirement: terraform.ProviderRequirement{Source: "getbusbar/busbar", Version: "0.1.0"}, Configuration: map[string]any{}}
				if _, err := store.Workspace(context.Background(), reader, managed, setup, provider.Resources["busbar_config"]); err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(filepath.Join(os.TempDir(), string(managed.GetUID()), "main.tf.json"))
				if err != nil {
					t.Fatal(err)
				}
				var output struct {
					Resource map[string]map[string]struct {
						Document  string `json:"document"`
						Lifecycle struct {
							IgnoreChanges []string `json:"ignore_changes"`
						} `json:"lifecycle"`
					} `json:"resource"`
				}
				if err := json.Unmarshal(data, &output); err != nil {
					t.Fatal(err)
				}
				got := output.Resource["busbar_config"]["config"]
				if got.Document != want || !reflect.DeepEqual(got.Lifecycle.IgnoreChanges, ignored) {
					t.Fatalf("workspace config=%#v; want document=%s ignored=%v", got, want, ignored)
				}
				if len(reader.selected) != len(keys) {
					t.Fatalf("resolved selectors = %#v; want keys%v", reader.selected, keys)
				}
				for i, key := range keys {
					expected := clusterRef
					expected.Key = key
					if reader.selected[i] != expected {
						t.Fatalf("resolved selector=%#v; want %#v", reader.selected[i], expected)
					}
				}
			})
		}
	}
}
