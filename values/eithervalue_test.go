package values

import (
	"testing"

	"github.com/speakeasy-api/openapi/pointer"
	"github.com/stretchr/testify/assert"
)

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
