package fix

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
	"go.yaml.in/yaml/v3"
)

// Mode controls how fixes are applied.
type Mode int

const (
	// ModeNone means no fixing (normal lint).
	ModeNone Mode = iota
	// ModeAuto applies only non-interactive fixes.
	ModeAuto
	// ModeInteractive applies all fixes, prompting for interactive ones.
	ModeInteractive
)

// Options configures fix engine behavior.
type Options struct {
	// Mode controls which fixes are applied.
	Mode Mode
	// DryRun when true reports what would be fixed without applying changes.
	// Acts as a modifier on ModeAuto or ModeInteractive.
	DryRun bool
}

// SkipReason explains why a fix was skipped.
type SkipReason int

const (
	// SkipInteractive means the fix requires user input but the mode is non-interactive.
	SkipInteractive SkipReason = iota
	// SkipConflict means another fix already modified the same location.
	SkipConflict
	// SkipUser means the user chose to skip the fix in interactive mode.
	SkipUser
)

// AppliedFix records a successfully applied fix.
type AppliedFix struct {
	Error  *validation.Error
	Fix    validation.Fix
	Before string // populated from ChangeDescriber if implemented
	After  string // populated from ChangeDescriber if implemented
}

// SkippedFix records a fix that was skipped.
type SkippedFix struct {
	Error  *validation.Error
	Fix    validation.Fix
	Reason SkipReason
}

// FailedFix records a fix that failed to apply.
type FailedFix struct {
	Error    *validation.Error
	Fix      validation.Fix
	FixError error
}

// Result tracks what the engine did.
type Result struct {
	Applied []AppliedFix
	Skipped []SkippedFix
	Failed  []FailedFix
}

// Engine applies fixes to an OpenAPI document.
type Engine struct {
	opts     Options
	prompter validation.Prompter
	registry *FixRegistry
}

// NewEngine creates a new fix engine.
func NewEngine(opts Options, prompter validation.Prompter, registry *FixRegistry) *Engine {
	return &Engine{
		opts:     opts,
		prompter: prompter,
		registry: registry,
	}
}

// conflictKey identifies a document location and rule for conflict detection.
// Including the rule allows independent fixes from different rules at the same
// YAML node to be applied without incorrectly skipping them as conflicts.
type conflictKey struct {
	Line   int
	Column int
	Rule   string
}

// ProcessErrors takes lint output errors and applies fixes where available.
// The doc is modified in-place by successful fixes.
//
// Pipeline ordering:
//  1. Fixable errors are collected from both Error.Fix fields and the FixRegistry.
//  2. Errors are sorted by document location (line, column ascending) so fixes
//     are applied in first-in-document-order. This ensures deterministic results.
//  3. Conflict detection: the key is {line, column, rule}. If two errors from
//     the same rule share a location, only the first (by sort order) is applied.
//     Different rules CAN independently fix the same location.
//  4. Interactive fixes are skipped in ModeAuto or when no prompter is available.
//  5. In dry-run mode, fixes are recorded without modifying the document but
//     conflict detection still operates.
func (e *Engine) ProcessErrors(ctx context.Context, doc *openapi.OpenAPI, errs []error) (*Result, error) {
	if e.opts.Mode == ModeNone {
		return &Result{}, nil
	}

	// Collect fixable errors
	type fixableError struct {
		vErr *validation.Error
		fix  validation.Fix
	}

	var fixable []fixableError

	for _, err := range errs {
		var vErr *validation.Error
		if !errors.As(err, &vErr) {
			continue
		}

		fix := vErr.Fix

		// If no fix attached to the error, check the registry
		if fix == nil && e.registry != nil {
			fix = e.registry.GetFix(vErr)
		}

		if fix != nil {
			fixable = append(fixable, fixableError{vErr: vErr, fix: fix})
		}
	}

	if len(fixable) == 0 {
		return &Result{}, nil
	}

	// Sort by document location for deterministic ordering
	sort.Slice(fixable, func(i, j int) bool {
		li, ci := fixable[i].vErr.GetLineNumber(), fixable[i].vErr.GetColumnNumber()
		lj, cj := fixable[j].vErr.GetLineNumber(), fixable[j].vErr.GetColumnNumber()
		if li != lj {
			return li < lj
		}
		return ci < cj
	})

	result := &Result{}
	modified := make(map[conflictKey]bool)
	skippedRules := make(map[string]bool)
	var nodeFixes []pendingNodeFix

fixLoop:
	for _, fe := range fixable {
		fix := fe.fix
		vErr := fe.vErr

		// Skip any remaining fixes for rules the user opted to skip.
		if skippedRules[vErr.Rule] {
			result.Skipped = append(result.Skipped, SkippedFix{
				Error:  vErr,
				Fix:    fix,
				Reason: SkipUser,
			})
			continue
		}

		// Check for conflicts at the same location
		key := conflictKey{Line: vErr.GetLineNumber(), Column: vErr.GetColumnNumber(), Rule: vErr.Rule}
		if key.Line >= 0 && modified[key] {
			result.Skipped = append(result.Skipped, SkippedFix{
				Error:  vErr,
				Fix:    fix,
				Reason: SkipConflict,
			})
			continue
		}

		// Skip interactive fixes in auto mode or when no prompter is available
		if fix.Interactive() && (e.opts.Mode == ModeAuto || e.prompter == nil) {
			result.Skipped = append(result.Skipped, SkippedFix{
				Error:  vErr,
				Fix:    fix,
				Reason: SkipInteractive,
			})
			continue
		}

		// Dry-run: record what would happen without applying
		if e.opts.DryRun {
			result.Applied = append(result.Applied, makeAppliedFix(vErr, fix))
			if key.Line >= 0 {
				modified[key] = true
			}
			continue
		}

		// Handle interactive input
		if fix.Interactive() && e.prompter != nil {
			responses, err := e.prompter.PromptFix(vErr, fix)
			if err != nil {
				if errors.Is(err, validation.ErrSkipFix) {
					result.Skipped = append(result.Skipped, SkippedFix{
						Error:  vErr,
						Fix:    fix,
						Reason: SkipUser,
					})
					continue
				}
				if errors.Is(err, validation.ErrSkipRule) {
					skippedRules[vErr.Rule] = true
					result.Skipped = append(result.Skipped, SkippedFix{
						Error:  vErr,
						Fix:    fix,
						Reason: SkipUser,
					})
					continue
				}
				if errors.Is(err, validation.ErrExitInteractive) {
					break fixLoop
				}
				result.Failed = append(result.Failed, FailedFix{
					Error: vErr, Fix: fix, FixError: err,
				})
				continue
			}

			if err := fix.SetInput(responses); err != nil {
				result.Failed = append(result.Failed, FailedFix{
					Error: vErr, Fix: fix, FixError: err,
				})
				continue
			}
		}

		// Node fixes are deferred until all model fixes have been applied, as
		// marshalling syncs the model back over the YAML nodes.
		// Their locations are only marked as modified once they apply.
		if nodeFix, ok := fix.(validation.NodeFix); ok && doc.GetRootNode() != nil {
			nodeFixes = append(nodeFixes, pendingNodeFix{vErr: vErr, fix: nodeFix, key: key, index: len(result.Applied)})
			result.Applied = append(result.Applied, makeAppliedFix(vErr, fix))
			continue
		}

		// Apply the fix
		applyErr := fix.Apply(doc)

		if applyErr != nil {
			result.Failed = append(result.Failed, FailedFix{
				Error: vErr, Fix: fix, FixError: applyErr,
			})
			continue
		}

		// Mark location as modified for conflict detection
		if key.Line >= 0 {
			modified[key] = true
		}

		result.Applied = append(result.Applied, makeAppliedFix(vErr, fix))
	}

	if err := applyNodeFixes(ctx, doc, result, nodeFixes, modified); err != nil {
		return result, err
	}

	return result, nil
}

type pendingNodeFix struct {
	vErr  *validation.Error
	fix   validation.NodeFix
	key   conflictKey
	index int // position of the fix in Result.Applied
}

// applyNodeFixes applies fixes that edit the YAML node tree directly. The model
// is synced to the nodes first so earlier model fixes are kept, and the model is
// rebuilt from the edited nodes afterwards so the edits survive marshalling.
// A pending fix is skipped as a conflict only if an earlier fix for the same rule
// and location applied, so a failed node fix does not block the ones after it.
func applyNodeFixes(ctx context.Context, doc *openapi.OpenAPI, result *Result, nodeFixes []pendingNodeFix, modified map[conflictKey]bool) error {
	if len(nodeFixes) == 0 {
		return nil
	}

	rootNode, err := marshaller.Sync(ctx, doc)
	if err != nil {
		return fmt.Errorf("failed to sync document before applying node fixes: %w", err)
	}
	if rootNode == nil {
		return errors.New("failed to sync document before applying node fixes: no root node")
	}

	notApplied := make(map[int]bool)
	for _, nf := range nodeFixes {
		if nf.key.Line >= 0 && modified[nf.key] {
			notApplied[nf.index] = true
			result.Skipped = append(result.Skipped, SkippedFix{
				Error:  nf.vErr,
				Fix:    nf.fix,
				Reason: SkipConflict,
			})
			continue
		}

		if err := nf.fix.ApplyNode(rootNode); err != nil {
			notApplied[nf.index] = true
			result.Failed = append(result.Failed, FailedFix{
				Error: nf.vErr, Fix: nf.fix, FixError: err,
			})
			continue
		}

		if nf.key.Line >= 0 {
			modified[nf.key] = true
		}
	}

	if len(notApplied) > 0 {
		applied := make([]AppliedFix, 0, len(result.Applied)-len(notApplied))
		for i, af := range result.Applied {
			if !notApplied[i] {
				applied = append(applied, af)
			}
		}
		result.Applied = applied
	}

	if len(notApplied) == len(nodeFixes) {
		return nil
	}

	// Rebuild from the document node, if there is one, so its comments are kept.
	node := rootNode
	if documentNode := doc.GetCore().DocumentNode; documentNode != nil && documentNode.Kind == yaml.DocumentNode {
		documentNode.Content = []*yaml.Node{rootNode}
		node = documentNode
	}

	config := doc.GetCore().GetConfig()
	*doc = openapi.OpenAPI{}
	doc.InitCache()
	doc.GetCore().SetConfig(config)
	if _, err := marshaller.UnmarshalNode(ctx, "", node, doc); err != nil {
		return fmt.Errorf("failed to rebuild document after applying node fixes: %w", err)
	}

	return nil
}

func makeAppliedFix(vErr *validation.Error, fix validation.Fix) AppliedFix {
	af := AppliedFix{Error: vErr, Fix: fix}
	if cd, ok := fix.(validation.ChangeDescriber); ok {
		af.Before, af.After = cd.DescribeChange()
	}
	return af
}

// ApplyNodeFix is a helper that applies a NodeFix if the fix implements the interface,
// otherwise falls back to Apply.
func ApplyNodeFix(fix validation.Fix, doc *openapi.OpenAPI, rootNode *yaml.Node) error {
	if nodeFix, ok := fix.(validation.NodeFix); ok && rootNode != nil {
		return nodeFix.ApplyNode(rootNode)
	}
	return fix.Apply(doc)
}
