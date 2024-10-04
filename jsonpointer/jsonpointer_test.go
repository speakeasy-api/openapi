package jsonpointer

import (
	"errors"
	"testing"

	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONPointer_Validate_Success(t *testing.T) {
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.args.j.Validate()
			require.NoError(t, err)
		})
	}
}

func TestJSONPointer_Validate_Error(t *testing.T) {
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
			name: "empty part in path",
			args: args{
				j: JSONPointer("/some//path"),
			},
			wantErr: errors.New("validation error -- jsonpointer part must not be empty: /some//path"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.args.j.Validate()
			assert.EqualError(t, err, tt.wantErr.Error())
		})
	}
}

func TestGetTarget_Success(t *testing.T) {
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, err := GetTarget(tt.args.source, tt.args.pointer, tt.args.opts...)
			require.NoError(t, err)
			assert.Equal(t, tt.want, target)
		})
	}
}

func TestGetTarget_Error(t *testing.T) {
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
			wantErr: errors.New("invalid path -- expected key, got index at /0"),
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
			wantErr: errors.New("invalid path -- expected map, slice, or struct, got int at /a"),
		},
		{
			name: "non string key in map",
			args: args{
				source:  map[any]any{1: 1},
				pointer: JSONPointer("/a"),
			},
			wantErr: errors.New("invalid path -- expected map key to be string, got interface at /a"),
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
			wantErr: errors.New("invalid path -- expected key, got index at /1"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, err := GetTarget(tt.args.source, tt.args.pointer, tt.args.opts...)
			assert.EqualError(t, err, tt.wantErr.Error())
			assert.Nil(t, target)
		})
	}
}
