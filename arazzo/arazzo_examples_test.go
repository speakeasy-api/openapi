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
	// arazzo: 1.0.1
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
	// [3:3] info.version is missing
	// [13:9] step at least one of operationId, operationPath or workflowId fields must be set
}
