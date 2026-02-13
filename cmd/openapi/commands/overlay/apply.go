package overlay

import (
	"fmt"
	"os"

	"github.com/speakeasy-api/openapi/cmd/openapi/commands/cmdutil"
	"github.com/speakeasy-api/openapi/overlay/loader"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	applyOverlayFlag string
	applySchemaFlag  string
	applyOutFlag     string
)

var applyCmd = &cobra.Command{
	Use:   "apply <overlay> [ <spec> ]",
	Short: "Given an overlay, it will apply it to the spec. If omitted, spec will be loaded via extends (only from local file system).",
	Args:  cobra.RangeArgs(0, 2),
	Run:   RunApply,
	Example: `  # Apply overlay using positional arguments
  openapi overlay apply overlay.yaml spec.yaml

  # Apply overlay using flags
  openapi overlay apply --overlay overlay.yaml --schema spec.yaml

  # Apply overlay with output to file
  openapi overlay apply --overlay overlay.yaml --schema spec.yaml --out modified-spec.yaml

  # Apply overlay when overlay has extends key set
  openapi overlay apply overlay.yaml

  # Pipe spec via stdin and provide overlay as file
  cat spec.yaml | openapi overlay apply overlay.yaml -
  cat spec.yaml | openapi overlay apply --overlay overlay.yaml --schema -`,
}

func init() {
	applyCmd.Flags().StringVar(&applyOverlayFlag, "overlay", "", "Path to the overlay file")
	applyCmd.Flags().StringVar(&applySchemaFlag, "schema", "", "Path to the OpenAPI specification file (use '-' for stdin)")
	applyCmd.Flags().StringVarP(&applyOutFlag, "out", "o", "", "Output file path (defaults to stdout)")
}

func RunApply(cmd *cobra.Command, args []string) {
	// Determine overlay file path from flag or positional argument
	var overlayFile string
	if applyOverlayFlag != "" {
		overlayFile = applyOverlayFlag
	} else if len(args) > 0 {
		overlayFile = args[0]
	} else {
		Dief("overlay file is required (use --overlay flag or provide as first argument)")
	}

	o, err := loader.LoadOverlay(overlayFile)
	if err != nil {
		Die(err)
	}

	// Determine spec file path from flag or positional argument
	var specFile string
	if applySchemaFlag != "" {
		specFile = applySchemaFlag
	} else {
		specFile = cmdutil.ArgAt(args, 1, "")
	}

	// Load spec from stdin or file
	var ys *yaml.Node
	specSource := specFile
	if cmdutil.IsStdin(specFile) || (specFile == "" && cmdutil.StdinIsPiped()) {
		fmt.Fprintf(os.Stderr, "Reading specification from stdin\n")
		specSource = "stdin"
		ys, err = loader.LoadSpecificationFromReader(os.Stdin)
		if err != nil {
			Die(err)
		}
	} else {
		ys, specSource, err = loader.LoadEitherSpecification(specFile, o)
		if err != nil {
			Die(err)
		}
	}

	err = o.ApplyTo(ys)
	if err != nil {
		Dief("Failed to apply overlay to spec %q: %v", specSource, err)
	}

	// Write to output file if specified, otherwise stdout
	out := os.Stdout
	if applyOutFlag != "" {
		f, err := os.Create(applyOutFlag)
		if err != nil {
			Dief("Failed to create output file %q: %v", applyOutFlag, err)
		}
		defer f.Close()
		out = f
	}

	err = yaml.NewEncoder(out).Encode(ys)
	if err != nil {
		Dief("Failed to encode spec %q: %v", specSource, err)
	}
}
