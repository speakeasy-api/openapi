package references

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefCache_Resolve_Success(t *testing.T) {
	t.Parallel()
	cache := &RefCache{}

	ref := Reference("#/components/schemas/User")
	targetLocation := "https://api.example.com/openapi.yaml"

	// First resolve - should cache the result
	result1, err := cache.Resolve(ref, targetLocation)
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com/openapi.yaml", result1.AbsoluteReference)
	assert.NotNil(t, result1.Classification)

	// Second resolve - should return cached result
	result2, err := cache.Resolve(ref, targetLocation)
	require.NoError(t, err)
	assert.Equal(t, result1.AbsoluteReference, result2.AbsoluteReference)

	// Verify they are different instances (copies)
	assert.NotSame(t, result1, result2, "cached results should be copies, not the same instance")

	// Verify cache has one entry
	stats := cache.GetStats()
	assert.Equal(t, int64(1), stats.Size)
}

func TestRefCache_Resolve_DifferentKeys(t *testing.T) {
	t.Parallel()
	cache := &RefCache{}

	// Different reference, same target
	ref1 := Reference("./schemas/user.yaml")
	ref2 := Reference("./schemas/product.yaml")
	targetLocation := "https://api.example.com/openapi.yaml"

	result1, err := cache.Resolve(ref1, targetLocation)
	require.NoError(t, err)

	result2, err := cache.Resolve(ref2, targetLocation)
	require.NoError(t, err)

	assert.NotEqual(t, result1.AbsoluteReference, result2.AbsoluteReference)

	// Should have two cache entries
	stats := cache.GetStats()
	assert.Equal(t, int64(2), stats.Size)

	// Same reference, different target
	ref3 := Reference("./schemas/user.yaml")
	targetLocation2 := "https://other.example.com/openapi.yaml"

	result3, err := cache.Resolve(ref3, targetLocation2)
	require.NoError(t, err)

	assert.NotEqual(t, result1.AbsoluteReference, result3.AbsoluteReference)

	// Should have three cache entries
	stats = cache.GetStats()
	assert.Equal(t, int64(3), stats.Size)
}

func TestRefCache_Resolve_EmptyReference(t *testing.T) {
	t.Parallel()
	cache := &RefCache{}

	ref := Reference("")
	targetLocation := "https://api.example.com/openapi.yaml"

	result, err := cache.Resolve(ref, targetLocation)
	require.NoError(t, err)
	assert.Equal(t, targetLocation, result.AbsoluteReference)
	assert.NotNil(t, result.Classification)
}

func TestRefCache_Resolve_AbsoluteURL(t *testing.T) {
	t.Parallel()
	cache := &RefCache{}

	ref := Reference("https://other.example.com/schema.yaml#/User")
	targetLocation := "https://api.example.com/openapi.yaml"

	result, err := cache.Resolve(ref, targetLocation)
	require.NoError(t, err)
	assert.Equal(t, "https://other.example.com/schema.yaml", result.AbsoluteReference)
	assert.NotNil(t, result.Classification)
	assert.True(t, result.Classification.IsURL)
}

func TestRefCache_Resolve_RelativePath(t *testing.T) {
	t.Parallel()
	cache := &RefCache{}

	ref := Reference("./schemas/user.yaml#/User")
	targetLocation := "https://api.example.com/openapi.yaml"

	result, err := cache.Resolve(ref, targetLocation)
	require.NoError(t, err)
	assert.Contains(t, result.AbsoluteReference, "schemas/user.yaml")
	assert.NotNil(t, result.Classification)
}

func TestRefCache_Concurrent_Access(t *testing.T) {
	t.Parallel()
	cache := &RefCache{}

	ref := Reference("#/components/schemas/User")
	targetLocation := "https://api.example.com/openapi.yaml"

	var wg sync.WaitGroup
	numGoroutines := 100
	results := make([]*AbsoluteReferenceResult, numGoroutines)
	errors := make([]error, numGoroutines)

	// Launch multiple goroutines to resolve the same reference concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index], errors[index] = cache.Resolve(ref, targetLocation)
		}(i)
	}

	wg.Wait()

	// Verify all results are successful and equivalent
	for i := 0; i < numGoroutines; i++ {
		require.NoError(t, errors[i], "goroutine %d should not have error", i)
		require.NotNil(t, results[i], "goroutine %d should have result", i)
		assert.Equal(t, "https://api.example.com/openapi.yaml", results[i].AbsoluteReference, "goroutine %d should have correct result", i)
	}

	// Verify cache only has one entry
	stats := cache.GetStats()
	assert.Equal(t, int64(1), stats.Size, "cache should only have one entry despite concurrent access")
}

func TestRefCache_Clear(t *testing.T) {
	t.Parallel()
	cache := &RefCache{}

	// Add some references to cache
	refs := []struct {
		ref    Reference
		target string
	}{
		{Reference("#/components/schemas/User"), "https://api1.example.com/openapi.yaml"},
		{Reference("#/components/schemas/Product"), "https://api2.example.com/openapi.yaml"},
		{Reference("./schema.yaml"), "https://api3.example.com/openapi.yaml"},
	}

	for _, r := range refs {
		_, err := cache.Resolve(r.ref, r.target)
		require.NoError(t, err)
	}

	// Verify cache has entries
	stats := cache.GetStats()
	assert.Equal(t, int64(3), stats.Size)

	// Clear cache
	cache.Clear()

	// Verify cache is empty
	stats = cache.GetStats()
	assert.Equal(t, int64(0), stats.Size)
}

func TestResolveAbsoluteReferenceCached_Global(t *testing.T) {
	t.Parallel()
	// Clear global cache before test
	ClearGlobalRefCache()

	ref := Reference("#/components/schemas/User")
	targetLocation := "https://api.example.com/openapi.yaml"

	// Resolve using global function
	result1, err := ResolveAbsoluteReferenceCached(ref, targetLocation)
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com/openapi.yaml", result1.AbsoluteReference)

	// Verify it's cached globally
	stats := GetRefCacheStats()
	assert.Equal(t, int64(1), stats.Size)

	// Resolve again - should use cache
	result2, err := ResolveAbsoluteReferenceCached(ref, targetLocation)
	require.NoError(t, err)
	assert.Equal(t, result1.AbsoluteReference, result2.AbsoluteReference)
	assert.NotSame(t, result1, result2, "should return copies")

	// Clean up
	ClearGlobalRefCache()
}

func TestResolveAbsoluteReference_UsesCache(t *testing.T) {
	t.Parallel()
	// Clear global cache before test
	ClearGlobalRefCache()

	ref := Reference("#/components/schemas/User")
	targetLocation := "https://api.example.com/openapi.yaml"

	// Call the main function - should use cache internally
	result1, err := ResolveAbsoluteReference(ref, targetLocation)
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com/openapi.yaml", result1.AbsoluteReference)

	// Verify it's cached
	stats := GetRefCacheStats()
	assert.Equal(t, int64(1), stats.Size)

	// Call again - should use cache
	result2, err := ResolveAbsoluteReference(ref, targetLocation)
	require.NoError(t, err)
	assert.Equal(t, result1.AbsoluteReference, result2.AbsoluteReference)

	// Clean up
	ClearGlobalRefCache()
}

func BenchmarkRefCache_Resolve_Cached(b *testing.B) {
	cache := &RefCache{}
	ref := Reference("#/components/schemas/User")
	targetLocation := "https://api.example.com/openapi.yaml"

	// Pre-populate cache
	_, err := cache.Resolve(ref, targetLocation)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cache.Resolve(ref, targetLocation)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRefCache_Resolve_Uncached(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// Use different reference each time to avoid caching
		ref := Reference(fmt.Sprintf("#/components/schemas/User%d", i))
		targetLocation := "https://api.example.com/openapi.yaml"
		_, err := resolveAbsoluteReferenceUncached(ref, targetLocation)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRefCache_vs_Uncached(b *testing.B) {
	ref := Reference("#/components/schemas/User")
	targetLocation := "https://api.example.com/openapi.yaml"

	b.Run("Uncached", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := resolveAbsoluteReferenceUncached(ref, targetLocation)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Cached", func(b *testing.B) {
		cache := &RefCache{}
		// Pre-populate cache
		_, err := cache.Resolve(ref, targetLocation)
		require.NoError(b, err)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := cache.Resolve(ref, targetLocation)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
