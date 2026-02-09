package openapi

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReportElapsed_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		action   string
		elapsed  time.Duration
		expected string
	}{
		{
			name:     "rounds to milliseconds",
			action:   "Linting",
			elapsed:  1250 * time.Microsecond,
			expected: "Linting completed in 1ms\n",
		},
		{
			name:     "uses minimum of one millisecond",
			action:   "Validation",
			elapsed:  300 * time.Microsecond,
			expected: "Validation completed in 1ms\n",
		},
		{
			name:     "supports second-scale durations",
			action:   "Linting",
			elapsed:  1234 * time.Millisecond,
			expected: "Linting completed in 1.234s\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			reportElapsed(&buf, tt.action, tt.elapsed)

			assert.Equal(t, tt.expected, buf.String(), "should render elapsed output")
		})
	}
}
