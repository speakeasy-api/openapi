package values

import (
	"testing"

	"github.com/speakeasy-api/openapi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CustomType is a test type that implements IsEqual method
type CustomType struct {
	Value string
}

// IsEqual implements custom equality logic for CustomType
func (c *CustomType) IsEqual(other *CustomType) bool {
	if c == nil && other == nil {
		return true
	}
	if c == nil || other == nil {
		return false
	}
	return c.Value == other.Value
}

// Test the IsLeft() method for nil safety and functionality
func TestEitherValue_IsLeft_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		either   *EitherValue[string, string, int, int]
		expected bool
	}{
		{
			name:     "nil EitherValue returns false",
			either:   nil,
			expected: false,
		},
		{
			name: "Left value set returns true",
			either: &EitherValue[string, string, int, int]{
				Left: pointer.From("test"),
			},
			expected: true,
		},
		{
			name: "Right value set returns false",
			either: &EitherValue[string, string, int, int]{
				Right: pointer.From(42),
			},
			expected: false,
		},
		{
			name: "Neither value set returns true (fallback to Left)",
			either: &EitherValue[string, string, int, int]{
				Left:  nil,
				Right: nil,
			},
			expected: true,
		},
		{
			name: "Both values set returns true (Left takes precedence)",
			either: &EitherValue[string, string, int, int]{
				Left:  pointer.From("test"),
				Right: pointer.From(42),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.either.IsLeft()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test the GetLeft() method for nil safety and functionality (returns pointer)
func TestEitherValue_GetLeft_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		either   *EitherValue[string, string, int, int]
		expected *string
	}{
		{
			name: "Left value set returns pointer to value",
			either: &EitherValue[string, string, int, int]{
				Left: pointer.From("test value"),
			},
			expected: pointer.From("test value"),
		},
		{
			name: "Left value nil returns nil",
			either: &EitherValue[string, string, int, int]{
				Left: nil,
			},
			expected: nil,
		},
		{
			name: "Right value set but Left nil returns nil",
			either: &EitherValue[string, string, int, int]{
				Left:  nil,
				Right: pointer.From(42),
			},
			expected: nil,
		},
		{
			name: "Empty string Left value returns pointer to empty string",
			either: &EitherValue[string, string, int, int]{
				Left: pointer.From(""),
			},
			expected: pointer.From(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.either.GetLeft()
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

// Test GetLeft() with nil EitherValue for nil safety
func TestEitherValue_GetLeft_NilSafety(t *testing.T) {
	t.Parallel()

	var either *EitherValue[string, string, int, int]

	// This should not panic even with nil EitherValue
	assert.NotPanics(t, func() {
		result := either.GetLeft()
		assert.Nil(t, result) // Should return nil for nil EitherValue
	})
}

// Test the LeftValue() method for nil safety and functionality (returns value)
func TestEitherValue_LeftValue_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		either   *EitherValue[string, string, int, int]
		expected string
	}{
		{
			name: "Left value set returns value",
			either: &EitherValue[string, string, int, int]{
				Left: pointer.From("test value"),
			},
			expected: "test value",
		},
		{
			name: "Left value nil returns zero value",
			either: &EitherValue[string, string, int, int]{
				Left: nil,
			},
			expected: "",
		},
		{
			name: "Right value set but Left nil returns zero value",
			either: &EitherValue[string, string, int, int]{
				Left:  nil,
				Right: pointer.From(42),
			},
			expected: "",
		},
		{
			name: "Empty string Left value returns empty string",
			either: &EitherValue[string, string, int, int]{
				Left: pointer.From(""),
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.either.LeftValue()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test LeftValue() with nil EitherValue for nil safety
func TestEitherValue_LeftValue_NilSafety(t *testing.T) {
	t.Parallel()

	var either *EitherValue[string, string, int, int]

	// This should not panic even with nil EitherValue
	assert.NotPanics(t, func() {
		result := either.LeftValue()
		assert.Empty(t, result) // Should return zero value for string
	})
}

// Test the IsRight() method for nil safety and functionality
func TestEitherValue_IsRight_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		either   *EitherValue[string, string, int, int]
		expected bool
	}{
		{
			name:     "nil EitherValue returns false",
			either:   nil,
			expected: false,
		},
		{
			name: "Right value set returns true",
			either: &EitherValue[string, string, int, int]{
				Right: pointer.From(42),
			},
			expected: true,
		},
		{
			name: "Left value set returns false",
			either: &EitherValue[string, string, int, int]{
				Left: pointer.From("test"),
			},
			expected: false,
		},
		{
			name: "Neither value set returns true (fallback to Right)",
			either: &EitherValue[string, string, int, int]{
				Left:  nil,
				Right: nil,
			},
			expected: true,
		},
		{
			name: "Both values set returns true (both are valid)",
			either: &EitherValue[string, string, int, int]{
				Left:  pointer.From("test"),
				Right: pointer.From(42),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.either.IsRight()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test the GetRight() method for nil safety and functionality (returns pointer)
func TestEitherValue_GetRight_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		either   *EitherValue[string, string, int, int]
		expected *int
	}{
		{
			name: "Right value set returns pointer to value",
			either: &EitherValue[string, string, int, int]{
				Right: pointer.From(42),
			},
			expected: pointer.From(42),
		},
		{
			name: "Right value nil returns nil",
			either: &EitherValue[string, string, int, int]{
				Right: nil,
			},
			expected: nil,
		},
		{
			name: "Left value set but Right nil returns nil",
			either: &EitherValue[string, string, int, int]{
				Left:  pointer.From("test"),
				Right: nil,
			},
			expected: nil,
		},
		{
			name: "Zero value Right returns pointer to zero",
			either: &EitherValue[string, string, int, int]{
				Right: pointer.From(0),
			},
			expected: pointer.From(0),
		},
		{
			name: "Negative Right value returns pointer to negative value",
			either: &EitherValue[string, string, int, int]{
				Right: pointer.From(-10),
			},
			expected: pointer.From(-10),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.either.GetRight()
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

// Test GetRight() with nil EitherValue for nil safety
func TestEitherValue_GetRight_NilSafety(t *testing.T) {
	t.Parallel()

	var either *EitherValue[string, string, int, int]

	// This should not panic even with nil EitherValue
	assert.NotPanics(t, func() {
		result := either.GetRight()
		assert.Nil(t, result) // Should return nil for nil EitherValue
	})
}

// Test the RightValue() method for nil safety and functionality (returns value)
func TestEitherValue_RightValue_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		either   *EitherValue[string, string, int, int]
		expected int
	}{
		{
			name: "Right value set returns value",
			either: &EitherValue[string, string, int, int]{
				Right: pointer.From(42),
			},
			expected: 42,
		},
		{
			name: "Right value nil returns zero value",
			either: &EitherValue[string, string, int, int]{
				Right: nil,
			},
			expected: 0,
		},
		{
			name: "Left value set but Right nil returns zero value",
			either: &EitherValue[string, string, int, int]{
				Left:  pointer.From("test"),
				Right: nil,
			},
			expected: 0,
		},
		{
			name: "Zero value Right returns zero",
			either: &EitherValue[string, string, int, int]{
				Right: pointer.From(0),
			},
			expected: 0,
		},
		{
			name: "Negative Right value returns negative value",
			either: &EitherValue[string, string, int, int]{
				Right: pointer.From(-10),
			},
			expected: -10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.either.RightValue()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test RightValue() with nil EitherValue for nil safety
func TestEitherValue_RightValue_NilSafety(t *testing.T) {
	t.Parallel()

	var either *EitherValue[string, string, int, int]

	// This should not panic even with nil EitherValue
	assert.NotPanics(t, func() {
		result := either.RightValue()
		assert.Equal(t, 0, result) // Should return zero value for int
	})
}

// Test logical consistency between IsLeft() and IsRight()
func TestEitherValue_LogicalConsistency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		either *EitherValue[string, string, int, int]
	}{
		{
			name:   "nil EitherValue",
			either: nil,
		},
		{
			name: "Left value only",
			either: &EitherValue[string, string, int, int]{
				Left: pointer.From("test"),
			},
		},
		{
			name: "Right value only",
			either: &EitherValue[string, string, int, int]{
				Right: pointer.From(42),
			},
		},
		{
			name: "Neither value set",
			either: &EitherValue[string, string, int, int]{
				Left:  nil,
				Right: nil,
			},
		},
		{
			name: "Both values set",
			either: &EitherValue[string, string, int, int]{
				Left:  pointer.From("test"),
				Right: pointer.From(42),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			isLeft := tt.either.IsLeft()
			isRight := tt.either.IsRight()

			// When neither value is set, both should be true (fallback behavior)
			if tt.either == nil {
				assert.False(t, isLeft, "IsLeft() should return false when EitherValue is nil")
				assert.False(t, isRight, "IsRight() should return false when EitherValue is nil")
			} else if tt.either.Left == nil && tt.either.Right == nil {
				assert.True(t, isLeft, "IsLeft() should return true when no values are set (fallback to Left)")
				assert.True(t, isRight, "IsRight() should return true when no values are set (fallback to Right)")
			}

			// When both values are set, both should return true
			if tt.either != nil && tt.either.Left != nil && tt.either.Right != nil {
				assert.True(t, isLeft, "IsLeft() should return true when Left is set")
				assert.True(t, isRight, "IsRight() should return true when Right is set")
			}

			// When only Left is set
			if tt.either != nil && tt.either.Left != nil && tt.either.Right == nil {
				assert.True(t, isLeft, "IsLeft() should return true when only Left is set")
				assert.False(t, isRight, "IsRight() should return false when only Left is set")
			}

			// When only Right is set
			if tt.either != nil && tt.either.Left == nil && tt.either.Right != nil {
				assert.False(t, isLeft, "IsLeft() should return false when only Right is set")
				assert.True(t, isRight, "IsRight() should return true when only Right is set")
			}
		})
	}
}

// Test with different types to ensure generics work properly
func TestEitherValue_DifferentTypes_Success(t *testing.T) {
	t.Parallel()

	t.Run("bool and float64", func(t *testing.T) {
		t.Parallel()
		either := &EitherValue[bool, bool, float64, float64]{
			Left: pointer.From(true),
		}

		assert.True(t, either.IsLeft())
		assert.False(t, either.IsRight())
		assert.True(t, either.LeftValue())
		assert.Nil(t, either.GetRight())                   // GetRight now returns pointer, so nil when not set
		assert.InDelta(t, 0.0, either.RightValue(), 0.001) // RightValue returns zero value
	})

	t.Run("slice and map", func(t *testing.T) {
		t.Parallel()

		either := &EitherValue[[]string, []string, map[string]int, map[string]int]{
			Right: &map[string]int{"key": 42},
		}

		assert.False(t, either.IsLeft())
		assert.True(t, either.IsRight())
		assert.Nil(t, either.GetLeft())                                 // GetLeft returns nil when not set
		assert.NotNil(t, either.GetRight())                             // GetRight returns pointer to map
		assert.Equal(t, map[string]int{"key": 42}, *either.GetRight())  // Dereference pointer
		assert.Equal(t, map[string]int{"key": 42}, either.RightValue()) // RightValue returns value directly
	})
}

// Test edge cases and boundary conditions
func TestEitherValue_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("empty struct types", func(t *testing.T) {
		t.Parallel()
		type EmptyStruct struct{}

		either := &EitherValue[EmptyStruct, EmptyStruct, EmptyStruct, EmptyStruct]{
			Left: &EmptyStruct{},
		}

		assert.True(t, either.IsLeft())
		assert.NotNil(t, either.GetLeft())                 // GetLeft returns pointer
		assert.Equal(t, EmptyStruct{}, *either.GetLeft())  // Dereference pointer
		assert.Equal(t, EmptyStruct{}, either.LeftValue()) // LeftValue returns value directly
	})

	t.Run("interface types", func(t *testing.T) {
		t.Parallel()

		either := &EitherValue[interface{}, interface{}, string, string]{
			Right: pointer.From("test"),
		}

		assert.False(t, either.IsLeft())
		assert.True(t, either.IsRight())
		assert.Nil(t, either.GetLeft())              // GetLeft returns nil when not set
		assert.NotNil(t, either.GetRight())          // GetRight returns pointer to string
		assert.Equal(t, "test", *either.GetRight())  // Dereference pointer
		assert.Equal(t, "test", either.RightValue()) // RightValue returns value directly
	})
}

// Test the IsEqual() method for comprehensive equality checking
func TestEitherValue_IsEqual_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		either1  *EitherValue[string, string, int, int]
		either2  *EitherValue[string, string, int, int]
		expected bool
	}{
		{
			name:     "both nil should be equal",
			either1:  nil,
			either2:  nil,
			expected: true,
		},
		{
			name:     "one nil should not be equal",
			either1:  nil,
			either2:  &EitherValue[string, string, int, int]{Left: pointer.From("test")},
			expected: false,
		},
		{
			name:     "nil vs non-nil should not be equal",
			either1:  &EitherValue[string, string, int, int]{Left: pointer.From("test")},
			either2:  nil,
			expected: false,
		},
		{
			name: "same left values should be equal",
			either1: &EitherValue[string, string, int, int]{
				Left: pointer.From("test"),
			},
			either2: &EitherValue[string, string, int, int]{
				Left: pointer.From("test"),
			},
			expected: true,
		},
		{
			name: "different left values should not be equal",
			either1: &EitherValue[string, string, int, int]{
				Left: pointer.From("test1"),
			},
			either2: &EitherValue[string, string, int, int]{
				Left: pointer.From("test2"),
			},
			expected: false,
		},
		{
			name: "same right values should be equal",
			either1: &EitherValue[string, string, int, int]{
				Right: pointer.From(42),
			},
			either2: &EitherValue[string, string, int, int]{
				Right: pointer.From(42),
			},
			expected: true,
		},
		{
			name: "different right values should not be equal",
			either1: &EitherValue[string, string, int, int]{
				Right: pointer.From(42),
			},
			either2: &EitherValue[string, string, int, int]{
				Right: pointer.From(24),
			},
			expected: false,
		},
		{
			name: "left vs right should not be equal",
			either1: &EitherValue[string, string, int, int]{
				Left: pointer.From("test"),
			},
			either2: &EitherValue[string, string, int, int]{
				Right: pointer.From(42),
			},
			expected: false,
		},
		{
			name: "both empty should be equal",
			either1: &EitherValue[string, string, int, int]{
				Left:  nil,
				Right: nil,
			},
			either2: &EitherValue[string, string, int, int]{
				Left:  nil,
				Right: nil,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.either1.IsEqual(tt.either2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test IsEqual with types that have IsEqual methods
func TestEitherValue_IsEqual_WithIsEqualMethod_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		either1  *EitherValue[CustomType, CustomType, int, int]
		either2  *EitherValue[CustomType, CustomType, int, int]
		expected bool
	}{
		{
			name: "custom types with same values should be equal",
			either1: &EitherValue[CustomType, CustomType, int, int]{
				Left: &CustomType{Value: "test"},
			},
			either2: &EitherValue[CustomType, CustomType, int, int]{
				Left: &CustomType{Value: "test"},
			},
			expected: true,
		},
		{
			name: "custom types with different values should not be equal",
			either1: &EitherValue[CustomType, CustomType, int, int]{
				Left: &CustomType{Value: "test1"},
			},
			either2: &EitherValue[CustomType, CustomType, int, int]{
				Left: &CustomType{Value: "test2"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.either1.IsEqual(tt.either2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test IsEqual with slice and map types (empty collection handling)
func TestEitherValue_IsEqual_EmptyCollections_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		either1  *EitherValue[[]string, []string, map[string]int, map[string]int]
		either2  *EitherValue[[]string, []string, map[string]int, map[string]int]
		expected bool
	}{
		{
			name: "nil slice vs empty slice should be equal",
			either1: &EitherValue[[]string, []string, map[string]int, map[string]int]{
				Left: nil,
			},
			either2: &EitherValue[[]string, []string, map[string]int, map[string]int]{
				Left: &[]string{},
			},
			expected: true,
		},
		{
			name: "empty slice vs nil slice should be equal",
			either1: &EitherValue[[]string, []string, map[string]int, map[string]int]{
				Left: &[]string{},
			},
			either2: &EitherValue[[]string, []string, map[string]int, map[string]int]{
				Left: nil,
			},
			expected: true,
		},
		{
			name: "nil map vs empty map should not be equal (reflect.DeepEqual behavior)",
			either1: &EitherValue[[]string, []string, map[string]int, map[string]int]{
				Right: nil,
			},
			either2: &EitherValue[[]string, []string, map[string]int, map[string]int]{
				Right: &map[string]int{},
			},
			expected: false,
		},
		{
			name: "empty map vs nil map should not be equal (reflect.DeepEqual behavior)",
			either1: &EitherValue[[]string, []string, map[string]int, map[string]int]{
				Right: &map[string]int{},
			},
			either2: &EitherValue[[]string, []string, map[string]int, map[string]int]{
				Right: nil,
			},
			expected: false,
		},
		{
			name: "non-empty slice vs empty slice should not be equal",
			either1: &EitherValue[[]string, []string, map[string]int, map[string]int]{
				Left: &[]string{"test"},
			},
			either2: &EitherValue[[]string, []string, map[string]int, map[string]int]{
				Left: &[]string{},
			},
			expected: false,
		},
		{
			name: "non-empty map vs empty map should not be equal",
			either1: &EitherValue[[]string, []string, map[string]int, map[string]int]{
				Right: &map[string]int{"key": 42},
			},
			either2: &EitherValue[[]string, []string, map[string]int, map[string]int]{
				Right: &map[string]int{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.either1.IsEqual(tt.either2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test the equalWithIsEqualMethod function indirectly through IsEqual
func TestEqualWithIsEqualMethod_Success(t *testing.T) {
	t.Parallel()

	// Test with types that don't have IsEqual method (fallback to reflect.DeepEqual)
	t.Run("fallback to reflect.DeepEqual", func(t *testing.T) {
		t.Parallel()

		either1 := &EitherValue[string, string, int, int]{
			Left: pointer.From("test"),
		}
		either2 := &EitherValue[string, string, int, int]{
			Left: pointer.From("test"),
		}

		assert.True(t, either1.IsEqual(either2))
	})

	// Test with nil values
	t.Run("both nil values", func(t *testing.T) {
		t.Parallel()

		either1 := &EitherValue[string, string, int, int]{
			Left: nil,
		}
		either2 := &EitherValue[string, string, int, int]{
			Left: nil,
		}

		assert.True(t, either1.IsEqual(either2))
	})

	// Test with one nil value
	t.Run("one nil value", func(t *testing.T) {
		t.Parallel()

		either1 := &EitherValue[string, string, int, int]{
			Left: nil,
		}
		either2 := &EitherValue[string, string, int, int]{
			Left: pointer.From("test"),
		}

		assert.False(t, either1.IsEqual(either2))
	})
}

// Test the isEmptyCollection function indirectly through IsEqual
func TestIsEmptyCollection_Success(t *testing.T) {
	t.Parallel()

	// Test with slice types
	t.Run("slice types", func(t *testing.T) {
		t.Parallel()

		// Test nil slice vs empty slice
		either1 := &EitherValue[[]string, []string, int, int]{
			Left: nil, // nil slice
		}
		either2 := &EitherValue[[]string, []string, int, int]{
			Left: &[]string{}, // empty slice
		}

		assert.True(t, either1.IsEqual(either2), "nil slice should equal empty slice")

		// Test non-empty slice vs empty slice
		either3 := &EitherValue[[]string, []string, int, int]{
			Left: &[]string{"test"}, // non-empty slice
		}
		either4 := &EitherValue[[]string, []string, int, int]{
			Left: &[]string{}, // empty slice
		}

		assert.False(t, either3.IsEqual(either4), "non-empty slice should not equal empty slice")
	})

	// Test with map types
	t.Run("map types", func(t *testing.T) {
		t.Parallel()

		// Test nil map vs empty map
		either1 := &EitherValue[map[string]int, map[string]int, int, int]{
			Left: nil, // nil map
		}
		either2 := &EitherValue[map[string]int, map[string]int, int, int]{
			Left: &map[string]int{}, // empty map
		}

		assert.True(t, either1.IsEqual(either2), "nil map should equal empty map")

		// Test non-empty map vs empty map
		either3 := &EitherValue[map[string]int, map[string]int, int, int]{
			Left: &map[string]int{"key": 42}, // non-empty map
		}
		either4 := &EitherValue[map[string]int, map[string]int, int, int]{
			Left: &map[string]int{}, // empty map
		}

		assert.False(t, either3.IsEqual(either4), "non-empty map should not equal empty map")
	})
}

// Test GetNavigableNode method
func TestEitherValue_GetNavigableNode_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		either      *EitherValue[string, string, int, int]
		expectedVal interface{}
		expectError bool
	}{
		{
			name: "left value set returns left",
			either: &EitherValue[string, string, int, int]{
				Left: pointer.From("test"),
			},
			expectedVal: pointer.From("test"),
			expectError: false,
		},
		{
			name: "right value set returns right",
			either: &EitherValue[string, string, int, int]{
				Right: pointer.From(42),
			},
			expectedVal: pointer.From(42),
			expectError: false,
		},
		{
			name: "both values set returns left (precedence)",
			either: &EitherValue[string, string, int, int]{
				Left:  pointer.From("test"),
				Right: pointer.From(42),
			},
			expectedVal: pointer.From("test"),
			expectError: false,
		},
		{
			name: "no values set returns error",
			either: &EitherValue[string, string, int, int]{
				Left:  nil,
				Right: nil,
			},
			expectedVal: nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := tt.either.GetNavigableNode()

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "EitherValue has no value set")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedVal, result)
			}
		})
	}
}
