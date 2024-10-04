package oas31

import "github.com/speakeasy-api/openapi/extensions"

type ExternalDoc struct {
	Description *string
	URL         string
	Extensions  *extensions.Extensions
}
