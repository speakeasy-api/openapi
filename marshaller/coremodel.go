package marshaller

import "gopkg.in/yaml.v3"

type CoreModeler interface {
	GetRootNode() *yaml.Node
	SetRootNode(rootNode *yaml.Node)
	GetValid() bool
	SetValid(valid bool)
}

type CoreModel struct {
	RootNode *yaml.Node
	Valid    bool
}

var _ CoreModeler = (*CoreModel)(nil)

func (c CoreModel) GetRootNode() *yaml.Node {
	return c.RootNode
}

func (c *CoreModel) SetRootNode(rootNode *yaml.Node) {
	c.RootNode = rootNode
}

func (c CoreModel) GetValid() bool {
	return c.Valid
}

func (c *CoreModel) SetValid(valid bool) {
	c.Valid = valid
}
