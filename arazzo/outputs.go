package arazzo

import (
	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

// Outputs is a map of friendly name to expressions that extract data from the workflows/steps.
type Outputs = *sequencedmap.Map[string, expression.Expression]
