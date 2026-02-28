package core

import "go.yaml.in/yaml/v4"

// Value represents a raw value in an OpenAPI or Arazzo document.
type Value = *yaml.Node
