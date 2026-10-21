package config

import "testing"

func TestLoad(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]string
		valid  bool
	}{
		{"defaults", map[string]string{}, true},
		{"configured", map[string]string{
			"APP_ORIGIN": "https://cu.example", "COOKIE_SECURE": "true", "APPROVAL_LIMIT_IDR": "1000000",
		}, true},
		{"origin path", map[string]string{"APP_ORIGIN": "http://localhost:3000/app"}, false},
		// #nosec G101 -- Synthetic invalid-origin fixture, not an application credential.
		{"origin credentials", map[string]string{"APP_ORIGIN": "http://user:secret@localhost"}, false},
		{"invalid cookie flag", map[string]string{"COOKIE_SECURE": "sometimes"}, false},
		{"negative limit", map[string]string{"APPROVAL_LIMIT_IDR": "-1"}, false},
		{"excessive limit", map[string]string{"APPROVAL_LIMIT_IDR": "1000000000001"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(func(key string) string { return tt.values[key] })
			if (err == nil) != tt.valid {
				t.Fatalf("valid=%v, err=%v", tt.valid, err)
			}
			if tt.name == "defaults" &&
				(cfg.ApprovalLimit != 0 || cfg.SecureCookies || cfg.Address != "127.0.0.1:8080") {
				t.Fatal("unsafe defaults")
			}
		})
	}
}
