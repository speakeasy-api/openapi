package jsonpointer

import (
	"errors"
	"fmt"
	"testing"

	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONPointer_Validate_Success(t *testing.T) {
	t.Parallel()

	type args struct {
		j JSONPointer
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "root",
			args: args{
				j: JSONPointer("/"),
			},
		},
		{
			name: "simple path",
			args: args{
				j: JSONPointer("/some/path"),
			},
		},
		{
			name: "path with indices",
			args: args{
				j: JSONPointer("/some/path/0/1"),
			},
		},
		{
			name: "escaped path",
			args: args{
				j: JSONPointer("/~0/some~1path"),
			},
		},
		{
			name: "complex statement",
			args: args{
				j: JSONPointer("/paths/~1special-events~1{eventId}/get"),
			},
		},
		{
			name: "empty tokens (consecutive slashes)",
			args: args{
				j: JSONPointer("/some//path"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.args.j.Validate()
			require.NoError(t, err)
		})
	}
}

func TestJSONPointer_Validate_Error(t *testing.T) {
	t.Parallel()

	type args struct {
		j JSONPointer
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "empty",
			args: args{
				j: JSONPointer(""),
			},
			wantErr: errors.New("validation error -- jsonpointer must not be empty"),
		},
		{
			name: "invalid beginning",
			args: args{
				j: JSONPointer("some/path"),
			},
			wantErr: errors.New("validation error -- jsonpointer must start with /: some/path"),
		},
		{
			name: "invalid path with unescaped tilde",
			args: args{
				j: JSONPointer("/~/some~path"),
			},
			wantErr: errors.New("validation error -- jsonpointer part must be a valid token [^(?:[\x00-.0-}\x7f-\uffff]|~[01])+$]: /~/some~path"),
		},
		{
			name: "invalid path with unescaped tilde in middle",
			args: args{
				j: JSONPointer("/some/~invalid/path"),
			},
			wantErr: errors.New("validation error -- jsonpointer part must be a valid token [^(?:[\x00-.0-}\x7f-\uffff]|~[01])+$]: /some/~invalid/path"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.args.j.Validate()
			require.EqualError(t, err, tt.wantErr.Error())
		})
	}
}

func TestGetTarget_Success(t *testing.T) {
	t.Parallel()

	type TestSimpleStructNoTags struct {
		A int
		B string
	}
	type TestSimpleStructWithTags struct {
		A int `key:"a"`
		B int `key:"b"`
	}
	type TestSimpleStructWithAlternateTags struct {
		A int `json:"a"`
		B int `json:"b"`
	}

	type TestStructLevel3 struct {
		A map[string]int `key:"a"`
	}
	type TestStructLevel2 struct {
		A bool              `key:"a"`
		B map[string]int    `key:"b"`
		C *TestStructLevel3 `key:"c"`
	}
	type TestStructLevel1 struct {
		A int                `key:"a"`
		B []TestStructLevel2 `key:"b"`
		C bool               `key:"c"`
	}
	type TestStructTopLevel struct {
		A map[string]any                              `key:"a"`
		B TestStructLevel1                            `key:"b"`
		C *sequencedmap.Map[string, TestStructLevel1] `key:"c"`
	}

	type args struct {
		source  any
		pointer JSONPointer
		opts    []option
	}
	tests := []struct {
		name string
		args args
		want any
	}{
		{
			name: "root finds primitive",
			args: args{
				source:  1,
				pointer: JSONPointer("/"),
			},
			want: 1,
		},
		{
			name: "root finds object",
			args: args{
				source:  map[string]any{"a": 1},
				pointer: JSONPointer("/"),
			},
			want: map[string]any{"a": 1},
		},
		{
			name: "simple path in top level map",
			args: args{
				source:  map[string]any{"a": 1},
				pointer: JSONPointer("/a"),
			},
			want: 1,
		},
		{
			name: "simple path in struct with no tags",
			args: args{
				source:  TestSimpleStructNoTags{A: 1},
				pointer: JSONPointer("/A"),
			},
			want: 1,
		},
		{
			name: "simple path in struct with tags",
			args: args{
				source:  TestSimpleStructWithTags{A: 1},
				pointer: JSONPointer("/a"),
			},
			want: 1,
		},
		{
			name: "simple path in struct with alternate tags",
			args: args{
				source:  TestSimpleStructWithAlternateTags{A: 1},
				pointer: JSONPointer("/a"),
				opts:    []option{WithStructTags("json")},
			},
			want: 1,
		},
		{
			name: "simple path in top level slice",
			args: args{
				source:  []any{1, 2, 3},
				pointer: JSONPointer("/1"),
			},
			want: 2,
		},
		{
			name: "path in map with / characters in key",
			args: args{
				source:  map[string]any{"a/b": 1},
				pointer: JSONPointer("/a~1b"),
			},
			want: 1,
		},
		{
			name: "path in map with ~ characters in key",
			args: args{
				source:  map[string]any{"a~b": 1},
				pointer: JSONPointer("/a~0b"),
			},
			want: 1,
		},
		{
			name: "complex path",
			args: args{
				source: TestStructTopLevel{
					A: map[string]any{
						"key1": TestStructLevel1{
							B: []TestStructLevel2{
								{
									C: &TestStructLevel3{
										A: map[string]int{
											"key2": 2,
										},
									},
								},
							},
						},
					},
				},
				pointer: JSONPointer("/a/key1/b/0/c/a/key2"),
			},
			want: 2,
		},
		{
			name: "works with sequenced maps",
			args: args{
				source:  sequencedmap.New(sequencedmap.NewElem("a", 1), sequencedmap.NewElem("b", 2)),
				pointer: JSONPointer("/a"),
			},
			want: 1,
		},
		{
			name: "works with sequenced maps inside struct",
			args: args{
				source: TestStructTopLevel{
					C: sequencedmap.New(sequencedmap.NewElem("a", TestStructLevel1{
						A: 3,
					}), sequencedmap.NewElem("b", TestStructLevel1{
						A: 4,
					})),
				},
				pointer: JSONPointer("/c/b/a"),
			},
			want: 4,
		},
		{
			name: "empty tokens in JSON pointer",
			args: args{
				source: map[string]any{
					"": map[string]any{
						"": "value",
					},
				},
				pointer: JSONPointer("//"),
			},
			want: "value",
		},
		{
			name: "numeric string as key in map",
			args: args{
				source:  map[string]any{"400": "Bad Request", "200": "OK"},
				pointer: JSONPointer("/400"),
			},
			want: "Bad Request",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			target, err := GetTarget(tt.args.source, tt.args.pointer, tt.args.opts...)
			require.NoError(t, err)
			assert.Equal(t, tt.want, target)
		})
	}
}

func TestGetTarget_Error(t *testing.T) {
	t.Parallel()

	type TestStruct struct {
		a int // unexported field should be ignored
	}
	type args struct {
		source  any
		pointer JSONPointer
		opts    []option
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "invalid pointer",
			args: args{
				source:  1,
				pointer: JSONPointer("some/path"),
			},
			wantErr: errors.New("validation error -- jsonpointer must start with /: some/path"),
		},
		{
			name: "key in array",
			args: args{
				source:  []any{1, 2, 3},
				pointer: JSONPointer("/key1"),
			},
			wantErr: errors.New("invalid path -- expected index, got key at /key1"),
		},
		{
			name: "index in map",
			args: args{
				source:  map[string]any{"key1": 1},
				pointer: JSONPointer("/0"),
			},
			wantErr: errors.New("not found -- key 0 not found in map at /0"),
		},
		{
			name: "nil map",
			args: args{
				source:  (map[string]any)(nil),
				pointer: JSONPointer("/key1"),
			},
			wantErr: errors.New("not found -- map is nil at /key1"),
		},
		{
			name: "nil slice",
			args: args{
				source:  ([]any)(nil),
				pointer: JSONPointer("/0"),
			},
			wantErr: errors.New("not found -- slice is nil at /0"),
		},
		{
			name: "nil struct",
			args: args{
				source:  (*TestStruct)(nil),
				pointer: JSONPointer("/a"),
			},
			wantErr: errors.New("not found -- struct is nil at /a"),
		},
		{
			name: "pointer through primitive",
			args: args{
				source:  1,
				pointer: JSONPointer("/a"),
			},
			wantErr: errors.New("invalid path -- expected map, slice, struct, or yaml.Node, got int at /a"),
		},
		{
			name: "non string key in map",
			args: args{
				source:  map[any]any{1: 1},
				pointer: JSONPointer("/a"),
			},
			wantErr: errors.New("invalid path -- unsupported map key type interface at /a"),
		},
		{
			name: "key not found in map",
			args: args{
				source:  map[string]any{"key1": 1},
				pointer: JSONPointer("/key2"),
			},
			wantErr: errors.New("not found -- key key2 not found in map at /key2"),
		},
		{
			name: "index out of range in slice",
			args: args{
				source:  []any{1, 2, 3},
				pointer: JSONPointer("/3"),
			},
			wantErr: errors.New("not found -- index 3 out of range for slice/array of length 3 at /3"),
		},
		{
			name: "key not found in struct",
			args: args{
				source: TestStruct{
					a: 1,
				},
				pointer: JSONPointer("/a"),
			},
			wantErr: errors.New("not found -- key a not found in jsonpointer.TestStruct at /a"),
		},
		{
			name: "index in struct",
			args: args{
				source:  TestStruct{},
				pointer: JSONPointer("/1"),
			},
			wantErr: errors.New("not found -- expected IndexNavigable, got struct at /1"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			target, err := GetTarget(tt.args.source, tt.args.pointer, tt.args.opts...)
			require.EqualError(t, err, tt.wantErr.Error())
			assert.Nil(t, target)
		})
	}
}

type InterfaceTestStruct struct {
	typ           string
	valuesByKey   map[string]any
	valuesByIndex []any
	Field1        any
	Field2        any
}

var (
	_ KeyNavigable   = (*InterfaceTestStruct)(nil)
	_ IndexNavigable = (*InterfaceTestStruct)(nil)
)

func (t InterfaceTestStruct) NavigateWithKey(key string) (any, error) {
	switch t.typ {
	case "map":
		return t.valuesByKey[key], nil
	case "struct":
		return nil, ErrSkipInterface
	case "slice":
		return nil, ErrInvalidPath
	default:
		return nil, fmt.Errorf("unknown type %s", t.typ)
	}
}

func (t InterfaceTestStruct) NavigateWithIndex(index int) (any, error) {
	switch t.typ {
	case "map":
		return nil, ErrInvalidPath
	case "struct":
		return nil, ErrSkipInterface
	case "slice":
		return t.valuesByIndex[index], nil
	default:
		return nil, fmt.Errorf("unknown type %s", t.typ)
	}
}

type NavigableNodeWrapper struct {
	typ           string
	NavigableNode InterfaceTestStruct
	Field1        any
	Field2        any
}

var _ NavigableNoder = (*NavigableNodeWrapper)(nil)

func (n NavigableNodeWrapper) GetNavigableNode() (any, error) {
	switch n.typ {
	case "wrapper":
		return n.NavigableNode, nil
	case "struct":
		return nil, ErrSkipInterface
	case "other":
		return nil, ErrInvalidPath
	default:
		return nil, fmt.Errorf("unknown type %s", n.typ)
	}
}

func TestGetTarget_WithInterfaces_Success(t *testing.T) {
	t.Parallel()

	type args struct {
		source  any
		pointer JSONPointer
		opts    []option
	}
	tests := []struct {
		name string
		args args
		want any
	}{
		{
			name: "KeyNavigable succeeds",
			args: args{
				source:  InterfaceTestStruct{typ: "map", valuesByKey: map[string]any{"key1": "value1"}},
				pointer: JSONPointer("/key1"),
			},
			want: "value1",
		},
		{
			name: "IndexNavigable succeeds",
			args: args{
				source:  InterfaceTestStruct{typ: "slice", valuesByIndex: []any{"value1", "value2"}},
				pointer: JSONPointer("/1"),
			},
			want: "value2",
		},
		{
			name: "Struct is navigable",
			args: args{
				source:  InterfaceTestStruct{typ: "struct", Field1: "value1"},
				pointer: JSONPointer("/Field1"),
			},
			want: "value1",
		},
		{
			name: "NavigableNoder succeeds",
			args: args{
				source:  NavigableNodeWrapper{typ: "wrapper", NavigableNode: InterfaceTestStruct{typ: "struct", Field1: "value1"}},
				pointer: JSONPointer("/Field1"),
			},
			want: "value1",
		},
		{
			name: "NavigableNoder struct is navigable",
			args: args{
				source:  NavigableNodeWrapper{typ: "struct", Field2: "value2"},
				pointer: JSONPointer("/Field2"),
			},
			want: "value2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			target, err := GetTarget(tt.args.source, tt.args.pointer, tt.args.opts...)
			require.NoError(t, err)
			assert.Equal(t, tt.want, target)
		})
	}
}

func TestGetTarget_WithInterfaces_Error(t *testing.T) {
	t.Parallel()

	type args struct {
		source  any
		pointer JSONPointer
		opts    []option
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "Error returned for invalid KeyNavigable type",
			args: args{
				source:  InterfaceTestStruct{typ: "slice", valuesByIndex: []any{"value1", "value2"}},
				pointer: JSONPointer("/key2"),
			},
			wantErr: errors.New("not found -- invalid path"),
		},
		{
			name: "Error returned for invalid IndexNavigable type",
			args: args{
				source:  InterfaceTestStruct{typ: "struct", Field1: "value1"},
				pointer: JSONPointer("/1"),
			},
			wantErr: errors.New("can't navigate by index on jsonpointer.InterfaceTestStruct at /1"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			target, err := GetTarget(tt.args.source, tt.args.pointer, tt.args.opts...)
			require.EqualError(t, err, tt.wantErr.Error())
			assert.Nil(t, target)
		})
	}
}

func TestEscapeString_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no special characters",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "tilde only",
			input:    "~",
			expected: "~0",
		},
		{
			name:     "slash only",
			input:    "/",
			expected: "~1",
		},
		{
			name:     "both tilde and slash",
			input:    "~/",
			expected: "~0~1",
		},
		{
			name:     "slash then tilde",
			input:    "/~",
			expected: "~1~0",
		},
		{
			name:     "multiple tildes",
			input:    "~~",
			expected: "~0~0",
		},
		{
			name:     "multiple slashes",
			input:    "//",
			expected: "~1~1",
		},
		{
			name:     "complex string with path-like structure",
			input:    "a/b~c",
			expected: "a~1b~0c",
		},
		{
			name:     "string with mixed characters",
			input:    "hello/world~test",
			expected: "hello~1world~0test",
		},
		{
			name:     "RFC6901 example - a/b",
			input:    "a/b",
			expected: "a~1b",
		},
		{
			name:     "RFC6901 example - m~n",
			input:    "m~n",
			expected: "m~0n",
		},
		{
			name:     "edge case - ~01 sequence",
			input:    "~01",
			expected: "~001",
		},
		{
			name:     "edge case - ~10 sequence",
			input:    "~10",
			expected: "~010",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := EscapeString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJSONPointer_String_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pointer  JSONPointer
		expected string
	}{
		{
			name:     "root pointer",
			pointer:  JSONPointer("/"),
			expected: "/",
		},
		{
			name:     "simple path",
			pointer:  JSONPointer("/some/path"),
			expected: "/some/path",
		},
		{
			name:     "empty string",
			pointer:  JSONPointer(""),
			expected: "",
		},
		{
			name:     "path with indices",
			pointer:  JSONPointer("/a/0/b"),
			expected: "/a/0/b",
		},
		{
			name:     "escaped characters",
			pointer:  JSONPointer("/~0/~1"),
			expected: "/~0/~1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.pointer.String()
			assert.Equal(t, tt.expected, result, "String() should return the pointer value")
		})
	}
}

func TestPartsToJSONPointer_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		parts    []string
		expected JSONPointer
	}{
		{
			name:     "empty parts",
			parts:    []string{},
			expected: JSONPointer(""),
		},
		{
			name:     "single part",
			parts:    []string{"a"},
			expected: JSONPointer("/a"),
		},
		{
			name:     "multiple parts",
			parts:    []string{"a", "b", "c"},
			expected: JSONPointer("/a/b/c"),
		},
		{
			name:     "parts with tilde",
			parts:    []string{"a~b"},
			expected: JSONPointer("/a~0b"),
		},
		{
			name:     "parts with slash",
			parts:    []string{"a/b"},
			expected: JSONPointer("/a~1b"),
		},
		{
			name:     "parts with both special chars",
			parts:    []string{"a~/b"},
			expected: JSONPointer("/a~0~1b"),
		},
		{
			name:     "numeric parts",
			parts:    []string{"0", "1", "2"},
			expected: JSONPointer("/0/1/2"),
		},
		{
			name:     "mixed parts",
			parts:    []string{"paths", "/users/{id}", "get"},
			expected: JSONPointer("/paths/~1users~1{id}/get"),
		},
		{
			name:     "empty string part",
			parts:    []string{""},
			expected: JSONPointer("/"),
		},
		{
			name:     "multiple empty parts",
			parts:    []string{"", ""},
			expected: JSONPointer("//"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := PartsToJSONPointer(tt.parts)
			assert.Equal(t, tt.expected, result, "PartsToJSONPointer should produce correct pointer")
		})
	}
}
