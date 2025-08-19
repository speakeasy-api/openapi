package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ExampleInfo holds information about an example function
type ExampleInfo struct {
	Name        string
	Title       string
	Description string
	Code        string
	Output      string
}

// PackageExamples holds all examples for a package
type PackageExamples struct {
	PackageName string
	Examples    []ExampleInfo
}

func main() {
	if err := updateExamples(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func updateExamples() error {
	fmt.Println("🔄 Updating examples in README files...")

	// Process all packages
	packages := []string{"openapi", "arazzo", "overlay"}

	for _, pkg := range packages {
		if err := processPackage(pkg); err != nil {
			return fmt.Errorf("failed to process package %s: %w", pkg, err)
		}
	}

	fmt.Println("🎉 Examples updated successfully!")
	return nil
}

func processPackage(packageName string) error {
	examplesFile := filepath.Join(packageName, packageName+"_examples_test.go")
	readmeFile := filepath.Join(packageName, "README.md")

	// Check if files exist
	if _, err := os.Stat(examplesFile); os.IsNotExist(err) {
		fmt.Printf("⚠️  No examples file found: %s\n", examplesFile)
		return nil
	}

	if _, err := os.Stat(readmeFile); os.IsNotExist(err) {
		fmt.Printf("⚠️  No README file found: %s\n", readmeFile)
		return nil
	}

	fmt.Printf("📝 Processing examples from %s\n", examplesFile)

	// Parse the examples file
	examples, err := parseExamplesFile(examplesFile)
	if err != nil {
		return fmt.Errorf("failed to parse examples file: %w", err)
	}

	// Generate README content
	content := generateReadmeContent(examples)

	// Update README file
	if err := updateReadmeFile(readmeFile, content); err != nil {
		return fmt.Errorf("failed to update README: %w", err)
	}

	fmt.Printf("✅ Updated %s\n", readmeFile)
	return nil
}

func parseExamplesFile(filename string) ([]ExampleInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var examples []ExampleInfo

	// Walk through all declarations in the order they appear in the file
	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if strings.HasPrefix(fn.Name.Name, "Example_") {
				example, err := extractExample(fset, fn)
				if err != nil {
					fmt.Printf("⚠️  Failed to extract example %s: %v\n", fn.Name.Name, err)
					continue
				}
				examples = append(examples, example)
			}
		}
	}

	return examples, nil
}

func extractExample(fset *token.FileSet, fn *ast.FuncDecl) (ExampleInfo, error) {
	example := ExampleInfo{
		Name: fn.Name.Name,
	}

	// Extract title and description from function comment
	if fn.Doc != nil {
		example.Title, example.Description = parseDocComment(fn.Doc.Text())
	}

	// If no title from comment, generate from function name
	if example.Title == "" {
		example.Title = generateTitleFromName(fn.Name.Name)
	}

	// Extract function body
	if fn.Body != nil {
		var buf bytes.Buffer
		if err := format.Node(&buf, fset, fn.Body); err != nil {
			return example, err
		}

		// Clean up the function body
		code := buf.String()
		code = strings.TrimPrefix(code, "{")
		code = strings.TrimSuffix(code, "}")
		code = strings.TrimSpace(code)

		// Remove one level of indentation
		lines := strings.Split(code, "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "\t") {
				lines[i] = line[1:]
			}
		}
		example.Code = strings.Join(lines, "\n")

		// Extract output comment if present
		example.Output = extractOutputComment(example.Code)
	}

	return example, nil
}

func parseDocComment(comment string) (title, description string) {
	lines := strings.Split(strings.TrimSpace(comment), "\n")
	if len(lines) == 0 {
		return "", ""
	}

	// First line is typically the title
	title = strings.TrimSpace(lines[0])

	// Extract title from comment patterns
	if strings.Contains(title, " demonstrates ") {
		parts := strings.Split(title, " demonstrates ")
		if len(parts) > 1 {
			title = strings.TrimSpace(parts[1])
			// Remove "how to " prefix and trailing periods
			title = strings.TrimPrefix(title, "how to ")
			title = strings.TrimSuffix(title, ".")
			// Capitalize first letter
			if len(title) > 0 {
				title = strings.ToUpper(title[:1]) + title[1:]
			}
		}
	}

	// Rest is description
	if len(lines) > 1 {
		description = strings.TrimSpace(strings.Join(lines[1:], "\n"))
	}

	return title, description
}

func generateTitleFromName(funcName string) string {
	// Remove "Example_" prefix
	name := strings.TrimPrefix(funcName, "Example_")

	// Convert camelCase to Title Case
	re := regexp.MustCompile(`([a-z])([A-Z])`)
	name = re.ReplaceAllString(name, "$1 $2")

	// Capitalize first letter
	if len(name) > 0 {
		name = strings.ToUpper(name[:1]) + name[1:]
	}

	return name
}

func extractOutputComment(code string) string {
	lines := strings.Split(code, "\n")
	var outputLines []string
	inOutput := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "// Output:" {
			inOutput = true
			continue
		}
		if inOutput {
			if strings.HasPrefix(trimmed, "//") {
				// Remove comment prefix and add to output
				output := strings.TrimPrefix(trimmed, "//")
				output = strings.TrimSpace(output)
				outputLines = append(outputLines, output)
			} else if trimmed != "" {
				// Non-comment line, stop collecting output
				break
			}
		}
	}

	return strings.Join(outputLines, "\n")
}

func generateReadmeContent(examples []ExampleInfo) string {
	var content strings.Builder

	// Generate content in the order examples appear in the file
	for _, example := range examples {
		content.WriteString(fmt.Sprintf("## %s\n\n", example.Title))

		// Add description if available
		if example.Description != "" {
			content.WriteString(example.Description)
			content.WriteString("\n\n")
		}

		content.WriteString("```go\n")
		content.WriteString(example.Code)
		content.WriteString("\n```\n\n")
	}

	return content.String()
}

func updateReadmeFile(filename, newContent string) error {
	// Read the current README
	data, err := os.ReadFile(filename) //nolint:gosec
	if err != nil {
		return err
	}

	content := string(data)

	// Find the start and end markers
	startMarker := "<!-- START USAGE EXAMPLES -->"
	endMarker := "<!-- END USAGE EXAMPLES -->"

	startIdx := strings.Index(content, startMarker)
	endIdx := strings.Index(content, endMarker)

	if startIdx == -1 || endIdx == -1 {
		return fmt.Errorf("could not find usage examples markers in %s", filename)
	}

	// Replace the content between markers
	before := content[:startIdx+len(startMarker)]
	after := content[endIdx:]

	newFileContent := before + "\n\n" + newContent + after

	// Write the updated content
	return os.WriteFile(filename, []byte(newFileContent), 0600)
}
