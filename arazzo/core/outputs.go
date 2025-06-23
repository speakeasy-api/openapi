package core

import (
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

type Outputs = *sequencedmap.Map[string, marshaller.Node[string]]
