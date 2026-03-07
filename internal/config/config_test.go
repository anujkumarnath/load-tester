package config

import "testing"

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"valid",             Config{URL: "https://example.com", Concurrency: 5, Duration: 1,  RPS: 0, Timeout: 1}, false},
		{"missing url",       Config{URL: "", Concurrency: 5, Duration: 1, RPS: 0, Timeout: 1},                     true},
		{"invalid c",         Config{URL: "https://example.com", Concurrency: -1, Duration: 1, RPS: 0, Timeout: 1}, true},
		{"0 c",               Config{URL: "https://example.com", Concurrency: 0, Duration: 1, RPS: 0, Timeout: 1},  true},
		{"negative d",        Config{URL: "https://example.com", Duration: -1, RPS: 0, Timeout: 1},                 true},
		{"negative rps",      Config{URL: "https://example.com", Duration: 1, RPS: -1, Timeout: 1},                 true},
		{"negative timeout",  Config{URL: "https://example.com", Duration: 1, RPS: 0, Timeout: -1},                 true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
