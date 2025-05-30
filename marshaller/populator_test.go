package marshaller_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type NestedSourceStruct struct {
	Value string
}

type ModelFromCoreTarget struct {
	Value string
}

func (m *ModelFromCoreTarget) FromCore(c any) error {
	if core, ok := c.(NestedSourceStruct); ok {
		m.Value = core.Value
	}
	return nil
}

func Test_PopulateModel_Success(t *testing.T) {
	// Test ModelFromCore interface
	source := NestedSourceStruct{Value: "from-core"}
	target := &ModelFromCoreTarget{}
	
	err := marshaller.PopulateModel(source, target)
	require.NoError(t, err)
	assert.Equal(t, "from-core", target.Value)
}

func Test_PopulateModel_SimpleStruct_Success(t *testing.T) {
	// Test basic struct population
	source := NestedSourceStruct{Value: "test"}
	target := &NestedSourceStruct{}
	
	err := marshaller.PopulateModel(source, target)
	require.NoError(t, err)
	assert.Equal(t, "test", target.Value)
}