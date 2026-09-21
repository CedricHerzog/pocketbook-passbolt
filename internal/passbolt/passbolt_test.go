package passbolt

import (
	"strings"
	"testing"
)

func TestParsePassword(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		isString bool
		want     string
	}{
		{"string secret", `p@ss {"word"}`, true, `p@ss {"word"}`},
		{"password", `{"password":"hunter2"}`, false, "hunter2"},
		{"password and description", `{"password":"hunter2","description":"note"}`, false, "hunter2"},
		{"totp only", `{"totp":{"secret_key":"ABC"}}`, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePassword(tt.data, tt.isString)
			if err != nil {
				t.Fatalf("parsePassword: %v", err)
			}
			if got != tt.want {
				t.Errorf("parsePassword = %q, want %q", got, tt.want)
			}
		})
	}
}

// The error is logged and shown on screen, so it must not quote the secret.
func TestParsePasswordInvalidJSON(t *testing.T) {
	// json.Unmarshal would report "invalid character 'Q' ...".
	_, err := parsePassword("Qz", false)
	if err == nil {
		t.Fatal("parsePassword: no error for invalid JSON")
	}
	if strings.ContainsAny(err.Error(), "Qz") {
		t.Errorf("error %q quotes the secret", err)
	}
}
