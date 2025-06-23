package marshaller

import (
	"reflect"
	"testing"
)

type TestStruct struct {
	Field1  string
	Field2  int
	Field3  bool
	Field4  float64
	Field5  []string
	Field6  map[string]interface{}
	Field7  *string
	Field8  interface{}
	Field9  uint64
	Field10 byte
}

func BenchmarkFieldByName(b *testing.B) {
	s := TestStruct{
		Field1:  "test",
		Field2:  42,
		Field3:  true,
		Field4:  3.14,
		Field5:  []string{"a", "b"},
		Field6:  map[string]interface{}{"key": "value"},
		Field7:  nil,
		Field8:  "interface",
		Field9:  123456,
		Field10: 255,
	}

	v := reflect.ValueOf(s)
	fieldNames := []string{"Field1", "Field2", "Field3", "Field4", "Field5", "Field6", "Field7", "Field8", "Field9", "Field10"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, name := range fieldNames {
			_ = v.FieldByName(name)
		}
	}
}

func BenchmarkFieldByIndex(b *testing.B) {
	s := TestStruct{
		Field1:  "test",
		Field2:  42,
		Field3:  true,
		Field4:  3.14,
		Field5:  []string{"a", "b"},
		Field6:  map[string]interface{}{"key": "value"},
		Field7:  nil,
		Field8:  "interface",
		Field9:  123456,
		Field10: 255,
	}

	v := reflect.ValueOf(s)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10; j++ {
			_ = v.Field(j)
		}
	}
}

func BenchmarkFieldByNameSingle(b *testing.B) {
	s := TestStruct{Field1: "test"}
	v := reflect.ValueOf(s)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.FieldByName("Field1")
	}
}

func BenchmarkFieldByIndexSingle(b *testing.B) {
	s := TestStruct{Field1: "test"}
	v := reflect.ValueOf(s)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.Field(0)
	}
}

func BenchmarkTypeFieldByName(b *testing.B) {
	s := TestStruct{}
	t := reflect.TypeOf(s)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = t.FieldByName("Field1")
	}
}

func BenchmarkTypeFieldByIndex(b *testing.B) {
	s := TestStruct{}
	t := reflect.TypeOf(s)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = t.Field(0)
	}
}
