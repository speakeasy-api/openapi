package core

import (
	"context"

	"gopkg.in/yaml.v3"
)

type CoreModel interface {
	Unmarshal(ctx context.Context, node *yaml.Node) error
}
