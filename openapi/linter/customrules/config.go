package customrules

import (
	"fmt"
	"os"
	"time"
)

// Config configures custom rule loading.
type Config struct {
	// Paths are glob patterns for rule files (e.g., "./rules/*.ts")
	Paths []string `yaml:"paths,omitempty" json:"paths,omitempty"`

	// Timeout is the maximum execution time per rule (default: 30s)
	Timeout time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`

	// Logger is used for console output from rules (programmatic only, not from YAML)
	Logger Logger `yaml:"-" json:"-"`
}

// DefaultTimeout is the default execution timeout for custom rules.
const DefaultTimeout = 30 * time.Second

// GetTimeout returns the configured timeout or the default.
func (c *Config) GetTimeout() time.Duration {
	if c == nil || c.Timeout == 0 {
		return DefaultTimeout
	}
	return c.Timeout
}

// GetLogger returns the configured logger or the default.
func (c *Config) GetLogger() Logger {
	if c == nil || c.Logger == nil {
		return &defaultLogger{}
	}
	return c.Logger
}

// Logger is the interface for custom rule console output.
type Logger interface {
	Log(args ...any)
	Warn(args ...any)
	Error(args ...any)
}

// defaultLogger writes to stderr with prefixes.
type defaultLogger struct{}

func (l *defaultLogger) Log(args ...any) {
	fmt.Fprintln(os.Stderr, "[custom-rule]", fmt.Sprint(args...))
}

func (l *defaultLogger) Warn(args ...any) {
	fmt.Fprintln(os.Stderr, "[custom-rule][WARN]", fmt.Sprint(args...))
}

func (l *defaultLogger) Error(args ...any) {
	fmt.Fprintln(os.Stderr, "[custom-rule][ERROR]", fmt.Sprint(args...))
}
