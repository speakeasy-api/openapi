package oas31

import (
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

type Discriminator struct {
	PropertyName string
	Mapping      *sequencedmap.Map[string, string]
	Extensions   *extensions.Extensions
}
