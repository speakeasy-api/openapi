package interfaces

import (
	"context"
	"iter"
	"reflect"

	"github.com/speakeasy-api/openapi/validation"
	"gopkg.in/yaml.v3"
)

type Validator[T any] interface {
	*T
	Validate(context.Context, ...validation.Option) []error
}

type Model[C any] interface {
	Validate(context.Context, ...validation.Option) []error
	GetCore() *C
}

type CoreModel interface {
	Unmarshal(ctx context.Context, node *yaml.Node) ([]error, error)
}

// sequencedMapInterface defines the interface that sequenced maps must implement
type SequencedMapInterface interface {
	Init()
	IsInitialized() bool
	SetUntyped(key, value any) error
	AllUntyped() iter.Seq2[any, any]
	GetKeyType() reflect.Type
	GetValueType() reflect.Type
	Len() int
	GetAny(key any) (any, bool)
	SetAny(key, value any)
	DeleteAny(key any)
	KeysAny() iter.Seq[any]
}

func ImplementsInterface[T any](t reflect.Type) bool {
	interfaceType := reflect.TypeOf((*T)(nil)).Elem()
	return t.Implements(interfaceType)
}
