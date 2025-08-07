package cache

import (
	"reflect"
	"testing"

	"github.com/speakeasy-api/openapi/internal/utils"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClearAllCaches_Success(t *testing.T) { //nolint:paralleltest
	// Populate all caches with test data
	populateURLCache(t)
	populateReferenceCache(t)
	populateFieldCache(t)

	// Verify caches have data
	stats := GetAllCacheStats()
	assert.Greater(t, stats.URLCacheSize, int64(0), "URL cache should have entries")
	assert.Greater(t, stats.ReferenceCacheSize, int64(0), "Reference cache should have entries")
	assert.Greater(t, stats.FieldCacheSize, int64(0), "Field cache should have entries")

	// Clear all caches
	ClearAllCaches()

	// Verify all caches are empty
	stats = GetAllCacheStats()
	assert.Equal(t, int64(0), stats.URLCacheSize, "URL cache should be empty")
	assert.Equal(t, int64(0), stats.ReferenceCacheSize, "Reference cache should be empty")
	assert.Equal(t, int64(0), stats.FieldCacheSize, "Field cache should be empty")
}

func TestClearURLCache_Success(t *testing.T) {
	t.Parallel()

	// Populate URL cache
	populateURLCache(t)

	// Verify cache has data
	stats := GetAllCacheStats()
	assert.Greater(t, stats.URLCacheSize, int64(0), "URL cache should have entries")

	// Clear only URL cache
	ClearURLCache()

	// Verify only URL cache is empty
	stats = GetAllCacheStats()
	assert.Equal(t, int64(0), stats.URLCacheSize, "URL cache should be empty")
}

func TestClearReferenceCache_Success(t *testing.T) {
	t.Parallel()

	// Populate reference cache
	populateReferenceCache(t)

	// Verify cache has data
	stats := GetAllCacheStats()
	assert.Greater(t, stats.ReferenceCacheSize, int64(0), "Reference cache should have entries")

	// Clear only reference cache
	ClearReferenceCache()

	// Verify only reference cache is empty
	stats = GetAllCacheStats()
	assert.Equal(t, int64(0), stats.ReferenceCacheSize, "Reference cache should be empty")
}

func TestClearFieldCache_Success(t *testing.T) {
	t.Parallel()

	// Populate field cache
	populateFieldCache(t)

	// Verify cache has data
	stats := GetAllCacheStats()
	assert.Greater(t, stats.FieldCacheSize, int64(0), "Field cache should have entries")

	// Clear only field cache
	ClearFieldCache()

	// Verify only field cache is empty
	stats = GetAllCacheStats()
	assert.Equal(t, int64(0), stats.FieldCacheSize, "Field cache should be empty")
}

// nolint:paralleltest
func TestGetAllCacheStats_Success(t *testing.T) {
	// Don't run in parallel since we're testing global cache state

	// Clear all caches first
	ClearAllCaches()

	// Verify all caches start empty
	stats := GetAllCacheStats()
	assert.Equal(t, int64(0), stats.URLCacheSize, "URL cache should start empty")
	assert.Equal(t, int64(0), stats.ReferenceCacheSize, "Reference cache should start empty")
	assert.Equal(t, int64(0), stats.FieldCacheSize, "Field cache should start empty")

	// Populate caches
	populateURLCache(t)
	populateReferenceCache(t)
	populateFieldCache(t)

	// Verify stats reflect populated caches
	stats = GetAllCacheStats()
	assert.Greater(t, stats.URLCacheSize, int64(0), "URL cache should have entries")
	assert.Greater(t, stats.ReferenceCacheSize, int64(0), "Reference cache should have entries")
	assert.Greater(t, stats.FieldCacheSize, int64(0), "Field cache should have entries")
}

// Helper functions to populate caches with test data

func populateURLCache(t *testing.T) {
	urls := []string{
		"https://example1.com/api/v1",
		"https://example2.com/api/v2",
		"https://example3.com/api/v3",
	}

	for _, url := range urls {
		_, err := utils.ParseURLCached(url)
		require.NoError(t, err, "should parse URL successfully")
	}
}

func populateReferenceCache(t *testing.T) {
	refs := []struct {
		ref    references.Reference
		target string
	}{
		{references.Reference("#/components/schemas/User"), "https://api1.example.com/openapi.yaml"},
		{references.Reference("#/components/schemas/Product"), "https://api2.example.com/openapi.yaml"},
		{references.Reference("./schema.yaml"), "https://api3.example.com/openapi.yaml"},
	}

	for _, r := range refs {
		_, err := references.ResolveAbsoluteReferenceCached(r.ref, r.target)
		require.NoError(t, err, "should resolve reference successfully")
	}
}

func populateFieldCache(t *testing.T) {
	// Define test struct types to populate the field cache
	type TestStruct1 struct {
		Name     string `key:"name" required:"true"`
		Value    int    `key:"value"`
		Optional string `key:"optional"`
	}

	type TestStruct2 struct {
		ID          string   `key:"id" required:"true"`
		Description string   `key:"description"`
		Tags        []string `key:"tags"`
	}

	type TestStruct3 struct {
		Title    string            `key:"title" required:"true"`
		Metadata map[string]string `key:"metadata"`
		Active   bool              `key:"active"`
	}

	// Register types to populate the field cache
	marshaller.RegisterType(func() *TestStruct1 { return &TestStruct1{} })
	marshaller.RegisterType(func() *TestStruct2 { return &TestStruct2{} })
	marshaller.RegisterType(func() *TestStruct3 { return &TestStruct3{} })

	// Access the cached field maps to ensure they're in the cache
	_ = reflect.TypeOf(TestStruct1{})
	_ = reflect.TypeOf(TestStruct2{})
	_ = reflect.TypeOf(TestStruct3{})
}
