package openapi

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/speakeasy-api/openapi/openapi/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnsureMutex_ConcurrentAccess verifies that ensureMutex is safe to call
// concurrently from multiple goroutines on a Reference with nil initMutex.
func TestEnsureMutex_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	ref := &Reference[PathItem, *PathItem, *core.PathItem]{}

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			ref.ensureMutex()
		}()
	}

	wg.Wait()

	assert.NotNil(t, ref.initMutex, "initMutex should be initialized")
	assert.NotNil(t, ref.cacheMutex, "cacheMutex should be initialized")
}

// TestEnsureMutex_CopiedReference verifies that a copied Reference
// with nil mutexes can safely initialize its own mutexes independently.
func TestEnsureMutex_CopiedReference(t *testing.T) {
	t.Parallel()

	original := &Reference[PathItem, *PathItem, *core.PathItem]{}
	original.ensureMutex()

	// Simulate a copy by creating a new Reference with nil mutexes
	copied := &Reference[PathItem, *PathItem, *core.PathItem]{}

	assert.Nil(t, copied.initMutex, "copied reference should have nil initMutex")
	assert.Nil(t, copied.cacheMutex, "copied reference should have nil cacheMutex")

	copied.ensureMutex()

	assert.NotNil(t, copied.initMutex, "copied reference should initialize its own initMutex")
	assert.NotNil(t, copied.cacheMutex, "copied reference should initialize its own cacheMutex")

	// Original should be unaffected
	assert.NotNil(t, original.initMutex, "original initMutex should still be set")
	assert.NotNil(t, original.cacheMutex, "original cacheMutex should still be set")
}

// TestResolveAllReferences_PointerTraversingItsOwnReference verifies that a
// $ref whose JSON pointer passes through the reference being resolved does not
// deadlock.
//
// Reference.resolve used to hold the reference's own write lock across
// references.Resolve. That call navigates the document, and navigating into a
// reference calls GetObject, which takes a read lock. sync.RWMutex is not
// reentrant, so a pointer whose prefix named the reference being resolved
// blocked forever on a lock its own goroutine held.
//
// Each case below is a document that hung before the fix. The expected result
// is an unresolved-reference error, which is what every other pointer that
// names nothing already produced.
func TestResolveAllReferences_PointerTraversingItsOwnReference(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec string
	}{
		{
			name: "prefix names the reference being resolved",
			spec: `openapi: 3.1.0
info: {title: t, version: "1"}
paths:
  /a: {$ref: '#/paths/~1a/t'}
`,
		},
		{
			name: "prefix names the reference and the pointer resolves",
			spec: `openapi: 3.1.0
info: {title: t, version: "1"}
paths:
  /a:
    $ref: '#/paths/~1a/get'
    get: {operationId: a, responses: {"200": {description: ok}}}
`,
		},
		{
			name: "prefix reaches the in-flight reference through a resolved one",
			spec: `openapi: 3.1.0
info: {title: t, version: "1"}
paths:
  /a: {$ref: '#/paths/~1b'}
  /b: {$ref: '#/paths/~1a/t'}
`,
		},
		{
			name: "components spelling",
			spec: `openapi: 3.1.0
info: {title: t, version: "1"}
paths:
  /a: {$ref: '#/components/pathItems/A'}
components:
  pathItems:
    A: {$ref: '#/components/pathItems/A/t'}
`,
		},
		{
			name: "webhooks spelling",
			spec: `openapi: 3.1.0
info: {title: t, version: "1"}
paths: {}
webhooks:
  onA: {$ref: '#/webhooks/onA/t'}
`,
		},
		{
			// GetJSONPointer trims the pointer, so this names /a and the
			// reference resolves to itself. Before the fix the resolution cache
			// pointed at its own reference and GetObject's delegation at the end
			// of the cache-hit branch recursed until the stack was exhausted,
			// taking the process with it rather than failing the test.
			name: "reference resolving to itself via a trimmed pointer",
			spec: `openapi: 3.1.0
info: {title: t, version: "1"}
paths:
  /a:
    $ref: '#/paths/~1a '
    get: {operationId: a, responses: {"200": {description: ok}}}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			doc, _, err := Unmarshal(ctx, strings.NewReader(tt.spec))
			require.NoError(t, err)

			type result struct {
				resolveErrs []error
				err         error
			}
			done := make(chan result, 1)
			go func() {
				resolveErrs, err := doc.ResolveAllReferences(ctx, ResolveAllOptions{
					OpenAPILocation:     "test.yaml",
					DisableExternalRefs: true,
				})
				done <- result{resolveErrs: resolveErrs, err: err}
			}()

			select {
			case got := <-done:
				assert.True(t, got.err != nil || len(got.resolveErrs) > 0,
					"a pointer that names nothing should report an error")
			case <-time.After(30 * time.Second):
				t.Fatal("ResolveAllReferences deadlocked resolving a reference whose pointer traverses itself")
			}
		})
	}
}
