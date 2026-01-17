package core

import (
	"testing"
)

type providerConfigStub struct {
	valid bool
}

func (p providerConfigStub) Validate() error {
	if p.valid {
		return nil
	}
	return ErrInvalidProviderName
}

func TestProviderConfigFromMap_NotRegistered(t *testing.T) {
	_, err := ProviderConfigFromMap("missing", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestProviderConfigFromMap_Registered(t *testing.T) {
	RegisterProviderConfigFactory("test-provider", func(_ map[string]interface{}) (ProviderConfig, error) {
		return providerConfigStub{valid: true}, nil
	})

	cfg, err := ProviderConfigFromMap("test-provider", map[string]interface{}{"k": "v"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
}
