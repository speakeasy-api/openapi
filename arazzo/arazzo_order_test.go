package arazzo_test

import (
	"bytes"
	"context"
	"os"
	"slices"
	"testing"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/expression"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestArazzo_ArrayOrdering_ReorderWorkflows_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	data, err := os.ReadFile("testdata/ordering/input.arazzo.yaml")
	require.NoError(t, err)

	a, validationErrs, err := arazzo.Unmarshal(ctx, bytes.NewBuffer(data), arazzo.WithSkipValidation())
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Move the first workflow to the end
	first := a.Workflows[0]
	a.Workflows = slices.Delete(a.Workflows, 0, 1)
	a.Workflows = append(a.Workflows, first)

	outBuf := bytes.NewBuffer([]byte{})
	err = arazzo.Marshal(ctx, a, outBuf)
	require.NoError(t, err)

	expected, err := os.ReadFile("testdata/ordering/reorder.expected.arazzo.yaml")
	require.NoError(t, err)

	assert.Equal(t, string(expected), outBuf.String())
}

func TestArazzo_ArrayOrdering_BasicRoundTrip_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	data, err := os.ReadFile("testdata/ordering/input.arazzo.yaml")
	require.NoError(t, err)

	a, validationErrs, err := arazzo.Unmarshal(ctx, bytes.NewBuffer(data), arazzo.WithSkipValidation())
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// DON'T modify anything - just unmarshal and marshal back
	outBuf := bytes.NewBuffer([]byte{})
	err = arazzo.Marshal(ctx, a, outBuf)
	require.NoError(t, err)

	assert.Equal(t, string(data), outBuf.String())
}

func TestArazzo_ArrayOrdering_ReorderWithoutSync_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	data, err := os.ReadFile("testdata/ordering/input.arazzo.yaml")
	require.NoError(t, err)

	a, validationErrs, err := arazzo.Unmarshal(ctx, bytes.NewBuffer(data), arazzo.WithSkipValidation())
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Move the first workflow to the end
	first := a.Workflows[0]
	a.Workflows = slices.Delete(a.Workflows, 0, 1)
	a.Workflows = append(a.Workflows, first)

	// Marshal WITHOUT calling Sync - directly marshal the core
	outBuf := bytes.NewBuffer([]byte{})
	err = a.GetCore().Marshal(ctx, outBuf)
	require.NoError(t, err)

	// When we don't sync, the core should still have the original order
	assert.Equal(t, string(data), outBuf.String())
}

func TestArazzo_ArrayOrdering_AddWorkflow_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	data, err := os.ReadFile("testdata/ordering/input.arazzo.yaml")
	require.NoError(t, err)

	a, validationErrs, err := arazzo.Unmarshal(ctx, bytes.NewBuffer(data), arazzo.WithSkipValidation())
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Add a new workflow at the end
	operationID := expression.Expression("new_operation")
	newWorkflow := &arazzo.Workflow{
		WorkflowID: "new_workflow",
		Steps: []*arazzo.Step{
			{
				StepID:      "new_step",
				OperationID: &operationID,
			},
		},
	}
	a.Workflows = append(a.Workflows, newWorkflow)

	outBuf := bytes.NewBuffer([]byte{})
	err = arazzo.Marshal(ctx, a, outBuf)
	require.NoError(t, err)

	expected, err := os.ReadFile("testdata/ordering/addition.expected.arazzo.yaml")
	require.NoError(t, err)

	assert.Equal(t, string(expected), outBuf.String())
}

func TestArazzo_ArrayOrdering_DeleteWorkflow_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	data, err := os.ReadFile("testdata/ordering/input.arazzo.yaml")
	require.NoError(t, err)

	a, validationErrs, err := arazzo.Unmarshal(ctx, bytes.NewBuffer(data), arazzo.WithSkipValidation())
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Create a new slice with only the workflows we want to keep (remove middle one)
	originalWorkflows := a.Workflows
	a.Workflows = []*arazzo.Workflow{originalWorkflows[0], originalWorkflows[2]}

	outBuf := bytes.NewBuffer([]byte{})
	err = arazzo.Marshal(ctx, a, outBuf)
	require.NoError(t, err)

	expected, err := os.ReadFile("testdata/ordering/deletion.expected.arazzo.yaml")
	require.NoError(t, err)

	assert.Equal(t, string(expected), outBuf.String())
}

func TestArazzo_ArrayOrdering_ComplexOperations_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	data, err := os.ReadFile("testdata/ordering/input.arazzo.yaml")
	require.NoError(t, err)

	a, validationErrs, err := arazzo.Unmarshal(ctx, bytes.NewBuffer(data), arazzo.WithSkipValidation())
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Original order: [workflow-a, workflow-b, workflow-c]

	// Step 1: Delete the middle workflow (index 1)
	originalWorkflows := a.Workflows
	a.Workflows = []*arazzo.Workflow{originalWorkflows[2]}
	// Now: [workflow-c]

	// Step 2: Add two new workflows
	operationID1 := expression.Expression("operation_1")
	operationID2 := expression.Expression("operation_2")
	inQuery := arazzo.InQuery
	contentType := "application/json"

	newWorkflow1 := &arazzo.Workflow{
		WorkflowID: "new_workflow_1",
		Steps: []*arazzo.Step{
			{
				StepID:      "step_1",
				OperationID: &operationID1,
				Parameters: []*arazzo.ReusableParameter{
					{
						Object: &arazzo.Parameter{
							Name:  "param1",
							In:    &inQuery,
							Value: &yaml.Node{Kind: yaml.ScalarNode, Value: "value1"},
						},
					},
				},
			},
		},
	}

	newWorkflow2 := &arazzo.Workflow{
		WorkflowID: "new_workflow_2",
		Steps: []*arazzo.Step{
			{
				StepID:      "step_2",
				OperationID: &operationID2,
				RequestBody: &arazzo.RequestBody{
					ContentType: &contentType,
					Payload: &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Value: "key"},
						{Kind: yaml.ScalarNode, Value: "value"},
					}},
				},
			},
		},
	}

	a.Workflows = append(a.Workflows, newWorkflow1, newWorkflow2)
	// Now: [workflow-c, new_workflow_1, new_workflow_2]

	// Step 3: Reorder - move first to end
	first := a.Workflows[0]
	a.Workflows = slices.Delete(a.Workflows, 0, 1)
	a.Workflows = append(a.Workflows, first)
	// Final: [new_workflow_1, new_workflow_2, workflow-c]

	outBuf := bytes.NewBuffer([]byte{})
	err = arazzo.Marshal(ctx, a, outBuf)
	require.NoError(t, err)

	expected, err := os.ReadFile("testdata/ordering/complex.expected.arazzo.yaml")
	require.NoError(t, err)

	assert.Equal(t, string(expected), outBuf.String())
}

func TestArazzo_MapOrdering_StressModification_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	data, err := os.ReadFile("testdata/ordering/input.arazzo.yaml")
	require.NoError(t, err)

	a, validationErrs, err := arazzo.Unmarshal(ctx, bytes.NewBuffer(data), arazzo.WithSkipValidation())
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Stress test: Multiple operations that should trigger sync

	// 1. Reorder parameters multiple times
	for i := 0; i < 3; i++ {
		paramA, _ := a.Components.Parameters.Get("param-a")
		paramB, _ := a.Components.Parameters.Get("param-b")
		paramC, _ := a.Components.Parameters.Get("param-c")

		a.Components.Parameters.Delete("param-a")
		a.Components.Parameters.Delete("param-b")
		a.Components.Parameters.Delete("param-c")

		// Different order each time
		switch i {
		case 0:
			a.Components.Parameters.Set("param-c", paramC)
			a.Components.Parameters.Set("param-a", paramA)
			a.Components.Parameters.Set("param-b", paramB)
		case 1:
			a.Components.Parameters.Set("param-b", paramB)
			a.Components.Parameters.Set("param-c", paramC)
			a.Components.Parameters.Set("param-a", paramA)
		case 2:
			a.Components.Parameters.Set("param-a", paramA)
			a.Components.Parameters.Set("param-c", paramC)
			a.Components.Parameters.Set("param-b", paramB)
		}
	}

	// 2. Modify parameter fields to force sync
	paramA, _ := a.Components.Parameters.Get("param-a")
	paramA.Name = "modified-param-a"
	paramA.Value = &yaml.Node{Kind: yaml.ScalarNode, Value: "modified-value"}
	a.Components.Parameters.Set("param-a", paramA)

	// 3. Add and remove a temporary parameter
	inHeader := arazzo.InHeader
	tempParam := &arazzo.Parameter{
		Name:  "temp-param",
		In:    &inHeader,
		Value: &yaml.Node{Kind: yaml.ScalarNode, Value: "temp-value"},
	}
	a.Components.Parameters.Set("temp-param", tempParam)
	a.Components.Parameters.Delete("temp-param")

	outBuf := bytes.NewBuffer([]byte{})
	err = arazzo.Marshal(ctx, a, outBuf)
	require.NoError(t, err)

	expected, err := os.ReadFile("testdata/ordering/map-stress.expected.arazzo.yaml")
	require.NoError(t, err)

	assert.Equal(t, string(expected), outBuf.String())
}

func TestArazzo_MapOrdering_AddParameter_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	data, err := os.ReadFile("testdata/ordering/input.arazzo.yaml")
	require.NoError(t, err)

	a, validationErrs, err := arazzo.Unmarshal(ctx, bytes.NewBuffer(data), arazzo.WithSkipValidation())
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Add a new parameter to the components
	inHeader := arazzo.InHeader
	newParam := &arazzo.Parameter{
		Name:  "param-new",
		In:    &inHeader,
		Value: &yaml.Node{Kind: yaml.ScalarNode, Value: "new-value"},
	}
	a.Components.Parameters.Set("param-new", newParam)

	outBuf := bytes.NewBuffer([]byte{})
	err = arazzo.Marshal(ctx, a, outBuf)
	require.NoError(t, err)

	expected, err := os.ReadFile("testdata/ordering/map-addition.expected.arazzo.yaml")
	require.NoError(t, err)

	assert.Equal(t, string(expected), outBuf.String())
}

func TestArazzo_MapOrdering_DeleteParameter_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	data, err := os.ReadFile("testdata/ordering/input.arazzo.yaml")
	require.NoError(t, err)

	a, validationErrs, err := arazzo.Unmarshal(ctx, bytes.NewBuffer(data), arazzo.WithSkipValidation())
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Delete parameters and success/failure actions (delete the middle ones)
	a.Components.Parameters.Delete("param-b")
	a.Components.SuccessActions.Delete("success-b")
	a.Components.FailureActions.Delete("failure-b")

	outBuf := bytes.NewBuffer([]byte{})
	err = arazzo.Marshal(ctx, a, outBuf)
	require.NoError(t, err)

	expected, err := os.ReadFile("testdata/ordering/map-deletion.expected.arazzo.yaml")
	require.NoError(t, err)

	assert.Equal(t, string(expected), outBuf.String())
}

func TestArazzo_MapOrdering_ReorderComponents_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	data, err := os.ReadFile("testdata/ordering/input.arazzo.yaml")
	require.NoError(t, err)

	a, validationErrs, err := arazzo.Unmarshal(ctx, bytes.NewBuffer(data), arazzo.WithSkipValidation())
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Get all parameters
	paramA, hasA := a.Components.Parameters.Get("param-a")
	paramB, hasB := a.Components.Parameters.Get("param-b")
	paramC, hasC := a.Components.Parameters.Get("param-c")

	require.True(t, hasA, "param-a should exist")
	require.True(t, hasB, "param-b should exist")
	require.True(t, hasC, "param-c should exist")

	// Clear and re-add in different order: c, a, b
	a.Components.Parameters.Delete("param-a")
	a.Components.Parameters.Delete("param-b")
	a.Components.Parameters.Delete("param-c")

	a.Components.Parameters.Set("param-c", paramC)
	a.Components.Parameters.Set("param-a", paramA)
	a.Components.Parameters.Set("param-b", paramB)

	outBuf := bytes.NewBuffer([]byte{})
	err = arazzo.Marshal(ctx, a, outBuf)
	require.NoError(t, err)

	expected, err := os.ReadFile("testdata/ordering/map-reorder.expected.arazzo.yaml")
	require.NoError(t, err)

	assert.Equal(t, string(expected), outBuf.String())
}
