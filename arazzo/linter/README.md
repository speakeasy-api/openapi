# Arazzo Linter

The Arazzo linter checks Arazzo workflow documents for semantic issues, style and security concerns beyond what
`arazzo.Validate` reports, such as steps that can never run, dependency cycles and unused components.

It uses the same engine, configuration format and output formats as the [OpenAPI linter](../../openapi/linter/README.md).

## Quick Start

### CLI

```bash
# Lint an Arazzo document, checking it against the documents its sourceDescriptions reference
openapi arazzo lint workflow.arazzo.yaml

# Only lint the document itself, without loading its sources
openapi arazzo lint --skip-sources workflow.arazzo.yaml

# Output as JSON
openapi arazzo lint --format json workflow.arazzo.yaml

# Use the recommended ruleset, and disable specific rules
openapi arazzo lint --ruleset recommended --disable style-step-success-criteria workflow.arazzo.yaml

# Apply the fixes that don't need input, or preview them first
openapi arazzo lint --fix workflow.arazzo.yaml
openapi arazzo lint --fix --dry-run workflow.arazzo.yaml

# Also be prompted for the fixes that need input
openapi arazzo lint --fix-interactive workflow.arazzo.yaml

# List the available rules
openapi arazzo list-rules
```

### Go API

```go
import (
    "context"
    "fmt"
    "os"

    "github.com/speakeasy-api/openapi/arazzo"
    arazzoLinter "github.com/speakeasy-api/openapi/arazzo/linter"
    "github.com/speakeasy-api/openapi/linter"
)

func main() {
    ctx := context.Background()

    f, _ := os.Open("workflow.arazzo.yaml")
    doc, validationErrors, _ := arazzo.Unmarshal(ctx, f)

    // Optionally load the sourceDescriptions, so findings can be checked against them and fixes can use them
    sources, loadErrors := arazzo.LoadSourceDescriptions(ctx, doc, arazzo.LoadSourcesOptions{DocumentLocation: "workflow.arazzo.yaml"})
    validationErrors = append(validationErrors, loadErrors...)
    validationErrors = append(validationErrors, doc.ValidateSources(ctx, sources)...)
    ctx = arazzoLinter.WithSourceDocuments(ctx, sources)

    lint := arazzoLinter.NewLinter(linter.NewConfig())
    output, _ := lint.Lint(ctx, linter.NewDocumentInfo(doc, "workflow.arazzo.yaml"), validationErrors, nil)

    fmt.Println(output.FormatText())
}
```

Fixes can be applied with the shared fix engine's `ProcessDocumentErrors`, then the document written back with
`arazzo.Marshal`.

## Available Rules

<!-- START LINT RULES -->

| Rule | Severity | Description |
|------|----------|-------------|
| <a name="security-source-description-https"></a>`security-source-description-https` | warning | Source descriptions should be loaded over HTTPS. A document fetched over plain HTTP can be modified in transit, which can redirect a workflow's steps to different operations or servers. Local addresses (localhost and loopback IPs) are not reported. |
| <a name="semantic-dependency-cycle"></a>`semantic-dependency-cycle` | error | Workflows and steps must not depend on each other in a cycle. A workflow or step only runs once everything in its dependsOn has completed, so none of the workflows or steps in a cycle can ever run. |
| <a name="semantic-unreachable-step"></a>`semantic-unreachable-step` | warning | Every step should be reachable. Steps run in order unless a success or failure action ends the workflow or goes to another step, so a step is unreachable when every path to it is cut off by an unconditional `end` or `goto` and no action goes to it. Unreachable steps are usually left over from a refactor, or a sign that an action's criteria are missing. |
| <a name="semantic-unused-component"></a>`semantic-unused-component` | warning | Components that are never referenced should be removed. Inputs are referenced with `$ref: '#/components/inputs/<name>'`, and parameters and actions with `reference: $components.<type>.<name>`. A component only referenced by itself is unused. |
| <a name="semantic-unused-step-output"></a>`semantic-unused-step-output` | hint | Step outputs should be used. A step's outputs can only be read within its workflow, by later steps or by the workflow's outputs, so an output that is never read has no effect. |
| <a name="semantic-unused-workflow-input"></a>`semantic-unused-workflow-input` | warning | Workflow inputs should be used. An input is only available to a workflow through `$inputs` expressions, so an input that is never read either isn't needed, or a step is missing the parameter or payload value that should use it. |
| <a name="style-info-description"></a>`style-info-description` | warning | The info section should include a description that explains what the workflows in the document do and when to use them, so readers don't have to infer it from the individual steps. |
| <a name="style-retry-after"></a>`style-retry-after` | hint | Retry failure actions should set retryAfter. Without it the step is retried immediately, which rarely helps with the transient failures retries are meant for (rate limits, eventual consistency) and adds load to a struggling API. |
| <a name="style-source-description-type"></a>`style-source-description-type` | warning | Source descriptions should declare their type. Without it, tools have to load the document to know whether it is an OpenAPI, AsyncAPI or Arazzo document, and some checks (such as which steps may use channelPath or action) can't be made from the Arazzo document alone. |
| <a name="style-step-success-criteria"></a>`style-step-success-criteria` | warning | Steps that call an operation or channel should declare successCriteria. Without them, any response is treated as success, so a workflow can carry on after an error response and fail later in a way that is harder to diagnose. Steps that run another workflow are not checked, as their success is decided by that workflow. |
| <a name="style-workflow-description"></a>`style-workflow-description` | hint | Workflows should have a summary or description that explains the outcome they achieve. Workflow IDs are often terse, and a short explanation helps people and tools pick the right workflow. |

<!-- END LINT RULES -->

## Rulesets

| Ruleset       | Rules                                                                                                                                                                      |
| ------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `all`         | Every rule (the default)                                                                                                                                                   |
| `recommended` | Every rule except the hint level rules (`semantic-unused-step-output`, `style-workflow-description` and `style-retry-after`), which are useful while authoring but noisier |

## Configuration

The CLI looks for a configuration file at `~/.openapi/arazzo-lint.yaml`, separate from the OpenAPI linter's
`~/.openapi/lint.yaml` as the rules and rulesets differ. Use `--config` to use a different file.

```yaml
extends: recommended

rules:
  # Change severity
  - id: style-step-success-criteria
    severity: error

  # Disable a rule
  - id: semantic-unused-step-output
    disabled: true

  # Filter by message pattern
  - id: security-source-description-https
    match: ".*internal.example.com.*"
    disabled: true

categories:
  # Disable an entire category
  style:
    enabled: false
```

Custom rules are not supported for Arazzo documents yet, so `custom_rules` is ignored.
