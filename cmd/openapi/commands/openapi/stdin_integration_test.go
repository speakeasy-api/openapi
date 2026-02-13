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

		processor := &OpenAPIProcessor{
			InputFile:     "-",
			ReadFromStdin: true,
			WriteToStdout: true,
			Stdin:         strings.NewReader(testOpenAPIDoc),
			Stderr:        &bytes.Buffer{},
		}

		doc, validationErrors, err := processor.LoadDocument(context.Background())

		require.NoError(t, err)
		require.NotNil(t, doc)
		assert.Empty(t, validationErrors)
		assert.Equal(t, "Test API", doc.Info.Title)
		assert.Equal(t, "1.0.0", doc.Info.Version)
	})

	t.Run("loads document from file", func(t *testing.T) {
		t.Parallel()

		tmpFile := writeTestFile(t, testOpenAPIDoc)

		processor := &OpenAPIProcessor{
			InputFile:     tmpFile,
			ReadFromStdin: false,
			WriteToStdout: true,
			Stderr:        &bytes.Buffer{},
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

	doc, _, err := openapi.Unmarshal(context.Background(), strings.NewReader(testOpenAPIDoc))
	require.NoError(t, err)

	var stdout bytes.Buffer
	processor := &OpenAPIProcessor{
		WriteToStdout: true,
		Stdout:        &stdout,
		Stderr:        &bytes.Buffer{},
	}

	writeErr := processor.WriteDocument(context.Background(), doc)

	require.NoError(t, writeErr)
	output := stdout.String()
	assert.Contains(t, output, "openapi:")
	assert.Contains(t, output, "Test API")
}

func TestStdinPipeline(t *testing.T) {
	t.Parallel()

	t.Run("stdin to stdout pipeline", func(t *testing.T) {
		t.Parallel()

		var stderr bytes.Buffer
		processor := &OpenAPIProcessor{
			InputFile:     "-",
			ReadFromStdin: true,
			WriteToStdout: true,
			Stdin:         strings.NewReader(testOpenAPIDoc),
			Stderr:        &stderr,
		}

		doc, _, loadErr := processor.LoadDocument(context.Background())
		require.NoError(t, loadErr)
		require.NotNil(t, doc)

		var stdout bytes.Buffer
		processor.Stdout = &stdout

		writeErr := processor.WriteDocument(context.Background(), doc)
		require.NoError(t, writeErr)

		output := stdout.String()
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
			Stdin:         strings.NewReader(testOpenAPIDoc),
			Stderr:        &bytes.Buffer{},
		}

		doc, _, loadErr := processor.LoadDocument(context.Background())
		require.NoError(t, loadErr)
		require.NotNil(t, doc)

		writeErr := processor.WriteDocument(context.Background(), doc)
		require.NoError(t, writeErr)

		contentBytes, err := os.ReadFile(outFile)
		require.NoError(t, err)
		content := string(contentBytes)
		assert.Contains(t, content, "openapi:")
		assert.Contains(t, content, "Test API")
	})
}

func TestStatusMessagesGoToStderr(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	processor := &OpenAPIProcessor{
		WriteToStdout: true,
		Stderr:        &stderr,
	}

	processor.PrintSuccess("test success")
	processor.PrintInfo("test info")
	processor.PrintWarning("test warning")

	output := stderr.String()
	assert.Contains(t, output, "test success")
	assert.Contains(t, output, "test info")
	assert.Contains(t, output, "test warning")
}

func writeTestFile(t *testing.T, content string) string {
	t.Helper()
	path := t.TempDir() + "/spec.yaml"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}
