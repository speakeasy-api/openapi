package customrules

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/go-sourcemap/sourcemap"
	baseLinter "github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
)

// typesShim is the JavaScript shim for @speakeasy-api/openapi-linter-types.
// It provides the Rule base class and re-exports runtime globals.
//
//go:embed shim/types-shim.js
var typesShim string

// Loader loads custom rules from TypeScript/JavaScript files.
type Loader struct {
	config *Config
	logger Logger

	// Cache of transpiled code and source maps
	transpiledCache map[string]*TranspiledRule
}

// TranspiledRule holds transpiled JavaScript code and its source map.
type TranspiledRule struct {
	SourceFile string
	Code       string
	SourceMap  *sourcemap.Consumer
}

// NewLoader creates a new custom rule loader.
func NewLoader(config *Config) *Loader {
	return &Loader{
		config:          config,
		logger:          config.GetLogger(),
		transpiledCache: make(map[string]*TranspiledRule),
	}
}

// LoadRules loads all custom rules from the configured paths.
func (l *Loader) LoadRules(baseConfig *baseLinter.CustomRulesConfig) ([]baseLinter.RuleRunner[*openapi.OpenAPI], error) {
	if baseConfig == nil || len(baseConfig.Paths) == 0 {
		return nil, nil
	}

	// Merge base config with extended config
	config := l.mergeConfig(baseConfig)

	// Resolve all rule files from glob patterns
	files, err := l.resolveFiles(config.Paths)
	if err != nil {
		return nil, fmt.Errorf("resolving rule files: %w", err)
	}

	if len(files) == 0 {
		return nil, nil
	}

	// Transpile all rule files
	transpiled, err := l.transpileFiles(files)
	if err != nil {
		return nil, fmt.Errorf("transpiling rules: %w", err)
	}

	// Load rules into runtime
	rules, err := l.loadRules(transpiled, config)
	if err != nil {
		return nil, fmt.Errorf("loading rules: %w", err)
	}

	return rules, nil
}

// mergeConfig merges the base YAML config with the extended programmatic config.
func (l *Loader) mergeConfig(base *baseLinter.CustomRulesConfig) *Config {
	config := &Config{
		Paths:   base.Paths,
		Timeout: base.Timeout,
	}

	// Apply programmatic overrides from l.config
	if l.config != nil {
		if l.config.Timeout > 0 {
			config.Timeout = l.config.Timeout
		}
		if l.config.Logger != nil {
			config.Logger = l.config.Logger
		}
	}

	return config
}

// resolveFiles resolves glob patterns to actual file paths.
func (l *Loader) resolveFiles(patterns []string) ([]string, error) {
	seen := make(map[string]bool)
	var files []string

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
		}

		for _, match := range matches {
			// Skip directories
			info, err := os.Stat(match)
			if err != nil {
				return nil, fmt.Errorf("stat %q: %w", match, err)
			}
			if info.IsDir() {
				continue
			}

			// Only process .ts and .js files
			ext := strings.ToLower(filepath.Ext(match))
			if ext != ".ts" && ext != ".js" {
				continue
			}

			// Deduplicate
			absPath, err := filepath.Abs(match)
			if err != nil {
				return nil, fmt.Errorf("abs path %q: %w", match, err)
			}
			if !seen[absPath] {
				seen[absPath] = true
				files = append(files, absPath)
			}
		}
	}

	return files, nil
}

// transpileFiles transpiles TypeScript files to JavaScript.
func (l *Loader) transpileFiles(files []string) ([]*TranspiledRule, error) {
	var transpiled []*TranspiledRule

	for _, file := range files {
		// Check cache first
		if cached, ok := l.transpiledCache[file]; ok {
			transpiled = append(transpiled, cached)
			continue
		}

		// Read the source file
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("reading %q: %w", file, err)
		}

		// Transpile if TypeScript, otherwise use as-is
		var code string
		var sm *sourcemap.Consumer

		ext := strings.ToLower(filepath.Ext(file))
		if ext == ".ts" {
			code, sm, err = l.transpileTypeScript(string(content), file)
			if err != nil {
				return nil, fmt.Errorf("transpiling %q: %w", file, err)
			}
		} else {
			code = string(content)
		}

		tr := &TranspiledRule{
			SourceFile: file,
			Code:       code,
			SourceMap:  sm,
		}

		// Cache the result
		l.transpiledCache[file] = tr
		transpiled = append(transpiled, tr)
	}

	return transpiled, nil
}

// transpileTypeScript transpiles TypeScript to JavaScript using esbuild.
func (l *Loader) transpileTypeScript(source, filename string) (string, *sourcemap.Consumer, error) {
	// Create a plugin that marks @speakeasy-api/openapi-linter-types as external
	// and resolves it to an empty module (since we inject the globals)
	typesPlugin := api.Plugin{
		Name: "openapi-linter-types",
		Setup: func(build api.PluginBuild) {
			// Mark the types package as external and resolve to a namespace
			build.OnResolve(api.OnResolveOptions{Filter: `^@speakeasy-api/openapi-linter-types$`},
				func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					return api.OnResolveResult{
						Path:      args.Path,
						Namespace: "openapi-linter-types",
					}, nil
				})

			// Return the types shim module
			// This provides the Rule base class and re-exports runtime globals
			build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "openapi-linter-types"},
				func(args api.OnLoadArgs) (api.OnLoadResult, error) {
					return api.OnLoadResult{
						Contents: &typesShim,
						Loader:   api.LoaderJS,
					}, nil
				})
		},
	}

	// Use Build API with stdin for better plugin support
	result := api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents:   source,
			Sourcefile: filename,
			Loader:     api.LoaderTS,
		},
		Bundle:         true,
		Write:          false,
		Target:         api.ES2020,
		Format:         api.FormatIIFE,
		Sourcemap:      api.SourceMapInline,
		SourcesContent: api.SourcesContentInclude,
		TreeShaking:    api.TreeShakingFalse,
		Plugins:        []api.Plugin{typesPlugin},
		LogLevel:       api.LogLevelSilent,
	})

	if len(result.Errors) > 0 {
		var errMsgs []string
		for _, e := range result.Errors {
			if e.Location != nil {
				errMsgs = append(errMsgs, fmt.Sprintf("%s:%d:%d: %s",
					e.Location.File, e.Location.Line, e.Location.Column, e.Text))
			} else {
				errMsgs = append(errMsgs, e.Text)
			}
		}
		return "", nil, fmt.Errorf("esbuild errors:\n%s", strings.Join(errMsgs, "\n"))
	}

	if len(result.OutputFiles) == 0 {
		return "", nil, fmt.Errorf("esbuild produced no output")
	}

	code := string(result.OutputFiles[0].Contents)

	// Extract the inline source map
	sm, err := ExtractInlineSourceMap(code)
	if err != nil {
		// Log but don't fail - source maps are nice to have
		l.logger.Warn("failed to extract source map from", filename, ":", err)
	}

	return code, sm, nil
}

// loadRules loads transpiled rules into goja and returns the rule runners.
func (l *Loader) loadRules(transpiled []*TranspiledRule, config *Config) ([]baseLinter.RuleRunner[*openapi.OpenAPI], error) {
	var rules []baseLinter.RuleRunner[*openapi.OpenAPI]

	for _, tr := range transpiled {
		// Create a new runtime for each rule file
		// (goja runtimes are not thread-safe)
		rt, err := NewRuntime(config.GetLogger())
		if err != nil {
			return nil, fmt.Errorf("creating runtime for %q: %w", tr.SourceFile, err)
		}

		// Execute the transpiled code
		_, err = rt.RunScript(tr.SourceFile, tr.Code)
		if err != nil {
			return nil, fmt.Errorf("executing %q: %w", tr.SourceFile, err)
		}

		// Get the registered rules
		jsRules := rt.GetRegisteredRules()
		if len(jsRules) == 0 {
			l.logger.Warn("no rules registered in", tr.SourceFile)
			continue
		}

		// Wrap each JS rule as a Go RuleRunner
		for _, jsRule := range jsRules {
			rule, err := NewCustomRule(rt, jsRule, tr.SourceFile, tr.SourceMap, config)
			if err != nil {
				return nil, fmt.Errorf("creating rule from %q: %w", tr.SourceFile, err)
			}
			rules = append(rules, rule)
		}
	}

	return rules, nil
}
