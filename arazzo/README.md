<p align="center">
  <p align="center">
    <img  width="200px" alt="Arazzo" src="https://github.com/user-attachments/assets/9b7ef9b6-ae9a-4e83-9980-2f4361e8bf4c">

  </p>
  <h1 align="center"><b>Arazzo Parser</b></h1>
  <p align="center">An API for working with <a href="https://www.speakeasy.com/openapi/arazzo">Arazzo documents</a> including: read, walk, create, mutate, and validate
</p>
  <p align="center">
    <!-- Arazzo Reference badge -->
     <a href="https://www.speakeasy.com/openapi/arazzo"><img alt="Arazzo reference" src="https://www.speakeasy.com/assets/badges/arazzo-reference.svg" /></a>
     <!-- Built By Speakeasy Badge -->
     <a href="https://speakeasy.com/"><img alt="Built by Speakeasy" src="https://www.speakeasy.com/assets/badges/built-by-speakeasy.svg" /></a>
    <a href="https://github.com/speakeasy-api/openapi/releases/latest"><img alt="Release" src="https://img.shields.io/github/release/speakeasy-api/openapi.svg?style=for-the-badge"></a>
    <a href="https://pkg.go.dev/github.com/speakeasy-api/openapi/arazzo?tab=doc"><img alt="Go Doc" src="https://img.shields.io/badge/godoc-reference-blue.svg?style=for-the-badge"></a>
   <br />
    <a href="https://github.com/speakeasy-api/openapi/actions/workflows/test.yaml"><img alt="GitHub Action: Test" src="https://img.shields.io/github/actions/workflow/status/speakeasy-api/openapi/test.yaml?style=for-the-badge"></a>
    <a href="https://goreportcard.com/report/github.com/speakeasy-api/openapi"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/speakeasy-api/openapi?style=for-the-badge"></a>
    <a href="/LICENSE"><img alt="Software License" src="https://img.shields.io/badge/license-MIT-blue.svg?style=for-the-badge"></a>
  </p>
</p>

## Supported versions

| Arazzo version | Support |
| -------------- | ------- |
| [1.1.x](https://spec.openapis.org/arazzo/v1.1.0) | Read, write, walk and validate, including AsyncAPI steps (`channelPath`, `action`, `correlationId`), `dependsOn`, `timeout`, Selector Objects, action `parameters`, `$self` and the stricter 1.1 runtime expression grammar. AsyncAPI documents are not loaded, so references into them are only checked structurally. |
| [1.0.x](https://spec.openapis.org/arazzo/v1.0.1) | Read, write, walk and validate. |

Any patch release of a supported minor version is accepted (including prerelease suffixes such as `1.1.0-rc1`). Validation follows the version declared by the document, so using an Arazzo 1.1 feature in a document that declares `arazzo: 1.0.x` is reported as an error. Use `arazzo.Upgrade` (or `openapi arazzo upgrade`) to move a 1.0.x document to 1.1.0. The upgrade pins `jsonpath` and `xpath` criteria to the versions Arazzo 1.0 implied, so their meaning doesn't change.

Where the Arazzo Specification text and its JSON Schema disagree, validation follows the specification text.

### Migrating step and workflow outputs

Arazzo 1.1 allows an output to be either a runtime expression or a Selector Object, so `arazzo.Outputs` is now a map of `*arazzo.OutputValue` instead of `expression.Expression`:

```go
// Before
step.Outputs = sequencedmap.New(sequencedmap.NewElem[string, expression.Expression]("petId", "$response.body#/id"))

// After
step.Outputs = sequencedmap.New(sequencedmap.NewElem("petId", arazzo.NewOutputValueFromExpression("$response.body#/id")))

for name, output := range step.Outputs.All() {
	if output.IsExpression() {
		fmt.Println(name, *output.GetExpression())
	}
	if output.IsSelector() {
		fmt.Println(name, output.GetSelector().Selector)
	}
}
```

<!-- START USAGE EXAMPLES -->

## Read and parse an Arazzo document from a file

Shows loading a document, handling validation errors, and making simple modifications.

```go
ctx := context.Background()

r, err := os.Open("testdata/simple.arazzo.yaml")
if err != nil {
	panic(err)
}
defer r.Close()

a, validationErrs, err := arazzo.Unmarshal(ctx, r)
if err != nil {
	panic(err)
}

for _, err := range validationErrs {
	fmt.Println(err.Error())
}

a.Info.Title = "Updated Simple Workflow"

buf := bytes.NewBuffer([]byte{})

if err := arazzo.Marshal(ctx, a, buf); err != nil {
	panic(err)
}

fmt.Println(buf.String())
```

## Create an Arazzo document from scratch

Shows building a basic workflow document with info and version programmatically.

```go
ctx := context.Background()

a := &arazzo.Arazzo{
	Arazzo: arazzo.Version,
	Info: arazzo.Info{
		Title:   "My Workflow",
		Summary: pointer.From("A summary"),
		Version: "1.0.1",
	},
}

buf := bytes.NewBuffer([]byte{})

err := arazzo.Marshal(ctx, a, buf)
if err != nil {
	panic(err)
}

fmt.Printf("%s", buf.String())
```

## Modify an existing Arazzo document

Shows loading a document, changing properties, and marshaling it back to YAML.

```go
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
```

## Traverse an Arazzo document using the iterator API

Shows how to match different types of objects like workflows during traversal.

```go
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
```

## Upgrade an Arazzo 1.0.x document to the latest supported version

Shows loading a document, upgrading it and validating the result.

```go
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
```

## Validate an Arazzo document

Shows loading an invalid document and handling validation errors.

```go
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
```

<!-- END USAGE EXAMPLES -->

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
