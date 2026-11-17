package rules_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/arazzo/linter/rules"
	"github.com/stretchr/testify/assert"
)

// workflowsDoc builds a document with the given workflows.
func workflowsDoc(workflows string) string {
	return `arazzo: 1.1.0
info:
  title: Workflows
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
  - name: shared
    url: https://example.com/shared.arazzo.yaml
    type: arazzo
workflows:
` + workflows
}

func TestDependencyCycleRule_Success(t *testing.T) {
	t.Parallel()

	doc := workflowsDoc(`  - workflowId: a
    dependsOn:
      - b
      - $sourceDescriptions.shared.c
      - missing
    steps:
      - stepId: one
        operationId: x
      - stepId: two
        operationId: x
        dependsOn:
          - one
          - $workflows.b.steps.one
          - $sourceDescriptions.shared.c.steps.one
          - $workflows.missing.steps.one
          - missing
          - two
      - null
  - workflowId: b
    steps:
      - stepId: one
        operationId: x
      - null
`)

	assert.Empty(t, runRule(t, &rules.DependencyCycleRule{}, doc), "acyclic dependencies and empty steps should not be reported")
}

func TestDependencyCycleRule_WorkflowAndStepCycle_Error(t *testing.T) {
	t.Parallel()

	// b's steps only run after a has completed, but a's step waits for one of b's steps.
	doc := workflowsDoc(`  - workflowId: a
    steps:
      - stepId: one
        operationId: x
        dependsOn:
          - $workflows.b.steps.one
  - workflowId: b
    dependsOn:
      - a
    steps:
      - stepId: one
        operationId: x
  - workflowId: c
    dependsOn:
      - b
    steps:
      - stepId: one
        operationId: x
        dependsOn:
          - $workflows.b.steps.one
`)

	expected := []string{"error: step.dependsOn creates a cycle: a.steps.one -> b.steps.one -> a.steps.one"}
	assert.Equal(t, expected, runRule(t, &rules.DependencyCycleRule{}, doc), "cycles across workflow and step dependencies should be reported")
}

func TestDependencyCycleRule_Error(t *testing.T) {
	t.Parallel()

	doc := workflowsDoc(`  - workflowId: a
    dependsOn:
      - b
    steps:
      - stepId: one
        operationId: x
        dependsOn:
          - $workflows.b.steps.one
  - workflowId: b
    dependsOn:
      - c
    steps:
      - stepId: one
        operationId: x
        dependsOn:
          - two
      - stepId: two
        operationId: x
        dependsOn:
          - $workflows.a.steps.one
  - workflowId: c
    dependsOn:
      - a
      - c
    steps:
      - stepId: one
        operationId: x
`)

	expected := []string{
		"error: workflow.dependsOn creates a cycle: a -> b -> c -> a",
		"error: workflow.dependsOn creates a cycle: c -> c",
		"error: step.dependsOn creates a cycle: a.steps.one -> b.steps.one -> b.steps.two -> a.steps.one",
	}
	assert.Equal(t, expected, runRule(t, &rules.DependencyCycleRule{}, doc), "cycles should be reported once each")
}
