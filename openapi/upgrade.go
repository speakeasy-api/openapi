package openapi

import (
	"context"
	"slices"
	"strings"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"gopkg.in/yaml.v3"
)

type UpgradeOptions struct {
	upgradeSamePatchVersion bool
}

// WithUpgradeSamePatchVersion will upgrade the same patch version of the OpenAPI document. For example 3.1.0 to 3.1.1.
func WithUpgradeSamePatchVersion() Option[UpgradeOptions] {
	return func(uo *UpgradeOptions) {
		uo.upgradeSamePatchVersion = true
	}
}

// Upgrade upgrades any OpenAPI 3x document to OpenAPI 3.1.1 (the latest version currently supported).
// It currently won't resolve any external references, so only this document itself will be upgraded.
func Upgrade(ctx context.Context, doc *OpenAPI, opts ...Option[UpgradeOptions]) (bool, error) {
	if doc == nil {
		return false, nil
	}

	o := UpgradeOptions{}
	for _, opt := range opts {
		opt(&o)
	}

	// Only upgrade if:
	// 1. Document is 3.0.x (always upgrade these)
	// 2. Document is 3.1.x and upgradeSamePatchVersion is true (upgrade to 3.1.1)
	if strings.HasPrefix(doc.OpenAPI, "3.0") {
		// Always upgrade 3.0.x versions
	} else if strings.HasPrefix(doc.OpenAPI, "3.1") && o.upgradeSamePatchVersion && doc.OpenAPI != Version {
		// Upgrade 3.1.x versions to 3.1.1 if option is set and not already 3.1.1
	} else {
		// Don't upgrade other versions
		return false, nil
	}

	for item := range Walk(ctx, doc) {
		_ = item.Match(Matcher{
			OpenAPI: func(o *OpenAPI) error {
				o.OpenAPI = Version
				return nil
			},
			Schema: func(js *oas3.JSONSchema[oas3.Referenceable]) error {
				upgradeSchema(js)
				return nil
			},
		})
	}

	_, err := marshaller.Sync(ctx, doc)
	return true, err
}

func upgradeSchema(js *oas3.JSONSchema[oas3.Referenceable]) {
	if js == nil || js.IsReference() || js.IsRight() {
		return
	}

	schema := js.GetResolvedSchema().GetLeft()

	upgradeExample(schema)
	upgradeExclusiveMinMax(schema)
	upgradeNullableSchema(schema)
}

func upgradeExample(schema *oas3.Schema) {
	if schema == nil || schema.Example == nil {
		return
	}

	if schema.Examples == nil {
		schema.Examples = []*yaml.Node{}
	}

	schema.Examples = append(schema.Examples, schema.Example)
	schema.Example = nil
}

func upgradeExclusiveMinMax(schema *oas3.Schema) {
	if schema.ExclusiveMaximum != nil && schema.ExclusiveMaximum.IsLeft() {
		if schema.Maximum == nil || !*schema.ExclusiveMaximum.GetLeft() {
			schema.ExclusiveMaximum = nil
		} else {
			schema.ExclusiveMaximum = oas3.NewExclusiveMaximumFromFloat64(*schema.Maximum)
			schema.Maximum = nil
		}
	}

	if schema.ExclusiveMinimum != nil && schema.ExclusiveMinimum.IsLeft() {
		if schema.Minimum == nil || !*schema.ExclusiveMinimum.GetLeft() {
			schema.ExclusiveMinimum = nil
		} else {
			schema.ExclusiveMinimum = oas3.NewExclusiveMinimumFromFloat64(*schema.Minimum)
			schema.Minimum = nil
		}
	}
}

func upgradeNullableSchema(schema *oas3.Schema) {
	if schema == nil {
		return
	}

	if schema.Nullable == nil || !*schema.Nullable {
		schema.Nullable = nil // clear it out if it was set to false
		return
	}

	schema.Nullable = nil

	switch {
	case len(schema.GetType()) > 0:
		if !slices.Contains(schema.GetType(), "null") {
			schema.Type = oas3.NewTypeFromArray(append(schema.GetType(), "null"))
		}
	case len(schema.AnyOf) > 0:
		nullSchema := createNullSchema()
		schema.AnyOf = append(schema.AnyOf, nullSchema)
	case len(schema.OneOf) > 0:
		nullSchema := createNullSchema()
		schema.OneOf = append(schema.OneOf, nullSchema)
	default:
		nullSchema := createNullSchema()
		clone := *schema
		newSchema := oas3.Schema{}
		newSchema.OneOf = []*oas3.JSONSchema[oas3.Referenceable]{nullSchema, oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&clone)}
		*schema = newSchema
	}
}

func createNullSchema() *oas3.JSONSchema[oas3.Referenceable] {
	return oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
		Type: oas3.NewTypeFromArray([]oas3.SchemaType{oas3.SchemaTypeNull}),
	})
}
