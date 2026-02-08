package rules

import (
	"context"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleStyleOAS3HostNotExample = "style-oas3-host-not-example"

type OAS3HostNotExampleRule struct{}

func (r *OAS3HostNotExampleRule) ID() string       { return RuleStyleOAS3HostNotExample }
func (r *OAS3HostNotExampleRule) Category() string { return CategoryStyle }
func (r *OAS3HostNotExampleRule) Description() string {
	return "Server URLs should not point to example.com domains, which are reserved for documentation purposes. Production API specifications should reference actual server endpoints where the API is hosted."
}
func (r *OAS3HostNotExampleRule) Summary() string {
	return "Server URLs should not point to example.com domains."
}
func (r *OAS3HostNotExampleRule) HowToFix() string {
	return "Replace example.com server URLs with real API endpoints."
}
func (r *OAS3HostNotExampleRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#style-oas3-host-not-example"
}
func (r *OAS3HostNotExampleRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *OAS3HostNotExampleRule) Versions() []string {
	// Applies to all OAS3 versions
	return nil
}

func (r *OAS3HostNotExampleRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
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

		url := strings.ToLower(server.GetURL())
		if !strings.Contains(url, "example.com") {
			continue
		}

		errNode := GetFieldValueNode(server, "url", doc)
		if errNode == nil {
			errNode = doc.GetRootNode()
		}

		errs = append(errs, &validation.Error{
			UnderlyingError: fmt.Errorf("server url %q must not point at example.com", server.GetURL()),
			Node:            errNode,
			Severity:        config.GetSeverity(r.DefaultSeverity()),
			Rule:            RuleStyleOAS3HostNotExample,
			Fix:             &replaceServerURLFix{urlNode: errNode},
		})
	}

	return errs
}
