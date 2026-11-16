package arazzo_test

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/pointer"
)

// Example_reading demonstrates how to read and parse an Arazzo document from a file.
// Shows loading a document, handling validation errors, and making simple modifications.
func Example_reading() {
	ctx := context.Background()

	r, err := os.Open("testdata/simple.arazzo.yaml")
	if err != nil {
		panic(err)
	}
	defer r.Close()

	// Unmarshal the Arazzo document which will also validate it against the Arazzo Specification
	a, validationErrs, err := arazzo.Unmarshal(ctx, r)
	if err != nil {
		panic(err)
	}

	// Validation errors are returned separately from any errors that block the document from being unmarshalled
	// allowing an invalid document to be mutated and fixed before being marshalled again
	for _, err := range validationErrs {
		fmt.Println(err.Error())
	}

	// Mutate the document by just modifying the returned Arazzo object
	a.Info.Title = "Updated Simple Workflow"

	buf := bytes.NewBuffer([]byte{})

	// Marshal the document to a writer
	if err := arazzo.Marshal(ctx, a, buf); err != nil {
		panic(err)
	}

	fmt.Println(buf.String())
	// Output:
	// arazzo: 1.0.0
	// info:
	//   title: Updated Simple Workflow
	//   version: 1.0.0
	// sourceDescriptions:
	//   - name: api
	//     url: https://api.example.com/openapi.yaml
	//     type: openapi
	// workflows:
	//   - workflowId: simpleWorkflow
	//     summary: A simple workflow
	//     steps:
	//       - stepId: step1
	//         operationId: getUser
	//         parameters:
	//           - name: id
	//             in: path
	//             value: "123"
}

// Example_creating demonstrates how to create an Arazzo document from scratch.
// Shows building a basic workflow document with info and version programmatically.
func Example_creating() {
	ctx := context.Background()

	a := &arazzo.Arazzo{
		Arazzo: arazzo.Version,
		Info: arazzo.Info{
			Title:   "My Workflow",
			Summary: pointer.From("A summary"),
			Version: "1.0.1",
		},
		// ...
	}

	buf := bytes.NewBuffer([]byte{})

	err := arazzo.Marshal(ctx, a, buf)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s", buf.String())
	// Output:
	// arazzo: 1.1.0
	// info:
	//   title: My Workflow
	//   summary: A summary
	//   version: 1.0.1
}

// Example_mutating demonstrates how to modify an existing Arazzo document.
// Shows loading a document, changing properties, and marshaling it back to YAML.
func Example_mutating() {
	ctx := context.Background()

	f, err := os.Open("testdata/simple.arazzo.yaml")
	if err != nil {
		panic(err)
	}

	a, _, err := arazzo.Unmarshal(ctx, f)
	if err != nil {
		panic(err)
	}

	a.Info.Title = "Updated Simple Workflow"

	buf := bytes.NewBuffer([]byte{})

	if err := arazzo.Marshal(ctx, a, buf); err != nil {
		panic(err)
	}

	fmt.Printf("%s", buf.String())
	// Output:
	// arazzo: 1.0.0
	// info:
	//   title: Updated Simple Workflow
	//   version: 1.0.0
	// sourceDescriptions:
	//   - name: api
	//     url: https://api.example.com/openapi.yaml
	//     type: openapi
	// workflows:
	//   - workflowId: simpleWorkflow
	//     summary: A simple workflow
	//     steps:
	//       - stepId: step1
	//         operationId: getUser
	//         parameters:
	//           - name: id
	//             in: path
	//             value: "123"
}

// Example_walking demonstrates how to traverse an Arazzo document using the iterator API.
// Shows how to match different types of objects like workflows during traversal.
func Example_walking() {
	ctx := context.Background()

	f, err := os.Open("testdata/simple.arazzo.yaml")
	if err != nil {
		panic(err)
	}

	a, _, err := arazzo.Unmarshal(ctx, f)
	if err != nil {
		panic(err)
	}

	for item := range arazzo.Walk(ctx, a) {
		err := item.Match(arazzo.Matcher{
			Workflow: func(workflow *arazzo.Workflow) error {
				fmt.Printf("Workflow: %s\n", workflow.WorkflowID)
				return nil
			},
		})
		if err != nil {
			panic(err)
		}
	}
	// Output:
	// Workflow: simpleWorkflow
}

// Example_upgrading demonstrates how to upgrade an Arazzo 1.0.x document to the latest supported version.
// Shows loading a document, upgrading it and validating the result.
func Example_upgrading() {
	ctx := context.Background()

	f, err := os.Open("testdata/simple.arazzo.yaml")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	a, _, err := arazzo.Unmarshal(ctx, f, arazzo.WithSkipValidation())
	if err != nil {
		panic(err)
	}

	originalVersion := a.Arazzo

	upgraded, err := arazzo.Upgrade(ctx, a)
	if err != nil {
		panic(err)
	}

	fmt.Printf("upgraded: %t, from: %s, to: %s\n", upgraded, originalVersion, a.Arazzo)
	fmt.Printf("validation errors: %d\n", len(a.Validate(ctx)))
	// Output:
	// upgraded: true, from: 1.0.0, to: 1.1.0
	// validation errors: 0
}

// Example_validating demonstrates how to validate an Arazzo document.
// Shows loading an invalid document and handling validation errors.
func Example_validating() {
	ctx := context.Background()

	f, err := os.Open("testdata/invalid.arazzo.yaml")
	if err != nil {
		panic(err)
	}

	_, validationErrs, err := arazzo.Unmarshal(ctx, f)
	if err != nil {
		panic(err)
	}

	for _, err := range validationErrs {
		fmt.Printf("%s\n", err.Error())
	}
	// Output:
	// [3:3] error validation-required-field `info.version` is required
	// [13:9] error validation-required-field step at least one of operationId, operationPath or workflowId fields must be set
	// [22:13] error validation-required-field parameter.in is required within a step when workflowId is not set
}

// Example_validatingSources demonstrates how to validate an Arazzo document against its source descriptions.
// Shows loading the OpenAPI and Arazzo documents a workflow references and checking its steps against them.
func Example_validatingSources() {
	ctx := context.Background()

	location := "testdata/sources/invalid.arazzo.yaml"

	f, err := os.Open(location)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	a, _, err := arazzo.Unmarshal(ctx, f)
	if err != nil {
		panic(err)
	}

	// Relative source description URLs are resolved against $self, or the document's location.
	sources, loadErrs := arazzo.LoadSourceDescriptions(ctx, a, arazzo.LoadSourcesOptions{DocumentLocation: location})
	fmt.Printf("load errors: %d\n", len(loadErrs))

	validationErrs := a.ValidateSources(ctx, sources)
	for _, err := range validationErrs[:3] {
		fmt.Printf("%s\n", err.Error())
	}
	// Output:
	// load errors: 0
	// [16:9] error validation-invalid-reference workflow.dependsOn references workflowId `missingWorkflow` which doesn't exist in sourceDescription `shared`
	// [19:22] error validation-operation-not-found step.operationId `missingOperation` not found in sourceDescription `petstore`
	// [21:24] error validation-operation-not-found step.operationPath `{$sourceDescriptions.petstore.url}#/paths/~1missing/get` doesn't reference an operation in sourceDescription `petstore`
}
