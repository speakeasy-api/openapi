package openapi

import (
	"fmt"
	"io"
	"time"
)

func reportElapsed(w io.Writer, action string, elapsed time.Duration) {
	roundedElapsed := elapsed.Round(time.Millisecond)
	if roundedElapsed < time.Millisecond {
		roundedElapsed = time.Millisecond
	}

	fmt.Fprintf(w, "%s completed in %s\n", action, roundedElapsed)
}
