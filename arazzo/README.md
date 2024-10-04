# github.com/speakeasy-api/openapi/arazzo

The Arazzo package provides an API for working with Arazzo documents including reading, creating, mutating, walking and validating them.

For more details on the Arazzo specification see [Speakeasy's Arazzo Documentation](https://www.speakeasy.com/openapi/arazzo).

For details on the API available via the `arazzo` package see [https://pkg.go.dev/github.com/speakeasy-api/openapi/arazzo](https://pkg.go.dev/github.com/speakeasy-api/openapi/arazzo).

## Reading

```go
package main

import (
 "context"
 "fmt"
 "os"

 "github.com/speakeasy-api/openapi/arazzo"
)

func main() {
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
```

## Creating

```go
package main

import (
 "context"
 "fmt"

 "github.com/speakeasy-api/openapi/arazzo"
 "github.com/speakeasy-api/openapi/pointer"
)

func main() {
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
```

## Mutating

```go
package main

import (
 "context"
 "fmt"

 "github.com/speakeasy-api/openapi/arazzo"
)

func main() {
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
```

## Walking

```go
package main

import (
 "context"
 "fmt"
 "os"

 "github.com/speakeasy-api/openapi/arazzo"
)

func main() {
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
```

## Validating

```go
package main

import (
 "context"
 "fmt"
 "os"

 "github.com/speakeasy-api/openapi/arazzo"
)

func main() {
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
```
