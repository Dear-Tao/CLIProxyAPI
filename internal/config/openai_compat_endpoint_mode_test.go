package config

import "testing"

func TestSanitizeOpenAICompatibility_NormalizesEndpointMode(t *testing.T) {
	cfg := &Config{
		OpenAICompatibility: []OpenAICompatibility{
			{
				Name:         "provider-a",
				BaseURL:      "https://example.com/v1",
				EndpointMode: " RESPONSES ",
			},
			{
				Name:         "provider-b",
				BaseURL:      "https://example.com/v2",
				EndpointMode: "unexpected",
			},
		},
	}

	cfg.SanitizeOpenAICompatibility()

	if got := cfg.OpenAICompatibility[0].EndpointMode; got != "responses" {
		t.Fatalf("endpoint mode = %q, want %q", got, "responses")
	}
	if got := cfg.OpenAICompatibility[1].EndpointMode; got != "auto" {
		t.Fatalf("endpoint mode = %q, want %q", got, "auto")
	}
}
