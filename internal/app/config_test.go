package app

import (
	"os"
	"strings"
	"testing"
)

func TestNewConfig(t *testing.T) {
	validArgs := []string{
		"-data-dir=/tmp/test",
		"-session-secret=a-very-long-secret-key-that-is-at-least-32-chars",
		"-discord-client-id=test-id",
		"-discord-client-secret=test-secret",
		"-discord-redirect-url=http://localhost/callback",
		"-admin-username=admin",
		"-admin-password=strongpassword",
	}

	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			args: validArgs,
		},
		{
			name: "missing discord-client-id",
			args: []string{
				"-data-dir=/tmp/test",
				"-session-secret=a-very-long-secret-key-that-is-at-least-32-chars",
				"-discord-client-secret=test-secret",
				"-discord-redirect-url=http://localhost/callback",
				"-admin-username=admin",
				"-admin-password=strongpassword",
			},
			wantErr:     true,
			errContains: "missing discord-client-id",
		},
		{
			name: "session-secret too short",
			args: []string{
				"-data-dir=/tmp/test",
				"-session-secret=tooshort",
				"-discord-client-id=test-id",
				"-discord-client-secret=test-secret",
				"-discord-redirect-url=http://localhost/callback",
				"-admin-username=admin",
				"-admin-password=strongpassword",
			},
			wantErr:     true,
			errContains: "session-secret must be at least 32 characters",
		},
		{
			name: "admin-password too short",
			args: []string{
				"-data-dir=/tmp/test",
				"-session-secret=a-very-long-secret-key-that-is-at-least-32-chars",
				"-discord-client-id=test-id",
				"-discord-client-secret=test-secret",
				"-discord-redirect-url=http://localhost/callback",
				"-admin-username=admin",
				"-admin-password=short",
			},
			wantErr:     true,
			errContains: "admin-password must be at least 12 characters",
		},
		{
			name: "missing session-secret",
			args: []string{
				"-data-dir=/tmp/test",
				"-discord-client-id=test-id",
				"-discord-client-secret=test-secret",
				"-discord-redirect-url=http://localhost/callback",
				"-admin-username=admin",
				"-admin-password=strongpassword",
			},
			wantErr:     true,
			errContains: "missing session-secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldArgs := os.Args
			os.Args = append([]string{"test"}, tt.args...)
			defer func() { os.Args = oldArgs }()

			cfg, err := NewConfig()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				} else if cfg == nil {
					t.Error("expected config, got nil")
				}
			}
		})
	}
}
