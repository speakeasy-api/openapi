package asyncapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/speakeasy-api/openapi/asyncapi"
	"github.com/speakeasy-api/openapi/cmd/openapi/commands/cmdutil"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate <file>",
	Short: "Validate an AsyncAPI 3.x document",
	Long: `Validate an AsyncAPI document for compliance with the AsyncAPI Specification.
AsyncAPI 3.0 and 3.1 are supported. Other versions, including 2.x, are reported as
unsupported and not checked further.

This command will parse and validate the provided AsyncAPI document, checking for:
- Structural validity, required fields and proper data types
- Schemas written in the AsyncAPI Schema format or JSON Schema Draft 07
- References within the document, which must exist and point to the right kind of object
- Operation and message traits, applied in order
- Rules spanning objects, such as an operation's messages belonging to its channel

By default nothing outside the document is read. References to other files or URLs
are reported as warnings and not followed. Use --resolve to load them, and
--base-uri to give a document read from stdin a location to resolve them from.

The output lists anything that couldn't be checked, such as references that weren't
followed and schemas or bindings in formats that aren't validated. Errors make
validation fail. Warnings only fail with --strict, and hints never do.

Stdin is supported — either pipe data directly or use '-' explicitly:
  cat asyncapi.yaml | openapi asyncapi validate
  cat asyncapi.yaml | openapi asyncapi validate --resolve --base-uri ./specs/asyncapi.yaml -`,
	Args: cmdutil.StdinOrFileArgs(1, 1),
	Run:  runValidate,
}

// validateOptions are the options for validating an AsyncAPI document.
type validateOptions struct {
	// resolve loads references to other documents.
	resolve bool
	// baseURI is the location references are resolved against, overriding the file's location.
	baseURI string
	// strict fails validation on warnings.
	strict bool
	// httpClient loads references to URLs.
	httpClient *http.Client
}

var (
	validateResolve bool
	validateBaseURI string
	validateStrict  bool
)

func init() {
	validateCmd.Flags().BoolVar(&validateResolve, "resolve", false, "load references to other files and URLs and validate them")
	validateCmd.Flags().StringVar(&validateBaseURI, "base-uri", "", "file path or URL to resolve relative references against, for example when reading from stdin (requires --resolve)")
	validateCmd.Flags().BoolVar(&validateStrict, "strict", false, "fail validation on warnings as well as errors")
}

func runValidate(cmd *cobra.Command, args []string) {
	opts := validateOptions{
		resolve:    validateResolve,
		baseURI:    validateBaseURI,
		strict:     validateStrict,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	if err := validateAsyncAPI(cmd.Context(), cmdutil.InputFileFromArgs(args), opts, os.Stdin, os.Stderr); err != nil {
		cmdutil.Die(err)
	}
}

func validateAsyncAPI(ctx context.Context, file string, opts validateOptions, stdin io.Reader, stderr io.Writer) error {
	if opts.baseURI != "" && !opts.resolve {
		return errors.New("--base-uri is only used to resolve references to other documents, use it with --resolve")
	}

	data, location, err := readInput(file, stdin, stderr)
	if err != nil {
		return err
	}
	if opts.baseURI != "" {
		location = opts.baseURI
	}
	if opts.resolve && location == "" {
		fmt.Fprintf(stderr, "Note: the document has no location, so relative references to other documents can't be resolved. Use --base-uri to set one.\n")
	}
	if location != "" && !isURL(location) {
		if location, err = filepath.Abs(location); err != nil {
			return fmt.Errorf("failed to resolve document location: %w", err)
		}
	}

	doc, validationErrs, err := asyncapi.Unmarshal(ctx, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to unmarshal file: %w", err)
	}

	// The rest of an unsupported version is checked against the 3.x model, which would only report noise.
	if versionErrs := unsupportedVersionErrors(validationErrs); len(versionErrs) > 0 {
		fmt.Fprintf(stderr, "❌ AsyncAPI document is not supported:\n\n")
		printErrors(stderr, versionErrs)
		fmt.Fprintf(stderr, "\nThe rest of the document was not checked.\n")
		return errors.New("asyncapi document version is not supported")
	}

	resolveErrs, err := doc.ResolveAllReferences(ctx, asyncapi.ResolveAllOptions{
		AsyncAPILocation:    location,
		DisableExternalRefs: !opts.resolve,
		HTTPClient:          opts.httpClient,
	})
	if err != nil {
		return fmt.Errorf("failed to resolve references: %w", err)
	}

	summary := summarize(append(validationErrs, resolveErrs...))
	summary.print(stderr)

	switch {
	case summary.errors > 0:
		return errors.New("asyncapi document validation failed")
	case opts.strict && summary.warnings > 0:
		return fmt.Errorf("asyncapi document has %d warnings and --strict is set", summary.warnings)
	}
	return nil
}

// readInput reads the document from the file or stdin, returning its location (empty for stdin).
func readInput(file string, stdin io.Reader, stderr io.Writer) ([]byte, string, error) {
	if cmdutil.IsStdin(file) {
		fmt.Fprintf(stderr, "Validating AsyncAPI document from stdin\n")
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, "", fmt.Errorf("failed to read stdin: %w", err)
		}
		return data, "", nil
	}

	cleanFile := filepath.Clean(file)
	fmt.Fprintf(stderr, "Validating AsyncAPI document: %s\n", cleanFile)
	data, err := os.ReadFile(cleanFile)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open file: %w", err)
	}
	return data, cleanFile, nil
}

func isURL(location string) bool {
	u, err := url.Parse(location)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https")
}

func unsupportedVersionErrors(errs []error) []error {
	versionErrs := []error{}
	for _, err := range errs {
		var vErr *validation.Error
		if errors.As(err, &vErr) && vErr.Rule == validation.RuleValidationSupportedVersion {
			versionErrs = append(versionErrs, err)
		}
	}
	return versionErrs
}

// validationSummary is the result of validating a document, including what couldn't be checked.
type validationSummary struct {
	errs     []error
	errors   int
	warnings int
	hints    int
	// unfollowed is the number of references to other documents that weren't followed.
	unfollowed int
	// unvalidatedFormats is the number of schemas, schema references and bindings in formats that aren't validated.
	unvalidatedFormats int
}

func summarize(errs []error) validationSummary {
	s := validationSummary{errs: errs}

	for _, err := range errs {
		var vErr *validation.Error
		if !errors.As(err, &vErr) {
			s.errors++
			continue
		}

		// A reference that wasn't followed isn't known to be broken, so it's reported as something not checked.
		if errors.Is(err, asyncapi.ErrExternalReferencesDisabled) {
			vErr.Severity = validation.SeverityWarning
			s.unfollowed++
		}
		if vErr.Rule == validation.RuleValidationUnsupportedFormat {
			s.unvalidatedFormats++
		}

		switch {
		case vErr.Severity.Rank() >= validation.SeverityError.Rank():
			s.errors++
		case vErr.Severity == validation.SeverityWarning:
			s.warnings++
		default:
			s.hints++
		}
	}

	validation.SortValidationErrors(s.errs)
	return s
}

func (s validationSummary) print(w io.Writer) {
	if s.errors > 0 {
		fmt.Fprintf(w, "❌ AsyncAPI document is invalid - %d errors", s.errors)
	} else {
		fmt.Fprintf(w, "✅ AsyncAPI document is valid - 0 errors")
	}
	if s.warnings > 0 {
		fmt.Fprintf(w, ", %d warnings", s.warnings)
	}
	if s.hints > 0 {
		fmt.Fprintf(w, ", %d hints", s.hints)
	}

	if len(s.errs) == 0 {
		fmt.Fprintln(w)
		return
	}

	fmt.Fprintf(w, ":\n\n")
	printErrors(w, s.errs)

	if s.unfollowed == 0 && s.unvalidatedFormats == 0 {
		return
	}
	fmt.Fprintf(w, "\nNot checked:\n")
	if s.unfollowed > 0 {
		fmt.Fprintf(w, "- %d references to other documents were not followed, use --resolve to validate them\n", s.unfollowed)
	}
	if s.unvalidatedFormats > 0 {
		fmt.Fprintf(w, "- %d schemas or bindings are in formats that are preserved but not validated\n", s.unvalidatedFormats)
	}
}

func printErrors(w io.Writer, errs []error) {
	for i, err := range errs {
		fmt.Fprintf(w, "%d. %s\n", i+1, err.Error())
	}
}

// GetValidateCommand returns the validate command for external use
func GetValidateCommand() *cobra.Command {
	return validateCmd
}
