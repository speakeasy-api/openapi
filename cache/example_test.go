package cache_test

import (
	"fmt"

	"github.com/speakeasy-api/openapi/cache"
	"github.com/speakeasy-api/openapi/internal/utils"
	"github.com/speakeasy-api/openapi/references"
)

// ExampleClearAllCaches demonstrates how to clear all global caches
func ExampleClearAllCaches() {
	// Start with clean caches for predictable output
	cache.ClearAllCaches()

	// Use some cached operations to populate caches
	_, _ = utils.ParseURLCached("https://example.com/api")
	_, _ = references.ResolveAbsoluteReferenceCached(
		references.Reference("#/components/schemas/User"),
		"https://api.example.com/openapi.yaml",
	)

	// Check cache stats before clearing
	stats := cache.GetAllCacheStats()
	fmt.Printf("Before clearing - URL cache: %d, Reference cache: %d, Field cache: %d\n",
		stats.URLCacheSize, stats.ReferenceCacheSize, stats.FieldCacheSize)

	// Clear all caches at once
	cache.ClearAllCaches()

	// Check cache stats after clearing
	stats = cache.GetAllCacheStats()
	fmt.Printf("After clearing - URL cache: %d, Reference cache: %d, Field cache: %d\n",
		stats.URLCacheSize, stats.ReferenceCacheSize, stats.FieldCacheSize)

	// Output:
	// Before clearing - URL cache: 2, Reference cache: 1, Field cache: 0
	// After clearing - URL cache: 0, Reference cache: 0, Field cache: 0
}

// ExampleClearURLCache demonstrates how to clear only the URL cache
func ExampleClearURLCache() {
	// Populate URL cache
	_, _ = utils.ParseURLCached("https://example.com/api/v1")
	_, _ = utils.ParseURLCached("https://example.com/api/v2")

	// Check URL cache size
	stats := cache.GetAllCacheStats()
	fmt.Printf("URL cache size before clearing: %d\n", stats.URLCacheSize)

	// Clear only URL cache
	cache.ClearURLCache()

	// Check URL cache size after clearing
	stats = cache.GetAllCacheStats()
	fmt.Printf("URL cache size after clearing: %d\n", stats.URLCacheSize)

	// Output:
	// URL cache size before clearing: 2
	// URL cache size after clearing: 0
}

// ExampleGetAllCacheStats demonstrates how to get statistics about all caches
func ExampleGetAllCacheStats() {
	// Clear all caches first for consistent output
	cache.ClearAllCaches()

	// Populate some caches
	_, _ = utils.ParseURLCached("https://example.com/api")
	_, _ = references.ResolveAbsoluteReferenceCached(
		references.Reference("#/components/schemas/User"),
		"https://api.example.com/openapi.yaml",
	)

	// Get cache statistics
	stats := cache.GetAllCacheStats()
	fmt.Printf("Cache Statistics:\n")
	fmt.Printf("  URL Cache: %d entries\n", stats.URLCacheSize)
	fmt.Printf("  Reference Cache: %d entries\n", stats.ReferenceCacheSize)
	fmt.Printf("  Field Cache: %d entries\n", stats.FieldCacheSize)

	// Output:
	// Cache Statistics:
	//   URL Cache: 2 entries
	//   Reference Cache: 1 entries
	//   Field Cache: 0 entries
}
