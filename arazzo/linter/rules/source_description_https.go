package rules

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleSecuritySourceDescriptionHTTPS = "security-source-description-https"

type SourceDescriptionHTTPSRule struct{}

func (r *SourceDescriptionHTTPSRule) ID() string       { return RuleSecuritySourceDescriptionHTTPS }
func (r *SourceDescriptionHTTPSRule) Category() string { return CategorySecurity }
func (r *SourceDescriptionHTTPSRule) Description() string {
	return "Source descriptions should be loaded over HTTPS. A document fetched over plain HTTP can be modified in transit, which can redirect a workflow's steps to different operations or servers. Local addresses (localhost and loopback IPs) are not reported."
}
func (r *SourceDescriptionHTTPSRule) Summary() string {
	return "Source description URLs should use HTTPS."
}
func (r *SourceDescriptionHTTPSRule) HowToFix() string {
	return "Serve the source document over HTTPS and update the sourceDescription url."
}
func (r *SourceDescriptionHTTPSRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/arazzo/linter/README.md#security-source-description-https"
}
func (r *SourceDescriptionHTTPSRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *SourceDescriptionHTTPSRule) Versions() []string { return nil }

func (r *SourceDescriptionHTTPSRule) Run(_ context.Context, docInfo *linter.DocumentInfo[*arazzo.Arazzo], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil {
		return nil
	}

	var errs []error
	for _, sd := range docInfo.Document.SourceDescriptions {
		if sd == nil || !isInsecureURL(sd.URL) {
			continue
		}
		core := sd.GetCore()
		errs = append(errs, validation.NewValueError(config.GetSeverity(r.DefaultSeverity()), RuleSecuritySourceDescriptionHTTPS, fmt.Errorf("sourceDescription `%s` url `%s` uses http instead of https", sd.Name, sd.URL), core, core.URL))
	}
	return errs
}

// isInsecureURL returns true for http URLs to hosts other than local addresses.
func isInsecureURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(u.Scheme, "http") {
		return false
	}
	host := u.Hostname()
	return host != "localhost" && host != "::1" && !strings.HasPrefix(host, "127.")
}
