package overlay

import (
	"fmt"
	"os"

	"github.com/speakeasy-api/openapi/cmd/openapi/commands/cmdutil"
	overlayPkg "github.com/speakeasy-api/openapi/overlay"
	"github.com/speakeasy-api/openapi/overlay/loader"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	compareBeforeFlag string
	compareAfterFlag  string
	compareOutFlag    string
)

var compareCmd = &cobra.Command{
	Use:   "compare <spec1> <spec2>",
	Short: "Given two specs, it will output an overlay that describes the differences between them",
	Args:  cobra.RangeArgs(0, 2),
	Run:   RunCompare,
	Example: `  # Compare specs using positional arguments
  openapi overlay compare spec1.yaml spec2.yaml

  # Compare specs using flags
  openapi overlay compare --before spec1.yaml --after spec2.yaml

  # Compare specs with output to file
  openapi overlay compare --before spec1.yaml --after spec2.yaml --out overlay.yaml

  # Pipe the "before" spec via stdin
  cat spec1.yaml | openapi overlay compare - spec2.yaml
  cat spec1.yaml | openapi overlay compare --after spec2.yaml`,
}

func init() {
	compareCmd.Flags().StringVar(&compareBeforeFlag, "before", "", "Path to the first (before) specification file (use '-' for stdin)")
	compareCmd.Flags().StringVar(&compareAfterFlag, "after", "", "Path to the second (after) specification file")
	compareCmd.Flags().StringVarP(&compareOutFlag, "out", "o", "", "Output file path (defaults to stdout)")
}

func RunCompare(cmd *cobra.Command, args []string) {
	// Determine first spec file path from flag or positional argument
	var spec1 string
	if compareBeforeFlag != "" {
		spec1 = compareBeforeFlag
	} else if len(args) > 0 {
		spec1 = args[0]
	} else if cmdutil.StdinIsPiped() {
		spec1 = cmdutil.StdinIndicator
	} else {
		Dief("first specification file is required (use --before flag or provide as first argument)")
	}

	// Determine second spec file path from flag or positional argument
	var spec2 string
	if compareAfterFlag != "" {
		spec2 = compareAfterFlag
	} else if len(args) > 1 {
		spec2 = args[1]
	} else {
		Dief("second specification file is required (use --after flag or provide as second argument)")
	}

	// Load first spec (may come from stdin)
	var y1 *yaml.Node
	spec1Source := spec1
	if cmdutil.IsStdin(spec1) {
		fmt.Fprintf(os.Stderr, "Reading before spec from stdin\n")
		spec1Source = "stdin"
		var err error
		y1, err = loader.LoadSpecificationFromReader(os.Stdin)
		if err != nil {
			Dief("Failed to load spec from stdin: %v", err)
		}
	} else {
		var err error
		y1, err = loader.LoadSpecification(spec1)
		if err != nil {
			Dief("Failed to load %q: %v", spec1, err)
		}
	}

	y2, err := loader.LoadSpecification(spec2)
	if err != nil {
		Dief("Failed to load %q: %v", spec2, err)
	}

	title := fmt.Sprintf("Overlay %s => %s", spec1Source, spec2)

	o, err := overlayPkg.Compare(title, y1, *y2)
	if err != nil {
		Dief("Failed to compare specs %q and %q: %v", spec1Source, spec2, err)
	}

	// Write to output file if specified, otherwise stdout
	out := os.Stdout
	if compareOutFlag != "" {
		f, err := os.Create(compareOutFlag)
		if err != nil {
			Dief("Failed to create output file %q: %v", compareOutFlag, err)
		}
		defer f.Close()
		out = f
	}

	err = o.Format(out)
	if err != nil {
		Dief("Failed to format overlay: %v", err)
	}
}
