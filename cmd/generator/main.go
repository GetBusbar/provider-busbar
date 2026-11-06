package main

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"github.com/crossplane/upjet/v2/pkg/pipeline"

	"github.com/GetBusbar/provider-busbar/config"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] == "" {
		panic("root directory is required to be given as argument")
	}
	rootDir := os.Args[1]
	absRootDir, err := filepath.Abs(rootDir)
	if err != nil {
		panic(fmt.Sprintf("cannot calculate the absolute path with %s", rootDir))
	}
	pipeline.Run(config.GetProvider(), config.GetProviderNamespaced(), absRootDir)
	if err := fixConfigSecretInit(absRootDir); err != nil {
		panic(err)
	}
}

// fixConfigSecretInit bridges the upstream generator's sensitive init fields to
// its Terraform lifecycle calculation. Secret values remain in Kubernetes Secrets;
// only field presence is exposed before Upjet resolves sensitive parameters.
// Keep this after pipeline.Run and before controller-gen so CRDs and deep copies
// are generated from the corrected optional selector shape.
func fixConfigSecretInit(root string) error {
	for _, scope := range []string{"cluster", "namespaced"} {
		dir := filepath.Join(root, "apis", scope, "busbar", "v1alpha1")
		selector := "v1.SecretKeySelector"
		if scope == "namespaced" {
			selector = "v1.LocalSecretKeySelector"
		}
		oldRule := "|| has(self.forProvider.documentSecretRef)"
		newRule := oldRule + " || (has(self.initProvider) && has(self.initProvider.documentSecretRef))"
		replacements := []struct {
			file, old, new string
			count          int
		}{
			{"zz_config_types.go", "DocumentSecretRef " + selector + " `json:\"documentSecretRef\" tf:\"-\"`", "DocumentSecretRef *" + selector + " `json:\"documentSecretRef,omitempty\" tf:\"-\"`", 2},
			{"zz_config_types.go", oldRule, newRule, 1},
			{"zz_config_types.go", "spec.forProvider.documentSecretRef is a required parameter", "spec.forProvider.documentSecretRef or spec.initProvider.documentSecretRef is required", 1},
		}
		for _, field := range []string{"ForProvider", "InitProvider"} {
			old := "p, err := json.TFParser.Marshal(tr.Spec." + field + `)
 if err != nil {
  return nil, err
 }
 base := map[string]any{}
 return base, json.TFParser.Unmarshal(p, &base)`
			new := "p, err := json.TFParser.Marshal(tr.Spec." + field + `)
 if err != nil {
  return nil, err
 }
 base := map[string]any{}
 if err := json.TFParser.Unmarshal(p, &base); err != nil {
  return nil, err
 }
 // Upjet computes ignore_changes before resolving Secret references.
 // Represent presence only; sensitive resolution replaces this marker.
 if tr.Spec.` + field + `.DocumentSecretRef != nil {
  base["document"] = ""
 }
 return base, nil`
			// Normalize fragments to the same indentation used by gofmt.
			normalize := func(body string) string {
				b, err := format.Source([]byte("package generated\nfunc f() {\n" + body + "\n}"))
				if err != nil {
					panic(err)
				}
				formatted := string(b)
				start := strings.Index(formatted, "{\n") + 2
				return strings.TrimSuffix(formatted[start:], "}\n")
			}
			replacements = append(replacements, struct {
				file, old, new string
				count          int
			}{"zz_config_terraformed.go", normalize(old), normalize(new), 1})
		}
		for _, r := range replacements {
			path := filepath.Join(dir, r.file)
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if n := strings.Count(string(data), r.old); n != r.count {
				return fmt.Errorf("Config secret init generator contract changed in %s: expected %d matches, got %d", path, r.count, n)
			}
			data, err = format.Source([]byte(strings.ReplaceAll(string(data), r.old, r.new)))
			if err != nil {
				return err
			}
			if err := os.WriteFile(path, data, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}
