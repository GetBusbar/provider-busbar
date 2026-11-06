package config

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestExamplesMatchProviderContract(t *testing.T) {
	for _, scope := range []string{"cluster", "namespaced"} {
		for _, name := range []string{"config", "hook", "virtualkey"} {
			t.Run(scope+"/"+name, func(t *testing.T) {
				data, err := os.ReadFile(filepath.Join("..", "examples", scope, "busbar", name+".yaml"))
				if err != nil {
					t.Fatal(err)
				}
				type example struct {
					Kind     string `json:"kind"`
					Metadata struct {
						Name      string `json:"name"`
						Namespace string `json:"namespace"`
					} `json:"metadata"`
					StringData map[string]string `json:"stringData"`
					Spec       struct {
						ForProvider       map[string]any `json:"forProvider"`
						ProviderConfigRef struct {
							Kind string `json:"kind"`
						} `json:"providerConfigRef"`
					} `json:"spec"`
				}
				var object example
				secrets := map[string]example{}
				decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)
				for {
					var item example
					if err := decoder.Decode(&item); err == io.EOF {
						break
					} else if err != nil {
						t.Fatal(err)
					}
					if item.Kind == "Secret" {
						secrets[item.Metadata.Namespace+"/"+item.Metadata.Name] = item
					} else {
						object = item
					}
				}
				group := "busbar.busbar.crossplane.io"
				if scope == "namespaced" {
					group = "busbar.busbar.m.crossplane.io"
				}
				plural := map[string]string{"config": "configs", "hook": "hooks", "virtualkey": "virtualkeys"}[name]
				crdData, err := os.ReadFile(filepath.Join("..", "package", "crds", group+"_"+plural+".yaml"))
				if err != nil {
					t.Fatal(err)
				}
				crdJSON, err := yaml.ToJSON(crdData)
				if err != nil {
					t.Fatal(err)
				}
				var crd apiextensions.CustomResourceDefinition
				if err := json.Unmarshal(crdJSON, &crd); err != nil {
					t.Fatal(err)
				}
				properties := crd.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"].Properties["forProvider"].Properties
				for key := range object.Spec.ForProvider {
					if _, ok := properties[key]; !ok {
						t.Fatalf("example field %q is absent from generated CRD", key)
					}
				}

				if scope == "namespaced" && object.Spec.ProviderConfigRef.Kind != "ClusterProviderConfig" {
					t.Fatal("namespaced example needs explicit provider config kind")
				}
				fields := object.Spec.ForProvider
				switch name {
				case "config":
					ref, ok := fields["documentSecretRef"].(map[string]any)
					if !ok {
						t.Fatal("missing config documentSecretRef")
					}
					for key := range ref {
						if _, ok := properties["documentSecretRef"].Properties[key]; !ok {
							t.Fatalf("selector field %q absent from generated CRD", key)
						}
					}
					namespace := object.Metadata.Namespace
					if scope == "cluster" {
						namespace, ok = ref["namespace"].(string)
						if !ok || namespace == "" {
							t.Fatal("cluster selector needs namespace")
						}
					} else if _, ok := ref["namespace"]; ok {
						t.Fatal("namespaced selector must be local")
					}
					secretName, ok := ref["name"].(string)
					if !ok || secretName == "" {
						t.Fatal("selector needs name")
					}
					key, ok := ref["key"].(string)
					if !ok || key == "" {
						t.Fatal("selector needs key")
					}
					secret, ok := secrets[namespace+"/"+secretName]
					if !ok {
						t.Fatal("matching Secret fixture missing")
					}
					document, ok := secret.StringData[key]
					if !ok {
						t.Fatal("Secret fixture document missing")
					}

					var body map[string]any
					if err := json.Unmarshal([]byte(document), &body); err != nil {
						t.Fatal(err)
					}
					if _, ok := body["config"].(map[string]any); !ok {
						t.Fatal("missing config write model")
					}
				case "hook":
					if fields["kind"] != "gate" || fields["module"] != "pii-redactor" || fields["prompt"] != "rw" || !reflect.DeepEqual(fields["phase"], []any{"request"}) {
						t.Fatal("invalid redactor hook contract")
					}
					for _, retired := range []string{"at", "webhook"} {
						if _, ok := fields[retired]; ok {
							t.Fatalf("retired hook field %q", retired)
						}
					}
					if _, ok := fields["name"]; ok {
						t.Fatal("hook name belongs in external-name annotation or metadata.name")
					}
				}
			})
		}
	}
}

func TestInstallExample(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "examples", "install.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	data, err = yaml.ToJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	var object struct {
		Spec struct {
			Package string `json:"package"`
		} `json:"spec"`
	}
	if err := json.Unmarshal(data, &object); err != nil {
		t.Fatal(err)
	}
	if object.Spec.Package != "ghcr.io/getbusbar/provider-busbar:v0.1.0" {
		t.Fatal("install example must use the published busbar package")
	}
}

func TestConfigSecretReferenceAdmissionRule(t *testing.T) {
	const requiredRule = "!('*' in self.managementPolicies || 'Create' in self.managementPolicies || 'Update' in self.managementPolicies) || has(self.forProvider.documentSecretRef) || (has(self.initProvider) && has(self.initProvider.documentSecretRef))"
	for _, group := range []string{"busbar.busbar.crossplane.io", "busbar.busbar.m.crossplane.io"} {
		t.Run(group, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "package", "crds", group+"_configs.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			data, err = yaml.ToJSON(data)
			if err != nil {
				t.Fatal(err)
			}
			var crd apiextensions.CustomResourceDefinition
			if err := json.Unmarshal(data, &crd); err != nil {
				t.Fatal(err)
			}
			spec := crd.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"]
			found := false
			for _, rule := range spec.XValidations {
				if rule.Rule == requiredRule {
					found = true
				}
			}
			if !found {
				t.Fatal("Config admission must accept either forProvider or initProvider documentSecretRef")
			}
			for _, key := range spec.Properties["forProvider"].Required {
				if key == "documentSecretRef" {
					t.Fatal("init-only SecretRef cannot be unconditionally required in forProvider")
				}
			}
			if _, ok := spec.Properties["initProvider"].Properties["documentSecretRef"]; !ok {
				t.Fatal("initProvider SecretRef schema missing")
			}
		})
	}
}
