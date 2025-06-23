package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
)

type Arazzo struct {
	marshaller.CoreModel

	Arazzo             marshaller.Node[string]               `key:"arazzo"`
	Info               marshaller.Node[Info]                 `key:"info"`
	SourceDescriptions marshaller.Node[[]*SourceDescription] `key:"sourceDescriptions" required:"true"`
	Workflows          marshaller.Node[[]*Workflow]          `key:"workflows" required:"true"`
	Components         marshaller.Node[*Components]          `key:"components"`
	Extensions         core.Extensions                       `key:"extensions"`
}
