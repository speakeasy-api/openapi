package rules

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
	"go.yaml.in/yaml/v4"
)

const RuleStyleTagsAlphabetical = "style-tags-alphabetical"

type TagsAlphabeticalRule struct{}

func (r *TagsAlphabeticalRule) ID() string {
	return RuleStyleTagsAlphabetical
}

func (r *TagsAlphabeticalRule) Description() string {
	return "Tags should be listed in alphabetical order to improve documentation organization and navigation. Alphabetical ordering makes it easier for developers to find specific tag groups in API documentation."
}

func (r *TagsAlphabeticalRule) Summary() string {
	return "Tags should be listed in alphabetical order."
}

func (r *TagsAlphabeticalRule) HowToFix() string {
	return "Sort the tags array alphabetically by tag name."
}

func (r *TagsAlphabeticalRule) Category() string {
	return CategoryStyle
}

func (r *TagsAlphabeticalRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}

func (r *TagsAlphabeticalRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#style-tags-alphabetical"
}

func (r *TagsAlphabeticalRule) Versions() []string {
	return nil // applies to all versions
}

func (r *TagsAlphabeticalRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil {
		return nil
	}

	var errs []error
	doc := docInfo.Document

	tags := doc.GetTags()
	if len(tags) < 2 {
		return nil // Need at least 2 tags to check ordering
	}

	// Check if tags are in alphabetical order by name
	for i := 0; i < len(tags)-1; i++ {
		currentTag := tags[i]
		nextTag := tags[i+1]

		if currentTag == nil || nextTag == nil {
			continue
		}

		currentName := currentTag.GetName()
		nextName := nextTag.GetName()

		// Compare case-insensitively
		if strings.Compare(strings.ToLower(currentName), strings.ToLower(nextName)) > 0 {
			// Get the node for the tags array
			tagsNode := doc.GetCore().Tags.ValueNode
			if tagsNode == nil {
				tagsNode = doc.GetRootNode()
			}

			errs = append(errs, &validation.Error{
				UnderlyingError: fmt.Errorf("tag `%s` must be placed before `%s` (alphabetical)", nextName, currentName),
				Node:            tagsNode,
				Severity:        config.GetSeverity(r.DefaultSeverity()),
				Rule:            RuleStyleTagsAlphabetical,
				Fix:             &sortTagsFix{tagsNode: tagsNode},
			})
			// Report only the first violation for deterministic behavior
			break
		}
	}

	return errs
}

// sortTagsFix sorts the tags sequence node alphabetically by tag name.
type sortTagsFix struct {
	tagsNode *yaml.Node
}

func (f *sortTagsFix) Description() string          { return "Sort tags alphabetically" }
func (f *sortTagsFix) Interactive() bool            { return false }
func (f *sortTagsFix) Prompts() []validation.Prompt { return nil }
func (f *sortTagsFix) SetInput([]string) error      { return nil }
func (f *sortTagsFix) Apply(doc any) error          { return nil }

func (f *sortTagsFix) ApplyNode(_ *yaml.Node) error {
	if f.tagsNode == nil || f.tagsNode.Kind != yaml.SequenceNode || len(f.tagsNode.Content) < 2 {
		return nil
	}
	sort.SliceStable(f.tagsNode.Content, func(i, j int) bool {
		nameI := getTagName(f.tagsNode.Content[i])
		nameJ := getTagName(f.tagsNode.Content[j])
		return strings.ToLower(nameI) < strings.ToLower(nameJ)
	})
	return nil
}

// getTagName extracts the "name" field value from a tag mapping node.
func getTagName(node *yaml.Node) string {
	if node == nil || node.Kind != yaml.MappingNode {
		return ""
	}
	_, valueNode, found := yml.GetMapElementNodes(context.Background(), node, "name")
	if !found || valueNode == nil {
		return ""
	}
	return valueNode.Value
}
