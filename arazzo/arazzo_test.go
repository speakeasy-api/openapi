package arazzo_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/arazzo/criterion"
	"github.com/speakeasy-api/openapi/arazzo/expression"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/speakeasy-api/openapi/jsonschema/oas31"
	jsonschema_core "github.com/speakeasy-api/openapi/jsonschema/oas31/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TODO make it possible to choose json or yaml output
var testArazzoInstance = &arazzo.Arazzo{
	Arazzo: arazzo.Version,
	Info: arazzo.Info{
		Title:   "My Workflow",
		Summary: pointer.From("A summary"),
		Version: "1.0.0",
		Extensions: extensions.New(extensions.NewElem("x-test", &yaml.Node{
			Value:  "some-value",
			Kind:   yaml.ScalarNode,
			Tag:    "!!str",
			Line:   6,
			Column: 11,
		})),
		Model: marshaller.Model[core.Info]{
			Valid: true,
		},
	},
	SourceDescriptions: []*arazzo.SourceDescription{
		{
			Name: "openapi",
			URL:  "https://openapi.com",
			Type: "openapi",
			Extensions: extensions.New(extensions.NewElem("x-test", &yaml.Node{
				Value:  "some-value",
				Kind:   yaml.ScalarNode,
				Tag:    "!!str",
				Line:   11,
				Column: 13,
			})),
			Model: marshaller.Model[core.SourceDescription]{
				Valid: true,
			},
		},
	},
	Workflows: []*arazzo.Workflow{
		{
			WorkflowID:  "workflow1",
			Summary:     pointer.From("A summary"),
			Description: pointer.From("A description"),
			Parameters: []*arazzo.ReusableParameter{
				{
					Object: &arazzo.Parameter{
						Name:  "parameter1",
						In:    pointer.From(arazzo.InQuery),
						Value: &yaml.Node{Value: "123", Kind: yaml.ScalarNode, Tag: "!!str", Line: 19, Column: 16, Style: yaml.DoubleQuotedStyle},
						Model: marshaller.Model[core.Parameter]{
							Valid: true,
						},
					},
					Model: marshaller.Model[core.Reusable[*core.Parameter]]{
						Valid: true,
					},
				},
			},
			Inputs: oas31.NewJSONSchemaFromSchema(&oas31.Schema{
				Type: oas31.NewTypeFromString("object"),
				Properties: sequencedmap.New(sequencedmap.NewElem("input1", oas31.NewJSONSchemaFromSchema(&oas31.Schema{
					Type: oas31.NewTypeFromString("string"),
					Model: marshaller.Model[jsonschema_core.Schema]{
						Valid: true,
					},
				}))),
				Required: []string{"input1"},
				Model: marshaller.Model[jsonschema_core.Schema]{
					Valid: true,
				},
			}),
			Steps: []*arazzo.Step{
				{
					StepID:      "step1",
					Description: pointer.From("A description"),
					OperationID: pointer.From[expression.Expression]("operation1"),
					Parameters: []*arazzo.ReusableParameter{
						{
							Reference: pointer.From[expression.Expression]("$components.parameters.userId"),
							Value:     &yaml.Node{Value: "456", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle, Line: 33, Column: 20},
							Model: marshaller.Model[core.Reusable[*core.Parameter]]{
								Valid: true,
							},
						},
					},
					RequestBody: &arazzo.RequestBody{
						ContentType: pointer.From("application/json"),
						Payload: &yaml.Node{Content: []*yaml.Node{
							{
								Value:  "a",
								Kind:   yaml.ScalarNode,
								Tag:    "!!str",
								Style:  yaml.DoubleQuotedStyle,
								Line:   36,
								Column: 21,
							},
							{
								Value:  "1",
								Kind:   yaml.ScalarNode,
								Tag:    "!!int",
								Line:   36,
								Column: 26,
							},
							{
								Value:  "b",
								Kind:   yaml.ScalarNode,
								Tag:    "!!str",
								Style:  yaml.DoubleQuotedStyle,
								Line:   36,
								Column: 29,
							},
							{
								Value:  "2",
								Kind:   yaml.ScalarNode,
								Tag:    "!!int",
								Line:   36,
								Column: 34,
							},
						}, Kind: yaml.MappingNode, Tag: "!!map", Style: yaml.FlowStyle, Line: 36, Column: 20},
						Replacements: []*arazzo.PayloadReplacement{
							{
								Target: jsonpointer.JSONPointer("/b"),
								Value:  &yaml.Node{Value: "3", Kind: yaml.ScalarNode, Tag: "!!int", Line: 39, Column: 22},
								Model: marshaller.Model[core.PayloadReplacement]{
									Valid: true,
								},
							},
						},
						Model: marshaller.Model[core.RequestBody]{
							Valid: true,
						},
					},
					SuccessCriteria: []*criterion.Criterion{{Condition: "$statusCode == 200", Type: criterion.CriterionTypeUnion{}, Model: marshaller.Model[core.Criterion]{Valid: true}}},
					OnSuccess: []*arazzo.ReusableSuccessAction{
						{
							Reference: pointer.From[expression.Expression]("$components.successActions.success"),
							Model: marshaller.Model[core.Reusable[*core.SuccessAction]]{
								Valid: true,
							},
						},
					},
					OnFailure: []*arazzo.ReusableFailureAction{
						{
							Reference: pointer.From[expression.Expression]("$components.failureActions.failure"),
							Model: marshaller.Model[core.Reusable[*core.FailureAction]]{
								Valid: true,
							},
						},
					},
					Outputs: sequencedmap.New(sequencedmap.NewElem[string, expression.Expression]("name", "$response.body#/name")),
					Model: marshaller.Model[core.Step]{
						Valid: true,
					},
				},
			},
			Outputs: sequencedmap.New(sequencedmap.NewElem[string, expression.Expression]("name", "$steps.step1.outputs.name")),
			Model: marshaller.Model[core.Workflow]{
				Valid: true,
			},
		},
	},
	Components: &arazzo.Components{
		Parameters: sequencedmap.New(sequencedmap.NewElem("userId", &arazzo.Parameter{
			Name:  "userId",
			In:    pointer.From(arazzo.InQuery),
			Value: &yaml.Node{Value: "123", Kind: yaml.ScalarNode, Tag: "!!str"},
			Model: marshaller.Model[core.Parameter]{
				Valid: true,
			},
		})),
		SuccessActions: sequencedmap.New(sequencedmap.NewElem("success", &arazzo.SuccessAction{
			Name: "success",
			Type: arazzo.SuccessActionTypeEnd,
			Criteria: []criterion.Criterion{{Context: pointer.From(expression.Expression("$statusCode")), Condition: "$statusCode == 200", Type: criterion.CriterionTypeUnion{
				Type: pointer.From(criterion.CriterionTypeSimple),
			}}},
			Model: marshaller.Model[core.SuccessAction]{
				Valid: true,
			},
		})),
		FailureActions: sequencedmap.New(sequencedmap.NewElem("failure", &arazzo.FailureAction{
			Name:       "failure",
			Type:       arazzo.FailureActionTypeRetry,
			RetryAfter: pointer.From(10.0),
			RetryLimit: pointer.From(3),
			Criteria: []criterion.Criterion{{Context: pointer.From(expression.Expression("$statusCode")), Condition: "$statusCode == 500", Type: criterion.CriterionTypeUnion{
				Type: pointer.From(criterion.CriterionTypeSimple),
			}}},
			Model: marshaller.Model[core.FailureAction]{
				Valid: true,
			},
		})),
		Model: marshaller.Model[core.Components]{
			Valid: true,
		},
	},
	Extensions: extensions.New(extensions.NewElem("x-test", &yaml.Node{
		Value:  "some-value",
		Kind:   yaml.ScalarNode,
		Tag:    "!!str",
		Line:   72,
		Column: 9,
	})),
	Model: marshaller.Model[core.Arazzo]{
		Valid: true,
	},
}

func TestArazzo_Unmarshal_Success(t *testing.T) {
	ctx := context.Background()

	data, err := os.ReadFile("testdata/test.arazzo.yaml")
	require.NoError(t, err)

	a, validationErrs, err := arazzo.Unmarshal(ctx, bytes.NewBuffer([]byte(fmt.Sprintf(string(data), ""))))
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	expected := testArazzoInstance

	assert.EqualExportedValues(t, expected, a)
	assert.EqualExportedValues(t, expected.Extensions, a.Extensions)
	assert.EqualExportedValues(t, expected.Info.Extensions, a.Info.Extensions)
	for i, sourceDescription := range expected.SourceDescriptions {
		assert.EqualExportedValues(t, sourceDescription.Extensions, a.SourceDescriptions[i].Extensions)
	}
}

func TestArazzo_RoundTrip_Success(t *testing.T) {
	ctx := context.Background()

	data, err := os.ReadFile("testdata/test.arazzo.yaml")
	require.NoError(t, err)

	doc := fmt.Sprintf(string(data), "")

	a, validationErrs, err := arazzo.Unmarshal(ctx, bytes.NewBuffer([]byte(doc)))
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	outBuf := bytes.NewBuffer([]byte{})

	err = arazzo.Marshal(ctx, a, outBuf)
	require.NoError(t, err)

	assert.Equal(t, doc, outBuf.String())
}

func TestArazzoUnmarshal_ValidationErrors(t *testing.T) {
	data := []byte(`arazzo: 1.0.2
x-test: some-value
info:
  title: My Workflow
sourceDescriptions:
  - name: openapi
    type: openapis
    x-test: some-value
`)

	ctx := context.Background()

	a, validationErrs, err := arazzo.Unmarshal(ctx, bytes.NewBuffer(data))
	require.NoError(t, err)

	assert.Equal(t, []error{
		&validation.Error{Line: 1, Column: 1, Message: "field workflows is missing"},
		&validation.Error{Line: 1, Column: 9, Message: "only Arazzo version 1.0.1 and below is supported"},
		&validation.Error{Line: 4, Column: 3, Message: "field version is missing"},
		&validation.Error{Line: 6, Column: 5, Message: "field url is missing"},
		&validation.Error{Line: 7, Column: 11, Message: "type must be one of [openapi, arazzo]"},
	}, validationErrs)

	expected := &arazzo.Arazzo{
		Arazzo: "1.0.2",
		Info: arazzo.Info{
			Title:   "My Workflow",
			Version: "",
			Model: marshaller.Model[core.Info]{
				Valid: false,
			},
		},
		SourceDescriptions: []*arazzo.SourceDescription{
			{
				Name: "openapi",
				Type: "openapis",
				Extensions: extensions.New(extensions.NewElem("x-test", &yaml.Node{
					Value:  "some-value",
					Kind:   yaml.ScalarNode,
					Tag:    "!!str",
					Line:   8,
					Column: 13,
				})),
			},
		},
		Extensions: extensions.New(extensions.NewElem("x-test", &yaml.Node{
			Value:  "some-value",
			Kind:   yaml.ScalarNode,
			Tag:    "!!str",
			Line:   2,
			Column: 9,
		})),
	}

	assert.EqualExportedValues(t, expected, a)
	assert.EqualExportedValues(t, expected.Extensions, a.Extensions)
	assert.EqualExportedValues(t, expected.Info.Extensions, a.Info.Extensions)
	for i, sourceDescription := range expected.SourceDescriptions {
		assert.EqualExportedValues(t, sourceDescription.Extensions, a.SourceDescriptions[i].Extensions)
	}
}

func TestArazzo_Mutate_Success(t *testing.T) {
	ctx := context.Background()

	data, err := os.ReadFile("testdata/test.arazzo.yaml")
	require.NoError(t, err)

	a, validationErrs, err := arazzo.Unmarshal(ctx, bytes.NewReader([]byte(fmt.Sprintf(string(data), ""))), arazzo.WithSkipValidation())
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	a.Arazzo = "1.0.0"
	a.Info.Title = "My updated workflow title"
	sd := a.SourceDescriptions[0]
	sd.Extensions.Set("x-test", yml.CreateOrUpdateScalarNode(ctx, "some-value", nil))
	a.SourceDescriptions[0] = sd

	a.Workflows[0].Steps[0].Parameters = nil
	a.Components.Parameters.Delete("userId")

	outBuf := bytes.NewBuffer([]byte{})

	err = arazzo.Marshal(ctx, a, outBuf)
	require.NoError(t, err)

	errs := a.Validate(ctx)
	require.Empty(t, errs)

	assert.Equal(t, `arazzo: 1.0.0
info:
  title: My updated workflow title
  summary: A summary
  version: 1.0.0
  x-test: some-value
sourceDescriptions:
  - name: openapi
    url: https://openapi.com
    type: openapi
    x-test: some-value
workflows:
  - workflowId: workflow1
    summary: A summary
    description: A description
    parameters:
      - name: parameter1
        in: query
        value: "123"
    inputs:
      type: object
      properties:
        input1:
          type: string
      required:
        - input1
    steps:
      - stepId: step1
        description: A description
        operationId: operation1
        requestBody:
          contentType: application/json
          payload: {"a": 1, "b": 2}
          replacements:
            - target: /b
              value: 3
        successCriteria:
          - condition: $statusCode == 200
        onSuccess:
          - reference: $components.successActions.success
        onFailure:
          - reference: $components.failureActions.failure
        outputs:
          name: $response.body#/name
    outputs:
      name: $steps.step1.outputs.name
components:
  parameters: {}
  successActions:
    success:
      name: success
      type: end
      criteria:
        - context: $statusCode
          condition: $statusCode == 200
          type: simple
  failureActions:
    failure:
      name: failure
      type: retry
      retryAfter: 10
      retryLimit: 3
      criteria:
        - context: $statusCode
          condition: $statusCode == 500
x-test: some-value
`, outBuf.String())
}

func TestArazzo_Create_Success(t *testing.T) {
	outBuf := bytes.NewBuffer([]byte{})

	ctx := context.Background()

	err := arazzo.Marshal(ctx, testArazzoInstance, outBuf)
	require.NoError(t, err)

	errs := testArazzoInstance.Validate(ctx)
	require.Empty(t, errs)

	data, err := os.ReadFile("testdata/test.arazzo.yaml")
	require.NoError(t, err)

	assert.Equal(t, fmt.Sprintf(string(data), "\n          type: simple"), outBuf.String())
}

func TestArazzo_Deconstruct_Success(t *testing.T) {
	data := []byte(`arazzo: 1.0.0
x-test: some-value
info:
  title: My Workflow
  summary: A summary
  version: 1.0.0
  x-test: some-value
sourceDescriptions:
  - name: openapi
    url: https://openapi.com
    type: openapi
    x-test: some-value
workflows: []
`)

	ctx := context.Background()

	a, validationErrs, err := arazzo.Unmarshal(ctx, bytes.NewReader(data))
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	a.Extensions = extensions.New()
	a.Info.Summary = nil
	a.Info.Extensions = extensions.New()
	a.SourceDescriptions = []*arazzo.SourceDescription{}

	outBuf := bytes.NewBuffer([]byte{})

	err = arazzo.Marshal(ctx, a, outBuf)
	require.NoError(t, err)

	assert.Equal(t, `arazzo: 1.0.0
info:
  title: My Workflow
  version: 1.0.0
sourceDescriptions: []
workflows: []
`, outBuf.String())
}

type args struct {
	location          string
	validationIgnores []string
	skipRoundTrip     bool
}

var stressTests = []struct {
	name      string
	args      args
	wantTitle string
}{
	{
		name: "Speakeasy Bar Workflows",
		args: args{
			location: "testdata/speakeasybar.arazzo.yaml",
		},
		wantTitle: "Speakeasy Bar Workflows",
	},
	{
		name: "Bump.sh Train Travel API",
		args: args{
			location: "testdata/traintravelapi.arazzo.yaml",
		},
		wantTitle: "Train Travel API - Book & Pay",
	},
	{
		name: "Redocly Museum API",
		args: args{
			location: "https://raw.githubusercontent.com/Redocly/museum-openapi-example/2770b2b2e59832d245c7b0eb0badf6568d7efb53/arazzo/museum-api.arazzo.yaml",
			validationIgnores: []string{
				"[71:24] invalid jsonpath expression: Error at line 1, column 7: unexpected token when parsing segment",  // legit invalid RFC 9535 syntax
				"[107:24] invalid jsonpath expression: Error at line 1, column 7: unexpected token when parsing segment", // legit invalid RFC 9535 syntax
			},
		},
		wantTitle: "Redocly Museum API Test Workflow",
	},
	{
		name: "Redocly Museum Tickets",
		args: args{
			location: "https://raw.githubusercontent.com/Redocly/museum-openapi-example/2770b2b2e59832d245c7b0eb0badf6568d7efb53/arazzo/museum-tickets.arazzo.yaml",
		},
		wantTitle: "Redocly Museum Tickets Workflow",
	},
	{
		name: "Redocly Warp API",
		args: args{
			location: "https://raw.githubusercontent.com/Redocly/warp-single-sidebar/b78fc09da52d7755e92e1bc8f990edd37421cbde/apis/arazzo.yaml",
			validationIgnores: []string{
				"[63:24] invalid jsonpath expression: Error at line 1, column 12: unexpected token when parsing segment", // legit invalid RFC 9535 syntax
			},
		},
		wantTitle: "Warp API",
	},
	{
		name: "Arazzo Extended Parameters Example",
		args: args{
			location: "https://raw.githubusercontent.com/OAI/Arazzo-Specification/23852b8b0d13ab1e3288a57a990611ffed45ab5d/examples/1.0.0/ExtendedParametersExample.arazzo.yaml",
		},
		wantTitle: "Public Zoo API",
	},
	{
		name: "Arazzo FAPI-PAR Example",
		args: args{
			location: "https://raw.githubusercontent.com/OAI/Arazzo-Specification/23852b8b0d13ab1e3288a57a990611ffed45ab5d/examples/1.0.0/FAPI-PAR.arazzo.yaml",
		},
		wantTitle: "PAR, Authorization and Token workflow",
	},
	{
		name: "Arazzo Login and Retrieve Pets Example",
		args: args{
			location: "https://raw.githubusercontent.com/OAI/Arazzo-Specification/23852b8b0d13ab1e3288a57a990611ffed45ab5d/examples/1.0.0/LoginAndRetrievePets.arazzo.yaml",
		},
		wantTitle: "A pet purchasing workflow",
	},
	{
		name: "Arazzo BNPL Example",
		args: args{
			location: "https://raw.githubusercontent.com/OAI/Arazzo-Specification/23852b8b0d13ab1e3288a57a990611ffed45ab5d/examples/1.0.0/bnpl-arazzo.yaml",
			validationIgnores: []string{
				"$response.headers.Location", // doc should be referencing `$response.header.Location`
			},
		},
		wantTitle: "BNPL Workflow description",
	},
	{
		name: "Arazzo OAuth Example",
		args: args{
			location: "https://raw.githubusercontent.com/OAI/Arazzo-Specification/23852b8b0d13ab1e3288a57a990611ffed45ab5d/examples/1.0.0/oauth.arazzo.yaml",
			validationIgnores: []string{
				"[65:24] invalid jsonpath expression: Error at line 1, column 15: unexpected token when parsing segment",  // legit invalid RFC 9535 syntax
				"[105:24] invalid jsonpath expression: Error at line 1, column 15: unexpected token when parsing segment", // legit invalid RFC 9535 syntax
				"[155:24] invalid jsonpath expression: Error at line 1, column 15: unexpected token when parsing segment", // legit invalid RFC 9535 syntax
				"[175:24] invalid jsonpath expression: Error at line 1, column 15: unexpected token when parsing segment", // legit invalid RFC 9535 syntax
			},
		},
		wantTitle: "Example OAuth service",
	},
	{
		name: "Arazzo Pet Coupons Example",
		args: args{
			location: "https://raw.githubusercontent.com/OAI/Arazzo-Specification/23852b8b0d13ab1e3288a57a990611ffed45ab5d/examples/1.0.0/pet-coupons.arazzo.yaml",
		},
		wantTitle: "Petstore - Apply Coupons",
	},
	{
		name: "Arazzo-Runner Basic ARZ Example",
		args: args{
			location: "https://raw.githubusercontent.com/AdrianMachado/arazzo-runner/4da957365496d213fba4c51b6245cc209af82bfa/tests/basic.arz.json",
		},
		wantTitle: "Simple Arazzo test",
	},
	{
		name: "Frank Kilcommins Online Store Example",
		args: args{
			location: "https://raw.githubusercontent.com/frankkilcommins/simple-spectral-arazzo-GA/4ec8856f1cf21c0f77597c715c150ef3e2772a89/apis/OnlineStore.arazzo.yaml",
			validationIgnores: []string{
				"field title is missing", // legit issue
				"operationId must be a valid expression if there are multiple OpenAPI source descriptions", // legit issue
				"$responses.body.menuItems[0].subcategories[0].id",                                         // legit issue
			},
			skipRoundTrip: true, // Has lots of validation errors that impact round tripping
		},
		wantTitle: "",
	},
	{
		name: "Itarazzo Library Example",
		args: args{
			location: "https://raw.githubusercontent.com/leidenheit/itarazzo-library/3b335e1c4293444add52b5f2476420e2d871b1a5/src/test/resources/test.arazzo.yaml",
			validationIgnores: []string{
				"expression is not valid, must begin with $: <root><id>4711</id><name>Chocolate</name></root>",          // legit issue
				"[32:24] invalid jsonpath expression: Error at line 1, column 0: unexpected token",                      // unsupported version: draft-goessner-dispatch-jsonpath-00
				"[36:24] invalid jsonpath expression: Error at line 1, column 5: unexpected token when parsing segment", // unsupported version: draft-goessner-dispatch-jsonpath-00
			},
		},
		wantTitle: "A cookie eating workflow",
	},
	{
		name: "Itarazzo Client Pet Store Example",
		args: args{
			location: "https://raw.githubusercontent.com/leidenheit/itarazzo-client/b3c126d28bf80ae7d74861c08509be33b83c5ddf/src/test/resources/pet-store.arazzo.yaml",
			validationIgnores: []string{
				"jsonpointer must start with /: $.status", // legit issues TODO: improve the error returned as it is wrong
				"jsonpointer must start with /: $.id",     // legit issues TODO: improve the error returned as it is wrong
				"[81:24] invalid jsonpath expression: Error at line 1, column 7: unexpected token when parsing segment",  // unsupported version: draft-goessner-dispatch-jsonpath-00
				"[110:24] invalid jsonpath expression: Error at line 1, column 5: unexpected token when parsing segment", // unsupported version: draft-goessner-dispatch-jsonpath-00
				"[114:24] invalid jsonpath expression: Error at line 1, column 9: unexpected token when parsing segment", // unsupported version: draft-goessner-dispatch-jsonpath-00
			},
		},
		wantTitle: "PetStore - Example of Workflows",
	},
	{
		name: "Ritza build-a-bot workflow",
		args: args{
			location: "https://raw.githubusercontent.com/ritza-co/e2e-testing-arazzo/c0615c3708a1e4c0fcaeb79edae78ddc4eb5ba82/arazzo.yaml",
			validationIgnores: []string{
				"[42:24] invalid jsonpath expression: Error at line 1, column 8: unexpected token", // legit invalid RFC 9535 syntax
			},
		},
		wantTitle: "Build-a-Bot Workflow",
	},
	{
		name: " API-Flows adyen-giving workflow",
		args: args{
			location: "https://raw.githubusercontent.com/API-Flows/openapi-workflow-registry/75c237ce1b155ba9f8dc7f065759df7ae1cbbbe5/root/adyen/adyen-giving.yaml",
			validationIgnores: []string{
				"in must be one of [path, query, header, cookie] but was body",
			},
		},
		wantTitle: "Adyen Giving",
	},
	{
		name: "API-Flows simple workflow",
		args: args{
			location: "https://raw.githubusercontent.com/API-Flows/openapi-workflow-parser/6b28ba4def262969c5a96bc54d08433e6c336643/src/test/resources/1.0.0/simple.workflow.yaml",
		},
		wantTitle: "simple",
	},
	// Disabled for now as it is currently failing round tripping due to missing conditions
	// {
	// 	name: "Kartikhub swap tokens workflow",
	// 	args: args{
	// 		location: "https://raw.githubusercontent.com/Kartikhub/web3-basics/be13fa7e6fdf386eef08bba2843d4a8b615561b9/swap-react/docs/swap-transaction-arazzo.yaml",
	// 		validationIgnores: []string{ // All valid issues
	// 			"field condition is missing",
	// 			"condition is required",
	// 			"field value is missing",
	// 			"expression is not valid, must begin with $",
	// 		},
	// 	},
	// 	wantTitle: "Swap Tokens",
	// },
}

func TestArazzo_StressTests_Validate(t *testing.T) {
	for _, tt := range stressTests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			var r io.ReadCloser
			if strings.HasPrefix(tt.args.location, "testdata/") {
				var err error
				r, err = os.Open(tt.args.location)
				require.NoError(t, err)
			} else {
				var err error
				r, err = downloadFile(tt.args.location)
				require.NoError(t, err)
			}
			defer r.Close()

			arazzo, validationErrs, err := arazzo.Unmarshal(ctx, r)
			require.NoError(t, err)

			handleValidationErrors(t, validationErrs, tt.args.validationIgnores)

			assert.Equal(t, tt.wantTitle, arazzo.Info.Title)
		})
	}
}

func TestArazzo_StressTests_RoundTrip(t *testing.T) {
	for _, tt := range stressTests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.skipRoundTrip {
				t.SkipNow()
			}

			ctx := context.Background()

			var r io.ReadCloser
			if strings.HasPrefix(tt.args.location, "testdata/") {
				var err error
				r, err = os.Open(tt.args.location)
				require.NoError(t, err)
			} else {
				var err error
				r, err = downloadFile(tt.args.location)
				require.NoError(t, err)
			}
			defer r.Close()

			inBuf := bytes.NewBuffer([]byte{})
			tee := io.TeeReader(r, inBuf)

			a, _, err := arazzo.Unmarshal(ctx, tee, arazzo.WithSkipValidation())
			require.NoError(t, err)

			outBuf := bytes.NewBuffer([]byte{})

			err = arazzo.Marshal(ctx, a, outBuf)
			require.NoError(t, err)

			sanitizedData := inBuf.Bytes()

			if a.GetCore().Config.OutputFormat == yml.OutputFormatYAML {
				sanitizedData, err = roundTripYamlOnly(sanitizedData)
				require.NoError(t, err)
			}

			assert.Equal(t, string(sanitizedData), outBuf.String())
		})
	}
}

func downloadFile(url string) (io.ReadCloser, error) {
	tempDir := filepath.Join(os.TempDir(), "speakeasy-api_arazzo")

	if err := os.MkdirAll(tempDir, os.ModePerm); err != nil {
		return nil, err
	}

	// hash url to create a unique filename
	hash := sha256.Sum256([]byte(url))
	filename := fmt.Sprintf("%x", hash)

	filepath := filepath.Join(tempDir, filename)

	// check if file exists and return it otherwise download it
	r, err := os.Open(filepath)
	if err == nil {
		return r, nil
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	buf := bytes.NewBuffer([]byte{})
	tee := io.TeeReader(resp.Body, buf)

	f, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	_, err = io.Copy(f, tee)

	return io.NopCloser(buf), err
}

func roundTripYamlOnly(data []byte) ([]byte, error) {
	var node yaml.Node

	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, err
	}

	b := bytes.NewBuffer([]byte{})
	enc := yaml.NewEncoder(b)

	cfg := yml.GetConfigFromDoc(data, &node)

	enc.SetIndent(cfg.Indentation)
	if err := enc.Encode(&node); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func handleValidationErrors(t *testing.T, errs []error, docSpecificIgnores []string) []error {
	t.Helper()

	errs = filterCurrentUncomfirmedValidationErrors(errs, docSpecificIgnores)
	if !assert.Empty(t, errs) {
		for _, err := range errs {
			t.Log(err.Error())
		}
		t.FailNow()
	}
	return errs
}

func filterCurrentUncomfirmedValidationErrors(validationErrs []error, docSpecificIgnores []string) []error {
	var filteredValidationErrs []error

	ignoreForNow := []string{
		"expression is not valid, only json pointers are allowed after $response.body",                    // If the error is about using dot notation after the body lets ignore it for now as this is an unconfirmed part of the spec
		"expression is not valid, json pointers are not allowed in current context: $sourceDescriptions.", // Currently a common error as until recently it wasn't documented correctly in the spec
		"operationPath must reference the url of a sourceDescription",                                     // Currently a common error as until recently it wasn't documented correctly in the spec (related to the above)
	}

	ignoreForNow = append(ignoreForNow, docSpecificIgnores...)

	for _, err := range validationErrs {
		ignored := false
		for _, ignore := range ignoreForNow {
			if strings.Contains(err.Error(), ignore) {
				ignored = true
				break
			}
		}

		if ignored {
			continue
		}

		filteredValidationErrs = append(filteredValidationErrs, err)
	}

	return filteredValidationErrs
}
