package core

import (
	"context"
	"testing"
)

// mockProvider is a test implementation of the Provider interface.
type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Validate(_ context.Context) error {
	return nil
}

func (m *mockProvider) Generate(_ context.Context, _ *GenerateOptions) (*Result, error) {
	return &Result{
		Provider:     m.name,
		FilesCreated: []string{},
	}, nil
}

func (m *mockProvider) Backup(_ context.Context) (string, error) {
	return "", nil
}

func (m *mockProvider) Restore(_ context.Context, _ string) error {
	return nil
}

func (m *mockProvider) Clean(_ context.Context) error {
	return nil
}

func TestRegistry_Register(t *testing.T) {
	tests := []struct {
		name        string
		provider    Provider
		expectError bool
	}{
		{
			name:        "valid provider",
			provider:    &mockProvider{name: "test"},
			expectError: false,
		},
		{
			name:        "empty name",
			provider:    &mockProvider{name: ""},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()
			err := registry.Register(tt.provider)

			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRegistry_DuplicateRegistration(t *testing.T) {
	registry := NewRegistry()
	provider := &mockProvider{name: "test"}

	if err := registry.Register(provider); err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	err := registry.Register(provider)
	if err == nil {
		t.Error("expected error for duplicate registration, got nil")
	}

	if _, ok := err.(*ErrProviderExists); !ok {
		t.Errorf("expected ErrProviderExists, got %T", err)
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()
	provider := &mockProvider{name: "test"}

	if err := registry.Register(provider); err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	got, err := registry.Get("test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got.Name() != "test" {
		t.Errorf("expected name 'test', got %q", got.Name())
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	registry := NewRegistry()

	_, err := registry.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent provider, got nil")
	}

	if _, ok := err.(*ErrProviderNotFound); !ok {
		t.Errorf("expected ErrProviderNotFound, got %T", err)
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	providers := []Provider{
		&mockProvider{name: "aws"},
		&mockProvider{name: "kubernetes"},
		&mockProvider{name: "ssh"},
	}

	for _, p := range providers {
		if err := registry.Register(p); err != nil {
			t.Fatalf("registration failed: %v", err)
		}
	}

	names := registry.List()
	if len(names) != 3 {
		t.Errorf("expected 3 providers, got %d", len(names))
	}

	// Verify alphabetical order
	expected := []string{"aws", "kubernetes", "ssh"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("expected %q at position %d, got %q", expected[i], i, name)
		}
	}
}

func TestRegistry_GetAll(t *testing.T) {
	registry := NewRegistry()

	provider1 := &mockProvider{name: "test1"}
	provider2 := &mockProvider{name: "test2"}

	if err := registry.Register(provider1); err != nil {
		t.Fatalf("failed to register provider1: %v", err)
	}
	if err := registry.Register(provider2); err != nil {
		t.Fatalf("failed to register provider2: %v", err)
	}

	providers := registry.GetAll()
	if len(providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(providers))
	}
}

func TestRegistry_Clear(t *testing.T) {
	registry := NewRegistry()

	if err := registry.Register(&mockProvider{name: "test"}); err != nil {
		t.Fatalf("failed to register provider: %v", err)
	}

	registry.Clear()

	providers := registry.List()
	if len(providers) != 0 {
		t.Errorf("expected 0 providers after clear, got %d", len(providers))
	}
}
