package marshaller

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRegistered_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		typ      reflect.Type
		expected bool
	}{
		{
			name:     "registered string type returns true",
			typ:      reflect.TypeOf(""),
			expected: true,
		},
		{
			name:     "registered pointer string type returns true",
			typ:      reflect.TypeOf((*string)(nil)),
			expected: true,
		},
		{
			name:     "registered int type returns true",
			typ:      reflect.TypeOf(0),
			expected: true,
		},
		{
			name:     "unregistered custom struct returns false",
			typ:      reflect.TypeOf(struct{ Name string }{}),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := IsRegistered(tt.typ)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestCreateInstance_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		typ      reflect.Type
		expected reflect.Type
	}{
		{
			name:     "create string instance",
			typ:      reflect.TypeOf(""),
			expected: reflect.TypeOf((*string)(nil)),
		},
		{
			name:     "create pointer string instance",
			typ:      reflect.TypeOf((*string)(nil)),
			expected: reflect.TypeOf((*string)(nil)),
		},
		{
			name:     "create int instance",
			typ:      reflect.TypeOf(0),
			expected: reflect.TypeOf((*int)(nil)),
		},
		{
			name:     "create bool instance",
			typ:      reflect.TypeOf(false),
			expected: reflect.TypeOf((*bool)(nil)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := CreateInstance(tt.typ)
			assert.Equal(t, tt.expected, actual.Type())
			assert.NotNil(t, actual.Interface())
		})
	}
}

func TestCreateInstance_UnregisteredType_Success(t *testing.T) {
	t.Parallel()

	// Test with an unregistered struct type
	type UnregisteredStruct struct {
		Name string
		Age  int
	}

	typ := reflect.TypeOf(UnregisteredStruct{})
	result := CreateInstance(typ)

	assert.Equal(t, reflect.TypeOf((*UnregisteredStruct)(nil)), result.Type())
	assert.NotNil(t, result.Interface())
}

func TestRegisterType_Success(t *testing.T) {
	t.Parallel()

	// Define a custom type for testing
	type TestCustomType struct {
		Value string
	}

	// Register the type
	RegisterType(func() *TestCustomType {
		return &TestCustomType{Value: "test"}
	})

	// Verify it's registered
	typ := reflect.TypeOf(TestCustomType{})
	assert.True(t, IsRegistered(typ))

	// Verify we can create instances
	instance := CreateInstance(typ)
	assert.Equal(t, reflect.TypeOf((*TestCustomType)(nil)), instance.Type())

	// Verify the factory function is used
	customInstance := instance.Interface().(*TestCustomType)
	assert.Equal(t, "test", customInstance.Value)
}

//nolint:paralleltest // Global state manipulation requires sequential execution
func TestClearGlobalFieldCache_Success(t *testing.T) {
	// Note: Not using t.Parallel() because this test manipulates global cache state
	// that could interfere with other parallel tests

	// Define a test struct to build cache for
	type TestStruct struct {
		Name string `key:"name"`
		Age  int    `key:"age"`
	}

	// Register the type to build field cache
	RegisterType(func() *TestStruct {
		return &TestStruct{}
	})

	// Verify cache has entries
	stats := GetFieldCacheStats()
	initialSize := stats.Size

	// Clear the cache
	ClearGlobalFieldCache()

	// Verify cache is empty
	stats = GetFieldCacheStats()
	assert.Equal(t, int64(0), stats.Size)
	assert.True(t, stats.Size < initialSize || initialSize == 0)
}

//nolint:paralleltest // Global state manipulation requires sequential execution
func TestGetFieldCacheStats_Success(t *testing.T) {
	// Note: Not using t.Parallel() because this test manipulates global cache state
	// that could interfere with other parallel tests

	// Clear cache first to get a clean state
	ClearGlobalFieldCache()

	// Define test structs to build cache for
	type TestStruct1 struct {
		Name string `key:"name"`
	}
	type TestStruct2 struct {
		Value int `key:"value"`
	}

	// Register types to build field cache
	RegisterType(func() *TestStruct1 {
		return &TestStruct1{}
	})
	RegisterType(func() *TestStruct2 {
		return &TestStruct2{}
	})

	// Get stats
	stats := GetFieldCacheStats()

	// Should have at least 2 entries (our test structs)
	assert.GreaterOrEqual(t, stats.Size, int64(2))
}

func TestBuildFieldCacheForType_Success(t *testing.T) {
	t.Parallel()

	// Define a test struct with various field types
	type TestStruct struct {
		Name          string  `key:"name" required:"true"`
		Age           int     `key:"age"`
		OptionalField *string `key:"optional"`
		// Extensions field (special handling)
		Extensions interface{} `key:"extensions"`
	}

	structType := reflect.TypeOf(TestStruct{})

	// Build cache for the type
	buildFieldCacheForType(structType)

	// Verify cache was built
	cached := getFieldMapCached(structType)

	assert.NotEmpty(t, cached.Fields)
	assert.Contains(t, cached.Fields, "name")
	assert.Contains(t, cached.Fields, "age")
	assert.Contains(t, cached.Fields, "optional")

	// Verify required field detection
	assert.True(t, cached.Fields["name"].Required)
	assert.False(t, cached.Fields["age"].Required)      // no required tag
	assert.False(t, cached.Fields["optional"].Required) // pointer type
}

func TestBuildFieldCacheForType_NonStruct_Success(t *testing.T) {
	t.Parallel()

	// Test with non-struct type (should not panic)
	intType := reflect.TypeOf(0)
	buildFieldCacheForType(intType)

	// Should not create cache entry for non-struct
	_, ok := fieldCache.Load(intType)
	assert.False(t, ok)
}

func TestGetFieldMapCached_CacheMiss_Success(t *testing.T) {
	t.Parallel()

	// Define a struct that hasn't been registered
	type UnregisteredStruct struct {
		Name string `key:"name"`
	}

	structType := reflect.TypeOf(UnregisteredStruct{})

	// This should build cache on-demand
	cached := getFieldMapCached(structType)

	assert.NotEmpty(t, cached.Fields)
	assert.Contains(t, cached.Fields, "name")
}

func TestIsTesting_Success(t *testing.T) {
	t.Parallel()

	// During test execution, isTesting() should return true
	result := isTesting()
	assert.True(t, result)
}

func TestCachedFieldInfo_Success(t *testing.T) {
	t.Parallel()

	// Test CachedFieldInfo struct creation
	info := CachedFieldInfo{
		Name:         "TestField",
		Index:        0,
		Required:     true,
		Tag:          "test",
		IsExported:   true,
		IsExtensions: false,
	}

	assert.Equal(t, "TestField", info.Name)
	assert.Equal(t, 0, info.Index)
	assert.True(t, info.Required)
	assert.Equal(t, "test", info.Tag)
	assert.True(t, info.IsExported)
	assert.False(t, info.IsExtensions)
}

func TestCachedFieldMaps_Success(t *testing.T) {
	t.Parallel()

	// Test CachedFieldMaps struct creation
	maps := CachedFieldMaps{
		Fields: map[string]CachedFieldInfo{
			"test": {
				Name:     "TestField",
				Index:    0,
				Required: true,
				Tag:      "test",
			},
		},
		ExtensionIndex: -1,
		HasExtensions:  false,
		FieldIndexes: map[string]int{
			"test": 0,
		},
		RequiredFields: map[string]bool{
			"test": true,
		},
	}

	assert.NotEmpty(t, maps.Fields)
	assert.Contains(t, maps.Fields, "test")
	assert.Equal(t, -1, maps.ExtensionIndex)
	assert.False(t, maps.HasExtensions)
	assert.NotEmpty(t, maps.FieldIndexes)
	assert.NotEmpty(t, maps.RequiredFields)
}

func TestFieldCacheStats_Success(t *testing.T) {
	t.Parallel()

	// Test FieldCacheStats struct
	stats := FieldCacheStats{
		Size: 42,
	}

	assert.Equal(t, int64(42), stats.Size)
}

func TestRegisterType_PointerType_Success(t *testing.T) {
	t.Parallel()

	// Test registering a pointer type
	type TestPointerType struct {
		Value string
	}

	// Register with pointer type
	RegisterType(func() *TestPointerType {
		return &TestPointerType{Value: "pointer-test"}
	})

	// Should be registered for the element type
	elemType := reflect.TypeOf(TestPointerType{})
	assert.True(t, IsRegistered(elemType))

	// Should also work with pointer type
	ptrType := reflect.TypeOf((*TestPointerType)(nil))
	assert.True(t, IsRegistered(ptrType))
}

func TestTypeFactory_Success(t *testing.T) {
	t.Parallel()

	// Test TypeFactory function type
	factory := TypeFactory(func() interface{} {
		return &struct{ Name string }{Name: "test"}
	})

	result := factory()
	assert.NotNil(t, result)

	// Verify the result is the expected type
	structPtr, ok := result.(*struct{ Name string })
	require.True(t, ok)
	assert.Equal(t, "test", structPtr.Name)
}
