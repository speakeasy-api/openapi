package core

import (
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type TestEitherValue[L any, R any] struct {
	Left  *L
	Right *R
}

func TestEitherValue_SyncChanges_Success(t *testing.T) {
	ctx := context.Background()

	source := TestEitherValue[string, string]{
		Left: pointer.From("some-value"),
	}
	var target EitherValue[string, string]
	outNode, err := marshaller.SyncValue(ctx, source, &target, nil)
	require.NoError(t, err)
	assert.Equal(t, &yaml.Node{
		Value: "some-value",
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
	}, outNode)
	assert.Equal(t, "some-value", *target.Left)
	assert.Nil(t, target.Right)
}
