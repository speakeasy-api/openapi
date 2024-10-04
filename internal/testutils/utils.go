package testutils

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// TODO use these more in tests
func CreateStringYamlNode(value string, line, column int) *yaml.Node {
	return &yaml.Node{
		Value:  value,
		Kind:   yaml.ScalarNode,
		Tag:    "!!str",
		Line:   line,
		Column: column,
	}
}

func CreateIntYamlNode(value int, line, column int) *yaml.Node {
	return &yaml.Node{
		Value:  fmt.Sprintf("%d", value),
		Kind:   yaml.ScalarNode,
		Tag:    "!!int",
		Line:   line,
		Column: column,
	}
}

func CreateBoolYamlNode(value bool, line, column int) *yaml.Node {
	return &yaml.Node{
		Value:  fmt.Sprintf("%t", value),
		Kind:   yaml.ScalarNode,
		Tag:    "!!bool",
		Line:   line,
		Column: column,
	}
}

func CreateMapYamlNode(contents []*yaml.Node, line, column int) *yaml.Node {
	return &yaml.Node{
		Content: contents,
		Kind:    yaml.MappingNode,
		Tag:     "!!map",
		Line:    line,
		Column:  column,
	}
}
