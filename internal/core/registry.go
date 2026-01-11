package core

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

// Registry manages registration and lookup of configuration providers.
// It ensures that provider names are unique and provides thread-safe access.
type Registry struct {
	providers map[string]Provider
	mu        sync.RWMutex
}

// NewRegistry creates a new provider registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry.
// Returns an error if a provider with the same name is already registered.
func (r *Registry) Register(provider Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := provider.Name()
	if name == "" {
		return ErrInvalidProviderName
	}

	if _, exists := r.providers[name]; exists {
		return &ProviderExistsError{Name: name}
	}

	r.providers[name] = provider
	return nil
}

// Get retrieves a provider by name.
// Returns ProviderNotFoundError if the provider doesn't exist.
func (r *Registry) Get(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, ok := r.providers[name]
	if !ok {
		return nil, &ProviderNotFoundError{Name: name}
	}
	return provider, nil
}

// List returns all registered provider names in alphabetical order.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetAll returns all registered providers.
func (r *Registry) GetAll() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		providers = append(providers, p)
	}

	return providers
}

// Clear removes all registered providers.
// This is primarily useful for testing.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers = make(map[string]Provider)
}

// ErrInvalidProviderName is returned when a provider has an empty name.
var ErrInvalidProviderName = errors.New("provider name cannot be empty")

// ProviderExistsError is returned when attempting to register a provider
// with a name that's already registered.
type ProviderExistsError struct {
	Name string
}

func (e *ProviderExistsError) Error() string {
	return fmt.Sprintf("provider %q is already registered", e.Name)
}

// ProviderNotFoundError is returned when a requested provider doesn't exist.
type ProviderNotFoundError struct {
	Name string
}

func (e *ProviderNotFoundError) Error() string {
	return fmt.Sprintf("provider %q not found", e.Name)
}
