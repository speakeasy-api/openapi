package format

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/speakeasy-api/openapi/validation"
)

type JSONFormatter struct{}

func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

type jsonOutput struct {
	Results []jsonResult `json:"results"`
	Summary jsonSummary  `json:"summary"`
}

type jsonResult struct {
	Rule     string       `json:"rule"`
	Category string       `json:"category"`
	Severity string       `json:"severity"`
	Message  string       `json:"message"`
	Location jsonLocation `json:"location"`
	Document string       `json:"document,omitempty"`
	Fix      *jsonFix     `json:"fix,omitempty"`
}

type jsonLocation struct {
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Pointer string `json:"pointer,omitempty"` // TODO: Add pointer support
}

type jsonFix struct {
	Description string `json:"description"`
}

type jsonSummary struct {
	Total    int `json:"total"`
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Hints    int `json:"hints"`
}

func (f *JSONFormatter) Format(results []error) (string, error) {
	output := jsonOutput{
		Results: make([]jsonResult, 0, len(results)),
	}

	for _, err := range results {
		var vErr *validation.Error
		if errors.As(err, &vErr) {
			category := "unknown"
			if idx := strings.Index(vErr.Rule, "-"); idx > 0 {
				category = vErr.Rule[:idx]
			}

			result := jsonResult{
				Rule:     vErr.Rule,
				Category: category,
				Severity: vErr.Severity.String(),
				Message:  vErr.UnderlyingError.Error(),
				Location: jsonLocation{
					Line:   vErr.GetLineNumber(),
					Column: vErr.GetColumnNumber(),
				},
			}

			if vErr.DocumentLocation != "" {
				result.Document = vErr.DocumentLocation
			}

			if vErr.Fix != nil {
				result.Fix = &jsonFix{
					Description: vErr.Fix.FixDescription(),
				}
			}

			output.Results = append(output.Results, result)

			switch vErr.Severity {
			case validation.SeverityError:
				output.Summary.Errors++
			case validation.SeverityWarning:
				output.Summary.Warnings++
			case validation.SeverityHint:
				output.Summary.Hints++
			}
		} else {
			// Non-validation error
			output.Results = append(output.Results, jsonResult{
				Rule:     "internal",
				Category: "internal",
				Severity: "error",
				Message:  err.Error(),
			})
			output.Summary.Errors++
		}
	}

	output.Summary.Total = len(results)

	bytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}
