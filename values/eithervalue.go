package values

import (
	"fmt"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/values/core"
)

type EitherValue[L any, LCore any, R any, RCore any] struct {
	marshaller.Model[core.EitherValue[LCore, RCore]]

	Left  *L
	Right *R
}

func (e *EitherValue[L, LCore, R, RCore]) IsLeft() bool {
	return e.Left != nil
}

func (e *EitherValue[L, LCore, R, RCore]) GetLeft() L {
	return *e.Left
}

func (e *EitherValue[L, LCore, R, RCore]) IsRight() bool {
	return e.Right != nil
}

func (e *EitherValue[L, LCore, R, RCore]) GetRight() R {
	return *e.Right
}

func (e *EitherValue[L, LCore, R, RCore]) Populate(source any) error {
	ec, ok := source.(*core.EitherValue[LCore, RCore])
	if !ok {
		return fmt.Errorf("source is not an %T", &core.EitherValue[LCore, RCore]{})
	}

	if ec.IsLeft {
		if err := marshaller.Populate(ec.Left, &e.Left); err != nil {
			return fmt.Errorf("failed to populate left: %w", err)
		}

		return nil
	}

	if err := marshaller.Populate(ec.Right, &e.Right); err != nil {
		return fmt.Errorf("failed to populate right: %w", err)
	}

	return nil
}

// GetNavigableNode implements the NavigableNoder interface to return the held value for JSON pointer navigation
func (e *EitherValue[L, LCore, R, RCore]) GetNavigableNode() (any, error) {
	if e.Left != nil {
		return e.Left, nil
	}
	if e.Right != nil {
		return e.Right, nil
	}
	return nil, fmt.Errorf("EitherValue has no value set")
}
