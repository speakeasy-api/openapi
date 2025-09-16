package openapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/internal/utils"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/system"
	"gopkg.in/yaml.v3"
)

// LocalizeNamingStrategy defines how external reference files should be named when localized.
type LocalizeNamingStrategy int

const (
	// LocalizeNamingPathBased uses path-based naming like "schemas-address.yaml" for conflicts
	LocalizeNamingPathBased LocalizeNamingStrategy = iota
	// LocalizeNamingCounter uses counter-based suffixes like "address_1.yaml" for conflicts
	LocalizeNamingCounter
)

// LocalizeOptions represents the options available when localizing an OpenAPI document.
type LocalizeOptions struct {
	// ResolveOptions are the options to use when resolving references during localization.
	ResolveOptions ResolveOptions
	// TargetDirectory is the directory where localized files will be written.
	TargetDirectory string
	// VirtualFS is the file system interface used for reading and writing files.
	VirtualFS system.WritableVirtualFS
	// NamingStrategy determines how external reference files are named when localized.
	NamingStrategy LocalizeNamingStrategy
}

// Localize transforms an OpenAPI document by copying all external reference files to a target directory
// and rewriting the references in the document to point to the localized files.
// This operation modifies the document in place.
//
// Why use Localize?
//
//   - **Create portable document bundles**: Copy all external dependencies into a single directory
//   - **Simplify deployment**: Package all API definition files together for easy distribution
//   - **Offline development**: Work with API definitions without external file dependencies
//   - **Version control**: Keep all related files in the same repository structure
//   - **CI/CD pipelines**: Ensure all dependencies are available in build environments
//   - **Documentation generation**: Bundle all files needed for complete API documentation
//
// What you'll get:
//
// Before localization:
//
//	main.yaml:
//	  paths:
//	    /users:
//	      get:
//	        responses:
//	          '200':
//	            content:
//	              application/json:
//	                schema:
//	                  $ref: "./components.yaml#/components/schemas/User"
//
//	components.yaml:
//	  components:
//	    schemas:
//	      User:
//	        properties:
//	          address:
//	            $ref: "./schemas/address.yaml#/Address"
//
// After localization (files copied to target directory):
//
//	target/main.yaml:
//	  paths:
//	    /users:
//	      get:
//	        responses:
//	          '200':
//	            content:
//	              application/json:
//	                schema:
//	                  $ref: "components.yaml#/components/schemas/User"
//
//	target/components.yaml:
//	  components:
//	    schemas:
//	      User:
//	        properties:
//	          address:
//	            $ref: "schemas-address.yaml#/Address"
//
//	target/schemas-address.yaml:
//	  Address:
//	    type: object
//	    properties:
//	      street: {type: string}
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - doc: The OpenAPI document to localize (modified in place)
//   - opts: Configuration options for localization
//
// Returns:
//   - error: Any error that occurred during localization
func Localize(ctx context.Context, doc *OpenAPI, opts LocalizeOptions) error {
	if doc == nil {
		return nil
	}

	if opts.VirtualFS == nil {
		opts.VirtualFS = &system.FileSystem{}
	}

	if opts.TargetDirectory == "" {
		return errors.New("target directory is required")
	}

	// Storage for tracking external references and their localized names
	localizeStorage := &localizeStorage{
		externalRefs:    sequencedmap.New[string, string](), // original ref -> localized filename
		usedFilenames:   make(map[string]bool),              // track used filenames to avoid conflicts
		resolvedContent: make(map[string][]byte),            // original ref -> file content
	}

	// Phase 1: Discover and collect all external references
	if err := discoverExternalReferences(ctx, doc, opts.ResolveOptions, localizeStorage); err != nil {
		return fmt.Errorf("failed to discover external references: %w", err)
	}

	// Phase 2: Generate conflict-free filenames for all external references
	generateLocalizedFilenames(localizeStorage, opts.NamingStrategy)

	// Phase 3: Copy external files to target directory
	if err := copyExternalFiles(ctx, localizeStorage, opts); err != nil {
		return fmt.Errorf("failed to copy external files: %w", err)
	}

	// Phase 4: Rewrite references in the document
	if err := rewriteReferencesToLocalized(ctx, doc, localizeStorage); err != nil {
		return fmt.Errorf("failed to rewrite references: %w", err)
	}

	return nil
}

type localizeStorage struct {
	externalRefs    *sequencedmap.Map[string, string] // original ref -> localized filename
	usedFilenames   map[string]bool                   // track used filenames to avoid conflicts
	resolvedContent map[string][]byte                 // original ref -> file content
}

// discoverExternalReferences walks through the document and collects all external references
func discoverExternalReferences(ctx context.Context, doc *OpenAPI, opts ResolveOptions, storage *localizeStorage) error {
	for item := range Walk(ctx, doc) {
		err := item.Match(Matcher{
			Schema: func(schema *oas3.JSONSchema[oas3.Referenceable]) error {
				return discoverSchemaReference(ctx, schema, opts, storage)
			},
			ReferencedPathItem: func(ref *ReferencedPathItem) error {
				return discoverGenericReference(ctx, ref, opts, storage)
			},
			ReferencedParameter: func(ref *ReferencedParameter) error {
				return discoverGenericReference(ctx, ref, opts, storage)
			},
			ReferencedExample: func(ref *ReferencedExample) error {
				return discoverGenericReference(ctx, ref, opts, storage)
			},
			ReferencedRequestBody: func(ref *ReferencedRequestBody) error {
				return discoverGenericReference(ctx, ref, opts, storage)
			},
			ReferencedResponse: func(ref *ReferencedResponse) error {
				return discoverGenericReference(ctx, ref, opts, storage)
			},
			ReferencedHeader: func(ref *ReferencedHeader) error {
				return discoverGenericReference(ctx, ref, opts, storage)
			},
			ReferencedCallback: func(ref *ReferencedCallback) error {
				return discoverGenericReference(ctx, ref, opts, storage)
			},
			ReferencedLink: func(ref *ReferencedLink) error {
				return discoverGenericReference(ctx, ref, opts, storage)
			},
			ReferencedSecurityScheme: func(ref *ReferencedSecurityScheme) error {
				return discoverGenericReference(ctx, ref, opts, storage)
			},
		})
		if err != nil {
			return fmt.Errorf("failed to discover references at %s: %w", item.Location.ToJSONPointer().String(), err)
		}
	}

	return nil
}

// discoverSchemaReference handles discovery of JSON schema references
func discoverSchemaReference(ctx context.Context, schema *oas3.JSONSchema[oas3.Referenceable], opts ResolveOptions, storage *localizeStorage) error {
	if !schema.IsReference() {
		return nil
	}

	ref, classification := handleReference(schema.GetRef(), "", opts.TargetLocation)
	if classification == nil || classification.IsFragment {
		return nil // Skip internal references
	}

	// Get the URI part (file path) from the reference
	refObj := references.Reference(ref)
	filePath := refObj.GetURI()

	// For URLs, use the full reference as the key, for file paths normalize
	var normalizedFilePath string
	if classification.Type == utils.ReferenceTypeURL {
		normalizedFilePath = ref // Use the full URL as the key
	} else {
		normalizedFilePath = normalizeFilePath(filePath)
	}

	// Always resolve the schema to enable recursive discovery
	resolveOpts := oas3.ResolveOptions{
		RootDocument:   opts.RootDocument,
		TargetDocument: opts.TargetDocument,
		TargetLocation: opts.TargetLocation,
		VirtualFS:      opts.VirtualFS,
		HTTPClient:     opts.HTTPClient,
	}

	if _, err := schema.Resolve(ctx, resolveOpts); err != nil {
		return fmt.Errorf("failed to resolve external schema reference %s: %w", ref, err)
	}

	// Only store the file content if we haven't processed this file before
	if !storage.externalRefs.Has(normalizedFilePath) {
		// Get the file content for this reference
		content, err := getFileContentForReference(ctx, normalizedFilePath, opts)
		if err != nil {
			return fmt.Errorf("failed to get content for reference %s: %w", normalizedFilePath, err)
		}

		storage.externalRefs.Set(normalizedFilePath, "") // Will be filled in filename generation phase
		storage.resolvedContent[normalizedFilePath] = content
	}

	// Get the resolved schema and recursively discover references within it
	resolvedSchema := schema.GetResolvedSchema()
	if resolvedSchema != nil {
		// Convert back to referenceable schema for recursive discovery
		resolvedRefSchema := (*oas3.JSONSchema[oas3.Referenceable])(resolvedSchema)

		targetDocInfo := schema.GetReferenceResolutionInfo()

		// Recursively discover references within the resolved schema using oas3.Walk
		for item := range oas3.Walk(ctx, resolvedRefSchema) {
			err := item.Match(oas3.SchemaMatcher{
				Schema: func(s *oas3.JSONSchema[oas3.Referenceable]) error {
					return discoverSchemaReference(ctx, s, ResolveOptions{
						RootDocument:   opts.RootDocument,
						TargetDocument: targetDocInfo.ResolvedDocument,
						TargetLocation: targetDocInfo.AbsoluteReference,
						VirtualFS:      opts.VirtualFS,
						HTTPClient:     opts.HTTPClient,
					}, storage)
				},
			})
			if err != nil {
				return fmt.Errorf("failed to discover nested schema reference: %w", err)
			}
		}
	}

	return nil
}

// discoverGenericReference handles discovery of generic OpenAPI component references
func discoverGenericReference[T any, V interfaces.Validator[T], C marshaller.CoreModeler](ctx context.Context, ref *Reference[T, V, C], opts ResolveOptions, storage *localizeStorage) error {
	if ref == nil || !ref.IsReference() {
		return nil
	}

	refStr, classification := handleReference(ref.GetReference(), "", opts.TargetLocation)
	if classification == nil || classification.IsFragment {
		return nil // Skip internal references
	}

	// Get the URI part (file path) from the reference
	refObj := references.Reference(refStr)
	filePath := refObj.GetURI()

	// For URLs, use the full reference as the key, for file paths normalize
	var normalizedFilePath string
	if classification.Type == utils.ReferenceTypeURL {
		normalizedFilePath = refStr // Use the full URL as the key
	} else {
		normalizedFilePath = normalizeFilePath(filePath)
	}

	// Check if we've already processed this file
	if storage.externalRefs.Has(normalizedFilePath) {
		return nil
	}

	// Resolve the external reference to ensure it's valid
	_, err := ref.Resolve(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to resolve external reference %s: %w", refStr, err)
	}

	// Get the file content for this reference
	content, err := getFileContentForReference(ctx, normalizedFilePath, opts)
	if err != nil {
		return fmt.Errorf("failed to get content for reference %s: %w", normalizedFilePath, err)
	}

	storage.externalRefs.Set(normalizedFilePath, "") // Will be filled in filename generation phase
	storage.resolvedContent[normalizedFilePath] = content

	// Get the resolved object and recursively discover references within it
	resolvedValue := ref.GetObject()
	if resolvedValue != nil {
		targetDocInfo := ref.GetReferenceResolutionInfo()

		// Recursively discover references within the resolved object using Walk
		for item := range Walk(ctx, resolvedValue) {
			err := item.Match(Matcher{
				Schema: func(s *oas3.JSONSchema[oas3.Referenceable]) error {
					return discoverSchemaReference(ctx, s, ResolveOptions{
						RootDocument:   opts.RootDocument,
						TargetDocument: targetDocInfo.ResolvedDocument,
						TargetLocation: targetDocInfo.AbsoluteReference,
						VirtualFS:      opts.VirtualFS,
						HTTPClient:     opts.HTTPClient,
					}, storage)
				},
				ReferencedPathItem: func(r *ReferencedPathItem) error {
					return discoverGenericReference(ctx, r, ResolveOptions{
						RootDocument:   opts.RootDocument,
						TargetDocument: targetDocInfo.ResolvedDocument,
						TargetLocation: targetDocInfo.AbsoluteReference,
						VirtualFS:      opts.VirtualFS,
						HTTPClient:     opts.HTTPClient,
					}, storage)
				},
				ReferencedParameter: func(r *ReferencedParameter) error {
					return discoverGenericReference(ctx, r, ResolveOptions{
						RootDocument:   opts.RootDocument,
						TargetDocument: targetDocInfo.ResolvedDocument,
						TargetLocation: targetDocInfo.AbsoluteReference,
						VirtualFS:      opts.VirtualFS,
						HTTPClient:     opts.HTTPClient,
					}, storage)
				},
				ReferencedExample: func(r *ReferencedExample) error {
					return discoverGenericReference(ctx, r, ResolveOptions{
						RootDocument:   opts.RootDocument,
						TargetDocument: targetDocInfo.ResolvedDocument,
						TargetLocation: targetDocInfo.AbsoluteReference,
						VirtualFS:      opts.VirtualFS,
						HTTPClient:     opts.HTTPClient,
					}, storage)
				},
				ReferencedRequestBody: func(r *ReferencedRequestBody) error {
					return discoverGenericReference(ctx, r, ResolveOptions{
						RootDocument:   opts.RootDocument,
						TargetDocument: targetDocInfo.ResolvedDocument,
						TargetLocation: targetDocInfo.AbsoluteReference,
						VirtualFS:      opts.VirtualFS,
						HTTPClient:     opts.HTTPClient,
					}, storage)
				},
				ReferencedResponse: func(r *ReferencedResponse) error {
					return discoverGenericReference(ctx, r, ResolveOptions{
						RootDocument:   opts.RootDocument,
						TargetDocument: targetDocInfo.ResolvedDocument,
						TargetLocation: targetDocInfo.AbsoluteReference,
						VirtualFS:      opts.VirtualFS,
						HTTPClient:     opts.HTTPClient,
					}, storage)
				},
				ReferencedHeader: func(r *ReferencedHeader) error {
					return discoverGenericReference(ctx, r, ResolveOptions{
						RootDocument:   opts.RootDocument,
						TargetDocument: targetDocInfo.ResolvedDocument,
						TargetLocation: targetDocInfo.AbsoluteReference,
						VirtualFS:      opts.VirtualFS,
						HTTPClient:     opts.HTTPClient,
					}, storage)
				},
				ReferencedCallback: func(r *ReferencedCallback) error {
					return discoverGenericReference(ctx, r, ResolveOptions{
						RootDocument:   opts.RootDocument,
						TargetDocument: targetDocInfo.ResolvedDocument,
						TargetLocation: targetDocInfo.AbsoluteReference,
						VirtualFS:      opts.VirtualFS,
						HTTPClient:     opts.HTTPClient,
					}, storage)
				},
				ReferencedLink: func(r *ReferencedLink) error {
					return discoverGenericReference(ctx, r, ResolveOptions{
						RootDocument:   opts.RootDocument,
						TargetDocument: targetDocInfo.ResolvedDocument,
						TargetLocation: targetDocInfo.AbsoluteReference,
						VirtualFS:      opts.VirtualFS,
						HTTPClient:     opts.HTTPClient,
					}, storage)
				},
				ReferencedSecurityScheme: func(r *ReferencedSecurityScheme) error {
					return discoverGenericReference(ctx, r, ResolveOptions{
						RootDocument:   opts.RootDocument,
						TargetDocument: targetDocInfo.ResolvedDocument,
						TargetLocation: targetDocInfo.AbsoluteReference,
						VirtualFS:      opts.VirtualFS,
						HTTPClient:     opts.HTTPClient,
					}, storage)
				},
			})
			if err != nil {
				return fmt.Errorf("failed to discover nested reference: %w", err)
			}
		}
	}

	return nil
}

// getFileContentForReference retrieves the content of a file referenced by the given file path
func getFileContentForReference(ctx context.Context, filePath string, opts ResolveOptions) ([]byte, error) {
	// First check if this is a URL or file path
	classification, err := utils.ClassifyReference(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to classify reference %s: %w", filePath, err)
	}

	var resolvedPath string
	if classification.Type == utils.ReferenceTypeURL {
		// For URLs, use the path as-is
		resolvedPath = filePath
	} else {
		// For file paths, check if we need to resolve relative to a URL target location
		resolvedPath = filePath
		if !filepath.IsAbs(filePath) && opts.TargetLocation != "" {
			// Check if target location is a URL
			if targetClassification, targetErr := utils.ClassifyReference(opts.TargetLocation); targetErr == nil && targetClassification.Type == utils.ReferenceTypeURL {
				// Resolve relative file path against URL target location
				resolved := resolveRelativeReference(filePath, opts.TargetLocation)
				// Re-classify the resolved reference
				if resolvedClassification, resolvedErr := utils.ClassifyReference(resolved); resolvedErr == nil {
					classification = resolvedClassification
					resolvedPath = resolved
				}
			} else {
				// Resolve relative to the target location directory (file path)
				targetDir := filepath.Dir(opts.TargetLocation)
				resolvedPath = filepath.Join(targetDir, filePath)
			}
		}
	}

	switch classification.Type {
	case utils.ReferenceTypeFilePath:
		// Read from file system
		file, err := opts.VirtualFS.Open(resolvedPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s: %w", resolvedPath, err)
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", resolvedPath, err)
		}

		return content, nil

	case utils.ReferenceTypeURL:
		// Fetch content via HTTP
		httpClient := opts.HTTPClient
		if httpClient == nil {
			httpClient = http.DefaultClient
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, resolvedPath, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP request for %s: %w", resolvedPath, err)
		}

		resp, err := httpClient.Do(req)
		if err != nil || resp == nil {
			return nil, fmt.Errorf("failed to fetch URL %s: %w", resolvedPath, err)
		}
		defer resp.Body.Close()

		// Check if the response was successful
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("HTTP request failed with status %d for URL %s", resp.StatusCode, resolvedPath)
		}

		content, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body from %s: %w", resolvedPath, err)
		}

		return content, nil

	default:
		return nil, fmt.Errorf("unsupported reference type for localization: %s", resolvedPath)
	}
}

// generateLocalizedFilenames creates conflict-free filenames for all external references
func generateLocalizedFilenames(storage *localizeStorage, strategy LocalizeNamingStrategy) {
	// First pass: collect all base filenames to detect conflicts
	baseFilenames := make(map[string][]string) // base filename -> list of full paths
	for ref := range storage.externalRefs.All() {
		refObj := references.Reference(ref)
		filePath := refObj.GetURI()
		baseFilename := filepath.Base(filePath)
		baseFilenames[baseFilename] = append(baseFilenames[baseFilename], ref)
	}

	// Second pass: assign filenames based on conflicts (using deterministic order from sequencedmap)
	processedBaseNames := make(map[string]bool) // track which base names we've processed

	for ref := range storage.externalRefs.All() {
		refObj := references.Reference(ref)
		filePath := refObj.GetURI()
		baseFilename := filepath.Base(filePath)
		conflictingRefs := baseFilenames[baseFilename]

		var filename string
		if len(conflictingRefs) == 1 {
			// No conflicts, use simple filename
			filename = baseFilename
		} else {
			// Has conflicts - for path-based naming, first file gets simple name, others get path prefix
			if strategy == LocalizeNamingPathBased && !processedBaseNames[baseFilename] {
				// First file with this base name gets the simple name
				filename = baseFilename
				processedBaseNames[baseFilename] = true
			} else {
				// Subsequent files or counter strategy get modified names
				filename = generateLocalizedFilenameWithConflictDetection(ref, strategy, baseFilenames, storage.usedFilenames)
			}
		}

		storage.externalRefs.Set(ref, filename)
		storage.usedFilenames[filename] = true
	}
}

// generateLocalizedFilenameWithConflictDetection creates a localized filename with proper conflict detection
func generateLocalizedFilenameWithConflictDetection(ref string, strategy LocalizeNamingStrategy, baseFilenames map[string][]string, usedFilenames map[string]bool) string {
	// Get the file path from the reference
	refObj := references.Reference(ref)
	filePath := refObj.GetURI()
	baseFilename := filepath.Base(filePath)

	// Check if there are conflicts for this base filename
	conflictingRefs := baseFilenames[baseFilename]
	hasConflicts := len(conflictingRefs) > 1

	switch strategy {
	case LocalizeNamingPathBased:
		return generatePathBasedFilenameWithConflictDetection(filePath, hasConflicts, usedFilenames)
	case LocalizeNamingCounter:
		return generateCounterBasedFilename(filePath, usedFilenames)
	default:
		return generatePathBasedFilenameWithConflictDetection(filePath, hasConflicts, usedFilenames)
	}
}

// generatePathBasedFilenameWithConflictDetection creates filenames with smart conflict resolution
func generatePathBasedFilenameWithConflictDetection(filePath string, _ bool, usedFilenames map[string]bool) string {
	// Check if this is a URL - if so, extract filename from URL path
	if classification, err := utils.ClassifyReference(filePath); err == nil && classification.Type == utils.ReferenceTypeURL {
		// For URLs, extract the filename from the URL path
		if lastSlash := strings.LastIndex(filePath, "/"); lastSlash != -1 {
			filename := filePath[lastSlash+1:]
			// Remove any query parameters or fragments
			if queryIdx := strings.Index(filename, "?"); queryIdx != -1 {
				filename = filename[:queryIdx]
			}
			if fragIdx := strings.Index(filename, "#"); fragIdx != -1 {
				filename = filename[:fragIdx]
			}
			return filename
		}
		// Fallback to a safe filename if we can't extract from URL
		return "remote-schema.yaml"
	}

	// Clean the path and get the base filename
	cleanPath := filepath.Clean(filePath)

	// Remove leading "./" if present
	cleanPath = strings.TrimPrefix(cleanPath, "./")

	// Handle parent directory references by replacing ".." with "parent"
	cleanPath = strings.ReplaceAll(cleanPath, "..", "parent")

	// Get the directory and filename
	dir := filepath.Dir(cleanPath)
	filename := filepath.Base(cleanPath)

	// For path-based naming, always use directory prefix if there's a directory
	// This ensures consistent naming regardless of processing order
	var result string
	if dir == "." || dir == "" {
		// No directory, use simple filename
		result = filename
	} else {
		// Replace path separators with hyphens
		dirPart := strings.ReplaceAll(dir, string(filepath.Separator), "-")
		dirPart = strings.ReplaceAll(dirPart, "/", "-")  // Handle forward slashes on Windows
		dirPart = strings.ReplaceAll(dirPart, "\\", "-") // Handle backslashes on Unix

		ext := filepath.Ext(filename)
		baseName := strings.TrimSuffix(filename, ext)
		result = dirPart + "-" + baseName + ext
	}

	// Ensure uniqueness
	originalResult := result
	counter := 1
	for usedFilenames[result] {
		ext := filepath.Ext(originalResult)
		baseName := strings.TrimSuffix(originalResult, ext)
		result = fmt.Sprintf("%s_%d%s", baseName, counter, ext)
		counter++
	}

	return result
}

// generateCounterBasedFilename creates filenames like "address_1.yaml" for conflicts
func generateCounterBasedFilename(filePath string, usedFilenames map[string]bool) string {
	filename := filepath.Base(filePath)

	// Ensure uniqueness
	result := filename
	counter := 1
	for usedFilenames[result] {
		ext := filepath.Ext(filename)
		baseName := strings.TrimSuffix(filename, ext)
		result = fmt.Sprintf("%s_%d%s", baseName, counter, ext)
		counter++
	}

	return result
}

// copyExternalFiles copies all external reference files to the target directory
func copyExternalFiles(_ context.Context, storage *localizeStorage, opts LocalizeOptions) error {
	for ref, localizedFilename := range storage.externalRefs.All() {
		content := storage.resolvedContent[ref]

		// Rewrite internal references within the copied file
		updatedContent, err := rewriteInternalReferences(content, ref, storage)
		if err != nil {
			return fmt.Errorf("failed to rewrite internal references in %s: %w", ref, err)
		}

		targetPath := filepath.Join(opts.TargetDirectory, localizedFilename)

		if err := opts.VirtualFS.WriteFile(targetPath, updatedContent, 0o644); err != nil {
			return fmt.Errorf("failed to write localized file %s: %w", targetPath, err)
		}
	}

	return nil
}

// rewriteInternalReferences updates references within a copied file to point to other localized files
func rewriteInternalReferences(content []byte, originalRef string, storage *localizeStorage) ([]byte, error) {
	// Parse the YAML content
	var node yaml.Node
	if err := yaml.Unmarshal(content, &node); err != nil {
		return nil, fmt.Errorf("failed to parse YAML content: %w", err)
	}

	// Walk through the YAML and update references
	if err := rewriteYAMLReferences(&node, originalRef, storage); err != nil {
		return nil, fmt.Errorf("failed to rewrite YAML references: %w", err)
	}

	// Marshal back to YAML
	updatedContent, err := yaml.Marshal(&node)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated YAML: %w", err)
	}

	return updatedContent, nil
}

// rewriteYAMLReferences recursively walks through YAML nodes and updates $ref values
func rewriteYAMLReferences(node *yaml.Node, originalRef string, storage *localizeStorage) error {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			if err := rewriteYAMLReferences(child, originalRef, storage); err != nil {
				return err
			}
		}
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			// Check if this is a $ref key
			if keyNode.Kind == yaml.ScalarNode && keyNode.Value == "$ref" && valueNode.Kind == yaml.ScalarNode {
				// Update the reference value
				updatedRef := rewriteReferenceValue(valueNode.Value, originalRef, storage)
				valueNode.Value = updatedRef
			} else {
				// Recursively process the value
				if err := rewriteYAMLReferences(valueNode, originalRef, storage); err != nil {
					return err
				}
			}
		}
	case yaml.SequenceNode:
		for _, child := range node.Content {
			if err := rewriteYAMLReferences(child, originalRef, storage); err != nil {
				return err
			}
		}
	}

	return nil
}

// rewriteReferenceValue updates a single reference value to point to localized files
func rewriteReferenceValue(refValue, originalRef string, storage *localizeStorage) string {
	// If this is an internal reference (starts with #), leave it as-is
	if strings.HasPrefix(refValue, "#") {
		return refValue
	}

	// Resolve the reference relative to the original file
	resolvedRef := resolveRelativeReference(refValue, originalRef)

	// Extract the file path from the resolved reference
	refObj := references.Reference(resolvedRef)
	filePath := refObj.GetURI()

	// For URLs, use the full reference as the key, for file paths normalize
	var normalizedFilePath string
	if classification, err := utils.ClassifyReference(resolvedRef); err == nil && classification.Type == utils.ReferenceTypeURL {
		normalizedFilePath = resolvedRef // Use the full URL as the key
	} else {
		normalizedFilePath = normalizeFilePath(filePath)
	}

	// Check if we have a localized version of this file
	if localizedFilename, exists := storage.externalRefs.Get(normalizedFilePath); exists {
		// Build new reference with localized filename
		if refObj.HasJSONPointer() {
			return localizedFilename + "#" + string(refObj.GetJSONPointer())
		}
		return localizedFilename
	}

	// If not found by full URL, try to find by just the reference value itself
	// This handles cases where the reference value is already a full URL
	if localizedFilename, exists := storage.externalRefs.Get(refValue); exists {
		// Build new reference with localized filename
		refObj := references.Reference(refValue)
		if refObj.HasJSONPointer() {
			return localizedFilename + "#" + string(refObj.GetJSONPointer())
		}
		return localizedFilename
	}

	// If not found in our localized references, return as-is
	return refValue
}

// resolveRelativeReference resolves a relative reference against a base reference
func resolveRelativeReference(ref, baseRef string) string {
	// Parse base reference to get the directory
	baseRefObj := references.Reference(baseRef)
	baseURI := baseRefObj.GetURI()

	// Parse the reference
	refObj := references.Reference(ref)
	refPath := refObj.GetURI()

	// Check if the base reference is a URL
	if classification, err := utils.ClassifyReference(baseURI); err == nil && classification.Type == utils.ReferenceTypeURL {
		// For URLs, use URL path joining instead of file path joining
		var resolvedPath string
		switch {
		case strings.HasPrefix(refPath, "./"):
			// Relative reference - resolve against base URL directory
			baseDir := baseURI
			if lastSlash := strings.LastIndex(baseURI, "/"); lastSlash != -1 {
				baseDir = baseURI[:lastSlash+1]
			}
			resolvedPath = baseDir + strings.TrimPrefix(refPath, "./")
		case strings.HasPrefix(refPath, "/"):
			// Absolute path reference - use as-is relative to URL root
			if idx := strings.Index(baseURI, "://"); idx != -1 {
				if hostEnd := strings.Index(baseURI[idx+3:], "/"); hostEnd != -1 {
					resolvedPath = baseURI[:idx+3+hostEnd] + refPath
				} else {
					resolvedPath = baseURI + refPath
				}
			} else {
				resolvedPath = refPath
			}
		default:
			// Simple filename - resolve against base URL directory
			baseDir := baseURI
			if lastSlash := strings.LastIndex(baseURI, "/"); lastSlash != -1 {
				baseDir = baseURI[:lastSlash+1]
			}
			resolvedPath = baseDir + refPath
		}

		// Add back the fragment if present
		if refObj.HasJSONPointer() {
			return resolvedPath + "#" + string(refObj.GetJSONPointer())
		}

		return resolvedPath
	} else {
		// For file paths, use the original file path logic
		baseDir := filepath.Dir(baseURI)

		// Resolve the path
		resolvedPath := filepath.Join(baseDir, refPath)
		resolvedPath = filepath.Clean(resolvedPath)

		// Add back the fragment if present
		if refObj.HasJSONPointer() {
			return resolvedPath + "#" + string(refObj.GetJSONPointer())
		}

		return resolvedPath
	}
}

// rewriteReferencesToLocalized updates all references in the document to point to localized files
func rewriteReferencesToLocalized(ctx context.Context, doc *OpenAPI, storage *localizeStorage) error {
	for item := range Walk(ctx, doc) {
		err := item.Match(Matcher{
			Schema: func(schema *oas3.JSONSchema[oas3.Referenceable]) error {
				if schema.IsReference() {
					ref := schema.GetRef()
					refObj := ref
					filePath := refObj.GetURI()

					// For URLs, use the full reference as the key, for file paths normalize
					var normalizedFilePath string
					if classification, err := utils.ClassifyReference(string(ref)); err == nil && classification.Type == utils.ReferenceTypeURL {
						normalizedFilePath = string(ref) // Use the full URL as the key
					} else {
						normalizedFilePath = normalizeFilePath(filePath)
					}

					if localizedFilename, exists := storage.externalRefs.Get(normalizedFilePath); exists {
						// Build new reference with localized filename
						if refObj.HasJSONPointer() {
							newRef := localizedFilename + "#" + string(refObj.GetJSONPointer())
							*schema.GetLeft().Ref = references.Reference(newRef)
						} else {
							*schema.GetLeft().Ref = references.Reference(localizedFilename)
						}
					}
				}
				return nil
			},
			ReferencedPathItem: func(ref *ReferencedPathItem) error {
				return updateGenericReference(ref, storage)
			},
			ReferencedParameter: func(ref *ReferencedParameter) error {
				return updateGenericReference(ref, storage)
			},
			ReferencedExample: func(ref *ReferencedExample) error {
				return updateGenericReference(ref, storage)
			},
			ReferencedRequestBody: func(ref *ReferencedRequestBody) error {
				return updateGenericReference(ref, storage)
			},
			ReferencedResponse: func(ref *ReferencedResponse) error {
				return updateGenericReference(ref, storage)
			},
			ReferencedHeader: func(ref *ReferencedHeader) error {
				return updateGenericReference(ref, storage)
			},
			ReferencedCallback: func(ref *ReferencedCallback) error {
				return updateGenericReference(ref, storage)
			},
			ReferencedLink: func(ref *ReferencedLink) error {
				return updateGenericReference(ref, storage)
			},
			ReferencedSecurityScheme: func(ref *ReferencedSecurityScheme) error {
				return updateGenericReference(ref, storage)
			},
		})
		if err != nil {
			return fmt.Errorf("failed to update reference at %s: %w", item.Location.ToJSONPointer().String(), err)
		}
	}

	return nil
}

// updateGenericReference updates a generic reference to point to the localized filename
func updateGenericReference[T any, V interfaces.Validator[T], C marshaller.CoreModeler](ref *Reference[T, V, C], storage *localizeStorage) error {
	if ref == nil || !ref.IsReference() {
		return nil
	}

	refObj := ref.GetReference()
	filePath := refObj.GetURI()

	// For URLs, use the full reference as the key, for file paths normalize
	var normalizedFilePath string
	if classification, err := utils.ClassifyReference(string(refObj)); err == nil && classification.Type == utils.ReferenceTypeURL {
		normalizedFilePath = string(refObj) // Use the full URL as the key
	} else {
		normalizedFilePath = normalizeFilePath(filePath)
	}

	if localizedFilename, exists := storage.externalRefs.Get(normalizedFilePath); exists {
		// Build new reference with localized filename
		if refObj.HasJSONPointer() {
			newRef := localizedFilename + "#" + string(refObj.GetJSONPointer())
			*ref.Reference = references.Reference(newRef)
		} else {
			*ref.Reference = references.Reference(localizedFilename)
		}
	}

	return nil
}

// normalizeFilePath normalizes a file path for consistent handling
func normalizeFilePath(filePath string) string {
	// Check if this is a URL - if so, don't apply file path normalization
	if classification, err := utils.ClassifyReference(filePath); err == nil && classification.Type == utils.ReferenceTypeURL {
		return filePath // Return URLs as-is
	}

	// Clean and normalize the file path
	cleanPath := filepath.Clean(filePath)

	// Remove leading "./" if present
	cleanPath = strings.TrimPrefix(cleanPath, "./")

	return cleanPath
}
