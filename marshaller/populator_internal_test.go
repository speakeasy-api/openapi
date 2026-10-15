package marshaller

import (
	"reflect"
	"testing"

	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type embeddedSeqMapHolder struct {
	sequencedmap.Map[string, string]
}

type embeddedSeqMapPtrHolder struct {
	*sequencedmap.Map[string, string]
}

type nonSeqMapEmbed struct {
	Name string
}

type nonSeqMapHolder struct {
	nonSeqMapEmbed
}

func TestGetSequencedMapInterface_ValueEmbed_Success(t *testing.T) {
	t.Parallel()

	holder := &embeddedSeqMapHolder{}

	seqMap := getSequencedMapInterface(reflect.ValueOf(holder).Elem().Field(0))
	require.NotNil(t, seqMap, "embedded sequenced map should be returned")
	assert.Same(t, &holder.Map, seqMap, "should return the holder's embedded map")
}

func TestGetSequencedMapInterface_PointerEmbed_Success(t *testing.T) {
	t.Parallel()

	holder := &embeddedSeqMapPtrHolder{}

	seqMap := getSequencedMapInterface(reflect.ValueOf(holder).Elem().Field(0))
	require.NotNil(t, holder.Map, "nil pointer embed should be allocated")
	assert.Same(t, holder.Map, seqMap, "should return the newly allocated embedded map")
}

func TestGetSequencedMapInterface_ReturnsNil(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		field reflect.Value
	}{
		{
			name:  "addressable field that is not a sequenced map",
			field: reflect.ValueOf(&nonSeqMapHolder{}).Elem().Field(0),
		},
		{
			name:  "non-addressable value that is not a sequenced map",
			field: reflect.ValueOf(nonSeqMapEmbed{Name: "x"}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Nil(t, getSequencedMapInterface(tt.field), "non sequenced map fields should return nil")
		})
	}
}

func TestGetSourceForPopulation_Success(t *testing.T) {
	t.Parallel()

	seqMap := sequencedmap.New(sequencedmap.NewElem("key", "value"))
	holder := &embeddedSeqMapHolder{Map: *seqMap}

	tests := []struct {
		name     string
		field    reflect.Value
		fieldInt any
		expected any
	}{
		{
			name:     "addressable field returns its address",
			field:    reflect.ValueOf(holder).Elem().Field(0),
			fieldInt: "unused",
			expected: &holder.Map,
		},
		{
			name:     "pointer field returns the pointer",
			field:    reflect.ValueOf(seqMap),
			fieldInt: "unused",
			expected: seqMap,
		},
		{
			name:     "non-addressable sequenced map value returns an addressable copy",
			field:    reflect.ValueOf(*seqMap),
			fieldInt: "unused",
			expected: seqMap,
		},
		{
			name:     "non-addressable other value returns the field interface",
			field:    reflect.ValueOf(nonSeqMapEmbed{Name: "x"}),
			fieldInt: "fallback",
			expected: "fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := getSourceForPopulation(tt.field, tt.fieldInt)
			assert.Equal(t, tt.expected, actual, "should return the expected population source")
		})
	}
}

func TestGetSourceForPopulation_CopiesNonAddressableValue(t *testing.T) {
	t.Parallel()

	seqMap := sequencedmap.New(sequencedmap.NewElem("key", "value"))

	actual := getSourceForPopulation(reflect.ValueOf(*seqMap), nil)
	copied, ok := actual.(*sequencedmap.Map[string, string])
	require.True(t, ok, "should return a pointer to a sequenced map")
	assert.NotSame(t, seqMap, copied, "should return a new addressable copy, not the original")
}
