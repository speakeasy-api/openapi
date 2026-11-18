# AsyncAPI Commands

Commands for working with AsyncAPI documents.

AsyncAPI documents describe message-driven APIs: the channels an application uses, the operations it performs on them and the messages it sends and receives. These commands work with documents written to the [AsyncAPI Specification](https://www.asyncapi.com/docs/reference/specification/latest) 3.0.x and 3.1.x, using the [asyncapi](../../../../asyncapi) package.

## Table of Contents

- [Table of Contents](#table-of-contents)
- [Available Commands](#available-commands)
  - [`validate`](#validate)
- [What is and isn't checked](#what-is-and-isnt-checked)
- [Exit status](#exit-status)

## Available Commands

### `validate`

Validate an AsyncAPI document for compliance with the AsyncAPI Specification.

```bash
# Validate a document without reading anything outside it
openapi asyncapi validate ./asyncapi.yaml

# Also load and validate the files and URLs the document references
openapi asyncapi validate --resolve ./asyncapi.yaml

# Fail on warnings as well as errors, for example in CI
openapi asyncapi validate --resolve --strict ./asyncapi.yaml

# Read from stdin, resolving relative references against where the document lives
cat ./specs/asyncapi.yaml | openapi asyncapi validate --resolve --base-uri ./specs/asyncapi.yaml
```

| Flag         | Description                                                                                                   |
| ------------ | ------------------------------------------------------------------------------------------------------------- |
| `--resolve`  | Load references to other files and URLs and validate them. Without it nothing outside the document is read    |
| `--base-uri` | File path or URL to resolve relative references against, instead of the file's location. Requires `--resolve` |
| `--strict`   | Fail validation on warnings as well as errors                                                                 |

Example output for a document that references another file, validated without `--resolve`:

```text
Validating AsyncAPI document: asyncapi.yaml
✅ AsyncAPI document is valid - 0 errors, 1 warnings:

1. [10:15] warning validation-invalid-reference reference `./messages.yaml#/components/messages/OrderPlaced` is to another document and external references are disabled

Not checked:
- 1 references to other documents were not followed, use --resolve to validate them
```

## What is and isn't checked

The command checks:

- Structural validity, required fields, allowed values and data types
- Schemas written in the AsyncAPI Schema format or JSON Schema Draft 07
- References within the document, which must exist and point to the right kind of object (with `--resolve`, references to other files and URLs too)
- Operation and message traits, applied in order, including problems that only appear once they're applied
- Rules that span objects, such as root operations using root channels and an operation's messages belonging to its channel

Some content is preserved but can't be checked. The output lists it under **Not checked** so a passing result isn't mistaken for a complete one:

- **References to other documents** aren't followed unless `--resolve` is set. They're reported as warnings, since they aren't known to be broken.
- **Schemas in other formats** (such as Avro or Protocol Buffers) and **protocol bindings** are kept as written but not validated.

Documents declaring a version other than 3.0.x or 3.1.x, including AsyncAPI 2.x, are reported as unsupported and the rest of the document isn't checked.

When reading from stdin with `--resolve`, the document has no location, so relative references to other documents can't be resolved unless `--base-uri` is set. Absolute file paths and URLs still resolve.

`--resolve` makes requests to the URLs the document references, with a 30 second timeout. Only use it on documents you trust.

## Exit status

| Result                                     | Exit status |
| ------------------------------------------ | ----------- |
| No errors (warnings and hints are allowed) | `0`         |
| Warnings, with `--strict`                  | `1`         |
| Errors, or an unsupported version          | `1`         |
| The document can't be read or parsed       | `1`         |

Hints, such as bindings that aren't validated, never fail validation.
