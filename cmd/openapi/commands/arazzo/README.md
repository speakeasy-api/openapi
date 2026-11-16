# Arazzo Commands

Commands for working with Arazzo workflow documents.

Arazzo workflows describe sequences of API calls and their dependencies. These commands help you validate and work with Arazzo documents according to the [Arazzo Specification](https://spec.openapis.org/arazzo/latest.html). Arazzo 1.0.x and 1.1.x documents are supported.

## Table of Contents

- [Table of Contents](#table-of-contents)
- [Available Commands](#available-commands)
  - [`validate`](#validate)
  - [`upgrade`](#upgrade)
- [What is Arazzo?](#what-is-arazzo)
  - [Example Arazzo Document](#example-arazzo-document)
- [Common Use Cases](#common-use-cases)
- [Common Options](#common-options)
- [Output Formats](#output-formats)

## Available Commands

### `validate`

Validate an Arazzo workflow document for compliance with the Arazzo Specification.

```bash
# Validate an Arazzo workflow file
openapi arazzo validate ./workflow.arazzo.yaml

# Validate with verbose output
openapi arazzo validate -v ./workflow.arazzo.yaml

# Only validate the document itself, without loading its source descriptions
openapi arazzo validate --skip-sources ./workflow.arazzo.yaml
```

This command checks for:

- Structural validity according to the Arazzo Specification
- Workflow step definitions and dependencies
- Parameter and expression syntax
- `$steps`, `$workflows` and `$inputs` expressions that reference steps, outputs and inputs that exist
- Parameters passed to workflows in the same document that match the workflow's inputs
- Source description references and validity
- Step success and failure action definitions

By default the OpenAPI and Arazzo documents referenced by `sourceDescriptions` are also loaded, and the workflows are checked against them:

- `operationId`, `operationPath` and `workflowId` targets (including `dependsOn` entries) exist
- Step parameters match the target operation's parameters by name and location
- Required parameters and request bodies are supplied
- Parameters passed to workflows in other Arazzo documents match their inputs

Relative source URLs are resolved against the document's `$self`, or its file location. A source that can't be loaded is reported as a warning, and AsyncAPI sources are reported with a hint and not checked. Only errors make validation fail.

| Flag             | Description                                                                                            |
| ---------------- | ------------------------------------------------------------------------------------------------------ |
| `--skip-sources` | Don't load source descriptions to validate the document against them, for example when working offline |

Documents are validated against the version they declare. Using an Arazzo 1.1 feature (such as `dependsOn` on a step or an AsyncAPI source description) in a document that declares `arazzo: 1.0.x` is reported as an error, use [`upgrade`](#upgrade) to move the document to 1.1.

### `upgrade`

Upgrade an Arazzo workflow document to the latest supported version (1.1.0).

```bash
# Upgrade to the latest version and print to stdout
openapi arazzo upgrade ./workflow.arazzo.yaml

# Upgrade and write to a new file
openapi arazzo upgrade ./workflow.arazzo.yaml ./workflow-1.1.arazzo.yaml

# Upgrade in-place
openapi arazzo upgrade -w ./workflow.arazzo.yaml

# Upgrade to a specific version
openapi arazzo upgrade --version 1.0.1 ./workflow.arazzo.yaml

# Only upgrade across minor versions (skip patch upgrades such as 1.1.0 → 1.1.1)
openapi arazzo upgrade --minor-only ./workflow.arazzo.yaml
```

| Flag            | Description                                                                         |
| --------------- | ----------------------------------------------------------------------------------- |
| `-w, --write`   | Write the result in-place to the input file                                         |
| `-V, --version` | Target Arazzo version to upgrade to (default latest)                                |
| `--minor-only`  | Only upgrade across minor versions, skip patch-level upgrades within the same minor |

The upgrade:

- Updates the `arazzo` version field
- Pins `jsonpath` and `xpath` criteria to the versions implied by Arazzo 1.0 (`draft-goessner-dispatch-jsonpath-00` and `xpath-30`), because the defaults changed to RFC 9535 and XPath 3.1 in Arazzo 1.1
- Validates the upgraded document and reports anything that needs manual changes, such as runtime expressions that don't match the stricter Arazzo 1.1 grammar (for example `$response.body.id` instead of `$response.body#/id`)

Before:

```yaml
arazzo: 1.0.1
# ...
        successCriteria:
          - context: $response.body
            condition: $[?@.status == 'available']
            type: jsonpath
```

After:

```yaml
arazzo: 1.1.0
# ...
        successCriteria:
          - context: $response.body
            condition: $[?@.status == 'available']
            type:
              type: jsonpath
              version: draft-goessner-dispatch-jsonpath-00
```

## What is Arazzo?

Arazzo is a specification for describing API workflows - sequences of API calls where the output of one call can be used as input to subsequent calls. It's designed to work alongside OpenAPI and AsyncAPI specifications to describe complex API interactions.

### Example Arazzo Document

```yaml
arazzo: 1.1.0
info:
  title: User Management Workflow
  version: 1.0.0

sourceDescriptions:
  - name: userAPI
    url: ./user-api.yaml
    type: openapi

workflows:
  - workflowId: createUserWorkflow
    summary: Create a new user and fetch their profile
    inputs:
      type: object
      properties:
        userName:
          type: string
        userEmail:
          type: string
    steps:
      - stepId: createUser
        description: Create a new user
        operationId: createUser
        requestBody:
          contentType: application/json
          payload:
            name: "{$inputs.userName}"
            email: "{$inputs.userEmail}"
        successCriteria:
          - condition: $statusCode == 201
        outputs:
          userId: $response.body#/id

      - stepId: getUserProfile
        description: Fetch the created user's profile
        operationId: getUser
        dependsOn:
          - createUser
        parameters:
          - name: userId
            in: path
            value: $steps.createUser.outputs.userId
        successCriteria:
          - condition: $statusCode == 200
        outputs:
          email:
            context: $response.body
            selector: $.email
            type: jsonpath
```

## Common Use Cases

**API Testing Workflows**: Test user registration and login flows
**Data Processing Pipelines**: Process data through multiple API endpoints  
**Integration Testing**: Test integration between multiple services
**Automated Workflows**: Chain API calls for business process automation

## Common Options

All commands support these common options:

- `-h, --help`: Show help for the command
- `-v, --verbose`: Enable verbose output (global flag)

## Output Formats

All commands work with both YAML and JSON input files. Validation results provide clear, structured feedback with specific error locations and descriptions.
