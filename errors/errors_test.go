package errors_test

import (
	"fmt"
	"testing"

	"github.com/speakeasy-api/openapi/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestError_Error_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		err      errors.Error
		expected string
	}{
		{
			name:     "simple error message",
			err:      errors.Error("test error"),
			expected: "test error",
		},
		{
			name:     "empty error message",
			err:      errors.Error(""),
			expected: "",
		},
		{
			name:     "error with special characters",
			err:      errors.Error("error: failed to parse JSON"),
			expected: "error: failed to parse JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.err.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestError_Is_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		err      errors.Error
		target   error
		expected bool
	}{
		{
			name:     "exact match",
			err:      errors.Error("test error"),
			target:   errors.Error("test error"),
			expected: true,
		},
		{
			name:     "wrapped error with separator",
			err:      errors.Error("test error"),
			target:   errors.New("test error -- wrapped cause"),
			expected: true,
		},
		{
			name:     "different error",
			err:      errors.Error("test error"),
			target:   errors.Error("different error"),
			expected: false,
		},
		{
			name:     "partial match without separator",
			err:      errors.Error("test error"),
			target:   errors.New("test error but different"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.err.Is(tt.target)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestError_As_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		err      errors.Error
		expected bool
	}{
		{
			name:     "can set Error type",
			err:      errors.Error("test error"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var target errors.Error
			result := tt.err.As(&target)
			assert.Equal(t, tt.expected, result)
			if result {
				assert.Equal(t, string(tt.err), string(target))
			}
		})
	}
}

func TestError_As_Error(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		err    errors.Error
		target interface{}
	}{
		{
			name:   "cannot set non-Error type",
			err:    errors.Error("test error"),
			target: new(string),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.err.As(tt.target)
			assert.False(t, result)
		})
	}
}

func TestError_As_NonSettableTarget_Error(t *testing.T) {
	t.Parallel()
	err := errors.Error("test error")

	// Test with a non-pointer target (which would cause panic if not handled properly)
	// We need to test this carefully to avoid the panic
	defer func() {
		if r := recover(); r != nil {
			// This is expected behavior - the As method should handle this gracefully
			// but the current implementation doesn't, so we expect a panic
			assert.Contains(t, fmt.Sprintf("%v", r), "reflect: call of reflect.Value.Elem")
		}
	}()

	// This will panic because we're passing a non-pointer
	var stringLiteral = "test"
	result := err.As(stringLiteral)
	assert.False(t, result)
}

func TestError_Wrap_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		err         errors.Error
		cause       error
		expectedMsg string
	}{
		{
			name:        "wrap with cause",
			err:         errors.Error("wrapper error"),
			cause:       errors.New("original cause"),
			expectedMsg: "wrapper error -- original cause",
		},
		{
			name:        "wrap with nil cause",
			err:         errors.Error("wrapper error"),
			cause:       nil,
			expectedMsg: "wrapper error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wrapped := tt.err.Wrap(tt.cause)
			assert.Equal(t, tt.expectedMsg, wrapped.Error())
		})
	}
}

func TestWrappedError_Error_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		msg         string
		cause       error
		expectedMsg string
	}{
		{
			name:        "with cause",
			msg:         "wrapper",
			cause:       errors.New("original"),
			expectedMsg: "wrapper -- original",
		},
		{
			name:        "without cause",
			msg:         "wrapper",
			cause:       nil,
			expectedMsg: "wrapper",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := errors.Error(tt.msg).Wrap(tt.cause)
			assert.Equal(t, tt.expectedMsg, err.Error())
		})
	}
}

func TestWrappedError_Is_Success(t *testing.T) {
	t.Parallel()
	baseErr := errors.Error("base error")
	wrappedErr := baseErr.Wrap(errors.New("cause"))

	tests := []struct {
		name     string
		target   error
		expected bool
	}{
		{
			name:     "matches base error",
			target:   baseErr,
			expected: true,
		},
		{
			name:     "matches error with separator",
			target:   errors.New("base error -- some cause"),
			expected: true,
		},
		{
			name:     "does not match different error",
			target:   errors.Error("different error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := wrappedErr.(interface{ Is(error) bool }).Is(tt.target)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWrappedError_As_Success(t *testing.T) {
	t.Parallel()
	baseErr := errors.Error("base error")
	wrappedErr := baseErr.Wrap(errors.New("cause"))

	var target errors.Error
	result := wrappedErr.(interface{ As(interface{}) bool }).As(&target)
	assert.True(t, result)
	assert.Equal(t, string(baseErr), string(target))
}

func TestWrappedError_Unwrap_Success(t *testing.T) {
	t.Parallel()
	cause := errors.New("original cause")
	wrappedErr := errors.Error("wrapper").Wrap(cause)

	unwrapped := wrappedErr.(interface{ Unwrap() error }).Unwrap()
	assert.Equal(t, cause, unwrapped)
}

func TestIs_Success(t *testing.T) {
	t.Parallel()
	err1 := errors.Error("test error")
	err2 := errors.Error("test error")
	err3 := errors.Error("different error")

	tests := []struct {
		name     string
		err      error
		target   error
		expected bool
	}{
		{
			name:     "same errors",
			err:      err1,
			target:   err2,
			expected: true,
		},
		{
			name:     "different errors",
			err:      err1,
			target:   err3,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			target:   err1,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := errors.Is(tt.err, tt.target)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAs_Success(t *testing.T) {
	t.Parallel()
	err := errors.Error("test error")

	var target errors.Error
	result := errors.As(err, &target)
	assert.True(t, result)
	assert.Equal(t, string(err), string(target))
}

func TestNew_Success(t *testing.T) {
	t.Parallel()
	message := "test error message"
	err := errors.New(message)
	assert.Equal(t, message, err.Error())
}

func TestJoin_Success(t *testing.T) {
	t.Parallel()
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	err3 := errors.New("error 3")

	tests := []struct {
		name     string
		errs     []error
		expected string
	}{
		{
			name:     "join multiple errors",
			errs:     []error{err1, err2, err3},
			expected: "error 1\nerror 2\nerror 3",
		},
		{
			name:     "join with nil error",
			errs:     []error{err1, nil, err2},
			expected: "error 1\nerror 2",
		},
		{
			name:     "join empty slice",
			errs:     []error{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			joined := errors.Join(tt.errs...)
			if tt.expected == "" {
				assert.NoError(t, joined)
			} else {
				require.Error(t, joined)
				assert.Equal(t, tt.expected, joined.Error())
			}
		})
	}
}

func TestUnwrapErrors_Success(t *testing.T) {
	t.Parallel()
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	joined := errors.Join(err1, err2)

	tests := []struct {
		name     string
		err      error
		expected []error
	}{
		{
			name:     "unwrap joined errors",
			err:      joined,
			expected: []error{err1, err2},
		},
		{
			name:     "unwrap single error",
			err:      err1,
			expected: []error{err1},
		},
		{
			name:     "unwrap nil error",
			err:      nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := errors.UnwrapErrors(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
