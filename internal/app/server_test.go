package app

import (
	"testing"
)

func TestExtractPublicBaseURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "https URL",
			input: "https://example.com/auth/callback",
			want:  "https://example.com",
		},
		{
			name:  "http URL",
			input: "http://localhost:5000/auth/callback",
			want:  "http://localhost:5000",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "invalid URL",
			input: "://invalid",
			want:  "",
		},
		{
			name:  "URL without host",
			input: "/relative/path",
			want:  "",
		},
		{
			name:  "URL with subdomain and port",
			input: "https://app.example.com:8443/auth/discord/callback",
			want:  "https://app.example.com:8443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPublicBaseURL(tt.input)
			if got != tt.want {
				t.Errorf("extractPublicBaseURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
