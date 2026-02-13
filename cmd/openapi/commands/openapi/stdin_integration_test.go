package openapi

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testOpenAPIDoc is a minimal valid OpenAPI 3.1 document for testing.
const testOpenAPIDoc = `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      summary: Get all users
      responses:
        "200":
          description: List of users
`

func TestLoadDocumentFromStdin(t *testing.T) {
	t.Parallel()

	t.Run("loads valid document from stdin", func(t *testing.T) {
		t.Parallel()

		// Create a processor configured for stdin
		processor := &OpenAPIProcessor{
			InputFile:     "-",
			ReadFromStdin: true,
			WriteToStdout: true,
		}

		// Replace os.Stdin with our test data
		oldStdin := os.Stdin
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdin = r

		// Write test data and close writer
		go func() {
			_, _ = w.Write([]byte(testOpenAPIDoc))
			w.Close()
		}()

		// Load the document
		doc, validationErrors, err := processor.LoadDocument(context.Background())
		os.Stdin = oldStdin

		require.NoError(t, err)
		require.NotNil(t, doc)
		assert.Empty(t, validationErrors)
		assert.Equal(t, "Test API", doc.Info.Title)
		assert.Equal(t, "1.0.0", doc.Info.Version)
	})

	t.Run("loads document from file", func(t *testing.T) {
		t.Parallel()

		// Write a temp file
		tmpFile, err := os.CreateTemp(t.TempDir(), "spec-*.yaml")
		require.NoError(t, err)
		_, err = tmpFile.Write([]byte(testOpenAPIDoc))
		require.NoError(t, err)
		tmpFile.Close()

		processor := &OpenAPIProcessor{
			InputFile:     tmpFile.Name(),
			ReadFromStdin: false,
			WriteToStdout: true,
		}

		doc, validationErrors, err := processor.LoadDocument(context.Background())
		require.NoError(t, err)
		require.NotNil(t, doc)
		assert.Empty(t, validationErrors)
		assert.Equal(t, "Test API", doc.Info.Title)
	})
}

func TestWriteDocumentToStdout(t *testing.T) {
	t.Parallel()

	// Parse a document to write
	doc, _, err := openapi.Unmarshal(context.Background(), strings.NewReader(testOpenAPIDoc))
	require.NoError(t, err)

	processor := &OpenAPIProcessor{
		WriteToStdout: true,
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	writeErr := processor.WriteDocument(context.Background(), doc)
	w.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = oldStdout

	require.NoError(t, writeErr)
	output := buf.String()
	assert.Contains(t, output, "openapi:")
	assert.Contains(t, output, "Test API")
}

func TestStdinPipeline(t *testing.T) {
	t.Parallel()

	t.Run("stdin to stdout pipeline", func(t *testing.T) {
		t.Parallel()

		// Simulate: cat spec.yaml | openapi spec inline -
		processor := &OpenAPIProcessor{
			InputFile:     "-",
			ReadFromStdin: true,
			WriteToStdout: true,
		}

		// Replace stdin
		oldStdin := os.Stdin
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdin = r

		go func() {
			_, _ = w.Write([]byte(testOpenAPIDoc))
			w.Close()
		}()

		// Load
		doc, _, loadErr := processor.LoadDocument(context.Background())
		os.Stdin = oldStdin

		require.NoError(t, loadErr)
		require.NotNil(t, doc)

		// Capture stdout for write
		oldStdout := os.Stdout
		rOut, wOut, err := os.Pipe()
		require.NoError(t, err)
		os.Stdout = wOut

		writeErr := processor.WriteDocument(context.Background(), doc)
		wOut.Close()

		var buf bytes.Buffer
		_, _ = buf.ReadFrom(rOut)
		os.Stdout = oldStdout

		require.NoError(t, writeErr)

		// The output should be a valid OpenAPI document
		output := buf.String()
		assert.Contains(t, output, "openapi:")
		assert.Contains(t, output, "Test API")
		assert.Contains(t, output, "/users")
	})

	t.Run("stdin to file", func(t *testing.T) {
		t.Parallel()

		outFile := t.TempDir() + "/output.yaml"
		processor := &OpenAPIProcessor{
			InputFile:     "-",
			OutputFile:    outFile,
			ReadFromStdin: true,
			WriteToStdout: false,
		}

		// Replace stdin
		oldStdin := os.Stdin
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdin = r

		go func() {
			_, _ = w.Write([]byte(testOpenAPIDoc))
			w.Close()
		}()

		doc, _, loadErr := processor.LoadDocument(context.Background())
		os.Stdin = oldStdin

		require.NoError(t, loadErr)
		require.NotNil(t, doc)

		writeErr := processor.WriteDocument(context.Background(), doc)
		require.NoError(t, writeErr)

		// Read the output file
		content, err := os.ReadFile(outFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "openapi:")
		assert.Contains(t, string(content), "Test API")
	})
}

func TestStatusMessagesGoToStderr(t *testing.T) {
	t.Parallel()

	// Capture stderr
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	processor := &OpenAPIProcessor{
		WriteToStdout: true,
	}

	processor.PrintSuccess("test success")
	processor.PrintInfo("test info")
	processor.PrintWarning("test warning")

	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stderr = oldStderr

	output := buf.String()
	assert.Contains(t, output, "test success")
	assert.Contains(t, output, "test info")
	assert.Contains(t, output, "test warning")
}
