package values

import "gopkg.in/yaml.v3"

// Value represents a raw value in an OpenAPI or Arazzo document.
type Value = *yaml.Node
