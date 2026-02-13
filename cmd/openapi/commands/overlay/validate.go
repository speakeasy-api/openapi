package overlay

import (
	"fmt"
	"os"

	"github.com/speakeasy-api/openapi/cmd/openapi/commands/cmdutil"
	"github.com/speakeasy-api/openapi/overlay/loader"
	"github.com/spf13/cobra"
)

var validateOverlayFlag string

var validateCmd = &cobra.Command{
	Use:   "validate <overlay>",
	Short: "Given an overlay, it will state whether it appears to be valid or describe the problems found",
	Args: func(cmd *cobra.Command, args []string) error {
		// Accept: --overlay flag, positional arg, or piped stdin
		if validateOverlayFlag != "" || len(args) > 0 || cmdutil.StdinIsPiped() {
			return nil
		}
		return fmt.Errorf("overlay file is required (use --overlay flag, provide as argument, or pipe to stdin)")
	},
	Run:   RunValidateOverlay,
	Example: `  # Validate overlay using positional argument
  openapi overlay validate overlay.yaml

  # Validate overlay using flag
  openapi overlay validate --overlay overlay.yaml

  # Validate overlay from stdin
  cat overlay.yaml | openapi overlay validate`,
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
	}

	source := overlayFile
	if source == "" || cmdutil.IsStdin(source) {
		// Read from stdin
		source = "stdin"
		fmt.Fprintf(os.Stderr, "Validating overlay from stdin\n")
		o, err := loader.LoadOverlayFromReader(os.Stdin)
		if err != nil {
			Die(err)
		}
		if err := o.Validate(); err != nil {
			Dief("Overlay %q failed validation:\n%v", source, err)
		}
	} else {
		o, err := loader.LoadOverlay(overlayFile)
		if err != nil {
			Die(err)
		}
		if err := o.Validate(); err != nil {
			Dief("Overlay %q failed validation:\n%v", source, err)
		}
	}

	fmt.Fprintf(os.Stderr, "Overlay %q is valid.\n", source)
}
