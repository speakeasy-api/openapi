package openapi

import (
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/stretchr/testify/assert"
)

func TestMediaType_GetItemSchema_Nil_Success(t *testing.T) {
	t.Parallel()

	var m *MediaType
	result := m.GetItemSchema()

	assert.Nil(t, result, "GetItemSchema on nil should return nil")
}

func TestMediaType_GetItemSchema_Set_Success(t *testing.T) {
	t.Parallel()

	schema := &oas3.JSONSchema[oas3.Referenceable]{}
	m := &MediaType{
		ItemSchema: schema,
	}
	result := m.GetItemSchema()

	assert.Equal(t, schema, result, "GetItemSchema should return ItemSchema")
}

func TestMediaType_GetPrefixEncoding_Nil_Success(t *testing.T) {
	t.Parallel()

	var m *MediaType
	result := m.GetPrefixEncoding()

	assert.Nil(t, result, "GetPrefixEncoding on nil should return nil")
}

func TestMediaType_GetPrefixEncoding_Set_Success(t *testing.T) {
	t.Parallel()

	enc := []*Encoding{{}, {}}
	m := &MediaType{
		PrefixEncoding: enc,
	}
	result := m.GetPrefixEncoding()

	assert.Equal(t, enc, result, "GetPrefixEncoding should return PrefixEncoding")
}

func TestMediaType_GetItemEncoding_Nil_Success(t *testing.T) {
	t.Parallel()

	var m *MediaType
	result := m.GetItemEncoding()

	assert.Nil(t, result, "GetItemEncoding on nil should return nil")
}

func TestMediaType_GetItemEncoding_Set_Success(t *testing.T) {
	t.Parallel()

	enc := &Encoding{}
	m := &MediaType{
		ItemEncoding: enc,
	}
	result := m.GetItemEncoding()

	assert.Equal(t, enc, result, "GetItemEncoding should return ItemEncoding")
}

func TestMediaType_GetSchema_Nil_Success(t *testing.T) {
	t.Parallel()

	var m *MediaType
	result := m.GetSchema()

	assert.Nil(t, result, "GetSchema on nil should return nil")
}

func TestMediaType_GetEncoding_Nil_Success(t *testing.T) {
	t.Parallel()

	var m *MediaType
	result := m.GetEncoding()

	assert.Nil(t, result, "GetEncoding on nil should return nil")
}

func TestMediaType_GetExamples_Nil_Success(t *testing.T) {
	t.Parallel()

	var m *MediaType
	result := m.GetExamples()

	assert.Nil(t, result, "GetExamples on nil should return nil")
}

func TestMediaType_GetExtensions_Nil_Success(t *testing.T) {
	t.Parallel()

	var m *MediaType
	result := m.GetExtensions()

	assert.NotNil(t, result, "GetExtensions on nil should return empty extensions")
}

func TestMediaType_GetExample_Nil_Success(t *testing.T) {
	t.Parallel()

	var m *MediaType
	result := m.GetExample()

	assert.Nil(t, result, "GetExample on nil should return nil")
}
