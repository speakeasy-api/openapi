package openapi_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/require"
)

func TestMediaType_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	yml := `
schema:
  type: object
  properties:
    name:
      type: string
    age:
      type: integer
example:
  name: John
  age: 30
examples:
  user1:
    value:
      name: Alice
      age: 25
    summary: First user example
  user2:
    value:
      name: Bob
      age: 35
    description: Second user example
encoding:
  profileImage:
    contentType: image/jpeg
    style: form
    explode: true
    allowReserved: false
    headers:
      X-Rate-Limit:
        schema:
          type: integer
x-test: some-value
`

	var mediaType openapi.MediaType

	validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(yml)), &mediaType)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	schema := mediaType.GetSchema()
	require.NotNil(t, schema)

	example := mediaType.GetExample()
	require.NotNil(t, example)

	examples := mediaType.GetExamples()
	require.NotNil(t, examples)
	user1Example, ok := examples.Get("user1")
	require.True(t, ok)
	require.Equal(t, "First user example", user1Example.Object.GetSummary())

	encoding := mediaType.GetEncoding()
	require.NotNil(t, encoding)
	profileImageEncoding, ok := encoding.Get("profileImage")
	require.True(t, ok)
	require.Equal(t, "image/jpeg", profileImageEncoding.GetContentTypeValue())
	require.Equal(t, openapi.SerializationStyleForm, profileImageEncoding.GetStyle())
	require.True(t, profileImageEncoding.GetExplode())
	require.False(t, profileImageEncoding.GetAllowReserved())

	ext, ok := mediaType.GetExtensions().Get("x-test")
	require.True(t, ok)
	require.Equal(t, "some-value", ext.Value)
}
