package clients

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseCredentials(t *testing.T) {
	for _, input := range []string{`{"endpoint":"https://example.test","token":"secret","insecure":true}`, `{"endpoint":"https://example.test","token":"secret","insecure":"true"}`, `{"endpoint":"https://example.test","token":"secret","insecure":"1"}`} {
		got, err := parseCredentials([]byte(input))
		want := map[string]any{"endpoint": "https://example.test", "token": "secret", "insecure": true}
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("credential types mismatch: %v", err)
		}
	}
	got, err := parseCredentials([]byte(`{"insecure":false,"client_key_pem":"private-key","ca_cert_pem":"CA"}`))
	if err != nil || got["insecure"] != false || got["client_key_pem"] != "private-key" || got["ca_cert_pem"] != "CA" {
		t.Fatal("TLS credentials did not round-trip")
	}
	for _, input := range []string{`{"insecure":"secret-invalid"}`, `{"token":123}`, `{"token":{"secret-invalid":true}}`, `{"token":"secret-invalid"`} {
		_, err := parseCredentials([]byte(input))
		if err == nil {
			t.Fatal("invalid credential accepted")
		}
		if strings.Contains(err.Error(), "secret-invalid") {
			t.Fatal("credential leaked in error")
		}
	}
}
