package core

import (
	"github.com/speakeasy-api/openapi/sequencedmap"
	values "github.com/speakeasy-api/openapi/values/core"
)

// OutputValue is either a Selector Object (Arazzo 1.1+) or a runtime expression string.
type OutputValue = *values.EitherValue[Selector, string]

type Outputs = *sequencedmap.Map[string, OutputValue]
