package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/spf13/cobra"
)

// joinWriteInPlace controls whether to write the joined document back to the main file
var joinWriteInPlace bool

// joinStrategy controls the conflict resolution strategy for joined components
var joinStrategy string

var joinCmd = &cobra.Command{
	Use:   "join <main-file> <document1> [document2...] [output-file]",
	Short: "Join multiple OpenAPI documents into a single document",
	Long: `Join combines multiple OpenAPI documents into a single unified document with intelligent conflict resolution.

This command merges OpenAPI specifications by:
• Combining all paths, components, and operations from multiple documents
• Resolving naming conflicts using configurable strategies
• Handling servers and security requirements intelligently
• Preserving external references while joining documents
• Maintaining document integrity and validation

The join operation supports two conflict resolution strategies:
• counter: Uses counter-based suffixes like User_1, User_2 for conflicts
• filepath: Uses file path-based naming like second_yaml~User

Smart conflict handling:
• Components: Identical components are merged, conflicts are renamed
• Operations: Path conflicts use fragment-based naming (/users~1)
• Servers/Security: Conflicts push settings to operation level
• Tags: Unique tags are appended, identical tags are preserved

Examples:
  # Join to stdout (pipe-friendly)
  openapi spec join ./main.yaml ./api1.yaml ./api2.yaml

  # Join to specific file
  openapi spec join ./main.yaml ./api1.yaml ./api2.yaml ./joined.yaml

  # Join in-place with counter strategy
  openapi spec join -w --strategy counter ./main.yaml ./api1.yaml

  # Join with filepath strategy (default)
  openapi spec join --strategy filepath ./main.yaml ./api1.yaml ./joined.yaml`,
	Args: cobra.MinimumNArgs(2),
	RunE: runJoinCommand,
}

func init() {
	joinCmd.Flags().BoolVarP(&joinWriteInPlace, "write", "w", false, "Write joined document back to main file")
	joinCmd.Flags().StringVar(&joinStrategy, "strategy", "counter", "Conflict resolution strategy (counter|filepath)")
}

func runJoinCommand(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Parse arguments - last arg might be output file if it doesn't exist as input
	mainFile := args[0]
	var documentFiles []string
	var outputFile string

	// Determine if last argument is an output file (doesn't exist) or input file (exists)
	if len(args) >= 3 {
		lastArg := args[len(args)-1]
		if _, err := os.Stat(lastArg); os.IsNotExist(err) {
			// Last argument doesn't exist, treat as output file
			documentFiles = args[1 : len(args)-1]
			outputFile = lastArg
		} else {
			// All arguments are input files
			documentFiles = args[1:]
		}
	} else {
		// Only main file and one document file
		documentFiles = args[1:]
	}

	// Validate strategy
	var strategy openapi.JoinConflictStrategy
	switch joinStrategy {
	case "counter":
		strategy = openapi.JoinConflictCounter
	case "filepath":
		strategy = openapi.JoinConflictFilePath
	default:
		return fmt.Errorf("invalid strategy: %s (must be 'counter' or 'filepath')", joinStrategy)
	}

	// Create processor
	processor, err := NewOpenAPIProcessor(mainFile, outputFile, joinWriteInPlace)
	if err != nil {
		return err
	}

	// Load main document
	mainDoc, validationErrors, err := processor.LoadDocument(ctx)
	if err != nil {
		return err
	}

	// Report validation errors for main document
	processor.ReportValidationErrors(validationErrors)

	// Load additional documents
	var documents []*openapi.OpenAPI
	var filePaths []string

	for _, docFile := range documentFiles {
		// Create a temporary processor for each document to load it
		docProcessor, err := NewOpenAPIProcessor(docFile, "", false)
		if err != nil {
			return fmt.Errorf("failed to create processor for %s: %w", docFile, err)
		}

		doc, docValidationErrors, err := docProcessor.LoadDocument(ctx)
		if err != nil {
			return fmt.Errorf("failed to load document %s: %w", docFile, err)
		}

		// Report validation errors for this document
		if len(docValidationErrors) > 0 && !processor.WriteToStdout {
			fmt.Printf("⚠️  Found %d validation errors in %s:\n", len(docValidationErrors), docFile)
			for i, validationErr := range docValidationErrors {
				fmt.Printf("  %d. %s\n", i+1, validationErr.Error())
			}
			fmt.Println()
		}

		documents = append(documents, doc)
		filePaths = append(filePaths, docFile)
	}

	// Prepare join options
	opts := openapi.JoinOptions{
		ConflictStrategy: strategy,
	}

	if strategy == openapi.JoinConflictFilePath {
		// Create document path mappings for filepath strategy
		opts.DocumentPaths = make(map[int]string)
		for i, path := range filePaths {
			opts.DocumentPaths[i] = path
		}
	}

	// Prepare document info slice
	var documentInfos []openapi.JoinDocumentInfo
	mainDir := filepath.Dir(mainFile)

	for i, doc := range documents {
		docInfo := openapi.JoinDocumentInfo{
			Document: doc,
		}
		if i < len(filePaths) {
			// Compute relative path from main document's directory
			relPath, err := filepath.Rel(mainDir, filePaths[i])
			if err != nil {
				// If we can't compute relative path, use the original path
				docInfo.FilePath = filePaths[i]
			} else {
				docInfo.FilePath = relPath
			}
		}
		documentInfos = append(documentInfos, docInfo)
	}

	// Perform the join operation (modifies mainDoc in place)
	if err := openapi.Join(ctx, mainDoc, documentInfos, opts); err != nil {
		return fmt.Errorf("failed to join documents: %w", err)
	}

	// Print success message
	processor.PrintSuccess(fmt.Sprintf("Successfully joined %d documents with %s strategy", len(documents)+1, joinStrategy))

	// Write the joined document (mainDoc was modified in place)
	if err := processor.WriteDocument(ctx, mainDoc); err != nil {
		return err
	}

	return nil
}
