package tests

import (
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/marshaller/tests/core"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/values"
	valuescore "github.com/speakeasy-api/openapi/values/core"
	"gopkg.in/yaml.v3"
)

// High-level model counterparts for population testing using marshaller.Model
type TestPrimitiveHighModel struct {
	marshaller.Model[core.TestPrimitiveModel]
	StringField     string
	StringPtrField  *string
	BoolField       bool
	BoolPtrField    *bool
	IntField        int
	IntPtrField     *int
	Float64Field    float64
	Float64PtrField *float64
	Extensions      *extensions.Extensions
}

type TestComplexHighModel struct {
	marshaller.Model[core.TestComplexModel]
	NestedModel            *TestPrimitiveHighModel
	NestedModelValue       TestPrimitiveHighModel
	ArrayField             []string
	NodeArrayField         []string
	StructArrayField       []*TestPrimitiveHighModel
	MapPrimitiveField      *sequencedmap.Map[string, string]
	MapNodeField           *sequencedmap.Map[string, string]
	MapStructField         *sequencedmap.Map[string, *TestPrimitiveHighModel]
	EitherField            *values.EitherValue[string, string, int, int]
	EitherModelOrPrimitive *values.EitherValue[TestPrimitiveHighModel, core.TestPrimitiveModel, int, int]
	RawNodeField           *yaml.Node
	ValueField             valuescore.Value
	ValuesField            []valuescore.Value
	Extensions             *extensions.Extensions
}

type TestEmbeddedMapHighModel struct {
	marshaller.Model[core.TestEmbeddedMapModel]
	sequencedmap.Map[string, string]
}

type TestEmbeddedMapWithFieldsHighModel struct {
	marshaller.Model[core.TestEmbeddedMapWithFieldsModel]
	sequencedmap.Map[string, *TestPrimitiveHighModel]
	NameField  string
	Extensions *extensions.Extensions
}

type TestValidationHighModel struct {
	marshaller.Model[core.TestValidationModel]
	RequiredField  string
	OptionalField  *string
	RequiredArray  []string
	OptionalArray  []string
	RequiredStruct *TestPrimitiveHighModel
	OptionalStruct *TestPrimitiveHighModel
	Extensions     *extensions.Extensions
}

type TestEitherValueHighModel struct {
	marshaller.Model[core.TestEitherValueModel]
	StringOrInt    *values.EitherValue[string, string, int, int]
	ArrayOrString  *values.EitherValue[[]string, []string, string, string]
	StructOrString *values.EitherValue[TestPrimitiveHighModel, core.TestPrimitiveModel, string, string]
	Extensions     *extensions.Extensions
}

type TestRequiredPointerHighModel struct {
	marshaller.Model[core.TestRequiredPointerModel]
	RequiredPtr *string
	OptionalPtr *string
	Extensions  *extensions.Extensions
}

type TestRequiredNilableHighModel struct {
	marshaller.Model[core.TestRequiredNilableModel]
	RequiredPtr     *string
	RequiredSlice   []string
	RequiredMap     *sequencedmap.Map[string, string]
	RequiredStruct  *TestPrimitiveHighModel
	RequiredEither  *values.EitherValue[string, string, int, int]
	RequiredRawNode *yaml.Node
	OptionalPtr     *string
	OptionalSlice   []string
	OptionalMap     *sequencedmap.Map[string, string]
	OptionalStruct  *TestPrimitiveHighModel
	Extensions      *extensions.Extensions
}

// HTTPMethod represents a custom string type for HTTP methods (like openapi.HTTPMethod)
type HTTPMethod string

const (
	HTTPMethodGet  HTTPMethod = "get"
	HTTPMethodPost HTTPMethod = "post"
	HTTPMethodPut  HTTPMethod = "put"
)

// TestTypeConversionHighModel represents high-level model with HTTPMethod keys (like openapi.PathItem)
// This reproduces the issue where high-level model expects HTTPMethod keys but core provides string keys
type TestTypeConversionHighModel struct {
	marshaller.Model[core.TestTypeConversionCoreModel]
	sequencedmap.Map[HTTPMethod, *TestPrimitiveHighModel]
	HTTPMethodField *HTTPMethod
	Extensions      *extensions.Extensions
}

// TestEmbeddedMapPointerHighModel represents high-level model with pointer embedded sequenced map
// This tests the legacy pointer embed pattern to ensure backward compatibility
type TestEmbeddedMapPointerHighModel struct {
	marshaller.Model[core.TestEmbeddedMapPointerModel]
	*sequencedmap.Map[string, string]
}

// TestEmbeddedMapWithFieldsPointerHighModel represents high-level model with pointer embedded sequenced map and additional fields
// This tests the legacy pointer embed pattern with fields to ensure backward compatibility
type TestEmbeddedMapWithFieldsPointerHighModel struct {
	marshaller.Model[core.TestEmbeddedMapWithFieldsPointerModel]
	*sequencedmap.Map[string, *TestPrimitiveHighModel]
	NameField  string
	Extensions *extensions.Extensions
}
