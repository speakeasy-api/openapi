# Arazzo Commands

Commands for working with Arazzo workflow documents.

Arazzo workflows describe sequences of API calls and their dependencies. These commands help you validate and work with Arazzo documents according to the [Arazzo Specification](https://spec.openapis.org/arazzo/v1.0.1).

## Table of Contents

- [Table of Contents](#table-of-contents)
- [Available Commands](#available-commands)
  - [`validate`](#validate)
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
```

This command checks for:

- Structural validity according to the Arazzo Specification
- Workflow step definitions and dependencies
- Parameter and expression syntax
- Source description references and validity
- Step success and failure action definitions

## What is Arazzo?

Arazzo is a specification for describing API workflows - sequences of API calls where the output of one call can be used as input to subsequent calls. It's designed to work alongside OpenAPI specifications to describe complex API interactions.

### Example Arazzo Document

```yaml
arazzo: 1.0.0
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
    steps:
      - stepId: createUser
        description: Create a new user
        operationId: userAPI.createUser
        requestBody:
          contentType: application/json
          payload:
            name: $inputs.userName
            email: $inputs.userEmail
        successCriteria:
          - condition: $statusCode == 201
        outputs:
          userId: $response.body.id
          
      - stepId: getUserProfile
        description: Fetch the created user's profile
        operationId: userAPI.getUser
        dependsOn: createUser
        parameters:
          - name: userId
            in: path
            value: $steps.createUser.outputs.userId
        successCriteria:
          - condition: $statusCode == 200
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