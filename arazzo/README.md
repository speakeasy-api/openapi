[![https://www.speakeasy.com](../.github/assets/speakeasy.png?raw=true)](https://www.speakeasy.com)

# [github.com/speakeasy-api/openapi/arazzo](https://github.com/speakeasy-api/openapi/arazzo)

[![Reference](https://godoc.org/github.com/speakeasy-api/openapi/arazzo?status.svg)](http://godoc.org/github.com/speakeasy-api/openapi/arazzo)
![Pipeline](https://github.com/speakeasy-api/openapi/workflows/test/badge.svg)
[![GoReportCard](https://goreportcard.com/badge/github.com/speakeasy-api/openapi)](https://goreportcard.com/report/github.com/speakeasy-api/openapi)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

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
