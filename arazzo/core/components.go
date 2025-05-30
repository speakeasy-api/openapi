package core

import (
	coreExtensions "github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/jsonschema/oas31/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

type Components struct {
	marshaller.CoreModel
	Inputs         marshaller.Node[*sequencedmap.Map[string, core.JSONSchema]] `key:"inputs"`
	Parameters     marshaller.Node[*sequencedmap.Map[string, *Parameter]]      `key:"parameters"`
	SuccessActions marshaller.Node[*sequencedmap.Map[string, *SuccessAction]]  `key:"successActions"`
	FailureActions marshaller.Node[*sequencedmap.Map[string, *FailureAction]]  `key:"failureActions"`
	Extensions     coreExtensions.Extensions                                   `key:"extensions"`
}
