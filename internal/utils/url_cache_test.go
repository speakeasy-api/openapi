package utils

import (
	"fmt"
	"net/url"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURLCache_Parse_Success(t *testing.T) {
	t.Parallel()
	cache := &URLCache{}

	testURL := "https://example.com/path?query=value"

	// First parse - should cache the result
	parsed1, err := cache.Parse(testURL)
	require.NoError(t, err)
	assert.Equal(t, "https", parsed1.Scheme)
	assert.Equal(t, "example.com", parsed1.Host)
	assert.Equal(t, "/path", parsed1.Path)
	assert.Equal(t, "query=value", parsed1.RawQuery)

	// Second parse - should return cached result
	parsed2, err := cache.Parse(testURL)
	require.NoError(t, err)
	assert.Equal(t, parsed1.String(), parsed2.String())

	// Verify they are different instances (copies)
	assert.NotSame(t, parsed1, parsed2, "cached URLs should be copies, not the same instance")

	// Modify one to ensure they don't affect each other
	parsed1.Host = "modified.com"
	assert.NotEqual(t, parsed1.Host, parsed2.Host, "modifying one URL should not affect the cached copy")
}

func TestURLCache_Parse_Error(t *testing.T) {
	t.Parallel()
	cache := &URLCache{}

	invalidURL := "://invalid-url"

	// Should return error and not cache invalid URLs
	_, err := cache.Parse(invalidURL)
	require.Error(t, err)

	// Verify it's not cached by checking stats
	stats := URLCacheStats{}
	cache.cache.Range(func(key, value interface{}) bool {
		stats.Size++
		return true
	})
	assert.Equal(t, int64(0), stats.Size, "invalid URLs should not be cached")
}

func TestURLCache_Concurrent_Access(t *testing.T) {
	t.Parallel()
	cache := &URLCache{}
	testURL := "https://concurrent-test.com"

	var wg sync.WaitGroup
	numGoroutines := 100
	results := make([]*url.URL, numGoroutines)
	errors := make([]error, numGoroutines)

	// Launch multiple goroutines to parse the same URL concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index], errors[index] = cache.Parse(testURL)
		}(i)
	}

	wg.Wait()

	// Verify all results are successful and equivalent
	for i := 0; i < numGoroutines; i++ {
		require.NoError(t, errors[i], "goroutine %d should not have error", i)
		require.NotNil(t, results[i], "goroutine %d should have result", i)
		assert.Equal(t, testURL, results[i].String(), "goroutine %d should have correct URL", i)
	}

	// Verify cache only has one entry
	var cacheSize int64
	cache.cache.Range(func(key, value interface{}) bool {
		cacheSize++
		return true
	})
	assert.Equal(t, int64(1), cacheSize, "cache should only have one entry despite concurrent access")
}

func TestURLCache_Clear(t *testing.T) {
	t.Parallel()
	cache := &URLCache{}

	// Add some URLs to cache
	urls := []string{
		"https://example1.com",
		"https://example2.com",
		"https://example3.com",
	}

	for _, u := range urls {
		_, err := cache.Parse(u)
		require.NoError(t, err)
	}

	// Verify cache has entries
	var sizeBefore int64
	cache.cache.Range(func(key, value interface{}) bool {
		sizeBefore++
		return true
	})
	assert.Equal(t, int64(3), sizeBefore)

	// Clear cache
	cache.Clear()

	// Verify cache is empty
	var sizeAfter int64
	cache.cache.Range(func(key, value interface{}) bool {
		sizeAfter++
		return true
	})
	assert.Equal(t, int64(0), sizeAfter)
}

//nolint:paralleltest
func TestParseURLCached_Global(t *testing.T) {
	// Don't run in parallel since we're testing global cache state

	// Clear global cache before test
	ClearGlobalURLCache()

	testURL := "https://global-test.com"

	// Parse using global function
	parsed1, err := ParseURLCached(testURL)
	require.NoError(t, err)
	assert.Equal(t, testURL, parsed1.String())

	// Verify it's cached globally
	stats := GetURLCacheStats()
	assert.Equal(t, int64(1), stats.Size)

	// Parse again - should use cache
	parsed2, err := ParseURLCached(testURL)
	require.NoError(t, err)
	assert.Equal(t, parsed1.String(), parsed2.String())
	assert.NotSame(t, parsed1, parsed2, "should return copies")

	// Clean up
	ClearGlobalURLCache()
}

func TestClassifyReference_WithCache(t *testing.T) {
	t.Parallel()
	// Clear global cache before test
	ClearGlobalURLCache()

	testURL := "https://api.example.com/openapi.yaml"

	// First classification - should cache URL parsing
	result1, err := ClassifyReference(testURL)
	require.NoError(t, err)
	assert.True(t, result1.IsURL)
	assert.Equal(t, ReferenceTypeURL, result1.Type)
	assert.NotNil(t, result1.ParsedURL)

	// Verify URL is cached
	stats := GetURLCacheStats()
	assert.Positive(t, stats.Size, "URL should be cached after classification")

	// Second classification - should use cached URL
	result2, err := ClassifyReference(testURL)
	require.NoError(t, err)
	assert.Equal(t, result1.IsURL, result2.IsURL)
	assert.Equal(t, result1.Type, result2.Type)
	assert.Equal(t, result1.ParsedURL.String(), result2.ParsedURL.String())

	// Clean up
	ClearGlobalURLCache()
}

func BenchmarkURLCache_Parse_Cached(b *testing.B) {
	cache := &URLCache{}
	testURL := "https://api.example.com/v1/openapi.yaml?version=3.0.0"

	// Pre-populate cache
	_, err := cache.Parse(testURL)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cache.Parse(testURL)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkURLCache_Parse_Uncached(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// Use different URL each time to avoid caching
		testURL := fmt.Sprintf("https://api.example.com/v1/openapi-%d.yaml", i)
		_, err := url.Parse(testURL)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkURLCache_vs_Standard_Parsing(b *testing.B) {
	testURL := "https://api.example.com/v1/openapi.yaml?version=3.0.0"

	b.Run("Standard", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := url.Parse(testURL)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Cached", func(b *testing.B) {
		cache := &URLCache{}
		// Pre-populate cache
		_, err := cache.Parse(testURL)
		require.NoError(b, err)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := cache.Parse(testURL)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
