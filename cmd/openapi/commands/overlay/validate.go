package overlay

import (
	"fmt"
	"os"

	"github.com/speakeasy-api/openapi/overlay/loader"
	"github.com/spf13/cobra"
)

var validateOverlayFlag string

var validateCmd = &cobra.Command{
	Use:   "validate <overlay>",
	Short: "Given an overlay, it will state whether it appears to be valid or describe the problems found",
	Args:  cobra.RangeArgs(0, 1),
	Run:   RunValidateOverlay,
	Example: `  # Validate overlay using positional argument
  openapi overlay validate overlay.yaml

  # Validate overlay using flag
  openapi overlay validate --overlay overlay.yaml`,
}

func init() {
	validateCmd.Flags().StringVar(&validateOverlayFlag, "overlay", "", "Path to the overlay file")
}

func RunValidateOverlay(cmd *cobra.Command, args []string) {
	// Determine overlay file path from flag or positional argument
	var overlayFile string
	if validateOverlayFlag != "" {
		overlayFile = validateOverlayFlag
	} else if len(args) > 0 {
		overlayFile = args[0]
	} else {
		Dief("overlay file is required (use --overlay flag or provide as first argument)")
	}

	o, err := loader.LoadOverlay(overlayFile)
	if err != nil {
		Die(err)
	}

	err = o.Validate()
	if err != nil {
		Dief("Overlay file %q failed validation:\n%v", overlayFile, err)
	}

	fmt.Fprintf(os.Stderr, "Overlay file %q is valid.\n", overlayFile)
}
