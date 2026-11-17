package arazzo

import (
	"github.com/speakeasy-api/openapi/arazzo/criterion"
	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/yml"
	"go.yaml.in/yaml/v3"
)

// Bootstrap creates a new Arazzo document with best practice examples.
// This serves as a template demonstrating a workflow with inputs and outputs, steps that pass values between each
// other, success criteria, and reusable components for parameters and failure actions.
func Bootstrap() *Arazzo {
	return &Arazzo{
		Arazzo: Version,
		Info: Info{
			Title:       "Pet adoption workflows",
			Summary:     pointer.From("Workflows for finding and adopting pets"),
			Description: pointer.From("A new Arazzo document template ready to populate. Replace the source description with your own OpenAPI document and update the workflows to call its operations."),
			Version:     "1.0.0",
		},
		SourceDescriptions: SourceDescriptions{
			{
				Name: "petStore",
				URL:  "https://api.example.com/openapi.yaml",
				Type: SourceDescriptionTypeOpenAPI,
			},
		},
		Workflows:  Workflows{createBootstrapAdoptPetWorkflow()},
		Components: createBootstrapComponents(),
	}
}

// createBootstrapAdoptPetWorkflow creates a workflow that passes the output of one step to the next.
func createBootstrapAdoptPetWorkflow() *Workflow {
	return &Workflow{
		WorkflowID:  "adoptPet",
		Summary:     pointer.From("Find an available pet and adopt it"),
		Description: pointer.From("Searches for an available pet of the requested species, then adopts the first one found."),
		Inputs: oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
			Type: oas3.NewTypeFromString(oas3.SchemaTypeObject),
			Properties: sequencedmap.New(
				sequencedmap.NewElem("species", oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
					Type:        oas3.NewTypeFromString(oas3.SchemaTypeString),
					Description: pointer.From("The species of pet to adopt"),
				})),
				sequencedmap.NewElem("ownerName", oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
					Type:        oas3.NewTypeFromString(oas3.SchemaTypeString),
					Description: pointer.From("The name of the new owner"),
				})),
			),
			Required: []string{"species", "ownerName"},
		}),
		Steps: Steps{
			{
				StepID:      "findPets",
				Description: pointer.From("Find available pets of the requested species"),
				OperationID: pointer.From(expression.Expression("$sourceDescriptions.petStore.findPets")),
				Parameters: []*ReusableParameter{
					{Reference: pointer.From(expression.Expression("$components.parameters.apiVersion"))},
					{Object: &Parameter{Name: "species", In: pointer.From(InQuery), Value: yml.CreateStringNode("$inputs.species")}},
					{Object: &Parameter{Name: "status", In: pointer.From(InQuery), Value: yml.CreateStringNode("available")}},
				},
				SuccessCriteria: []*criterion.Criterion{
					{Condition: "$statusCode == 200"},
					{
						Context:   pointer.From(expression.Expression("$response.body")),
						Condition: "$[?@.status == 'available']",
						Type: criterion.CriterionTypeUnion{
							ExpressionType: &criterion.CriterionExpressionType{Type: criterion.CriterionTypeJsonPath, Version: criterion.CriterionTypeVersionRFC9535},
						},
					},
				},
				Outputs: sequencedmap.New(
					sequencedmap.NewElem("petId", NewOutputValueFromExpression("$response.body#/0/id")),
				),
			},
			{
				StepID:      "adoptPet",
				Description: pointer.From("Adopt the first pet found"),
				OperationID: pointer.From(expression.Expression("$sourceDescriptions.petStore.adoptPet")),
				Parameters: []*ReusableParameter{
					{Reference: pointer.From(expression.Expression("$components.parameters.apiVersion"))},
					{Object: &Parameter{Name: "petId", In: pointer.From(InPath), Value: yml.CreateStringNode("$steps.findPets.outputs.petId")}},
				},
				RequestBody: &RequestBody{
					ContentType: pointer.From("application/json"),
					Payload: &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
						yml.CreateStringNode("ownerName"),
						yml.CreateStringNode("$inputs.ownerName"),
					}},
				},
				SuccessCriteria: []*criterion.Criterion{{Condition: "$statusCode == 201"}},
				Outputs: sequencedmap.New(
					sequencedmap.NewElem("adoptionId", NewOutputValueFromExpression("$response.body#/id")),
				),
			},
		},
		FailureActions: []*ReusableFailureAction{
			{Reference: pointer.From(expression.Expression("$components.failureActions.retryWhenBusy"))},
		},
		Outputs: sequencedmap.New(
			sequencedmap.NewElem("petId", NewOutputValueFromExpression("$steps.findPets.outputs.petId")),
			sequencedmap.NewElem("adoptionId", NewOutputValueFromExpression("$steps.adoptPet.outputs.adoptionId")),
		),
	}
}

// createBootstrapComponents creates reusable components shared by the workflow's steps.
func createBootstrapComponents() *Components {
	return &Components{
		Parameters: sequencedmap.New(
			sequencedmap.NewElem("apiVersion", &Parameter{
				Name:  "X-API-Version",
				In:    pointer.From(InHeader),
				Value: yml.CreateStringNode("2025-01-01"),
			}),
		),
		FailureActions: sequencedmap.New(
			sequencedmap.NewElem("retryWhenBusy", &FailureAction{
				Name:       "retryWhenBusy",
				Type:       FailureActionTypeRetry,
				RetryAfter: pointer.From(1.0),
				RetryLimit: pointer.From(3),
				Criteria:   []criterion.Criterion{{Condition: "$statusCode == 503"}},
			}),
		),
	}
}
