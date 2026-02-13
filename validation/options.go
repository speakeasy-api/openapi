package validation

import (
	"reflect"
)

type Option func(o *Options)

type Options struct {
	ContextObjects map[reflect.Type]any
}

func WithContextObject[T any](obj *T) Option {
	return func(o *Options) {
		if o.ContextObjects == nil {
			o.ContextObjects = make(map[reflect.Type]any)
		}
		o.ContextObjects[reflect.TypeOf((*T)(nil)).Elem()] = obj
	}
}

func NewOptions(opts ...Option) *Options {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func GetContextObject[T any](o *Options) *T {
	var z T

	if o.ContextObjects == nil {
		return nil
	}

	obj, ok := o.ContextObjects[reflect.TypeOf(z)]
	if !ok {
		return nil
	}

	return obj.(*T)
}
