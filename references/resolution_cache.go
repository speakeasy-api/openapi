package references

import (
	"path/filepath"
	"sync"

	"github.com/speakeasy-api/openapi/internal/utils"
)

// RefCacheKey represents a unique key for caching reference resolution results
type RefCacheKey struct {
	RefURI         string
	TargetLocation string
}

// RefCache provides a thread-safe cache for reference resolution results
type RefCache struct {
	cache sync.Map // map[RefCacheKey]*AbsoluteReferenceResult
}

// Global reference resolution cache instance
var globalRefCache = &RefCache{}

// ResolveAbsoluteReferenceCached resolves a reference to an absolute reference string
// using a cache to avoid repeated resolution of the same (reference, target) pairs.
func ResolveAbsoluteReferenceCached(ref Reference, targetLocation string) (*AbsoluteReferenceResult, error) {
	return globalRefCache.Resolve(ref, targetLocation)
}

// Resolve resolves a reference using the cache. If the (ref, target) pair has been
// resolved before, it returns a copy of the cached result. Otherwise, it resolves
// the reference, caches it, and returns the result.
func (c *RefCache) Resolve(ref Reference, targetLocation string) (*AbsoluteReferenceResult, error) {
	key := RefCacheKey{
		RefURI:         ref.GetURI(),
		TargetLocation: targetLocation,
	}

	// Check cache first
	if cached, ok := c.cache.Load(key); ok {
		// Return a copy to prevent mutation of cached result
		cachedResult := cached.(*AbsoluteReferenceResult)
		resultCopy := &AbsoluteReferenceResult{
			AbsoluteReference: cachedResult.AbsoluteReference,
			Classification:    cachedResult.Classification, // Classification is read-only, safe to share
		}
		return resultCopy, nil
	}

	// Resolve using the original implementation
	result, err := resolveAbsoluteReferenceUncached(ref, targetLocation)
	if err != nil {
		return nil, err
	}

	// Store in cache
	c.cache.Store(key, result)

	return result, nil
}

// resolveAbsoluteReferenceUncached is the original ResolveAbsoluteReference implementation
// moved here to avoid infinite recursion when caching
func resolveAbsoluteReferenceUncached(ref Reference, targetLocation string) (*AbsoluteReferenceResult, error) {
	uri := ref.GetURI()

	// If the reference is empty, it's relative to the target document
	if uri == "" {
		classification, err := utils.ClassifyReference(targetLocation)
		if err != nil {
			return nil, err
		}
		return &AbsoluteReferenceResult{
			AbsoluteReference: targetLocation,
			Classification:    classification,
		}, nil
	}

	classification, err := utils.ClassifyReference(targetLocation)
	if err != nil {
		return nil, err
	}

	// Check if the URI is already absolute - if so, use it as-is instead of joining
	var absRef string
	var finalClassification *utils.ReferenceClassification
	uriClassification, uriErr := utils.ClassifyReference(uri)
	if uriErr == nil && uriClassification.Type == utils.ReferenceTypeURL {
		// URI is an absolute URL - use it directly
		absRef = uri
		finalClassification = uriClassification
	} else if uriErr == nil && uriClassification.Type == utils.ReferenceTypeFilePath && filepath.IsAbs(uri) {
		// URI is an absolute file path - use it directly
		absRef = uri
		finalClassification = uriClassification
	} else {
		// URI is relative - join with root location
		absRef, err = classification.JoinWith(uri)
		if err != nil {
			return nil, err
		}
		finalClassification = classification
	}

	return &AbsoluteReferenceResult{
		AbsoluteReference: absRef,
		Classification:    finalClassification,
	}, nil
}

// Clear clears all cached reference resolutions. Useful for testing or memory management.
func (c *RefCache) Clear() {
	c.cache.Range(func(key, value interface{}) bool {
		c.cache.Delete(key)
		return true
	})
}

// Stats returns basic statistics about the cache
type RefCacheStats struct {
	Size int64
}

// GetStats returns statistics about the cache
func (c *RefCache) GetStats() RefCacheStats {
	var size int64
	c.cache.Range(func(key, value interface{}) bool {
		size++
		return true
	})
	return RefCacheStats{Size: size}
}

// GetRefCacheStats returns statistics about the global reference cache
func GetRefCacheStats() RefCacheStats {
	return globalRefCache.GetStats()
}

// ClearGlobalRefCache clears the global reference cache
func ClearGlobalRefCache() {
	globalRefCache.Clear()
}
