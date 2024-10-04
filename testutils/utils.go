package testutils

import (
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

func CreateMapYamlNode(contents []*yaml.Node, line, column int) *yaml.Node {
	return &yaml.Node{
		Content: contents,
		Kind:    yaml.MappingNode,
		Tag:     "!!map",
		Line:    line,
		Column:  column,
	}
}
