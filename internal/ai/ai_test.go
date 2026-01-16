package ai

import (
	"testing"
)

func TestParseProvider(t *testing.T) {
	tests := []struct {
		input    string
		expected Provider
		ok       bool
	}{
		{"claude", ProviderClaude, true},
		{"codex", ProviderCodex, true},
		{"gemini", ProviderGemini, true},
		{"unknown", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := ParseProvider(tt.input)
			if ok != tt.ok {
				t.Errorf("ParseProvider(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			}
			if got != tt.expected {
				t.Errorf("ParseProvider(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestAllProviders(t *testing.T) {
	providers := AllProviders()
	if len(providers) != 3 {
		t.Errorf("AllProviders() returned %d providers, want 3", len(providers))
	}

	// Check order
	expected := []Provider{ProviderClaude, ProviderCodex, ProviderGemini}
	for i, p := range providers {
		if p != expected[i] {
			t.Errorf("AllProviders()[%d] = %v, want %v", i, p, expected[i])
		}
	}
}

func TestProviderString(t *testing.T) {
	if ProviderClaude.String() != "claude" {
		t.Errorf("ProviderClaude.String() = %q, want %q", ProviderClaude.String(), "claude")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Default != "auto" {
		t.Errorf("Default = %q, want %q", cfg.Default, "auto")
	}

	if cfg.Claude.Bin != "claude" {
		t.Errorf("Claude.Bin = %q, want %q", cfg.Claude.Bin, "claude")
	}

	if cfg.Codex.Bin != "codex" {
		t.Errorf("Codex.Bin = %q, want %q", cfg.Codex.Bin, "codex")
	}

	if cfg.Gemini.Bin != "gemini" {
		t.Errorf("Gemini.Bin = %q, want %q", cfg.Gemini.Bin, "gemini")
	}

	if cfg.Timeout.Seconds() != 60 {
		t.Errorf("Timeout = %v, want 60s", cfg.Timeout)
	}
}

func TestNewClient(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		provider Provider
		name     string
	}{
		{ProviderClaude, "claude"},
		{ProviderCodex, "codex"},
		{ProviderGemini, "gemini"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.provider, cfg)
			if client == nil {
				t.Fatal("NewClient returned nil")
			}
			if client.Name() != tt.name {
				t.Errorf("Name() = %q, want %q", client.Name(), tt.name)
			}
		})
	}
}

func TestNewClientUnknownProvider(t *testing.T) {
	cfg := DefaultConfig()
	client := NewClient("unknown", cfg)
	if client != nil {
		t.Error("NewClient with unknown provider should return nil")
	}
}
