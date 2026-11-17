package rules_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/arazzo/linter/rules"
	"github.com/stretchr/testify/assert"
)

// sourcesDoc builds a document with the given source descriptions.
func sourcesDoc(arazzoVersion, sources string) string {
	return `arazzo: ` + arazzoVersion + `
info:
  title: Sources
  version: 1.0.0
sourceDescriptions:
` + sources + `workflows: []
`
}

func TestSourceDescriptionHTTPSRule_Success(t *testing.T) {
	t.Parallel()

	doc := sourcesDoc("1.1.0", `  - name: secure
    url: https://example.com/openapi.yaml
  - name: relative
    url: ./openapi.yaml
  - name: localhost
    url: http://LOCALHOST:8080/openapi.yaml
  - name: loopback
    url: http://127.0.0.1/openapi.yaml
  - name: ipv6
    url: http://[::1]/openapi.yaml
  - name: invalid
    url: "http://[::1"
`)
	assert.Empty(t, runRule(t, &rules.SourceDescriptionHTTPSRule{}, doc), "secure and local URLs should not be reported")
}

func TestSourceDescriptionHTTPSRule_Error(t *testing.T) {
	t.Parallel()

	doc := sourcesDoc("1.1.0", `  - name: insecure
    url: HTTP://example.com/openapi.yaml
  - name: lookalike
    url: http://127.evil.example/openapi.yaml
  - name: remoteIP
    url: http://10.0.0.1/openapi.yaml
`)
	expected := []string{
		"warning: sourceDescription `insecure` url `HTTP://example.com/openapi.yaml` uses http instead of https",
		"warning: sourceDescription `lookalike` url `http://127.evil.example/openapi.yaml` uses http instead of https",
		"warning: sourceDescription `remoteIP` url `http://10.0.0.1/openapi.yaml` uses http instead of https",
	}
	assert.Equal(t, expected, runRule(t, &rules.SourceDescriptionHTTPSRule{}, doc), "http URLs to non-local hosts should be reported")
}
