package openapi

import (
	"sync"
	"testing"

	"github.com/speakeasy-api/openapi/openapi/core"
	"github.com/stretchr/testify/assert"
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
