package arazzo_test

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/require"
)

// validationMessages unmarshals and validates the provided Arazzo document and returns the underlying messages of
// any validation errors (without line/column information).
func validationMessages(t *testing.T, doc string) []string {
	t.Helper()

	_, validationErrs, err := arazzo.Unmarshal(t.Context(), bytes.NewBufferString(doc))
	require.NoError(t, err, "unmarshal should succeed")

	messages := make([]string, 0, len(validationErrs))
	for _, validationErr := range validationErrs {
		var vErr *validation.Error
		if errors.As(validationErr, &vErr) {
			messages = append(messages, vErr.UnderlyingError.Error())
			continue
		}
		messages = append(messages, validationErr.Error())
	}

	return messages
}

// assertMessagesContain asserts that there is exactly one message per expected substring, with each expected
// substring contained in a distinct message.
func assertMessagesContain(t *testing.T, expected []string, messages []string) {
	t.Helper()

	require.Len(t, messages, len(expected), "number of validation errors should match, got: %v", messages)

	remaining := append([]string{}, messages...)
	for _, e := range expected {
		found := -1
		for i, m := range remaining {
			if strings.Contains(m, e) {
				found = i
				break
			}
		}
		require.GreaterOrEqual(t, found, 0, "expected a validation error containing %q, got: %v", e, messages)
		remaining = append(remaining[:found], remaining[found+1:]...)
	}
}

// sprintfDoc formats a document template containing a single %s verb.
func sprintfDoc(template string, value string) string {
	return fmt.Sprintf(template, value)
}

// fmtPayloadReplacementDoc formats the payload replacement test document with the content type and replacements.
func fmtPayloadReplacementDoc(contentType, replacements string) string {
	return fmt.Sprintf(payloadReplacementTestDoc, contentType, replacements)
}
