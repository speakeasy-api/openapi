package arazzo_test

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/speakeasy-api/openapi/arazzo"
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
