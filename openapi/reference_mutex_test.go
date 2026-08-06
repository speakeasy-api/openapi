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
// Each case below is a document that hung before the fix.
//
// Resolution is only half of it. A pointer that names a reference already in
// the chain also leaves that reference's resolution cache pointing back into
// the chain, so every case asserts GetObject afterwards: walking a cycle there
// exhausts the goroutine stack, which aborts the test binary outright rather
// than failing a single case.
func TestResolveAllReferences_PointerTraversingItsOwnReference(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		spec        string
		expectedErr string
	}{
		{
			name: "prefix names the reference being resolved",
			spec: `openapi: 3.1.0
info: {title: t, version: "1"}
paths:
  /a: {$ref: '#/paths/~1a/t'}
`,
			expectedErr: "unresolved reference",
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
			expectedErr: "unresolved reference",
		},
		{
			name: "prefix reaches the in-flight reference through a resolved one",
			spec: `openapi: 3.1.0
info: {title: t, version: "1"}
paths:
  /a: {$ref: '#/paths/~1b'}
  /b: {$ref: '#/paths/~1a/t'}
`,
			expectedErr: "unresolved reference",
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
			expectedErr: "unresolved reference",
		},
		{
			name: "webhooks spelling",
			spec: `openapi: 3.1.0
info: {title: t, version: "1"}
paths: {}
webhooks:
  onA: {$ref: '#/webhooks/onA/t'}
`,
			expectedErr: "unresolved reference",
		},
		{
			// GetJSONPointer trims the pointer, so this names /a and the
			// reference resolves to itself. The tracker reports that, but the
			// reference is left holding a resolution cache that points at
			// itself, so it is GetObject below that this case guards.
			name: "reference resolving to itself via a trimmed pointer",
			spec: `openapi: 3.1.0
info: {title: t, version: "1"}
paths:
  /a:
    $ref: '#/paths/~1a '
    get: {operationId: a, responses: {"200": {description: ok}}}
`,
			expectedErr: "circular reference detected: test.yaml#/paths/~1a -> test.yaml#/paths/~1a",
		},
		{
			// Three nodes, so the cycle closes a hop beyond anything a
			// pairwise check would notice.
			name: "three references forming a cycle via trimmed pointers",
			spec: `openapi: 3.1.0
info: {title: t, version: "1"}
paths:
  /a:
    $ref: '#/paths/~1b '
    get: {operationId: a, responses: {"200": {description: ok}}}
  /b:
    $ref: '#/paths/~1c '
    get: {operationId: b, responses: {"200": {description: ok}}}
  /c:
    $ref: '#/paths/~1a '
    get: {operationId: c, responses: {"200": {description: ok}}}
`,
			expectedErr: "circular reference detected: test.yaml#/paths/~1b -> test.yaml#/paths/~1c -> test.yaml#/paths/~1a -> test.yaml#/paths/~1b",
		},
		{
			// Same shape one hop wider: /a's cache points at /b and /b's points
			// back at /a. The tracker only notices on the third hop, by which
			// point both caches are published, so neither reference is a
			// self-reference and the cycle only shows up when walking them.
			name: "two references resolving to each other via trimmed pointers",
			spec: `openapi: 3.1.0
info: {title: t, version: "1"}
paths:
  /a:
    $ref: '#/paths/~1b '
    get: {operationId: a, responses: {"200": {description: ok}}}
  /b:
    $ref: '#/paths/~1a '
    get: {operationId: b, responses: {"200": {description: ok}}}
`,
			expectedErr: "circular reference detected: test.yaml#/paths/~1b -> test.yaml#/paths/~1a -> test.yaml#/paths/~1b",
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
				require.Error(t, got.err)
				assert.Contains(t, got.err.Error(), tt.expectedErr)
				assert.Empty(t, got.resolveErrs)
			case <-time.After(30 * time.Second):
				t.Fatal("ResolveAllReferences deadlocked resolving a reference whose pointer traverses itself")
			}

			// None of these resolved, so none of them have an object. Reaching
			// that verdict must not walk a cycle, and neither must walking the
			// parent links the failed resolution left behind.
			for path, pathItem := range doc.Paths.All() {
				assert.Nil(t, pathItem.GetObject(), "path %s should have no resolved object", path)
				assertParentLinksTerminate(t, path, pathItem)
			}
			for name, webhook := range doc.Webhooks.All() {
				assert.Nil(t, webhook.GetObject(), "webhook %s should have no resolved object", name)
				assertParentLinksTerminate(t, name, webhook)
			}
		})
	}
}

// assertParentLinksTerminate walks the parent links a resolution attempt left
// on ref and fails if they lead back to a reference already walked. A failed
// resolution still publishes links, and callers reach them through the public
// GetParent and GetTopLevelParent.
func assertParentLinksTerminate(t *testing.T, label string, ref *ReferencedPathItem) {
	t.Helper()

	seen := map[*ReferencedPathItem]bool{}
	for current := ref; current != nil; current = current.GetParent() {
		require.False(t, seen[current], "%s: parent links cycle", label)
		seen[current] = true
	}

	// The top-level parent is the head of the chain, never the reference itself.
	assert.NotSame(t, ref, ref.GetTopLevelParent(), "%s: top-level parent points at itself", label)
}

// TestResolve_SeparateCallsOverCycle covers the parent links when each member of
// a cycle is resolved by its own call to the public Resolve.
//
// ResolveAllReferences walks the document once and skips references already
// marked resolved, so it never revisits the second member. Resolve has no such
// guard: it starts a fresh chain every time, and the links from the earlier call
// are the only record that the two references already descend from each other.
func TestResolve_SeparateCallsOverCycle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		spec  string
		order []string
	}{
		{
			name:  "two-node cycle, /a first",
			spec:  twoNodeCycleSpec,
			order: []string{"/a", "/b"},
		},
		{
			name:  "two-node cycle, /b first",
			spec:  twoNodeCycleSpec,
			order: []string{"/b", "/a"},
		},
		{
			name:  "three-node cycle, every member in turn",
			spec:  threeNodeCycleSpec,
			order: []string{"/a", "/b", "/c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			doc, _, err := Unmarshal(ctx, strings.NewReader(tt.spec))
			require.NoError(t, err)

			opts := ResolveOptions{
				RootDocument:        doc,
				TargetDocument:      doc,
				TargetLocation:      "test.yaml",
				DisableExternalRefs: true,
			}

			// Every call reports the cycle; it is what they leave behind that matters.
			for _, path := range tt.order {
				ref, ok := doc.Paths.Get(path)
				require.True(t, ok)

				_, err := ref.Resolve(ctx, opts)
				require.Error(t, err, "resolving %s", path)
				assert.Contains(t, err.Error(), "circular reference detected")
			}

			for path, ref := range doc.Paths.All() {
				assertParentLinksTerminate(t, path, ref)
				assert.Nil(t, ref.GetObject(), "%s should have no resolved object", path)
			}
		})
	}
}

// TestResolve_ValidChainParentLinks pins the parent links a chain that resolves
// cleanly is expected to leave, so that guarding against cycles cannot quietly
// start dropping legitimate links. Resolving twice must not disturb them.
func TestResolve_ValidChainParentLinks(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	doc, _, err := Unmarshal(ctx, strings.NewReader(validChainSpec))
	require.NoError(t, err)

	opts := ResolveOptions{
		RootDocument:        doc,
		TargetDocument:      doc,
		TargetLocation:      "test.yaml",
		DisableExternalRefs: true,
	}

	root, ok := doc.Paths.Get("/a")
	require.True(t, ok)
	hopA, ok := doc.Components.PathItems.Get("A")
	require.True(t, ok)
	hopB, ok := doc.Components.PathItems.Get("B")
	require.True(t, ok)

	for i := range 2 {
		_, err := root.Resolve(ctx, opts)
		require.NoError(t, err, "resolve %d", i+1)

		obj := root.GetObject()
		require.NotNil(t, obj, "resolve %d", i+1)
		assert.Equal(t, "c", obj.Get().GetOperationID(), "resolve %d", i+1)

		// /a heads the chain, so it has no parent of its own.
		assert.Nil(t, root.GetParent())
		assert.Nil(t, root.GetTopLevelParent())

		assert.Same(t, root, hopA.GetParent())
		assert.Same(t, root, hopA.GetTopLevelParent())

		assert.Same(t, hopA, hopB.GetParent())
		assert.Same(t, root, hopB.GetTopLevelParent())
	}
}

// TestResolve_ConcurrentCallsOverCycle starts both members of a cycle resolving
// together. Each resolver decides whether an edge is safe by reading links that
// the other is writing, so the check and the write have to be one operation: two
// resolvers that both see no ancestry would otherwise publish opposite edges and
// rebuild the cycle between them.
func TestResolve_ConcurrentCallsOverCycle(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	doc, _, err := Unmarshal(ctx, strings.NewReader(twoNodeCycleSpec))
	require.NoError(t, err)

	opts := ResolveOptions{
		RootDocument:        doc,
		TargetDocument:      doc,
		TargetLocation:      "test.yaml",
		DisableExternalRefs: true,
	}

	pathA, ok := doc.Paths.Get("/a")
	require.True(t, ok)
	pathB, ok := doc.Paths.Get("/b")
	require.True(t, ok)

	// A barrier, so both resolvers are inside the check at the same time.
	// Sequencing them would not exercise anything the tests above do not.
	start := make(chan struct{})
	var wg sync.WaitGroup

	for _, ref := range []*ReferencedPathItem{pathA, pathB} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, _ = ref.Resolve(ctx, opts)
		}()
	}

	close(start)
	wg.Wait()

	for path, ref := range doc.Paths.All() {
		assertParentLinksTerminate(t, path, ref)
		assert.Nil(t, ref.GetObject(), "%s should have no resolved object", path)
	}
}

// TestResolve_ConcurrentCallsSameReference resolves one valid reference from many
// goroutines at once. Releasing the lock during resolution lets the work happen
// more than once; every caller still has to end up with the resolved object.
func TestResolve_ConcurrentCallsSameReference(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	doc, _, err := Unmarshal(ctx, strings.NewReader(validChainSpec))
	require.NoError(t, err)

	opts := ResolveOptions{
		RootDocument:        doc,
		TargetDocument:      doc,
		TargetLocation:      "test.yaml",
		DisableExternalRefs: true,
	}

	root, ok := doc.Paths.Get("/a")
	require.True(t, ok)

	const resolvers = 16

	start := make(chan struct{})
	errs := make([]error, resolvers)
	objs := make([]*PathItem, resolvers)

	var wg sync.WaitGroup
	for i := range resolvers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, errs[i] = root.Resolve(ctx, opts)
			objs[i] = root.GetObject()
		}()
	}

	close(start)
	wg.Wait()

	for i := range resolvers {
		require.NoError(t, errs[i], "resolver %d", i)
		require.NotNil(t, objs[i], "resolver %d", i)
		assert.Equal(t, "c", objs[i].Get().GetOperationID(), "resolver %d", i)
	}

	assertParentLinksTerminate(t, "/a", root)
}

const twoNodeCycleSpec = `openapi: 3.1.0
info: {title: t, version: "1"}
paths:
  /a:
    $ref: '#/paths/~1b '
    get: {operationId: a, responses: {"200": {description: ok}}}
  /b:
    $ref: '#/paths/~1a '
    get: {operationId: b, responses: {"200": {description: ok}}}
`

const threeNodeCycleSpec = `openapi: 3.1.0
info: {title: t, version: "1"}
paths:
  /a:
    $ref: '#/paths/~1b '
    get: {operationId: a, responses: {"200": {description: ok}}}
  /b:
    $ref: '#/paths/~1c '
    get: {operationId: b, responses: {"200": {description: ok}}}
  /c:
    $ref: '#/paths/~1a '
    get: {operationId: c, responses: {"200": {description: ok}}}
`

const validChainSpec = `openapi: 3.1.0
info: {title: t, version: "1"}
paths:
  /a: {$ref: '#/components/pathItems/A'}
components:
  pathItems:
    A: {$ref: '#/components/pathItems/B'}
    B: {$ref: '#/components/pathItems/C'}
    C: {get: {operationId: c, responses: {"200": {description: ok}}}}
`

// TestGetObject_ChainWalking covers the two ends of GetObject's chain walk: a
// chain of references that terminates has to be followed all the way to the
// object, and one that does not terminate has to give up.
func TestGetObject_ChainWalking(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		spec        string
		expectedOp  string
		expectedErr string
	}{
		{
			name: "multi-hop chain reaches the object",
			spec: `openapi: 3.1.0
info: {title: t, version: "1"}
paths:
  /a: {$ref: '#/components/pathItems/A'}
components:
  pathItems:
    A: {$ref: '#/components/pathItems/B'}
    B: {$ref: '#/components/pathItems/C'}
    C: {get: {operationId: c, responses: {"200": {description: ok}}}}
`,
			expectedOp: "c",
		},
		{
			name: "circular chain reports no object",
			spec: `openapi: 3.1.0
info: {title: t, version: "1"}
paths:
  /a: {$ref: '#/components/pathItems/A'}
components:
  pathItems:
    A: {$ref: '#/components/pathItems/B'}
    B: {$ref: '#/components/pathItems/A'}
`,
			expectedErr: "circular reference detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			doc, _, err := Unmarshal(ctx, strings.NewReader(tt.spec))
			require.NoError(t, err)

			_, err = doc.ResolveAllReferences(ctx, ResolveAllOptions{
				OpenAPILocation:     "test.yaml",
				DisableExternalRefs: true,
			})

			pathItem, ok := doc.Paths.Get("/a")
			require.True(t, ok)

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Nil(t, pathItem.GetObject())
				return
			}

			require.NoError(t, err)
			obj := pathItem.GetObject()
			require.NotNil(t, obj)
			assert.Equal(t, tt.expectedOp, obj.Get().GetOperationID())
		})
	}
}
