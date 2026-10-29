package openapi

import (
	"path/filepath"
	"regexp"
)

var (
	// unsafeComponentFileNameChars matches characters that are replaced when turning a file path into part of a component name.
	unsafeComponentFileNameChars = regexp.MustCompile(`[^a-zA-Z0-9_]`)
	// unsafeFragmentFileNameChars matches characters that are replaced when turning a file name into a URL fragment or document name.
	unsafeFragmentFileNameChars = regexp.MustCompile(`[^a-zA-Z0-9_-]`)
)

// fileStem returns the base name of a file path without its extension, e.g. "some/path/User.yaml" becomes "User".
func fileStem(path string) string {
	baseName := filepath.Base(path)
	return baseName[:len(baseName)-len(filepath.Ext(baseName))]
}

// sanitizeFileNameForComponent replaces every character outside [a-zA-Z0-9_] with "_".
func sanitizeFileNameForComponent(name string) string {
	return unsafeComponentFileNameChars.ReplaceAllString(name, "_")
}

// sanitizeFileNameForFragment replaces every character outside [a-zA-Z0-9_-] with "_".
func sanitizeFileNameForFragment(name string) string {
	return unsafeFragmentFileNameChars.ReplaceAllString(name, "_")
}

// filePathToComponentPrefix converts a cleaned file path into a component name prefix, keeping the
// extension for uniqueness, e.g. "some/path/external.yaml" becomes "some_path_external_yaml".
func filePathToComponentPrefix(cleanPath string) string {
	if ext := filepath.Ext(cleanPath); ext != "" {
		cleanPath = cleanPath[:len(cleanPath)-len(ext)] + "_" + ext[1:]
	}
	return sanitizeFileNameForComponent(cleanPath)
}
