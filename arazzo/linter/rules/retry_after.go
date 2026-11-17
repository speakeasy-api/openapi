package rules

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleStyleRetryAfter = "style-retry-after"

type RetryAfterRule struct{}

func (r *RetryAfterRule) ID() string       { return RuleStyleRetryAfter }
func (r *RetryAfterRule) Category() string { return CategoryStyle }
func (r *RetryAfterRule) Description() string {
	return "Retry failure actions should set retryAfter. Without it the step is retried immediately, which rarely helps with the transient failures retries are meant for (rate limits, eventual consistency) and adds load to a struggling API."
}
func (r *RetryAfterRule) Summary() string {
	return "Retry failure actions should set retryAfter."
}
func (r *RetryAfterRule) HowToFix() string {
	return "Set retryAfter to the number of seconds to wait before retrying."
}
func (r *RetryAfterRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/arazzo/linter/README.md#style-retry-after"
}
func (r *RetryAfterRule) DefaultSeverity() validation.Severity { return validation.SeverityHint }
func (r *RetryAfterRule) Versions() []string                   { return nil }

func (r *RetryAfterRule) Run(_ context.Context, docInfo *linter.DocumentInfo[*arazzo.Arazzo], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil {
		return nil
	}

	var errs []error
	check := func(action *arazzo.FailureAction) {
		if action == nil || action.Type != arazzo.FailureActionTypeRetry || action.RetryAfter != nil {
			return
		}
		core := action.GetCore()
		errs = append(errs, validation.NewValueError(config.GetSeverity(r.DefaultSeverity()), RuleStyleRetryAfter, fmt.Errorf("retry failureAction `%s` has no retryAfter, so it retries immediately", action.Name), core, core.Type))
	}

	a := docInfo.Document
	for _, w := range a.Workflows {
		if w == nil {
			continue
		}
		// Referenced component actions are checked once with the components.
		for _, action := range w.FailureActions {
			if action != nil {
				check(action.Object)
			}
		}
		for _, s := range w.Steps {
			if s == nil {
				continue
			}
			for _, action := range s.OnFailure {
				if action != nil {
					check(action.Object)
				}
			}
		}
	}
	if a.Components != nil {
		for _, action := range a.Components.FailureActions.All() {
			check(action)
		}
	}

	return errs
}
