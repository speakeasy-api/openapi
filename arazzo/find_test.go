package arazzo_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/stretchr/testify/assert"
)

func TestWorkflows_Find_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		workflows arazzo.Workflows
		id        string
		expected  *arazzo.Workflow
	}{
		{
			name:      "empty workflows returns nil",
			workflows: arazzo.Workflows{},
			id:        "test",
			expected:  nil,
		},
		{
			name: "finds workflow by id",
			workflows: arazzo.Workflows{
				{WorkflowID: "workflow1"},
				{WorkflowID: "workflow2"},
				{WorkflowID: "workflow3"},
			},
			id:       "workflow2",
			expected: &arazzo.Workflow{WorkflowID: "workflow2"},
		},
		{
			name: "returns nil when workflow not found",
			workflows: arazzo.Workflows{
				{WorkflowID: "workflow1"},
			},
			id:       "nonexistent",
			expected: nil,
		},
		{
			name: "returns first match when multiple workflows with same id",
			workflows: arazzo.Workflows{
				{WorkflowID: "duplicate", Summary: pointer.From("first")},
				{WorkflowID: "duplicate", Summary: pointer.From("second")},
			},
			id:       "duplicate",
			expected: &arazzo.Workflow{WorkflowID: "duplicate", Summary: pointer.From("first")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.workflows.Find(tt.id)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected.WorkflowID, result.WorkflowID)
				if tt.expected.Summary != nil {
					assert.Equal(t, *tt.expected.Summary, *result.Summary)
				}
			}
		})
	}
}

func TestSteps_Find_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		steps    arazzo.Steps
		id       string
		expected *arazzo.Step
	}{
		{
			name:     "empty steps returns nil",
			steps:    arazzo.Steps{},
			id:       "test",
			expected: nil,
		},
		{
			name: "finds step by id",
			steps: arazzo.Steps{
				{StepID: "step1"},
				{StepID: "step2"},
				{StepID: "step3"},
			},
			id:       "step2",
			expected: &arazzo.Step{StepID: "step2"},
		},
		{
			name: "returns nil when step not found",
			steps: arazzo.Steps{
				{StepID: "step1"},
			},
			id:       "nonexistent",
			expected: nil,
		},
		{
			name: "returns first match when multiple steps with same id",
			steps: arazzo.Steps{
				{StepID: "duplicate", Description: pointer.From("first")},
				{StepID: "duplicate", Description: pointer.From("second")},
			},
			id:       "duplicate",
			expected: &arazzo.Step{StepID: "duplicate", Description: pointer.From("first")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.steps.Find(tt.id)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected.StepID, result.StepID)
				if tt.expected.Description != nil {
					assert.Equal(t, *tt.expected.Description, *result.Description)
				}
			}
		})
	}
}

func TestSourceDescriptions_Find_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		sourceDescriptions arazzo.SourceDescriptions
		findName           string
		expected           *arazzo.SourceDescription
	}{
		{
			name:               "empty source descriptions returns nil",
			sourceDescriptions: arazzo.SourceDescriptions{},
			findName:           "test",
			expected:           nil,
		},
		{
			name: "finds source description by name",
			sourceDescriptions: arazzo.SourceDescriptions{
				{Name: "apiOne", URL: "https://api1.example.com"},
				{Name: "apiTwo", URL: "https://api2.example.com"},
				{Name: "apiThree", URL: "https://api3.example.com"},
			},
			findName: "apiTwo",
			expected: &arazzo.SourceDescription{Name: "apiTwo", URL: "https://api2.example.com"},
		},
		{
			name: "returns nil when source description not found",
			sourceDescriptions: arazzo.SourceDescriptions{
				{Name: "apiOne", URL: "https://api1.example.com"},
			},
			findName: "nonexistent",
			expected: nil,
		},
		{
			name: "returns first match when multiple source descriptions with same name",
			sourceDescriptions: arazzo.SourceDescriptions{
				{Name: "duplicate", URL: "https://first.example.com"},
				{Name: "duplicate", URL: "https://second.example.com"},
			},
			findName: "duplicate",
			expected: &arazzo.SourceDescription{Name: "duplicate", URL: "https://first.example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.sourceDescriptions.Find(tt.findName)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected.Name, result.Name)
				assert.Equal(t, tt.expected.URL, result.URL)
			}
		})
	}
}
