package validation

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types for options testing
type TestConfig struct {
	Name  string
	Value int
}

type AnotherConfig struct {
	Data string
}

// Test WithContextObject function
func TestWithContextObject_Success(t *testing.T) {
	t.Parallel()

	testConfig := &TestConfig{
		Name:  "test",
		Value: 42,
	}

	option := WithContextObject(testConfig)
	options := &Options{
		ContextObjects: make(map[reflect.Type]any),
	}

	option(options)

	expectedType := reflect.TypeOf((*TestConfig)(nil)).Elem()
	storedObj, exists := options.ContextObjects[expectedType]
	require.True(t, exists, "object should be stored in context")
	assert.Equal(t, testConfig, storedObj)
}

// Test NewOptions function
func TestNewOptions_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		opts     []Option
		validate func(t *testing.T, options *Options)
	}{
		{
			name: "no options",
			opts: nil,
			validate: func(t *testing.T, options *Options) {
				t.Helper()
				assert.Nil(t, options.ContextObjects, "map is lazy-initialized on first write")
			},
		},
		{
			name: "single option",
			opts: []Option{
				WithContextObject(&TestConfig{Name: "test", Value: 123}),
			},
			validate: func(t *testing.T, options *Options) {
				t.Helper()
				assert.NotNil(t, options.ContextObjects)
				assert.Len(t, options.ContextObjects, 1)

				expectedType := reflect.TypeOf((*TestConfig)(nil)).Elem()
				storedObj, exists := options.ContextObjects[expectedType]
				require.True(t, exists)

				config, ok := storedObj.(*TestConfig)
				require.True(t, ok)
				assert.Equal(t, "test", config.Name)
				assert.Equal(t, 123, config.Value)
			},
		},
		{
			name: "multiple options",
			opts: []Option{
				WithContextObject(&TestConfig{Name: "test1", Value: 100}),
				WithContextObject(&AnotherConfig{Data: "data1"}),
			},
			validate: func(t *testing.T, options *Options) {
				t.Helper()
				assert.NotNil(t, options.ContextObjects)
				assert.Len(t, options.ContextObjects, 2)

				// Check TestConfig
				testConfigType := reflect.TypeOf((*TestConfig)(nil)).Elem()
				storedTestObj, exists := options.ContextObjects[testConfigType]
				require.True(t, exists)
				testConfig, ok := storedTestObj.(*TestConfig)
				require.True(t, ok)
				assert.Equal(t, "test1", testConfig.Name)
				assert.Equal(t, 100, testConfig.Value)

				// Check AnotherConfig
				anotherConfigType := reflect.TypeOf((*AnotherConfig)(nil)).Elem()
				storedAnotherObj, exists := options.ContextObjects[anotherConfigType]
				require.True(t, exists)
				anotherConfig, ok := storedAnotherObj.(*AnotherConfig)
				require.True(t, ok)
				assert.Equal(t, "data1", anotherConfig.Data)
			},
		},
		{
			name: "overwrite same type",
			opts: []Option{
				WithContextObject(&TestConfig{Name: "first", Value: 1}),
				WithContextObject(&TestConfig{Name: "second", Value: 2}),
			},
			validate: func(t *testing.T, options *Options) {
				t.Helper()
				assert.NotNil(t, options.ContextObjects)
				assert.Len(t, options.ContextObjects, 1)

				expectedType := reflect.TypeOf((*TestConfig)(nil)).Elem()
				storedObj, exists := options.ContextObjects[expectedType]
				require.True(t, exists)

				config, ok := storedObj.(*TestConfig)
				require.True(t, ok)
				// Should have the second (last) value
				assert.Equal(t, "second", config.Name)
				assert.Equal(t, 2, config.Value)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			options := NewOptions(tt.opts...)
			tt.validate(t, options)
		})
	}
}

// Test GetContextObject function
func TestGetContextObject_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func() *Options
		validate func(t *testing.T, options *Options)
	}{
		{
			name: "get existing object",
			setup: func() *Options {
				return NewOptions(
					WithContextObject(&TestConfig{Name: "test", Value: 42}),
				)
			},
			validate: func(t *testing.T, options *Options) {
				t.Helper()
				result := GetContextObject[TestConfig](options)
				require.NotNil(t, result)
				assert.Equal(t, "test", result.Name)
				assert.Equal(t, 42, result.Value)
			},
		},
		{
			name: "get non-existing object",
			setup: func() *Options {
				return NewOptions(
					WithContextObject(&TestConfig{Name: "test", Value: 42}),
				)
			},
			validate: func(t *testing.T, options *Options) {
				t.Helper()
				result := GetContextObject[AnotherConfig](options)
				assert.Nil(t, result)
			},
		},
		{
			name: "get from nil context objects",
			setup: func() *Options {
				return &Options{
					ContextObjects: nil,
				}
			},
			validate: func(t *testing.T, options *Options) {
				t.Helper()
				result := GetContextObject[TestConfig](options)
				assert.Nil(t, result)
			},
		},
		{
			name: "get from empty context objects",
			setup: func() *Options {
				return &Options{
					ContextObjects: make(map[reflect.Type]any),
				}
			},
			validate: func(t *testing.T, options *Options) {
				t.Helper()
				result := GetContextObject[TestConfig](options)
				assert.Nil(t, result)
			},
		},
		{
			name: "get multiple different types",
			setup: func() *Options {
				return NewOptions(
					WithContextObject(&TestConfig{Name: "test", Value: 100}),
					WithContextObject(&AnotherConfig{Data: "some data"}),
				)
			},
			validate: func(t *testing.T, options *Options) {
				t.Helper()
				testResult := GetContextObject[TestConfig](options)
				require.NotNil(t, testResult)
				assert.Equal(t, "test", testResult.Name)
				assert.Equal(t, 100, testResult.Value)

				anotherResult := GetContextObject[AnotherConfig](options)
				require.NotNil(t, anotherResult)
				assert.Equal(t, "some data", anotherResult.Data)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			options := tt.setup()
			tt.validate(t, options)
		})
	}
}

// Test edge cases and type safety
func TestOptions_EdgeCases_Success(t *testing.T) {
	t.Parallel()

	t.Run("nil pointer handling", func(t *testing.T) {
		t.Parallel()

		var nilConfig *TestConfig
		option := WithContextObject(nilConfig)
		options := NewOptions(option)

		result := GetContextObject[TestConfig](options)
		assert.Nil(t, result) // Should return the nil pointer
	})

	t.Run("different pointer types to same struct", func(t *testing.T) {
		t.Parallel()

		config1 := &TestConfig{Name: "config1", Value: 1}
		config2 := &TestConfig{Name: "config2", Value: 2}

		options := NewOptions(
			WithContextObject(config1),
			WithContextObject(config2), // This should overwrite config1
		)

		result := GetContextObject[TestConfig](options)
		require.NotNil(t, result)
		assert.Equal(t, "config2", result.Name)
		assert.Equal(t, 2, result.Value)
	})
}

// Test with interface types
type TestInterface interface {
	GetName() string
}

type TestImplementation struct {
	name string
}

func (t *TestImplementation) GetName() string {
	return t.name
}

func TestOptions_WithInterfaces_Success(t *testing.T) {
	t.Parallel()

	impl := &TestImplementation{name: "test implementation"}

	options := NewOptions(
		WithContextObject(impl),
	)

	result := GetContextObject[TestImplementation](options)
	require.NotNil(t, result)
	assert.Equal(t, "test implementation", result.GetName())
}
