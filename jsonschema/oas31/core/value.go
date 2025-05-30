package core

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"gopkg.in/yaml.v3"
)

type EitherValue[L any, R any] struct {
	marshaller.CoreModel
	Left  *L `populatorValue:"true"`
	Right *R `populatorValue:"true"`
}

var _ interfaces.CoreModel = (*EitherValue[any, any])(nil)

func (v *EitherValue[L, R]) Unmarshal(ctx context.Context, node *yaml.Node) error {
	errs := []error{}

	l, lErrs := unmarshalValue[L](ctx, node)
	if l != nil && len(lErrs) == 0 {
		v.Left = l
		v.SetRootNode(node)
		return nil
	}
	errs = append(errs, lErrs...)

	r, rErrs := unmarshalValue[R](ctx, node)
	if r != nil && len(rErrs) == 0 {
		v.Right = r
		v.SetRootNode(node)
		return nil
	}
	errs = append(errs, rErrs...)

	return fmt.Errorf("unable to marshal into either %s or %s: %w", reflect.TypeOf((*L)(nil)).Elem().Name(), reflect.TypeOf((*R)(nil)).Elem().Name(), errors.Join(errs...))
}

func (v *EitherValue[L, R]) SyncChanges(ctx context.Context, model any, valueNode *yaml.Node) (*yaml.Node, error) {
	mv := reflect.ValueOf(model)

	if mv.Kind() == reflect.Ptr {
		mv = mv.Elem()
	}

	if mv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %s", mv.Kind())
	}

	lf := mv.FieldByName("Left")
	rf := mv.FieldByName("Right")

	lv, err := marshaller.SyncValue(ctx, lf.Interface(), &v.Left, valueNode, false)
	if err != nil {
		return nil, err
	}

	rv, err := marshaller.SyncValue(ctx, rf.Interface(), &v.Right, valueNode, false)
	if err != nil {
		return nil, err
	}

	if lv != nil {
		v.SetRootNode(lv)
		return lv, nil
	} else {
		v.SetRootNode(rv)
		return rv, nil
	}
}

func (v *EitherValue[L, R]) GetNavigableNode() (any, error) {
	if v.Left != nil {
		return v.Left, nil
	}
	return v.Right, nil
}

func unmarshalValue[T any](ctx context.Context, node *yaml.Node) (*T, []error) {
	errs := []error{}

	if reflect.TypeOf((*T)(nil)).Implements(reflect.TypeOf((*marshaller.Unmarshallable)(nil)).Elem()) {
		var v T

		unmarshallable, ok := any(&v).(marshaller.Unmarshallable)
		if ok {
			if err := unmarshallable.Unmarshal(ctx, node); err != nil {
				errs = append(errs, err)
			} else {
				return &v, nil
			}
		}
	}

	var v T
	if err := node.Decode(&v); err != nil {
		errs = append(errs, err)
	} else {
		return &v, nil
	}

	return nil, errs
}

type Value = *yaml.Node
