package rules

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleStyleOAS3APIServers = "style-oas3-api-servers"

type OAS3APIServersRule struct{}

func (r *OAS3APIServersRule) ID() string {
	return RuleStyleOAS3APIServers
}

func (r *OAS3APIServersRule) Description() string {
	return "OpenAPI 3.x specifications should define at least one server with a valid URL where the API can be accessed. Server definitions help API consumers understand where to send requests and support multiple environments like production, staging, and development."
}

func (r *OAS3APIServersRule) Summary() string {
	return "OpenAPI 3.x specs should define at least one valid server URL."
}

func (r *OAS3APIServersRule) HowToFix() string {
	return "Define at least one valid server URL in the servers list."
}

func (r *OAS3APIServersRule) Category() string {
	return CategoryStyle
}

func (r *OAS3APIServersRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}

func (r *OAS3APIServersRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#style-oas3-api-servers"
}

func (r *OAS3APIServersRule) Versions() []string {
	return []string{"3.0.0", "3.0.1", "3.0.2", "3.0.3", "3.1.0", "3.1.1", "3.2.0"}
}

func (r *OAS3APIServersRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil {
		return nil
	}

	doc := docInfo.Document
	var errs []error

	servers := doc.GetServers()

	// Check if servers is nil or empty (note: GetServers returns a default server if empty)
	// We need to check the actual Servers field on the document
	if len(doc.Servers) == 0 {
		// Get the root node for error reporting
		rootNode := doc.GetRootNode()
		errs = append(errs, validation.NewValidationError(
			config.GetSeverity(r.DefaultSeverity()),
			RuleStyleOAS3APIServers,
			errors.New("no servers defined for the specification"),
			rootNode,
		))
		return errs
	}

	// Check each server has a valid URL
	i := 0
	for _, server := range servers {
		if server == nil {
			continue
		}

		serverURL := server.GetURL()
		if serverURL == "" {
			errNode := GetFieldValueNode(server, "url", doc)
			if errNode == nil {
				errNode = server.GetRootNode()
			}

			errs = append(errs, validation.NewValidationError(
				config.GetSeverity(r.DefaultSeverity()),
				RuleStyleOAS3APIServers,
				errors.New("server definition is missing a URL"),
				errNode,
			))
			i++
			continue
		}

		// Skip validation for URLs with template variables
		if strings.Contains(serverURL, "{") && strings.Contains(serverURL, "}") {
			i++
			continue
		}

		// Validate URL can be parsed
		parsed, err := url.Parse(serverURL)
		if err != nil {
			errNode := GetFieldValueNode(server, "url", doc)
			if errNode == nil {
				errNode = server.GetRootNode()
			}

			errs = append(errs, validation.NewValidationError(
				config.GetSeverity(r.DefaultSeverity()),
				RuleStyleOAS3APIServers,
				fmt.Errorf("server URL %q cannot be parsed: %s", serverURL, err.Error()),
				errNode,
			))
			i++
			continue
		}

		// Check that either host or path is provided
		if parsed.Host == "" && parsed.Path == "" {
			errNode := GetFieldValueNode(server, "url", doc)
			if errNode == nil {
				errNode = server.GetRootNode()
			}

			errs = append(errs, validation.NewValidationError(
				config.GetSeverity(r.DefaultSeverity()),
				RuleStyleOAS3APIServers,
				fmt.Errorf("server URL %q is not valid: no hostname or path provided", serverURL),
				errNode,
			))
		}

		i++
	}

	return errs
}
