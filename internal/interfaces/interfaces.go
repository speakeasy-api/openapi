package interfaces

import (
	"context"
	"iter"
	"reflect"
	"sync"

	"github.com/speakeasy-api/openapi/validation"
	"go.yaml.in/yaml/v3"
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
	Unmarshal(ctx context.Context, parentName string, node *yaml.Node) ([]error, error)
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
	return TypeImplements(t, reflect.TypeFor[T]())
}

type implementsKey struct {
	t     reflect.Type
	iface reflect.Type
}

// implementsCache memoizes reflect.Type.Implements, which is costly for model
// types with large method sets and is queried for every node during population.
var implementsCache sync.Map // map[implementsKey]bool

// TypeImplements reports whether t implements the interface type iface.
// It is equivalent to t.Implements(iface) but caches the result per type pair.
func TypeImplements(t, iface reflect.Type) bool {
	key := implementsKey{t: t, iface: iface}
	if v, ok := implementsCache.Load(key); ok {
		implements, _ := v.(bool)
		return implements
	}

	result := t.Implements(iface)
	implementsCache.Store(key, result)
	return result
}
