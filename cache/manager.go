package cache

import (
	"github.com/speakeasy-api/openapi/internal/utils"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/references"
)

// Manager provides centralized cache management for all global caches in the system
type Manager struct{}

// ClearAllCaches clears all global caches in the system.
// This includes:
// - URL parsing cache (internal/utils)
// - Reference resolution cache (references)
// - Field mapping cache (marshaller)
//
// This function is thread-safe and can be called from multiple goroutines.
// It's particularly useful for:
// - Testing scenarios where clean state is needed
// - Memory management when caches are no longer needed
// - Development/debugging when cache invalidation is required
func ClearAllCaches() {
	ClearURLCache()
	ClearReferenceCache()
	ClearFieldCache()
}

// ClearURLCache clears the global URL parsing cache.
// This cache stores parsed URL objects to avoid repeated parsing of the same URLs.
func ClearURLCache() {
	utils.ClearGlobalURLCache()
}

// ClearReferenceCache clears the global reference resolution cache.
// This cache stores resolved reference results to avoid repeated resolution
// of the same (reference, target) pairs.
func ClearReferenceCache() {
	references.ClearGlobalRefCache()
}

// ClearFieldCache clears the global field mapping cache.
// This cache stores pre-computed field maps for struct types to avoid
// expensive reflection operations during unmarshalling.
func ClearFieldCache() {
	marshaller.ClearGlobalFieldCache()
}

// GetCacheStats returns statistics about all global caches
type CacheStats struct {
	URLCacheSize       int64
	ReferenceCacheSize int64
	FieldCacheSize     int64
}

// GetAllCacheStats returns statistics about all global caches in the system
func GetAllCacheStats() CacheStats {
	return CacheStats{
		URLCacheSize:       utils.GetURLCacheStats().Size,
		ReferenceCacheSize: references.GetRefCacheStats().Size,
		FieldCacheSize:     marshaller.GetFieldCacheStats().Size,
	}
}
