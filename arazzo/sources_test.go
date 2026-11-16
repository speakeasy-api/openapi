package arazzo_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"
	"testing/fstest"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sourcesTestOpenAPI = `openapi: 3.1.0
info:
  title: Pets
  version: 1.0.0
paths: {}
`

// mockHTTPClient returns canned responses keyed by URL.
type mockHTTPClient struct {
	responses  map[string]string
	status     int
	err        error
	noResponse bool
}

func (c *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if c.err != nil || c.noResponse {
		return nil, c.err
	}

	body, ok := c.responses[req.URL.String()]
	status := c.status
	switch {
	case !ok:
		status = http.StatusNotFound
	case status == 0:
		status = http.StatusOK
	}

	return &http.Response{StatusCode: status, Status: http.StatusText(status), Body: io.NopCloser(bytes.NewBufferString(body))}, nil
}

// sourcesTestDoc builds an Arazzo document with a single source description.
func sourcesTestDoc(t *testing.T, self, sourceType, url string) *arazzo.Arazzo {
	t.Helper()

	doc := "arazzo: 1.1.0\n"
	if self != "" {
		doc += "$self: " + self + "\n"
	}
	doc += "info:\n  title: Sources\n  version: 1.0.0\nsourceDescriptions:\n  - name: api\n    url: " + url + "\n"
	if sourceType != "" {
		doc += "    type: " + sourceType + "\n"
	}
	doc += "workflows:\n  - workflowId: main\n    steps:\n      - stepId: first\n        operationId: getPet\n"

	a, _, err := arazzo.Unmarshal(t.Context(), bytes.NewBufferString(doc), arazzo.WithSkipValidation())
	require.NoError(t, err, "unmarshal should succeed")
	return a
}

// severityMessages returns `<severity>: <message>` for each validation error.
func severityMessages(errs []error) []string {
	messages := make([]string, 0, len(errs))
	for _, err := range errs {
		var vErr *validation.Error
		if errors.As(err, &vErr) {
			messages = append(messages, string(vErr.Severity)+": "+vErr.UnderlyingError.Error())
			continue
		}
		messages = append(messages, err.Error())
	}
	return messages
}

func TestLoadSourceDescriptions_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		self             string
		sourceType       string
		url              string
		documentLocation string
		files            fstest.MapFS
		responses        map[string]string
		expectedLocation string
		expectedType     arazzo.SourceDescriptionType
	}{
		{
			name:             "relative file resolved against the document location",
			url:              "petstore.yaml",
			documentLocation: "specs/main.arazzo.yaml",
			files:            fstest.MapFS{"specs/petstore.yaml": {Data: []byte(sourcesTestOpenAPI)}},
			expectedLocation: "specs/petstore.yaml",
			expectedType:     arazzo.SourceDescriptionTypeOpenAPI,
		},
		{
			name:             "relative url resolved against $self",
			self:             "https://example.com/flows/main.arazzo.yaml",
			url:              "petstore.yaml",
			documentLocation: "specs/main.arazzo.yaml",
			responses:        map[string]string{"https://example.com/flows/petstore.yaml": sourcesTestOpenAPI},
			expectedLocation: "https://example.com/flows/petstore.yaml",
			expectedType:     arazzo.SourceDescriptionTypeOpenAPI,
		},
		{
			name:             "absolute path resolved against the host of $self",
			self:             "https://example.com/flows/main.arazzo.yaml",
			url:              "/openapi.yaml",
			responses:        map[string]string{"https://example.com/openapi.yaml": sourcesTestOpenAPI},
			expectedLocation: "https://example.com/openapi.yaml",
			expectedType:     arazzo.SourceDescriptionTypeOpenAPI,
		},
		{
			name:             "arazzo source detected from content",
			url:              "https://example.com/shared.arazzo.yaml",
			responses:        map[string]string{"https://example.com/shared.arazzo.yaml": "arazzo: 1.1.0\ninfo:\n  title: Shared\n  version: 1.0.0\n"},
			expectedLocation: "https://example.com/shared.arazzo.yaml",
			expectedType:     arazzo.SourceDescriptionTypeArazzo,
		},
		{
			name:             "declared type matches content",
			sourceType:       "openapi",
			url:              "https://example.com/openapi.yaml",
			responses:        map[string]string{"https://example.com/openapi.yaml": sourcesTestOpenAPI},
			expectedLocation: "https://example.com/openapi.yaml",
			expectedType:     arazzo.SourceDescriptionTypeOpenAPI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := sourcesTestDoc(t, tt.self, tt.sourceType, tt.url)
			sources, errs := arazzo.LoadSourceDescriptions(t.Context(), a, arazzo.LoadSourcesOptions{
				DocumentLocation: tt.documentLocation,
				VirtualFS:        tt.files,
				HTTPClient:       &mockHTTPClient{responses: tt.responses},
			})
			assert.Empty(t, errs, "loading should succeed")

			doc := sources.Get("api")
			require.NotNil(t, doc, "source should be loaded")
			assert.Equal(t, "api", doc.Name, "name should match")
			assert.Equal(t, tt.expectedLocation, doc.Location, "location should match")
			assert.Equal(t, tt.expectedType, doc.Type, "type should match")
			assert.Equal(t, tt.expectedType == arazzo.SourceDescriptionTypeOpenAPI, doc.OpenAPI != nil, "OpenAPI document should be set for OpenAPI sources")
			assert.Equal(t, tt.expectedType == arazzo.SourceDescriptionTypeArazzo, doc.Arazzo != nil, "Arazzo document should be set for Arazzo sources")
		})
	}
}

func TestLoadSourceDescriptions_NotLoaded(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		self             string
		sourceType       string
		url              string
		documentLocation string
		files            fstest.MapFS
		client           *mockHTTPClient
		expected         []string
	}{
		{
			name:     "missing file",
			url:      "missing.yaml",
			files:    fstest.MapFS{},
			expected: []string{"warning: sourceDescription `api` could not be loaded from `missing.yaml`: open missing.yaml: file does not exist"},
		},
		{
			name:     "http error status",
			url:      "https://example.com/missing.yaml",
			client:   &mockHTTPClient{},
			expected: []string{"warning: sourceDescription `api` could not be loaded from `https://example.com/missing.yaml`: unexpected HTTP status Not Found"},
		},
		{
			name:     "http client error",
			url:      "https://example.com/openapi.yaml",
			client:   &mockHTTPClient{err: errors.New("connection refused")},
			expected: []string{"warning: sourceDescription `api` could not be loaded from `https://example.com/openapi.yaml`: connection refused"},
		},
		{
			name:     "http client without a response",
			url:      "https://example.com/openapi.yaml",
			client:   &mockHTTPClient{noResponse: true},
			expected: []string{"warning: sourceDescription `api` could not be loaded from `https://example.com/openapi.yaml`: no response received"},
		},
		{
			name:             "unresolvable relative url",
			url:              "petstore.yaml",
			documentLocation: "http://[::1/main.arazzo.yaml",
			expected:         []string{"warning: sourceDescription `api` url can't be resolved: invalid base reference: invalid reference format: parse \"http://[::1/main.arazzo.yaml\": missing ']' in host"},
		},
		{
			name:             "absolute url isn't resolved against a file location",
			url:              "https://example.com/missing.yaml",
			documentLocation: "specs/main.arazzo.yaml",
			client:           &mockHTTPClient{},
			expected:         []string{"warning: sourceDescription `api` could not be loaded from `https://example.com/missing.yaml`: unexpected HTTP status Not Found"},
		},
		{
			name:             "unresolvable $self",
			self:             "main.arazzo.yaml",
			url:              "https://example.com/missing.yaml",
			documentLocation: "http://[::1/main.arazzo.yaml",
			client:           &mockHTTPClient{},
			expected: []string{
				"warning: `$self` can't be used to resolve source descriptions: invalid base reference: invalid reference format: parse \"http://[::1/main.arazzo.yaml\": missing ']' in host",
				"warning: sourceDescription `api` could not be loaded from `https://example.com/missing.yaml`: unexpected HTTP status Not Found",
			},
		},
		{
			name:       "declared asyncapi source",
			sourceType: "asyncapi",
			url:        "asyncapi.yaml",
			expected:   []string{"hint: sourceDescription `api` is an AsyncAPI document, steps targeting it are not checked against the source"},
		},
		{
			name:     "detected asyncapi source",
			url:      "events.yaml",
			files:    fstest.MapFS{"events.yaml": {Data: []byte("asyncapi: 3.0.0\n")}},
			expected: []string{"hint: sourceDescription `api` is an AsyncAPI document, steps targeting it are not checked against the source"},
		},
		{
			name:     "unknown document",
			url:      "other.yaml",
			files:    fstest.MapFS{"other.yaml": {Data: []byte("swagger: \"2.0\"\n")}},
			expected: []string{"warning: sourceDescription `api` at `other.yaml` is not an OpenAPI 3.x, AsyncAPI or Arazzo document"},
		},
		{
			name:     "non mapping document",
			url:      "list.yaml",
			files:    fstest.MapFS{"list.yaml": {Data: []byte("- openapi\n")}},
			expected: []string{"warning: sourceDescription `api` at `list.yaml` is not an OpenAPI 3.x, AsyncAPI or Arazzo document"},
		},
		{
			name:     "invalid yaml",
			url:      "broken.yaml",
			files:    fstest.MapFS{"broken.yaml": {Data: []byte("openapi: [\n")}},
			expected: []string{"warning: sourceDescription `api` at `broken.yaml` is not an OpenAPI 3.x, AsyncAPI or Arazzo document"},
		},
		{
			name:       "declared type doesn't match content",
			sourceType: "openapi",
			url:        "shared.yaml",
			files:      fstest.MapFS{"shared.yaml": {Data: []byte("arazzo: 1.1.0\n")}},
			expected:   []string{"error: sourceDescription `api` is declared as type `openapi` but `shared.yaml` is an arazzo document"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := arazzo.LoadSourcesOptions{DocumentLocation: tt.documentLocation}
			if tt.files != nil {
				opts.VirtualFS = tt.files
			}
			if tt.client != nil {
				opts.HTTPClient = tt.client
			}

			a := sourcesTestDoc(t, tt.self, tt.sourceType, tt.url)
			sources, errs := arazzo.LoadSourceDescriptions(t.Context(), a, opts)
			assert.Equal(t, tt.expected, severityMessages(errs), "load errors should match")
			assert.Nil(t, sources.Get("api"), "source should not be loaded")
		})
	}
}

func TestLoadSourceDescriptions_UnresolvedReferences_Success(t *testing.T) {
	t.Parallel()

	files := fstest.MapFS{"petstore.yaml": {Data: []byte(`openapi: 3.1.0
info:
  title: Pets
  version: 1.0.0
paths:
  /pets:
    get:
      operationId: listPets
      parameters:
        - $ref: "./missing.yaml#/components/parameters/Limit"
      responses:
        "200":
          description: OK
`)}}

	a := sourcesTestDoc(t, "", "", "petstore.yaml")
	sources, errs := arazzo.LoadSourceDescriptions(t.Context(), a, arazzo.LoadSourcesOptions{VirtualFS: files})
	require.Len(t, errs, 1, "should warn about the unresolved reference")
	assert.Contains(t, severityMessages(errs)[0], "warning: sourceDescription `api` has references that could not be resolved, some checks may be skipped", "warning should match")
	assert.NotNil(t, sources.Get("api"), "source should still be loaded")
}

func TestLoadSourceDescriptions_SkipsIncompleteSourceDescriptions_Success(t *testing.T) {
	t.Parallel()

	a := &arazzo.Arazzo{
		Arazzo: "1.1.0",
		SourceDescriptions: arazzo.SourceDescriptions{
			nil,
			{Name: "", URL: "petstore.yaml"},
			{Name: "api", URL: ""},
		},
		Self: pointer.From(""),
	}

	sources, errs := arazzo.LoadSourceDescriptions(t.Context(), a, arazzo.LoadSourcesOptions{})
	assert.Empty(t, errs, "incomplete source descriptions should be skipped")
	assert.Nil(t, sources.Get("api"), "nothing should be loaded")
}

func TestLoadSourceDescriptions_NilDocument_Success(t *testing.T) {
	t.Parallel()

	sources, errs := arazzo.LoadSourceDescriptions(t.Context(), nil, arazzo.LoadSourcesOptions{})
	assert.Empty(t, errs, "nil document should have no errors")
	assert.Nil(t, sources.Get("api"), "nothing should be loaded")

	var nilSources *arazzo.SourceDocuments
	assert.Nil(t, nilSources.Get("api"), "nil sources should be nil safe")
}
