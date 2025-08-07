package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

type Paths struct {
	marshaller.CoreModel
	sequencedmap.Map[string, *Reference[*PathItem]]

	Extensions core.Extensions `key:"extensions"`
}

func NewPaths() *Paths {
	return &Paths{
		Map: *sequencedmap.New[string, *Reference[*PathItem]](),
	}
}

type PathItem struct {
	marshaller.CoreModel
	sequencedmap.Map[string, *Operation]

	Summary     marshaller.Node[*string] `key:"summary"`
	Description marshaller.Node[*string] `key:"description"`

	Servers    marshaller.Node[[]*Server]                `key:"servers"`
	Parameters marshaller.Node[[]*Reference[*Parameter]] `key:"parameters"`

	Extensions core.Extensions `key:"extensions"`
}

func NewPathItem() *PathItem {
	return &PathItem{
		Map: *sequencedmap.New[string, *Operation](),
	}
}
