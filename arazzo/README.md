<p align="center">
  <p align="center">
    <img  width="200px" alt="Arazzo" src="https://github.com/user-attachments/assets/ded936d5-3fd9-439f-925a-a2959735b71a">
  </p>
  <h1 align="center"><b>Arazzo Parser</b></h1>
  <p align="center">An API for working with <a href="https://www.speakeasy.com/openapi/arazzo">Arazzo documents</a> including: read, walk, create, mutate, and validate
</p>
  <p align="center">
    <a href="https://speakeasy.com/"><img src="https://custom-icon-badges.demolab.com/badge/-Built%20By%20Speakeasy-212015?style=for-the-badge&logoColor=FBE331&logo=speakeasy&labelColor=545454" /></a>
    <a href="https://github.com/speakeasy-api/openapi/releases/latest"><img alt="Release" src="https://img.shields.io/github/release/goadesign/goa.svg?style=for-the-badge"></a>
    <a href="https://pkg.go.dev/github.com/speakeasy-api/openapi/arazzo?tab=doc"><img alt="Go Doc" src="https://img.shields.io/badge/godoc-reference-blue.svg?style=for-the-badge"></a>
   <br />
    <a href="https://github.com/speakeasy-api/openapi/actions/workflows/test.yaml"><img alt="GitHub Action: Test" src="https://img.shields.io/github/actions/workflow/status/speakeasy-api/openapi/test.yaml?style=for-the-badge"></a>
    <a href="https://goreportcard.com/report/github.com/speakeasy-api/openapi"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/speakeasy-api/openapi?style=for-the-badge"></a>
    <a href="/LICENSE"><img alt="Software License" src="https://img.shields.io/badge/license-MIT-blue.svg?style=for-the-badge"></a>
  </p>
</p>

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

## Contributing

This repository is maintained by Speakeasy, but we welcome and encourage contributions from the community to help improve its capabilities and stability.

### How to Contribute

1. **Open Issues**: Found a bug or have a feature suggestion? Open an issue to describe what you'd like to see changed.

2. **Pull Requests**: We welcome pull requests! If you'd like to contribute code:
   - Fork the repository
   - Create a new branch for your feature/fix
   - Submit a PR with a clear description of the changes and any related issues

3. **Feedback**: Share your experience using the packages or suggest improvements.

All contributions, whether they're bug reports, feature requests, or code changes, help make this project better for everyone.

Please ensure your contributions adhere to our coding standards and include appropriate tests where applicable.
