package utils

import (
	"net/url"
	"sync"
)

// URLCache provides a thread-safe cache for parsed URLs to avoid repeated parsing
type URLCache struct {
	cache sync.Map // map[string]*url.URL
}

// Global URL cache instance
var globalURLCache = &URLCache{}

// ParseURLCached parses a URL string using a cache to avoid repeated parsing of the same URLs.
// This is particularly beneficial when the same base URLs are parsed thousands of times.
func ParseURLCached(rawURL string) (*url.URL, error) {
	return globalURLCache.Parse(rawURL)
}

// Parse parses a URL string using the cache. If the URL has been parsed before,
// it returns a copy of the cached result. Otherwise, it parses the URL, caches it,
// and returns the result.
func (c *URLCache) Parse(rawURL string) (*url.URL, error) {
	// Check cache first
	if cached, ok := c.cache.Load(rawURL); ok {
		// Return a copy to prevent mutation of cached URL
		cachedURL := cached.(*url.URL)
		urlCopy := *cachedURL
		return &urlCopy, nil
	}

	// Parse the URL
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	// Store a copy in cache to prevent mutation issues
	urlCopy := *parsed
	c.cache.Store(rawURL, &urlCopy)

	// Return the original parsed URL
	return parsed, nil
}

// Clear clears all cached URLs. Useful for testing or memory management.
func (c *URLCache) Clear() {
	c.cache.Range(func(key, value interface{}) bool {
		c.cache.Delete(key)
		return true
	})
}

// Stats returns basic statistics about the cache
type URLCacheStats struct {
	Size int64
}

// GetStats returns statistics about the global URL cache
func GetURLCacheStats() URLCacheStats {
	var size int64
	globalURLCache.cache.Range(func(key, value interface{}) bool {
		size++
		return true
	})
	return URLCacheStats{Size: size}
}

// ClearGlobalURLCache clears the global URL cache
func ClearGlobalURLCache() {
	globalURLCache.Clear()
}
