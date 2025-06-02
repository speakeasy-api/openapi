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

type PopulatorTarget struct {
	Value string
}

func (m *PopulatorTarget) Populate(c any) error {
	if core, ok := c.(NestedSourceStruct); ok {
		m.Value = core.Value
	}
	return nil
}

func Test_PopulateModel_Success(t *testing.T) {
	// Test Populator interface
	source := NestedSourceStruct{Value: "from-core"}
	target := &PopulatorTarget{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)
	assert.Equal(t, "from-core", target.Value)
}

func Test_PopulateModel_SimpleStruct_Success(t *testing.T) {
	// Test basic struct population
	source := NestedSourceStruct{Value: "test"}
	target := &NestedSourceStruct{}

	err := marshaller.Populate(source, target)
	require.NoError(t, err)
	assert.Equal(t, "test", target.Value)
}
