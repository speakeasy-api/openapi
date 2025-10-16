package openapi_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// When top-level tags include tags unused by any operation, Clean should remove the unused ones
// and keep only those referenced by operations, preserving order.
func TestClean_RemoveUnusedTopLevelTags_KeepReferenced_PreserveOrder(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	const yml = `
openapi: 3.1.0
info:
  title: Tags Test
  version: 1.0.0
tags:
  - name: users
    description: "Users related operations"
  - name: admin
    description: "Administrative operations"
  - name: pets
    description: "Pet operations"
paths:
  /users:
    get:
      tags: ["users"]
      responses:
        "200":
          description: ok
  /admin:
    post:
      tags: ["admin"]
      responses:
        "201":
          description: created
`

	doc, validationErrs, err := openapi.Unmarshal(ctx, strings.NewReader(yml))
	require.NoError(t, err, "unmarshal should succeed")
	require.Empty(t, validationErrs)

	err = openapi.Clean(ctx, doc)
	require.NoError(t, err, "clean should succeed")

	var buf bytes.Buffer
	err = openapi.Marshal(ctx, doc, &buf)
	require.NoError(t, err, "marshal should succeed")
	actual := buf.String()

	// Expect only users and admin tags remain (pets removed), preserve original order
	const expected = `openapi: 3.1.0
info:
  title: Tags Test
  version: 1.0.0
tags:
  - name: users
    description: "Users related operations"
  - name: admin
    description: "Administrative operations"
paths:
  /users:
    get:
      tags: ["users"]
      responses:
        "200":
          description: ok
  /admin:
    post:
      tags: ["admin"]
      responses:
        "201":
          description: created
`

	assert.Equal(t, expected, actual, "unused top-level tags should be removed; referenced tags kept in original order")
}

// When no operation references any tag, Clean should remove the entire top-level tags array.
func TestClean_RemoveAllTopLevelTags_WhenUnused(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	const yml = `
openapi: 3.1.0
info:
  title: Tags Test
  version: 1.0.0
tags:
  - name: users
  - name: admin
paths:
  /ping:
    get:
      responses:
        "200":
          description: pong
`

	doc, validationErrs, err := openapi.Unmarshal(ctx, strings.NewReader(yml))
	require.NoError(t, err, "unmarshal should succeed")
	require.Empty(t, validationErrs)

	err = openapi.Clean(ctx, doc)
	require.NoError(t, err, "clean should succeed")

	var buf bytes.Buffer
	err = openapi.Marshal(ctx, doc, &buf)
	require.NoError(t, err, "marshal should succeed")
	actual := buf.String()

	// Expect the tags array to be removed completely
	const expected = `openapi: 3.1.0
info:
  title: Tags Test
  version: 1.0.0
paths:
  /ping:
    get:
      responses:
        "200":
          description: pong
`

	assert.Equal(t, expected, actual, "top-level tags should be removed entirely when unused")
}
