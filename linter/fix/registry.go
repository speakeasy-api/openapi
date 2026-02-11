package fix

import (
	"sync"

	"github.com/speakeasy-api/openapi/validation"
)

// FixProvider creates a Fix for a specific validation error.
// It receives the error and returns a Fix, or nil if no fix is applicable
// for this particular error instance.
type FixProvider func(err *validation.Error) validation.Fix

// FixRegistry maps validation rule IDs to fix providers.
// This allows registering fix providers for pre-existing validation errors
// that don't come from linter rules (e.g., errors from unmarshal/indexing).
type FixRegistry struct {
	mu        sync.RWMutex
	providers map[string][]FixProvider
}

// NewFixRegistry creates a new fix registry.
func NewFixRegistry() *FixRegistry {
	return &FixRegistry{
		providers: make(map[string][]FixProvider),
	}
}

// Register registers a fix provider for a validation rule ID.
// Multiple providers can be registered for the same rule ID; the first one
// that returns a non-nil Fix wins.
func (r *FixRegistry) Register(ruleID string, provider FixProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[ruleID] = append(r.providers[ruleID], provider)
}

// GetFix returns a Fix for the given validation error, or nil if no provider
// can fix it.
func (r *FixRegistry) GetFix(err *validation.Error) validation.Fix {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers, ok := r.providers[err.Rule]
	if !ok {
		return nil
	}

	for _, provider := range providers {
		if fix := provider(err); fix != nil {
			return fix
		}
	}

	return nil
}
