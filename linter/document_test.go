package linter_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
)

func TestNewDocumentInfo(t *testing.T) {
	t.Parallel()

	doc := &MockDoc{ID: "test-doc"}
	location := "/path/to/openapi.yaml"

	docInfo := linter.NewDocumentInfo(doc, location)

	assert.NotNil(t, docInfo)
	assert.Equal(t, doc, docInfo.Document)
	assert.Equal(t, location, docInfo.Location)
	assert.Nil(t, docInfo.Index)
}

func TestNewDocumentInfoWithIndex(t *testing.T) {
	t.Parallel()

	doc := &MockDoc{ID: "test-doc"}
	location := "/path/to/openapi.yaml"
	index := &openapi.Index{}

	docInfo := linter.NewDocumentInfoWithIndex(doc, location, index)

	assert.NotNil(t, docInfo)
	assert.Equal(t, doc, docInfo.Document)
	assert.Equal(t, location, docInfo.Location)
	assert.Equal(t, index, docInfo.Index)
}
