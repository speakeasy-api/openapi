package utils

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// ReferenceType represents the type of reference string
type ReferenceType int

const (
	ReferenceTypeUnknown ReferenceType = iota
	ReferenceTypeURL
	ReferenceTypeFilePath
	ReferenceTypeFragment
)

// ReferenceClassification holds the result of classifying a reference string
type ReferenceClassification struct {
	Type       ReferenceType
	IsURL      bool
	IsFile     bool
	IsFragment bool
	Original   string
	ParsedURL  *url.URL // Cached parsed URL to avoid re-parsing
}

// ClassifyReference determines if a string represents a URL, file path, or JSON Pointer fragment.
// It returns detailed classification information and any parsing errors.
func ClassifyReference(ref string) (*ReferenceClassification, error) {
	// Handle empty strings
	if ref == "" {
		return nil, errors.New("empty reference")
	}

	result := &ReferenceClassification{
		Original: ref,
	}

	// Try parsing as URL first using cached parsing
	u, err := ParseURLCached(ref)
	if err != nil {
		return nil, fmt.Errorf("invalid reference format: %w", err)
	}

	// Check for URL schemes, but exclude Windows drive letters
	if u.Scheme != "" {
		// Check if this is a Windows drive letter (single letter followed by colon)
		if len(u.Scheme) == 1 && strings.Contains(ref, "\\") {
			// This is likely a Windows path like C:\path\to\file
			result.Type = ReferenceTypeFilePath
			result.IsFile = true
			return result, nil
		}

		switch strings.ToLower(u.Scheme) {
		case "http", "https", "ftp", "ftps", "file":
			result.Type = ReferenceTypeURL
			result.IsURL = true
			result.ParsedURL = u // Cache the parsed URL
			return result, nil
		default:
			// Unknown scheme, might be custom protocol
			result.Type = ReferenceTypeURL
			result.IsURL = true
			result.ParsedURL = u // Cache the parsed URL
			return result, nil
		}
	}

	// Check for fragment-only reference (#/components/schemas/User)
	if strings.HasPrefix(ref, "#") {
		result.Type = ReferenceTypeFragment
		result.IsFragment = true
		return result, nil
	}

	// No scheme - check for file path patterns
	if strings.Contains(ref, "/") ||
		strings.Contains(ref, "\\") ||
		strings.HasPrefix(ref, "./") ||
		strings.HasPrefix(ref, "../") ||
		filepath.IsAbs(ref) {
		result.Type = ReferenceTypeFilePath
		result.IsFile = true
		return result, nil
	}

	// Ambiguous case - could be relative file or just a name
	// Default to file path for relative references
	result.Type = ReferenceTypeFilePath
	result.IsFile = true
	return result, nil
}

// IsURL returns true if the reference string represents a URL
func IsURL(ref string) bool {
	classification, err := ClassifyReference(ref)
	if err != nil {
		return false
	}
	return classification.IsURL
}

// IsFilePath returns true if the reference string represents a file path
func IsFilePath(ref string) bool {
	classification, err := ClassifyReference(ref)
	if err != nil {
		return false
	}
	return classification.IsFile
}

// IsFragment returns true if the reference string represents a JSON Pointer fragment
func IsFragment(ref string) bool {
	classification, err := ClassifyReference(ref)
	if err != nil {
		return false
	}
	return classification.IsFragment
}

// JoinWith joins this classified reference with a relative reference.
// It uses the cached classification and parsed URL (if available) to avoid re-parsing.
// For URLs, it uses the cached ParsedURL and ResolveReference. For file paths, it uses filepath.Join.
// Fragments are handled specially and can be combined with both URLs and file paths.
func (rc *ReferenceClassification) JoinWith(relative string) (string, error) {
	if relative == "" {
		return rc.Original, nil
	}

	// Handle fragment-only relative references
	if strings.HasPrefix(relative, "#") {
		// Strip any existing fragment from base and append the new one
		base := rc.Original
		if idx := strings.Index(base, "#"); idx != -1 {
			base = base[:idx]
		}
		return base + relative, nil
	}

	// Use classification to determine joining strategy
	if rc.IsURL {
		return rc.joinURL(relative)
	}

	if rc.IsFile {
		return rc.joinFilePath(relative)
	}

	// If base is a fragment, treat relative as the new reference
	if rc.IsFragment {
		return relative, nil
	}

	// Fallback: treat as file path
	return rc.joinFilePath(relative)
}

// joinURL joins this URL reference with a relative reference using the cached ParsedURL
func (rc *ReferenceClassification) joinURL(relative string) (string, error) {
	// Use cached ParsedURL if available to avoid re-parsing
	var baseURL *url.URL
	if rc.ParsedURL != nil {
		baseURL = rc.ParsedURL
	} else {
		// Fallback to parsing if not cached (shouldn't happen in normal usage)
		var err error
		baseURL, err = ParseURLCached(rc.Original)
		if err != nil {
			return "", fmt.Errorf("invalid base URL: %w", err)
		}
	}

	relativeURL, err := ParseURLCached(relative)
	if err != nil {
		return "", fmt.Errorf("invalid relative URL: %w", err)
	}

	resolvedURL := baseURL.ResolveReference(relativeURL)
	return resolvedURL.String(), nil
}

// joinFilePath joins this file path reference with a relative path using cross-platform path handling
func (rc *ReferenceClassification) joinFilePath(relative string) (string, error) {
	// If relative path is absolute, return it as-is
	// Check for both OS-specific absolute paths and Unix-style absolute paths (for cross-platform compatibility)
	if filepath.IsAbs(relative) || strings.HasPrefix(relative, "/") || rc.isWindowsAbsolutePath(relative) {
		return relative, nil
	}

	// Determine the path separator style from the original path
	isWindowsStyle := strings.Contains(rc.Original, "\\") && !strings.Contains(rc.Original, "/")

	// Get the directory part of the original path using cross-platform logic
	var baseDir string
	if isWindowsStyle {
		// Handle Windows-style paths manually for cross-platform compatibility
		baseDir = rc.getWindowsDir()
	} else {
		// Use standard filepath.Dir for Unix-style paths
		baseDir = filepath.Dir(rc.Original)
	}

	// Join the paths
	var joined string
	if isWindowsStyle {
		// Manual Windows-style path joining
		joined = rc.joinWindowsPaths(baseDir, relative)
	} else {
		// Use standard filepath.Join for Unix-style paths
		joined = filepath.Join(baseDir, relative)
		// Convert to forward slashes for OpenAPI/JSON Schema compatibility
		joined = strings.ReplaceAll(joined, "\\", "/")
	}

	return joined, nil
}

// getWindowsDir extracts the directory part from a Windows-style path
func (rc *ReferenceClassification) getWindowsDir() string {
	path := rc.Original
	// Find the last backslash
	lastSlash := strings.LastIndex(path, "\\")
	if lastSlash == -1 {
		return "." // No directory separator found
	}
	return path[:lastSlash]
}

// joinWindowsPaths joins Windows-style paths manually
func (rc *ReferenceClassification) joinWindowsPaths(base, relative string) string {
	// Handle relative path navigation
	// Split by both forward and backward slashes to handle cross-platform relative paths
	var parts []string
	if strings.Contains(relative, "/") {
		// Unix-style path with forward slashes
		parts = strings.Split(relative, "/")
	} else {
		// Windows-style path with backslashes
		parts = strings.Split(relative, "\\")
	}

	baseParts := strings.Split(base, "\\")

	for _, part := range parts {
		switch part {
		case ".":
			// Current directory, do nothing
			continue
		case "..":
			// Parent directory, remove last part from base
			if len(baseParts) > 1 {
				baseParts = baseParts[:len(baseParts)-1]
			}
		default:
			// Regular path component
			baseParts = append(baseParts, part)
		}
	}

	return strings.Join(baseParts, "\\")
}

// isWindowsAbsolutePath checks if a path is a Windows absolute path (e.g., C:\path or \\server\share)
func (rc *ReferenceClassification) isWindowsAbsolutePath(path string) bool {
	// Check for drive letter paths (C:\, D:\, etc.)
	if len(path) >= 3 && path[1] == ':' && (path[2] == '\\' || path[2] == '/') {
		return true
	}
	// Check for UNC paths (\\server\share)
	if strings.HasPrefix(path, "\\\\") {
		return true
	}
	return false
}

// JoinReference is a convenience function that classifies the base reference and joins it with a relative reference.
// For better performance when you already have a classification, use ReferenceClassification.JoinWith() instead.
func JoinReference(base, relative string) (string, error) {
	if base == "" {
		return relative, nil
	}

	baseClassification, err := ClassifyReference(base)
	if err != nil {
		return "", fmt.Errorf("invalid base reference: %w", err)
	}

	return baseClassification.JoinWith(relative)
}
