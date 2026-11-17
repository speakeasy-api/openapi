<h1 align="center"><b>AsyncAPI Parser</b></h1>
<p align="center">An API for working with <a href="https://www.asyncapi.com/docs/reference/specification/v3.1.0">AsyncAPI 3.x documents</a> including: read, walk, create, mutate, resolve and validate.</p>

> [!NOTE]
> This package is under active development. Its API may still change before it's listed as a supported specification in the repository README.

## Supported versions

| AsyncAPI version                                                      | Support                                                        |
| --------------------------------------------------------------------- | -------------------------------------------------------------- |
| [3.1.x](https://www.asyncapi.com/docs/reference/specification/v3.1.0) | Read, write, walk, resolve and validate. Adds `ros2` bindings. |
| [3.0.x](https://www.asyncapi.com/docs/reference/specification/v3.0.0) | Read, write, walk, resolve and validate.                       |

Any patch release of a supported minor version is accepted. AsyncAPI 2.x documents are reported as unsupported. Where the specification text and its JSON Schema disagree, validation follows the specification text.

## Usage

### Read, edit, validate and write a document

`Unmarshal` returns a fatal error only when the document can't be read or parsed. Problems with its content are returned as validation errors, with error, warning or hint severity.

```go
doc, validationErrs, err := asyncapi.Unmarshal(ctx, f)
if err != nil {
	return err
}
for _, vErr := range validationErrs {
	fmt.Println(vErr)
}

doc.Info.Title = "Order events"

if err := asyncapi.Marshal(ctx, doc, os.Stdout); err != nil {
	return err
}
```

`Unmarshal`, `Validate` and `Walk` never read files or make network requests. They check each object locally and leave references unresolved.

### Resolve references

`ResolveAllReferences` is the explicit step that loads referenced resources. It follows chains of references, applies traits and runs the checks that need referenced objects, such as whether an operation's messages belong to its channel.

```go
validationErrs, err := doc.ResolveAllReferences(ctx, asyncapi.ResolveAllOptions{
	AsyncAPILocation: "/specs/orders.yaml", // relative references resolve against this
	VirtualFS:        fsys,                 // optional, defaults to the local file system
	HTTPClient:       client,               // optional, defaults to http.DefaultClient
})

operation := doc.Operations.GetOrZero("placeOrder").GetObject()
channel := operation.Channel.GetChannel()
for _, message := range operation.Messages {
	fmt.Println(message.GetMessage().GetName())
}
```

- Reference problems (missing or wrong-kind targets, cycles, disallowed external references) are returned as validation errors at the referring `$ref`. Diagnostics found in other documents set `DocumentLocation`.
- Set `DisableExternalRefs` to keep resolution within the document. Without `AsyncAPILocation`, relative references to other documents are reported rather than resolved against the working directory.
- The default HTTP client isn't a sandbox. If a document might come from someone you don't trust, provide a client and file system that only allow the locations you expect.
- Each call rebuilds resolution from the current document, so edits are picked up. Resolution never changes the document, and references stay as written.

### Effective operations and messages

After resolution, `GetEffective` returns an operation or message with its traits applied, as a separate snapshot:

```go
effective := operation.GetEffective()
fmt.Println(effective.GetSummary())

index, ok := operation.GetTraitSource("summary") // index into Traits, or -1 if the operation sets it
```

Traits are applied with JSON Merge Patch in declaration order, and the object's own values win:

```text
effective = {}
for each trait: effective = MergePatch(effective, trait)
effective = MergePatch(effective, object without traits)
```

So objects merge recursively, later traits win over earlier ones, arrays are replaced and `null` deletes a value, including the object's own `null` deleting an inherited value. Fields a trait must not set (`action`, `channel`, `messages` and `traits` for operations, `payload` and `traits` for messages) are reported by `Validate` and ignored when merging. Diagnostics that only appear once traits are applied are reported by `ResolveAllReferences`.

### Walk a document

```go
for item := range asyncapi.Walk(ctx, doc) {
	err := item.Match(asyncapi.Matcher{
		Operation: func(o *asyncapi.Operation) error {
			fmt.Println(item.Location.ToJSONPointer(), o.GetAction())
			return nil
		},
	})
	if err != nil {
		return err
	}
}
```

Walking visits what was written, in document order. It doesn't follow references, even after they're resolved.

## Support limits

| Area                 | Support                                                                                                                                                                                                                                 |
| -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Document objects     | All AsyncAPI 3.x objects are modelled, with getters, editing and round-tripping. Unknown properties are kept and reported as warnings.                                                                                                  |
| References           | All Reference Objects are resolved, within the document and across files and URLs, including chains and relative references between documents.                                                                                          |
| Schemas              | Native AsyncAPI and JSON Schema Draft 07 schemas are parsed and validated. A schema `$ref` and a Multi Format Schema Object whose `schema` is a `$ref` are resolved. `$ref`s nested inside schemas aren't resolved yet.                 |
| Other schema formats | Avro, RAML, OpenAPI and other formats are kept unchanged and reported as not validated. A `$ref` to a JSON or YAML resource in those formats is loaded and kept unchanged. References to Protocol Buffers text resources aren't loaded. |
| Bindings             | Kept unchanged for every protocol. Only their shape and `bindingVersion` are checked; protocol-specific fields aren't validated.                                                                                                        |
| Effective views      | References in an effective view are resolved when all of its traits come from the same document as the object.                                                                                                                          |
| Message instances    | Examples and runtime messages aren't validated against their schemas.                                                                                                                                                                   |
