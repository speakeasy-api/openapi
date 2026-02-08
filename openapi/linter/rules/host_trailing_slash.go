package rules

import (
	"context"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
	"gopkg.in/yaml.v3"
)

const RuleStyleOAS3HostTrailingSlash = "style-oas3-host-trailing-slash"

type OAS3HostTrailingSlashRule struct{}

func (r *OAS3HostTrailingSlashRule) ID() string {
	return RuleStyleOAS3HostTrailingSlash
}

func (r *OAS3HostTrailingSlashRule) Description() string {
	return "Server URLs should not end with a trailing slash to avoid ambiguity when combining with path templates. Trailing slashes can lead to double slashes in final URLs when paths are appended, potentially causing routing issues."
}

func (r *OAS3HostTrailingSlashRule) Summary() string {
	return "Server URLs should not end with a trailing slash."
}

func (r *OAS3HostTrailingSlashRule) HowToFix() string {
	return "Remove trailing slashes from server URLs."
}

func (r *OAS3HostTrailingSlashRule) Category() string {
	return CategoryStyle
}

func (r *OAS3HostTrailingSlashRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}

func (r *OAS3HostTrailingSlashRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#style-oas3-host-trailing-slash"
}

func (r *OAS3HostTrailingSlashRule) Versions() []string {
	return []string{"3.0.0", "3.0.1", "3.0.2", "3.0.3", "3.1.0", "3.1.1", "3.2.0"}
}

func (r *OAS3HostTrailingSlashRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	doc := docInfo.Document
	var errs []error

	for _, serverNode := range docInfo.Index.Servers {
		server := serverNode.Node
		if server == nil {
			continue
		}

		url := server.GetURL()
		if strings.HasSuffix(url, "/") {
			errNode := GetFieldValueNode(server, "url", doc)
			if errNode == nil {
				errNode = server.GetRootNode()
			}

			errs = append(errs, &validation.Error{
				UnderlyingError: fmt.Errorf("server url %q should not have a trailing slash", url),
				Node:            errNode,
				Severity:        config.GetSeverity(r.DefaultSeverity()),
				Rule:            RuleStyleOAS3HostTrailingSlash,
				Fix:             &removeHostTrailingSlashFix{node: errNode},
			})
		}
	}

	return errs
}

// removeHostTrailingSlashFix removes the trailing slash from a server URL node.
type removeHostTrailingSlashFix struct {
	node *yaml.Node
}

func (f *removeHostTrailingSlashFix) Description() string {
	return "Remove trailing slash from server URL"
}
func (f *removeHostTrailingSlashFix) Interactive() bool            { return false }
func (f *removeHostTrailingSlashFix) Prompts() []validation.Prompt { return nil }
func (f *removeHostTrailingSlashFix) SetInput([]string) error      { return nil }
func (f *removeHostTrailingSlashFix) Apply(doc any) error          { return nil }

func (f *removeHostTrailingSlashFix) ApplyNode(_ *yaml.Node) error {
	if f.node != nil {
		f.node.Value = strings.TrimRight(f.node.Value, "/")
	}
	return nil
}
