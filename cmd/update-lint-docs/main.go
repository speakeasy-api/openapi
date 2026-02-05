package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	openapiLinter "github.com/speakeasy-api/openapi/openapi/linter"
)

func main() {
	if err := updateLintDocs(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func updateLintDocs() error {
	fmt.Println("🔄 Updating lint rules in README files...")

	if err := updateOpenAPILintDocs(); err != nil {
		return fmt.Errorf("failed to update OpenAPI lint docs: %w", err)
	}

	if err := updateRuleLinks(); err != nil {
		return fmt.Errorf("failed to update rule links: %w", err)
	}

	fmt.Println("🎉 Lint docs updated successfully!")
	return nil
}

func updateOpenAPILintDocs() error {
	readmeFile := "openapi/linter/README.md"

	// Check if README exists
	if _, err := os.Stat(readmeFile); os.IsNotExist(err) {
		fmt.Printf("⚠️  No README file found: %s\n", readmeFile)
		return nil
	}

	// Create linter to get the registry
	config := linter.NewConfig()
	lint, err := openapiLinter.NewLinter(config)
	if err != nil {
		return fmt.Errorf("failed to create linter: %w", err)
	}
	docGen := linter.NewDocGenerator(lint.Registry())

	// Generate rules table
	content := generateRulesTable(docGen)

	// Update README file
	if err := updateReadmeFile(readmeFile, content); err != nil {
		return fmt.Errorf("failed to update README: %w", err)
	}

	fmt.Printf("✅ Updated %s\n", readmeFile)
	return nil
}

func generateRulesTable(docGen *linter.DocGenerator[*openapi.OpenAPI]) string {
	docs := docGen.GenerateAllRuleDocs()

	// Sort rules alphabetically by ID
	sort.Slice(docs, func(i, j int) bool {
		return docs[i].ID < docs[j].ID
	})

	var content strings.Builder
	content.WriteString("| Rule | Severity | Description |\n")
	content.WriteString("|------|----------|-------------|\n")

	for _, doc := range docs {
		// Escape pipe characters in description
		desc := strings.ReplaceAll(doc.Description, "|", "\\|")
		// Replace newlines with spaces
		desc = strings.ReplaceAll(desc, "\n", " ")
		content.WriteString(fmt.Sprintf("| <a name=\"%s\"></a>`%s` | %s | %s |\n", doc.ID, doc.ID, doc.DefaultSeverity, desc))
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
	startMarker := "<!-- START LINT RULES -->"
	endMarker := "<!-- END LINT RULES -->"

	startIdx := strings.Index(content, startMarker)
	endIdx := strings.Index(content, endMarker)

	if startIdx == -1 || endIdx == -1 {
		return fmt.Errorf("could not find lint rules markers in %s", filename)
	}

	// Replace the content between markers
	before := content[:startIdx+len(startMarker)]
	after := content[endIdx:]

	newFileContent := before + "\n\n" + newContent + "\n" + after

	// Write the updated content
	return os.WriteFile(filename, []byte(newFileContent), 0600)
}

func updateRuleLinks() error {
	const baseURL = "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md"
	rulesDir := "openapi/linter/rules"

	// Get all rule files
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		return fmt.Errorf("failed to read rules directory: %w", err)
	}

	// Pattern to match Link() method - captures receiver and return value
	linkPattern := regexp.MustCompile(`func (\([^)]+\)) Link\(\) string \{\s*return "[^"]*"\s*\}`)

	updatedCount := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		filePath := filepath.Join(rulesDir, entry.Name())

		// Read the file
		data, err := os.ReadFile(filePath) //nolint:gosec
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", filePath, err)
		}

		content := string(data)

		// Find the rule ID constant
		ruleIDPattern := regexp.MustCompile(`const (Rule\w+) = "([^"]+)"`)
		matches := ruleIDPattern.FindStringSubmatch(content)
		if len(matches) < 3 {
			continue // Skip if no rule ID found
		}
		ruleID := matches[2]

		// Create the new link
		newLink := fmt.Sprintf("%s#%s", baseURL, ruleID)

		// Replace the Link() method, preserving the receiver
		newContent := linkPattern.ReplaceAllStringFunc(content, func(match string) string {
			receiverMatch := regexp.MustCompile(`func (\([^)]+\))`).FindStringSubmatch(match)
			if len(receiverMatch) > 1 {
				return fmt.Sprintf(`func %s Link() string {
	return "%s"
}`, receiverMatch[1], newLink)
			}
			return match
		})

		// Only write if content changed
		if newContent != content {
			if err := os.WriteFile(filePath, []byte(newContent), 0600); err != nil {
				return fmt.Errorf("failed to write %s: %w", filePath, err)
			}
			updatedCount++
			fmt.Printf("✅ Updated link in %s\n", filePath)
		}
	}

	fmt.Printf("✅ Updated links in %d rule files\n", updatedCount)
	return nil
}
