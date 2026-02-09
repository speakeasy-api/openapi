# Agent Development Guidelines

This document provides guidelines for AI agents working on this codebase.

## Running Tests

This project uses [mise](https://mise.jdx.dev/) for running tests with enhanced output formatting via gotestsum.

### Run All Tests

```bash
mise test
```

This runs all tests in the project with race detection enabled and provides clean, organized test output.

### Run Tests for Specific Packages

The `mise test` command accepts the same arguments as `go test`, allowing you to target specific packages or use any `go test` flags:

```bash
# Run tests for a specific package
mise test ./openapi/core

# Run tests matching a pattern
mise test -run TestGetMapKeyNodeOrRoot ./openapi/core

# Run tests with verbose output
mise test -v ./marshaller

# Run tests for multiple packages
mise test ./openapi/core ./marshaller

# Use any go test flags
mise test -race -count=1 ./...
```

### Common Test Commands

```bash
# Run all tests in current directory
mise test .

# Run specific test function
mise test -run TestSecurityRequirement_GetMapKeyNodeOrRoot_Success ./openapi/core

# Run tests with coverage
mise run test-coverage

# Run tests without cache
mise test -count=1 ./...
```

### Why Use Mise for Testing?

- **Enhanced Output**: Uses gotestsum for better formatted, more readable test results
- **Consistent Environment**: Ensures correct Go version and tool versions
- **Race Detection**: Automatically enables race detection to catch concurrency issues
- **Submodule Awareness**: Checks for and warns about uninitialized test submodules

## Pre-Commit CI Check

**Always run `mise ci` before committing changes.** This runs the full CI pipeline locally (format, lint, test, build) and ensures your changes won't break CI.

```bash
mise ci
```

## Git Commit Conventions

**Always use single-line conventional commits.** Do not create multi-line commit messages. Do not add `Co-Authored-By` trailers.

### Commit Message Format

```
<type>: <description>
```

### Common Types

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `refactor:` - Code refactoring
- `test:` - Adding or updating tests
- `chore:` - Maintenance tasks
- `perf:` - Performance improvements

### Examples

#### ✅ Good: Single-line conventional commits

```bash
git commit -m "feat: add prefixEncoding and itemEncoding support for OpenAPI 3.2 multipart media types"
git commit -m "fix: correct validation logic for encoding field mutual exclusivity"
git commit -m "test: add comprehensive tests for multipart encoding validation"
git commit -m "refactor: simplify media type context passing in validation"
```

#### ❌ Bad: Multi-line commits

```bash
git commit -m "feat: implement prefixEncoding and itemEncoding for OpenAPI 3.2

- Add PrefixEncoding and ItemEncoding fields to MediaType
- Implement validation for mutual exclusivity
- Add comprehensive tests"
```

### Why Single-Line Commits?

1. **Simplicity**: Easy to read in git log and GitHub UI
2. **Consistency**: All commits follow the same pattern
3. **Searchability**: Easier to search and filter commits
4. **Tool Compatibility**: Works better with automated tools and scripts

## Linter Rules

This project uses `golangci-lint` with strict rules. Run `mise lint` to check. The most common violations are listed below. **When you encounter a new common lint pattern not documented here, add it to this section so future sessions avoid the same mistakes.**

### perfsprint — Avoid `fmt.Sprintf` for Simple String Operations

The `perfsprint` linter flags unnecessary `fmt.Sprintf` calls. Use string concatenation or `strconv` instead.

#### ❌ Bad

```go
// Single %s — just use concatenation
msg := fmt.Sprintf("prefix: %s", value)

// Single %d — use strconv
msg := fmt.Sprintf("%d", count)

// Writing formatted string to a writer
b.WriteString(fmt.Sprintf("hello %s world %d", name, n))
```

#### ✅ Good

```go
// String concatenation
msg := "prefix: " + value

// strconv for numbers
msg := strconv.Itoa(count)

// fmt.Fprintf writes directly to the writer
fmt.Fprintf(b, "hello %s world %d", name, n)

// For string-only format with multiple args, concatenation is fine
b.WriteString(indent + "const x = " + varName + ";\n")
```

**Rule of thumb:** If `fmt.Sprintf` has a single `%s` or `%d` verb and nothing else complex, replace it with concatenation or `strconv`. If writing to an `io.Writer`/`strings.Builder`, use `fmt.Fprintf` directly instead of `WriteString(fmt.Sprintf(...))`.

### staticcheck — Common Issues

- **QF1012**: Use `fmt.Fprintf(w, ...)` instead of `w.WriteString(fmt.Sprintf(...))` — writes directly to the writer without an intermediate string allocation.
- **QF1003**: Use tagged `switch` instead of `if-else` chains on the same variable.
- **S1016**: Use type conversion `TargetType(value)` instead of struct literal when types have identical fields.

### predeclared — Don't Shadow Built-in Identifiers

Avoid using `min`, `max`, `new`, `len`, `cap`, `copy`, `delete`, `error`, `any` as variable names. Use descriptive alternatives like `minVal`, `maxVal`.

### testifylint — Test Assertion Best Practices

- Use `assert.Empty(t, val)` instead of `assert.Equal(t, "", val)`
- Use `assert.True(t, val)` / `assert.False(t, val)` instead of `assert.Equal(t, true/false, val)`
- Use `require.Error(t, err)` instead of `assert.Error(t, err)` for error checks
- Use `assert.Len(t, slice, n)` instead of `assert.Equal(t, n, len(slice))`

### gocritic — Code Style

- Convert `if-else if` chains to `switch` statements when comparing the same variable.

## Testing

Follow these testing conventions when writing Go tests in this project. Run newly added or modified test immediately after changes to make sure they work as expected before continuing with more work.

### Test File Organization

**Keep tests localized to the files they are testing.** Each source file should have a corresponding test file in the same directory.

- `responses.go` → `responses_test.go`
- `paths.go` → `paths_test.go`
- `security.go` → `security_test.go`

This makes it easy to find tests and understand what functionality is being tested.

### Test Simplicity

**Keep tests simple by avoiding branching logic.** Tests should be straightforward and easy to understand.

#### ❌ Bad: Branching in tests

```go
func TestExample(t *testing.T) {
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var model Model
            if tt.shouldInitialize {
                model = initializeModel()
            } else {
                model = Model{}
            }
            // test logic...
        })
    }
}
```

#### ✅ Good: Separate test functions

```go
func TestExample_Initialized(t *testing.T) {
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            model := initializeModel()
            // test logic...
        })
    }
}

func TestExample_Uninitialized(t *testing.T) {
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            model := Model{}
            // test logic...
        })
    }
}
```

### Parallel Test Execution

**Always use `t.Parallel()` for parallel test execution.** This speeds up test runs and ensures tests are independent.

```go
func TestExample_Success(t *testing.T) {
    t.Parallel()  // At the top level

    tests := []struct {
        name string
        // ...
    }{
        // test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()  // In each subtest
            // test logic...
        })
    }
}
```

### Context Usage

**Use `t.Context()` instead of `context.Background()`.** This provides better test lifecycle management and cancellation.

#### ❌ Bad

```go
ctx := context.Background()
```

#### ✅ Good

```go
ctx := t.Context()
```

### Table-Driven Tests

Use table-driven tests where possible and when they make sense (don't over-complicate the main test implementation).

```go
func TestFeature_Success(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name     string
        input    InputType
        expected ExpectedType
    }{
        {
            name:     "descriptive test case name",
            input:    // test input,
            expected: // expected output,
        },
        // more test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            ctx := t.Context()
            
            actual := FunctionUnderTest(ctx, tt.input)
            assert.Equal(t, tt.expected, actual, "should return expected value")
        })
    }
}
```

### Test Function Naming

Use `_Success` and `_Error` (or `_ReturnsRoot`, `_ReturnsDefault`, etc.) suffixes to denote different test scenarios.

#### Examples

- `TestGetMapKeyNodeOrRoot_Success` - Tests happy path scenarios
- `TestGetMapKeyNodeOrRoot_ReturnsRoot` - Tests when root is returned
- `TestParseConfig_Success` - Tests successful parsing
- `TestParseConfig_Error` - Tests parsing failures

### Assertions

Use the testify assert/require libraries for cleaner assertions.

```go
import (
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)
```

#### Usage Guidelines

- Use `assert.Equal()` for value comparisons with descriptive messages
- Use `assert.Nil()` and `assert.NotNil()` for pointer checks
- Use `require.*` when the test should stop on failure (e.g., setup operations)
- **Use `require.Error()` for error assertions** - The linter enforces this via testifylint
- **Always include descriptive error messages**

```go
// Good: Clear assertions with messages
require.NoError(t, err, "unmarshal should succeed")
require.NotNil(t, result, "result should not be nil")
assert.Equal(t, expected, actual, "should return correct value")
```

### Exact Object Assertions

**Assert against exact objects rather than using complex setup functions.** This makes tests clearer and easier to debug.

#### ❌ Bad: Complex setup with branching

```go
tests := []struct {
    name  string
    setup func() *Model
}{
    {
        name: "test case",
        setup: func() *Model {
            if someCondition {
                return &Model{Field: "value1"}
            }
            return &Model{Field: "value2"}
        },
    },
}
```

#### ✅ Good: Direct object creation

```go
tests := []struct {
    name     string
    yaml     string
    key      string
    expected string
}{
    {
        name:     "returns key when exists",
        yaml:     `key: value`,
        key:      "key",
        expected: "key",
    },
}
```

### Leverage Existing Project Packages

**Use existing project packages for test setup instead of reinventing the wheel.** The project provides utilities for common testing needs.

#### Marshaller Package

Use `marshaller.UnmarshalCore()` to create properly initialized core models:

```go
func TestCoreModel_Success(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name string
        yaml string
        key  string
    }{
        {
            name: "test case",
            yaml: `
key1: value1
key2: value2
`,
            key: "key1",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            ctx := t.Context()
            
            var model CoreModel
            _, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &model)
            require.NoError(t, err, "unmarshal should succeed")

            // Test logic using the model
            result := model.SomeMethod(tt.key)
            assert.Equal(t, tt.key, result.Value, "should return correct value")
        })
    }
}

// Helper function for parsing YAML
func parseYAML(t *testing.T, yml string) *yaml.Node {
    t.Helper()
    var node yaml.Node
    err := yaml.Unmarshal([]byte(yml), &node)
    require.NoError(t, err)
    return &node
}
```

#### YML Package

Use the `yml` package for creating and manipulating YAML nodes:

```go
import "github.com/speakeasy-api/openapi/yml"

// Create scalar nodes
stringNode := yml.CreateStringNode("value")
intNode := yml.CreateIntNode(42)
boolNode := yml.CreateBoolNode(true)

// Create map nodes
ctx := t.Context()
mapNode := yml.CreateMapNode(ctx, []*yaml.Node{
    yml.CreateStringNode("key1"),
    yml.CreateStringNode("value1"),
})

// Get map elements
keyNode, valueNode, found := yml.GetMapElementNodes(ctx, mapNode, "key1")
```

#### General Principles

- **Don't recreate existing functionality** - Check if the project already has utilities for what you need
- **Use project-specific helpers** - Packages like `marshaller`, `yml`, `sequencedmap`, etc. provide tested utilities
- **Follow existing patterns** - Look at how other tests in the project construct test data
- **Reuse helper functions** - If a test file has a `parseYAML` helper, use it rather than duplicating

#### Examples of Project Packages to Leverage

- `marshaller` - For unmarshalling and working with models
- `yml` - For creating and manipulating YAML nodes
- `sequencedmap` - For creating ordered maps
- `extensions` - For working with OpenAPI extensions
- `validation` - For validation utilities

### Test Coverage

Test cases should cover:

- **Happy path scenarios** - Various valid inputs
- **Edge cases** - Empty inputs, boundary values
- **Error conditions** - Nil inputs, invalid parameters
- **Integration scenarios** - Where applicable

### Why These Conventions Matter

1. **Consistency**: All tests follow the same pattern, making them easier to read and maintain
2. **Clarity**: Clear naming and simple logic make it obvious what each test covers
3. **Maintainability**: Table tests make it easy to add new test cases
4. **Performance**: Parallel execution speeds up test runs
5. **Debugging**: testify assertions and clear structure provide helpful failure messages
6. **Reliability**: Using `t.Context()` ensures proper test lifecycle management
