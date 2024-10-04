package arazzo_test

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/pointer"
)

func Example_readAndMutate() {
	ctx := context.Background()

	r, err := os.Open("testdata/speakeasybar.arazzo.yaml")
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
	a.Info.Title = "Speakeasy Bar Workflows"

	buf := bytes.NewBuffer([]byte{})

	// Marshal the document to a writer
	if err := arazzo.Marshal(ctx, a, buf); err != nil {
		panic(err)
	}

	fmt.Println(buf.String())
}

// The below examples should be copied into the README.md file if every changed TODO: automate this
func Example_reading() {
	ctx := context.Background()

	f, err := os.Open("arazzo.yaml")
	if err != nil {
		panic(err)
	}

	arazzo, validationErrs, err := arazzo.Unmarshal(ctx, f)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", arazzo)
	fmt.Printf("%+v\n", validationErrs)
}

func Example_creating() {
	ctx := context.Background()

	arazzo := &arazzo.Arazzo{
		Arazzo: arazzo.Version,
		Info: arazzo.Info{
			Title:   "My Workflow",
			Summary: pointer.From("A summary"),
			Version: "1.0.0",
		},
		// ...
	}

	buf := bytes.NewBuffer([]byte{})

	err := arazzo.Marshal(ctx, buf)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s", buf.String())
}

func Example_mutating() {
	ctx := context.Background()

	f, err := os.Open("arazzo.yaml")
	if err != nil {
		panic(err)
	}

	arazzo, _, err := arazzo.Unmarshal(ctx, f)
	if err != nil {
		panic(err)
	}

	arazzo.Info.Title = "My updated workflow title"

	buf := bytes.NewBuffer([]byte{})

	if err := arazzo.Marshal(ctx, buf); err != nil {
		panic(err)
	}

	fmt.Printf("%s", buf.String())
}

func Example_walking() {
	ctx := context.Background()

	f, err := os.Open("arazzo.yaml")
	if err != nil {
		panic(err)
	}

	a, _, err := arazzo.Unmarshal(ctx, f)
	if err != nil {
		panic(err)
	}

	err = arazzo.Walk(ctx, a, func(ctx context.Context, node, parent arazzo.MatchFunc, a *arazzo.Arazzo) error {
		return node(arazzo.Matcher{
			Workflow: func(workflow *arazzo.Workflow) error {
				fmt.Printf("Workflow: %s\n", workflow.WorkflowID)
				return nil
			},
		})
	})
	if err != nil {
		panic(err)
	}
}

func Example_validating() {
	ctx := context.Background()

	f, err := os.Open("arazzo.yaml")
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
}
