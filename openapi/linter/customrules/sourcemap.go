package customrules

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"github.com/go-sourcemap/sourcemap"
)

// inlineSourceMapPrefix is the prefix for inline source maps.
const inlineSourceMapPrefix = "//# sourceMappingURL=data:application/json;base64,"

// ExtractInlineSourceMap extracts and parses an inline source map from JavaScript code.
func ExtractInlineSourceMap(code string) (*sourcemap.Consumer, error) {
	// Find the source map comment
	idx := strings.LastIndex(code, inlineSourceMapPrefix)
	if idx == -1 {
		return nil, nil // No inline source map
	}

	// Extract the base64 data
	b64Data := strings.TrimSpace(code[idx+len(inlineSourceMapPrefix):])

	// Handle potential newlines at the end
	if newlineIdx := strings.IndexAny(b64Data, "\r\n"); newlineIdx != -1 {
		b64Data = b64Data[:newlineIdx]
	}

	// Decode base64
	jsonData, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		return nil, fmt.Errorf("decoding source map base64: %w", err)
	}

	// Parse using go-sourcemap library
	consumer, err := sourcemap.Parse("", jsonData)
	if err != nil {
		// Ignore "mappings are empty" errors
		if strings.Contains(err.Error(), "mappings are empty") {
			return nil, nil
		}
		return nil, fmt.Errorf("parsing source map: %w", err)
	}

	return consumer, nil
}

// MappedError wraps an error with original source location.
type MappedError struct {
	Original   error
	SourceFile string
	Line       int
	Column     int
	Message    string
}

func (e *MappedError) Error() string {
	if e.SourceFile != "" {
		return fmt.Sprintf("%s:%d:%d: %s", e.SourceFile, e.Line, e.Column, e.Message)
	}
	return e.Message
}

func (e *MappedError) Unwrap() error {
	return e.Original
}

// MapException maps a goja exception to original TypeScript source locations.
func MapException(exc *goja.Exception, sourceFile string, sm *sourcemap.Consumer) *MappedError {
	message := exc.String()

	// Try to extract location from the exception value
	if obj := exc.Value(); obj != nil {
		if objVal, ok := obj.(*goja.Object); ok {
			// Try to get line and column from error object
			lineVal := objVal.Get("lineNumber")
			colVal := objVal.Get("columnNumber")

			if lineVal != nil && !goja.IsUndefined(lineVal) && colVal != nil && !goja.IsUndefined(colVal) {
				line := int(lineVal.ToInteger())
				col := int(colVal.ToInteger())

				// Try to remap using source map
				if sm != nil {
					_, _, remappedLine, remappedColumn, ok := sm.Source(line, col)
					if ok {
						return &MappedError{
							Original:   exc,
							SourceFile: sourceFile,
							Line:       remappedLine,
							Column:     remappedColumn,
							Message:    message,
						}
					}
				}

				return &MappedError{
					Original:   exc,
					SourceFile: sourceFile,
					Line:       line,
					Column:     col,
					Message:    message,
				}
			}
		}
	}

	// Fallback: just return the message without location
	return &MappedError{
		Original:   exc,
		SourceFile: sourceFile,
		Message:    message,
	}
}
