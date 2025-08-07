package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

type Callback struct {
	marshaller.CoreModel
	sequencedmap.Map[string, *Reference[*PathItem]]

	Extensions core.Extensions `key:"extensions"`
}
