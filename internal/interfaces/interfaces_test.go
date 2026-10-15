package interfaces

import (
	"context"
	"iter"
	"reflect"
	"testing"

	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Test implementations for Model interface
type testModelCore struct{}

type testModelImpl struct {
	core *testModelCore
}

func (t *testModelImpl) Validate(ctx context.Context, opts ...validation.Option) []error {
	return nil
}

func (t *testModelImpl) GetCore() *testModelCore {
	return t.core
}

// Test implementations for CoreModel interface
type testCoreModelImpl struct{}

func (t *testCoreModelImpl) Unmarshal(ctx context.Context, parentName string, node *yaml.Node) ([]error, error) {
	return nil, nil
}

// Test implementations for SequencedMapInterface
type testSequencedMapImpl struct {
	initialized bool
}

func (t *testSequencedMapImpl) Init() {
	t.initialized = true
}

func (t *testSequencedMapImpl) IsInitialized() bool {
	return t.initialized
}

func (t *testSequencedMapImpl) SetUntyped(key, value any) error {
	return nil
}

func (t *testSequencedMapImpl) AllUntyped() iter.Seq2[any, any] {
	return func(yield func(any, any) bool) {}
}

func (t *testSequencedMapImpl) GetKeyType() reflect.Type {
	return reflect.TypeOf("")
}

func (t *testSequencedMapImpl) GetValueType() reflect.Type {
	return reflect.TypeOf("")
}

func (t *testSequencedMapImpl) Len() int {
	return 0
}

func (t *testSequencedMapImpl) GetAny(key any) (any, bool) {
	return nil, false
}

func (t *testSequencedMapImpl) SetAny(key, value any) {}

func (t *testSequencedMapImpl) DeleteAny(key any) {}

func (t *testSequencedMapImpl) KeysAny() iter.Seq[any] {
	return func(yield func(any) bool) {}
}

// Test types that do NOT implement interfaces
type testNonModel struct{}

type testNonCoreModel struct{}

type testNonSequencedMap struct{}

func TestImplementsInterface_Model_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		typeToCheck     reflect.Type
		shouldImplement bool
	}{
		{
			name:            "pointer to struct implements Model",
			typeToCheck:     reflect.TypeOf(&testModelImpl{}),
			shouldImplement: true,
		},
		{
			name:            "struct does not implement Model (needs pointer receiver)",
			typeToCheck:     reflect.TypeOf(testModelImpl{}),
			shouldImplement: false,
		},
		{
			name:            "non-model type does not implement Model",
			typeToCheck:     reflect.TypeOf(&testNonModel{}),
			shouldImplement: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := ImplementsInterface[Model[testModelCore]](tt.typeToCheck)
			assert.Equal(t, tt.shouldImplement, result, "should correctly identify Model implementation")
		})
	}
}

func TestImplementsInterface_CoreModel_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		typeToCheck     reflect.Type
		shouldImplement bool
	}{
		{
			name:            "pointer to struct implements CoreModel",
			typeToCheck:     reflect.TypeOf(&testCoreModelImpl{}),
			shouldImplement: true,
		},
		{
			name:            "struct does not implement CoreModel (needs pointer receiver)",
			typeToCheck:     reflect.TypeOf(testCoreModelImpl{}),
			shouldImplement: false,
		},
		{
			name:            "non-core-model type does not implement CoreModel",
			typeToCheck:     reflect.TypeOf(&testNonCoreModel{}),
			shouldImplement: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := ImplementsInterface[CoreModel](tt.typeToCheck)
			assert.Equal(t, tt.shouldImplement, result, "should correctly identify CoreModel implementation")
		})
	}
}

func TestImplementsInterface_SequencedMapInterface_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		typeToCheck     reflect.Type
		shouldImplement bool
	}{
		{
			name:            "pointer to struct implements SequencedMapInterface",
			typeToCheck:     reflect.TypeOf(&testSequencedMapImpl{}),
			shouldImplement: true,
		},
		{
			name:            "struct does not implement SequencedMapInterface (needs pointer receiver)",
			typeToCheck:     reflect.TypeOf(testSequencedMapImpl{}),
			shouldImplement: false,
		},
		{
			name:            "non-sequenced-map type does not implement SequencedMapInterface",
			typeToCheck:     reflect.TypeOf(&testNonSequencedMap{}),
			shouldImplement: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := ImplementsInterface[SequencedMapInterface](tt.typeToCheck)
			assert.Equal(t, tt.shouldImplement, result, "should correctly identify SequencedMapInterface implementation")
		})
	}
}

func TestImplementsInterface_BuiltInTypes_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		typeToCheck     reflect.Type
		shouldImplement bool
	}{
		{
			name:            "string does not implement CoreModel",
			typeToCheck:     reflect.TypeOf(""),
			shouldImplement: false,
		},
		{
			name:            "int does not implement CoreModel",
			typeToCheck:     reflect.TypeOf(0),
			shouldImplement: false,
		},
		{
			name:            "map does not implement SequencedMapInterface",
			typeToCheck:     reflect.TypeOf(map[string]string{}),
			shouldImplement: false,
		},
		{
			name:            "slice does not implement CoreModel",
			typeToCheck:     reflect.TypeOf([]string{}),
			shouldImplement: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := ImplementsInterface[CoreModel](tt.typeToCheck)
			assert.Equal(t, tt.shouldImplement, result, "should correctly identify that built-in types do not implement interfaces")
		})
	}
}

// typeImplementsProber and its implementations are used only by
// TestTypeImplements_MatchesReflect_Success so no other test can warm the
// shared implements cache for these type pairs.
type typeImplementsProber interface {
	probeTypeImplements()
}

type typeImplementsPointerProbe struct{}

func (*typeImplementsPointerProbe) probeTypeImplements() {}

type typeImplementsValueProbe struct{}

func (typeImplementsValueProbe) probeTypeImplements() {}

type typeImplementsNonProbe struct{}

func TestTypeImplements_MatchesReflect_Success(t *testing.T) {
	t.Parallel()

	proberType := reflect.TypeFor[typeImplementsProber]()

	tests := []struct {
		name     string
		typ      reflect.Type
		expected bool
	}{
		{name: "pointer receiver type implements via pointer", typ: reflect.TypeFor[*typeImplementsPointerProbe](), expected: true},
		{name: "pointer receiver type does not implement via value", typ: reflect.TypeFor[typeImplementsPointerProbe](), expected: false},
		{name: "value receiver type implements via value", typ: reflect.TypeFor[typeImplementsValueProbe](), expected: true},
		{name: "value receiver type implements via pointer", typ: reflect.TypeFor[*typeImplementsValueProbe](), expected: true},
		{name: "unrelated type does not implement", typ: reflect.TypeFor[typeImplementsNonProbe](), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Clear the entry so the first lookup always takes the uncached path,
			// including on repeated runs in the same process (-count > 1).
			key := implementsKey{t: tt.typ, iface: proberType}
			implementsCache.Delete(key)

			assert.Equal(t, tt.expected, TypeImplements(tt.typ, proberType), "uncached lookup should match reflect")

			cachedValue, cached := implementsCache.Load(key)
			require.True(t, cached, "type pair should be cached after the first lookup")
			assert.Equal(t, tt.expected, cachedValue, "cached entry should match reflect")
			assert.Equal(t, tt.expected, TypeImplements(tt.typ, proberType), "cached lookup should match reflect")
		})
	}
}
