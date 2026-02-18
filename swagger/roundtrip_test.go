package swagger_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/internal/testutils"
	"github.com/speakeasy-api/openapi/json"
	"github.com/speakeasy-api/openapi/swagger"
	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

type roundTripTest struct {
	name              string
	location          string
	skipRoundTrip     bool
	needsSanitization bool // If true we will put the input document through the go marshallers as well to reduce whitespace diffs (as the YAML library doesn't preserve whitespace well)
}

var roundTripTests = []roundTripTest{
	{
		name:     "Comprehensive Test Swagger",
		location: "testdata/test.swagger.json",
	},
	{
		name:     "Swagger Petstore",
		location: "https://raw.githubusercontent.com/swagger-api/swagger-ui/04224150734be88f70a0bbd3f61bbe444606b657/test/unit/core/plugins/spec/assets/petstore.json",
	},
	{
		name:     "Twilio API",
		location: "https://github.com/dreamfactorysoftware/df-service-apis/raw/0dfd7df7ae217041c642bd045461cf5ed35a548b/twilio/twilio.json",
	},
	{
		name:     "eBay Key Management API",
		location: "https://developer.ebay.com/api-docs/master/developer/key-management/openapi/2/developer_key_management_v1_oas2.yaml",
	},
	{
		name:              "Docker Engine API",
		location:          "https://github.com/docker-archive/engine/raw/25381123d3483eacfa02e989381bd36939b02d1d/api/swagger.yaml",
		needsSanitization: true,
	},
	{
		name:     "DocuSign Admin API",
		location: "https://github.com/docusign/OpenAPI-Specifications/raw/10056bd7c07c2f8e41f5cb382be85d31e233f1ec/admin.rest.swagger-v2.1.json",
	},
}

func TestSwagger_RoundTrip(t *testing.T) {
	t.Parallel()
	for _, tt := range roundTripTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.skipRoundTrip {
				t.SkipNow()
			}

			ctx := t.Context()

			var r io.ReadCloser
			if strings.HasPrefix(tt.location, "testdata/") {
				var err error
				r, err = os.Open(tt.location)
				require.NoError(t, err)
			} else {
				var err error
				r, err = testutils.DownloadFile(tt.location, "SWAGGER_CACHE_DIR", "speakeasy-api_swagger")
				require.NoError(t, err)
			}
			defer r.Close()

			inBuf := bytes.NewBuffer([]byte{})
			tee := io.TeeReader(r, inBuf)

			s, validationErrs, err := swagger.Unmarshal(ctx, tee, swagger.WithSkipValidation())
			require.NoError(t, err)
			assert.Empty(t, validationErrs)

			outBuf := bytes.NewBuffer([]byte{})

			err = swagger.Marshal(ctx, s, outBuf)
			require.NoError(t, err)

			if tt.needsSanitization {
				sanitizedData, err := sanitize(inBuf.Bytes())
				require.NoError(t, err)
				inBuf = bytes.NewBuffer(sanitizedData)
			}

			assert.Equal(t, inBuf.String(), outBuf.String())
		})
	}
}

func sanitize(data []byte) ([]byte, error) {
	var node yaml.Node

	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, err
	}

	cfg := yml.GetConfigFromDoc(data, &node)

	b := bytes.NewBuffer([]byte{})

	if cfg.OriginalFormat == yml.OutputFormatYAML {
		enc := yaml.NewEncoder(b)

		enc.SetIndent(cfg.Indentation)
		if err := enc.Encode(&node); err != nil {
			return nil, err
		}
	} else {
		if err := json.YAMLToJSONWithConfig(&node, cfg.IndentationStyle.ToIndent(), cfg.Indentation, true, b); err != nil {
			return nil, err
		}
	}

	return b.Bytes(), nil
}
