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

 data, err := os.ReadFile("arazzo.yaml")
 if err != nil {
  panic(err)
 }

 arazzo, validationErrs, err := arazzo.Unmarshal(ctx, data, "arazzo.yaml")
 if err != nil {
  panic(err)
 }

 fmt.Printf("%+v\n", arazzo)
 fmt.Printf("%+v\n", validationErrs)
}
```

## Creating

```go
package main

import (
 "context"
 "fmt"

 "github.com/speakeasy-api/openapi/arazzo"
)

func main() {
 ctx := context.Background()

 arazzo := &arazzo.Arazzo{
  Arazzo: Version,
  Info: arazzo.Info{
   Title:   "My Workflow",
   Summary: arazzo.pointer.From("A summary"),
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

 data, err := os.ReadFile("arazzo.yaml")
 if err != nil {
  panic(err)
 }

 arazzo, _, err := arazzo.Unmarshal(ctx, data, "arazzo.yaml")
 if err != nil {
  panic(err)
 }
 
 arazzo.Info.Title = "My updated workflow title"

 buf := bytes.NewBuffer([]byte{})

 err := arazzo.Marshal(ctx, buf)
 if err != nil {
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

 data, err := os.ReadFile("arazzo.yaml")
 if err != nil {
  panic(err)
 }

 arazzo, _, err := arazzo.Unmarshal(ctx, data, "arazzo.yaml")
 if err != nil {
  panic(err)
 }

 err = arazzo.Walk(ctx, func(ctx context.Context, node, parent arazzo.MatchFunc, arazzo *arazzo.Arazzo) error {
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

 data, err := os.ReadFile("arazzo.yaml")
 if err != nil {
  panic(err)
 }

 arazzo, validationErrs, err := arazzo.Unmarshal(ctx, data, "arazzo.yaml")
 if err != nil {
  panic(err)
 }
 
 for _, err := range validationErrs {
  fmt.Printf("%s\n", err.Error())
 }
}
```
