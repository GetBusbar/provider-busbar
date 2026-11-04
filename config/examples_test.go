package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

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
				var object struct {
					Spec struct {
						ForProvider       map[string]any `json:"forProvider"`
						ProviderConfigRef struct {
							Kind string `json:"kind"`
						} `json:"providerConfigRef"`
					} `json:"spec"`
				}
				data, err = yaml.ToJSON(data)
				if err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(data, &object); err != nil {
					t.Fatal(err)
				}
				if scope == "namespaced" && object.Spec.ProviderConfigRef.Kind != "ClusterProviderConfig" {
					t.Fatal("namespaced example needs explicit provider config kind")
				}
				fields := object.Spec.ForProvider
				switch name {
				case "config":
					document, ok := fields["document"].(string)
					if !ok {
						t.Fatal("missing config document")
					}
					var body map[string]any
					if err := json.Unmarshal([]byte(document), &body); err != nil {
						t.Fatal(err)
					}
					if _, ok := body["config"].(map[string]any); !ok {
						t.Fatal("missing config write model")
					}
				case "hook":
					if fields["kind"] != "gate" || fields["at"] != "request" || fields["prompt"] != "rw" || fields["webhook"] == nil {
						t.Fatal("invalid redactor hook contract")
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
