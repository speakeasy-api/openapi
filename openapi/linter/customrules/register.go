package customrules

import (
	"sync"

	baseLinter "github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	openapiLinter "github.com/speakeasy-api/openapi/openapi/linter"
)

// defaultConfig holds programmatic options for the loader.
// Access is protected by defaultConfigMu.
var (
	defaultConfig   *Config
	defaultConfigMu sync.RWMutex
)

// init registers the custom rule loader with the OpenAPI linter.
// This is called automatically when the package is imported.
func init() {
	openapiLinter.RegisterCustomRuleLoader(loadCustomRules)
}

// loadCustomRules is the registered loader function called by the OpenAPI linter.
// Each call creates a new loader to ensure thread safety during parallel execution.
func loadCustomRules(config *baseLinter.CustomRulesConfig) ([]baseLinter.RuleRunner[*openapi.OpenAPI], error) {
	if config == nil || len(config.Paths) == 0 {
		return nil, nil
	}

	// Get any programmatic config options
	defaultConfigMu.RLock()
	cfg := defaultConfig
	defaultConfigMu.RUnlock()

	// Create a new loader for each call to ensure thread safety
	loader := NewLoader(cfg)
	return loader.LoadRules(config)
}

// SetDefaultConfig sets the default config for the loader.
// This allows customizing the loader with programmatic options (like Logger).
// Call this before creating any linters to take effect.
// This function is thread-safe.
func SetDefaultConfig(config *Config) {
	defaultConfigMu.Lock()
	defaultConfig = config
	defaultConfigMu.Unlock()
}
