package arazzo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/cmd/openapi/commands/cmdutil"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade <input-file> [output-file]",
	Short: "Upgrade an Arazzo workflow document to the latest supported version",
	Long: `Upgrade an Arazzo workflow document to the latest supported version (` + arazzo.Version + `).

Stdin is supported — either pipe data directly or use '-' explicitly:
  cat workflow.yaml | openapi arazzo upgrade
  cat workflow.yaml | openapi arazzo upgrade -

By default, upgrades all versions including patch-level upgrades:
- 1.0.x → ` + arazzo.Version + `
- 1.1.x → ` + arazzo.Version + ` (patch upgrade if newer patch exists)

With --minor-only, only performs cross-minor version upgrades:
- 1.0.x → ` + arazzo.Version + ` (cross-minor upgrade)
- 1.1.x → no change (same minor version, skip patch upgrades)

With --version, upgrades to a specific Arazzo version instead of the latest:
- openapi arazzo upgrade --version 1.0.1 workflow.yaml

Note: --version and --minor-only are mutually exclusive.

The upgrade process includes:
- Updating the arazzo version field
- Pinning ` + "`jsonpath`" + ` and ` + "`xpath`" + ` criteria to the versions implied by Arazzo 1.0
  (the defaults changed to RFC 9535 and XPath 3.1 in Arazzo 1.1)
- Validating the upgraded document and reporting anything that needs manual changes,
  such as runtime expressions that don't match the stricter Arazzo 1.1 grammar

Output options:
- No output file specified: writes to stdout (pipe-friendly)
- Output file specified: writes to the specified file
- --write flag: writes in-place to the input file`,
	Args: cmdutil.StdinOrFileArgs(1, 2),
	Run:  runUpgrade,
}

var (
	upgradeMinorOnly     bool
	upgradeWriteInPlace  bool
	upgradeTargetVersion string
)

func init() {
	upgradeCmd.Flags().BoolVar(&upgradeMinorOnly, "minor-only", false, "only upgrade across minor versions, skip patch-level upgrades within same minor")
	upgradeCmd.Flags().BoolVarP(&upgradeWriteInPlace, "write", "w", false, "write result in-place to input file")
	upgradeCmd.Flags().StringVarP(&upgradeTargetVersion, "version", "V", "", "target Arazzo version to upgrade to (default latest)")
}

func runUpgrade(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	inputFile := cmdutil.InputFileFromArgs(args)
	outputFile := cmdutil.ArgAt(args, 1, "")

	if upgradeTargetVersion != "" && upgradeMinorOnly {
		cmdutil.Dief("--version and --minor-only are mutually exclusive")
	}

	if upgradeWriteInPlace {
		if cmdutil.IsStdin(inputFile) {
			cmdutil.Dief("cannot use --write flag when reading from stdin")
		}
		if outputFile != "" {
			cmdutil.Dief("cannot specify output file when using --write flag")
		}
		outputFile = inputFile
	}

	var opts []arazzo.Option[arazzo.UpgradeOptions]
	if upgradeTargetVersion != "" {
		// Enable same-minor upgrades so patch-level targets work as expected.
		opts = append(opts, arazzo.WithUpgradeTargetVersion(upgradeTargetVersion), arazzo.WithUpgradeSameMinorVersion())
	} else if !upgradeMinorOnly {
		opts = append(opts, arazzo.WithUpgradeSameMinorVersion())
	}

	input, err := readInput(inputFile)
	if err != nil {
		cmdutil.Die(err)
	}

	output := bytes.NewBuffer(nil)
	if err := upgradeArazzo(ctx, bytes.NewReader(input), output, os.Stderr, opts...); err != nil {
		cmdutil.Die(err)
	}

	if outputFile == "" {
		if _, err := os.Stdout.Write(output.Bytes()); err != nil {
			cmdutil.Die(err)
		}
		return
	}

	cleanOutputFile := filepath.Clean(outputFile)
	if err := os.WriteFile(cleanOutputFile, output.Bytes(), 0o644); err != nil { //nolint:gosec // output files are user specified
		cmdutil.Die(fmt.Errorf("failed to write output file: %w", err))
	}
	fmt.Fprintf(os.Stderr, "📄 Document written to: %s\n", cleanOutputFile)
}

func readInput(inputFile string) ([]byte, error) {
	if cmdutil.IsStdin(inputFile) {
		fmt.Fprintf(os.Stderr, "Processing Arazzo document from stdin\n")
		return io.ReadAll(os.Stdin)
	}

	cleanInputFile := filepath.Clean(inputFile)
	fmt.Fprintf(os.Stderr, "Processing Arazzo document: %s\n", cleanInputFile)

	data, err := os.ReadFile(cleanInputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open input file: %w", err)
	}
	return data, nil
}

// upgradeArazzo upgrades the Arazzo document read from in and writes the result to out, reporting progress and any
// validation errors found in the upgraded document to stderr. Validation errors don't fail the upgrade.
func upgradeArazzo(ctx context.Context, in io.Reader, out io.Writer, stderr io.Writer, opts ...arazzo.Option[arazzo.UpgradeOptions]) error {
	doc, _, err := arazzo.Unmarshal(ctx, in, arazzo.WithSkipValidation())
	if err != nil {
		return fmt.Errorf("failed to unmarshal Arazzo document: %w", err)
	}
	if doc == nil {
		return errors.New("failed to parse Arazzo document: document is nil")
	}

	originalVersion := doc.Arazzo
	upgraded, err := arazzo.Upgrade(ctx, doc, opts...)
	if err != nil {
		return fmt.Errorf("failed to upgrade Arazzo document: %w", err)
	}

	if upgraded {
		fmt.Fprintf(stderr, "✅ Successfully upgraded from %s to %s\n", originalVersion, doc.Arazzo)
	} else {
		fmt.Fprintf(stderr, "📋 No upgrade needed - document is already at version %s\n", originalVersion)
	}

	if validationErrs := doc.Validate(ctx); len(validationErrs) > 0 {
		fmt.Fprintf(stderr, "⚠️  Found %d validation errors in the upgraded document that need to be fixed manually:\n", len(validationErrs))
		for i, validationErr := range validationErrs {
			fmt.Fprintf(stderr, "  %d. %s\n", i+1, validationErr.Error())
		}
		fmt.Fprintln(stderr)
	}

	return arazzo.Marshal(ctx, doc, out)
}

// GetUpgradeCommand returns the upgrade command for external use
func GetUpgradeCommand() *cobra.Command {
	return upgradeCmd
}
