package sequencedmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllOrdered_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		setup      func() *Map[string, int]
		order      OrderType
		expected   []string
		expectVals []int
	}{
		{
			name: "OrderAdded with string keys",
			setup: func() *Map[string, int] {
				m := New[string, int]()
				m.Set("charlie", 3)
				m.Set("alpha", 1)
				m.Set("beta", 2)
				return m
			},
			order:      OrderAdded,
			expected:   []string{"charlie", "alpha", "beta"},
			expectVals: []int{3, 1, 2},
		},
		{
			name: "OrderAddedReverse with string keys",
			setup: func() *Map[string, int] {
				m := New[string, int]()
				m.Set("charlie", 3)
				m.Set("alpha", 1)
				m.Set("beta", 2)
				return m
			},
			order:      OrderAddedReverse,
			expected:   []string{"beta", "alpha", "charlie"},
			expectVals: []int{2, 1, 3},
		},
		{
			name: "OrderKeyAsc with string keys",
			setup: func() *Map[string, int] {
				m := New[string, int]()
				m.Set("charlie", 3)
				m.Set("alpha", 1)
				m.Set("beta", 2)
				return m
			},
			order:      OrderKeyAsc,
			expected:   []string{"alpha", "beta", "charlie"},
			expectVals: []int{1, 2, 3},
		},
		{
			name: "OrderKeyDesc with string keys",
			setup: func() *Map[string, int] {
				m := New[string, int]()
				m.Set("charlie", 3)
				m.Set("alpha", 1)
				m.Set("beta", 2)
				return m
			},
			order:      OrderKeyDesc,
			expected:   []string{"charlie", "beta", "alpha"},
			expectVals: []int{3, 2, 1},
		},
		{
			name: "OrderKeyAsc with numeric keys",
			setup: func() *Map[string, int] {
				m := New[string, int]()
				m.Set("30", 30)
				m.Set("10", 10)
				m.Set("20", 20)
				return m
			},
			order:      OrderKeyAsc,
			expected:   []string{"10", "20", "30"},
			expectVals: []int{10, 20, 30},
		},
		{
			name: "Empty map with OrderAdded",
			setup: func() *Map[string, int] {
				return New[string, int]()
			},
			order:      OrderAdded,
			expected:   nil,
			expectVals: nil,
		},
		{
			name: "Single element with OrderKeyDesc",
			setup: func() *Map[string, int] {
				m := New[string, int]()
				m.Set("single", 42)
				return m
			},
			order:      OrderKeyDesc,
			expected:   []string{"single"},
			expectVals: []int{42},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := tt.setup()

			var actualKeys []string
			var actualVals []int

			for k, v := range m.AllOrdered(tt.order) {
				actualKeys = append(actualKeys, k)
				actualVals = append(actualVals, v)
			}

			assert.Equal(t, tt.expected, actualKeys, "keys should match expected order")
			assert.Equal(t, tt.expectVals, actualVals, "values should match expected order")
			assert.Equal(t, len(tt.expected), len(actualKeys), "length should match")
		})
	}
}

func TestAllOrdered_Error(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func() *Map[string, int]
		order OrderType
	}{
		{
			name: "nil map with OrderAdded",
			setup: func() *Map[string, int] {
				return nil
			},
			order: OrderAdded,
		},
		{
			name: "nil map with OrderAddedReverse",
			setup: func() *Map[string, int] {
				return nil
			},
			order: OrderAddedReverse,
		},
		{
			name: "nil map with OrderKeyAsc",
			setup: func() *Map[string, int] {
				return nil
			},
			order: OrderKeyAsc,
		},
		{
			name: "nil map with OrderKeyDesc",
			setup: func() *Map[string, int] {
				return nil
			},
			order: OrderKeyDesc,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := tt.setup()

			count := 0
			for range m.AllOrdered(tt.order) {
				count++
			}

			assert.Equal(t, 0, count, "nil map should yield no elements")
		})
	}
}

func TestAllOrdered_IntegerKeys_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		order      OrderType
		expected   []int
		expectVals []string
	}{
		{
			name:       "OrderKeyAsc with integer keys",
			order:      OrderKeyAsc,
			expected:   []int{10, 20, 30}, // String sort: "10", "20", "30"
			expectVals: []string{"ten", "twenty", "thirty"},
		},
		{
			name:       "OrderKeyDesc with integer keys",
			order:      OrderKeyDesc,
			expected:   []int{30, 20, 10}, // String sort desc: "30", "20", "10"
			expectVals: []string{"thirty", "twenty", "ten"},
		},
		{
			name:       "OrderAdded with integer keys",
			order:      OrderAdded,
			expected:   []int{30, 10, 20}, // Insertion order
			expectVals: []string{"thirty", "ten", "twenty"},
		},
		{
			name:       "OrderAddedReverse with integer keys",
			order:      OrderAddedReverse,
			expected:   []int{20, 10, 30}, // Reverse insertion order
			expectVals: []string{"twenty", "ten", "thirty"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := New[int, string]()
			m.Set(30, "thirty")
			m.Set(10, "ten")
			m.Set(20, "twenty")

			var actualKeys []int
			var actualVals []string

			for k, v := range m.AllOrdered(tt.order) {
				actualKeys = append(actualKeys, k)
				actualVals = append(actualVals, v)
			}

			assert.Equal(t, tt.expected, actualKeys, "keys should match expected order")
			assert.Equal(t, tt.expectVals, actualVals, "values should match expected order")
		})
	}
}

func TestAllOrdered_EarlyExit_Success(t *testing.T) {
	t.Parallel()
	m := New[string, int]()
	m.Set("alpha", 1)
	m.Set("beta", 2)
	m.Set("gamma", 3)

	tests := []struct {
		name         string
		order        OrderType
		stopAfter    int
		expectedKeys []string
	}{
		{
			name:         "Early exit after 1 element OrderAdded",
			order:        OrderAdded,
			stopAfter:    1,
			expectedKeys: []string{"alpha"},
		},
		{
			name:         "Early exit after 2 elements OrderKeyAsc",
			order:        OrderKeyAsc,
			stopAfter:    2,
			expectedKeys: []string{"alpha", "beta"},
		},
		{
			name:         "Early exit after 1 element OrderAddedReverse",
			order:        OrderAddedReverse,
			stopAfter:    1,
			expectedKeys: []string{"gamma"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var actualKeys []string
			count := 0

			for k := range m.AllOrdered(tt.order) {
				actualKeys = append(actualKeys, k)
				count++
				if count >= tt.stopAfter {
					break
				}
			}

			assert.Equal(t, tt.expectedKeys, actualKeys, "keys should match expected with early exit")
			assert.Equal(t, tt.stopAfter, len(actualKeys), "should stop after specified count")
		})
	}
}

func TestAllOrdered_CompareWithAll_Success(t *testing.T) {
	t.Parallel()
	m := New[string, int]()
	m.Set("charlie", 3)
	m.Set("alpha", 1)
	m.Set("beta", 2)

	t.Run("OrderAdded should match All() behavior", func(t *testing.T) {
		var allKeys []string
		var allVals []int
		for k, v := range m.All() {
			allKeys = append(allKeys, k)
			allVals = append(allVals, v)
		}

		var orderedKeys []string
		var orderedVals []int
		for k, v := range m.AllOrdered(OrderAdded) {
			orderedKeys = append(orderedKeys, k)
			orderedVals = append(orderedVals, v)
		}

		assert.Equal(t, allKeys, orderedKeys, "AllOrdered(OrderAdded) should match All()")
		assert.Equal(t, allVals, orderedVals, "values should also match")
	})
}
