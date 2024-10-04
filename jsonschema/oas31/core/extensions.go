package core

import (
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

type Extensions = *sequencedmap.Map[string, marshaller.Node[extensions.Extension]]
