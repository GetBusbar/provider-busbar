package config

import "testing"

func TestHookExternalName(t *testing.T) {
	e := ExternalNameConfigs["busbar_hook"]
	got, err := e.GetExternalNameFn(map[string]any{"name": "redactor"})
	if err != nil || got != "redactor" {
		t.Fatalf("hook identity = %q, %v", got, err)
	}
	for _, state := range []map[string]any{{}, {"name": nil}, {"name": true}, {"name": ""}} {
		if _, err := e.GetExternalNameFn(state); err == nil {
			t.Fatal("invalid hook identity accepted")
		}
	}
	args := map[string]any{}
	e.SetIdentifierArgumentFn(args, "redactor")
	if args["name"] != "redactor" {
		t.Fatal("hook name not configured")
	}
}
