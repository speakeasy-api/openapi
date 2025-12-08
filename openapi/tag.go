package openapi

import (
	"context"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi/core"
	"github.com/speakeasy-api/openapi/validation"
)

// Tag represent the metadata for a single tag relating to operations in the API.
type Tag struct {
	marshaller.Model[core.Tag]

	// The name of the tag.
	Name string
	// A short summary of the tag, used for display purposes.
	Summary *string
	// A description for the tag. May contain CommonMark syntax.
	Description *string
	// External documentation for this tag.
	ExternalDocs *oas3.ExternalDocumentation
	// The name of a tag that this tag is nested under. The named tag must exist in the API description.
	Parent *string
	// A machine-readable string to categorize what sort of tag it is.
	Kind *string

	// Extensions provides a list of extensions to the Tag object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Tag] = (*Tag)(nil)

// GetName returns the value of the Name field. Returns empty string if not set.
func (t *Tag) GetName() string {
	if t == nil {
		return ""
	}
	return t.Name
}

// GetDescription returns the value of the Description field. Returns empty string if not set.
func (t *Tag) GetDescription() string {
	if t == nil || t.Description == nil {
		return ""
	}
	return *t.Description
}

// GetExternalDocs returns the value of the ExternalDocs field. Returns nil if not set.
func (t *Tag) GetExternalDocs() *oas3.ExternalDocumentation {
	if t == nil {
		return nil
	}
	return t.ExternalDocs
}

// GetSummary returns the value of the Summary field. Returns empty string if not set.
func (t *Tag) GetSummary() string {
	if t == nil || t.Summary == nil {
		return ""
	}
	return *t.Summary
}

// GetParent returns the value of the Parent field. Returns empty string if not set.
func (t *Tag) GetParent() string {
	if t == nil || t.Parent == nil {
		return ""
	}
	return *t.Parent
}

// GetKind returns the value of the Kind field. Returns empty string if not set.
func (t *Tag) GetKind() string {
	if t == nil || t.Kind == nil {
		return ""
	}
	return *t.Kind
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (t *Tag) GetExtensions() *extensions.Extensions {
	if t == nil || t.Extensions == nil {
		return extensions.New()
	}
	return t.Extensions
}

// Validate will validate the Tag object against the OpenAPI Specification.
func (t *Tag) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := t.GetCore()
	errs := []error{}

	if core.Name.Present && t.Name == "" {
		errs = append(errs, validation.NewValueError(validation.NewMissingValueError("tag.name is required"), core, core.Name))
	}

	if t.ExternalDocs != nil {
		errs = append(errs, t.ExternalDocs.Validate(ctx, opts...)...)
	}

	// Get OpenAPI object from validation options to check parent relationships
	o := validation.NewOptions(opts...)
	openapi := validation.GetContextObject[OpenAPI](o)

	// If we have an OpenAPI object with tags, validate parent relationships
	if openapi != nil && openapi.Tags != nil && t.Parent != nil && *t.Parent != "" {
		allTags := openapi.Tags

		// Check if parent tag exists
		parentExists := false
		for _, tag := range allTags {
			if tag != nil && tag.Name == *t.Parent {
				parentExists = true
				break
			}
		}

		if !parentExists {
			errs = append(errs, validation.NewValueError(
				validation.NewMissingValueError("parent tag '%s' does not exist", *t.Parent),
				core, core.Parent))
		}

		// Check for circular references
		if t.hasCircularParentReference(allTags, make(map[string]bool)) {
			errs = append(errs, validation.NewValueError(
				validation.NewValueValidationError("circular parent reference detected for tag '%s'", t.Name),
				core, core.Parent))
		}
	}

	t.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

// hasCircularParentReference checks if this tag has a circular parent reference
func (t *Tag) hasCircularParentReference(allTags []*Tag, visited map[string]bool) bool {
	if t == nil || t.Parent == nil || *t.Parent == "" {
		return false
	}

	// If we've already visited this tag, we have a circular reference
	if visited[t.Name] {
		return true
	}

	// Mark this tag as visited
	visited[t.Name] = true

	// Find the parent tag and recursively check
	for _, tag := range allTags {
		if tag != nil && tag.Name == *t.Parent {
			return tag.hasCircularParentReference(allTags, visited)
		}
	}

	return false
}
