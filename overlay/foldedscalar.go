package overlay

import (
	"github.com/speakeasy-api/openapi/yml"
)

// stabilizeFoldedScalars restyles folded block scalars in the overlay's own update
// payloads so that serializing the overlay round trips unchanged.
func (o *Overlay) stabilizeFoldedScalars() {
	if o == nil {
		return
	}

	for i := range o.Actions {
		yml.StabilizeFoldedScalars(&o.Actions[i].Update)
	}
}
