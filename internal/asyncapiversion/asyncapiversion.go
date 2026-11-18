// Package asyncapiversion provides the version and schema format policy shared by the asyncapi
// document package and the jsonschema/asyncapi schema package.
//
// It deliberately imports neither package so both can depend on it without an import cycle.
package asyncapiversion

import (
	"fmt"
	"mime"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Latest is the latest AsyncAPI Specification version supported, used as the default for new documents.
const Latest = "3.1.0"

// supportedMinorVersions are the `major.minor` AsyncAPI Specification versions supported.
// Kept private so callers cannot change the version policy process-wide.
var supportedMinorVersions = []string{"3.0", "3.1"}

// SupportedMinorVersions returns a copy of the `major.minor` AsyncAPI Specification versions supported.
// Any patch release (and permitted suffix) of a supported minor version is accepted, as the
// specification states the patch version is not considered by tooling.
func SupportedMinorVersions() []string {
	return slices.Clone(supportedMinorVersions)
}

// versionRegex matches the AsyncAPI version string grammar `major.minor.patch` where the patch
// may be suffixed by a hyphen and extra alphanumeric characters.
var versionRegex = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-[0-9A-Za-z]+)?$`)

// Version is a parsed AsyncAPI version string.
type Version struct {
	Major  int
	Minor  int
	Patch  int
	Suffix string
}

// Parse parses an AsyncAPI version string such as `3.1.0` or `3.0.0-rc1`.
func Parse(v string) (*Version, error) {
	m := versionRegex.FindStringSubmatch(v)
	if m == nil {
		return nil, fmt.Errorf("invalid version `%s`: expected `major.minor.patch` with an optional `-suffix`", v)
	}

	major, err := strconv.Atoi(m[1])
	if err != nil {
		return nil, fmt.Errorf("invalid major version in `%s`: %w", v, err)
	}
	minor, err := strconv.Atoi(m[2])
	if err != nil {
		return nil, fmt.Errorf("invalid minor version in `%s`: %w", v, err)
	}
	patch, err := strconv.Atoi(m[3])
	if err != nil {
		return nil, fmt.Errorf("invalid patch version in `%s`: %w", v, err)
	}

	return &Version{
		Major:  major,
		Minor:  minor,
		Patch:  patch,
		Suffix: strings.TrimPrefix(m[4], "-"),
	}, nil
}

// String returns the version string.
func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Suffix != "" {
		s += "-" + v.Suffix
	}
	return s
}

// MinorLine returns the `major.minor` line of the version.
func (v Version) MinorLine() string {
	return fmt.Sprintf("%d.%d", v.Major, v.Minor)
}

// IsSupported returns true if the version belongs to a supported minor line.
func (v Version) IsSupported() bool {
	return slices.Contains(supportedMinorVersions, v.MinorLine())
}

// IsSupported returns true if the provided AsyncAPI version string is valid and belongs to a supported minor line.
func IsSupported(v string) bool {
	parsed, err := Parse(v)
	if err != nil {
		return false
	}
	return parsed.IsSupported()
}

// SupportedVersionPatterns returns the supported versions in a human readable form, e.g. `3.0.x`.
func SupportedVersionPatterns() []string {
	patterns := make([]string, 0, len(supportedMinorVersions))
	for _, v := range supportedMinorVersions {
		patterns = append(patterns, v+".x")
	}
	return patterns
}

// SchemaFormatKind identifies the schema language named by a Multi Format Schema Object `schemaFormat`.
type SchemaFormatKind string

const (
	// SchemaFormatKindAsyncAPI is the native AsyncAPI Schema Object.
	SchemaFormatKindAsyncAPI SchemaFormatKind = "asyncapi"
	// SchemaFormatKindJSONSchema is JSON Schema.
	SchemaFormatKindJSONSchema SchemaFormatKind = "jsonschema"
	// SchemaFormatKindAvro is an Apache Avro schema.
	SchemaFormatKindAvro SchemaFormatKind = "avro"
	// SchemaFormatKindOpenAPI is an OpenAPI Schema Object.
	SchemaFormatKindOpenAPI SchemaFormatKind = "openapi"
	// SchemaFormatKindRAML is a RAML data type.
	SchemaFormatKindRAML SchemaFormatKind = "raml"
	// SchemaFormatKindProtobuf is a Protocol Buffers schema.
	SchemaFormatKindProtobuf SchemaFormatKind = "protobuf"
	// SchemaFormatKindCustom is any other (custom) schema format.
	SchemaFormatKindCustom SchemaFormatKind = "custom"
	// SchemaFormatKindInvalid is a value that is not a valid media type.
	SchemaFormatKindInvalid SchemaFormatKind = "invalid"
)

// JSONSchemaDraft07 is the only JSON Schema version listed in the AsyncAPI schema formats table.
const JSONSchemaDraft07 = "draft-07"

// SchemaFormat is a parsed `schemaFormat` value.
type SchemaFormat struct {
	// Raw is the authored value.
	Raw string
	// Kind is the schema language the media type names.
	Kind SchemaFormatKind
	// MediaType is the media type without parameters, e.g. `application/vnd.aai.asyncapi+json`.
	MediaType string
	// Version is the `version` media type parameter, if any.
	Version string
}

var knownSchemaMediaTypes = map[string]SchemaFormatKind{
	"application/vnd.aai.asyncapi":      SchemaFormatKindAsyncAPI,
	"application/vnd.aai.asyncapi+json": SchemaFormatKindAsyncAPI,
	"application/vnd.aai.asyncapi+yaml": SchemaFormatKindAsyncAPI,
	"application/schema+json":           SchemaFormatKindJSONSchema,
	"application/schema+yaml":           SchemaFormatKindJSONSchema,
	"application/vnd.apache.avro":       SchemaFormatKindAvro,
	"application/vnd.apache.avro+json":  SchemaFormatKindAvro,
	"application/vnd.apache.avro+yaml":  SchemaFormatKindAvro,
	"application/vnd.oai.openapi":       SchemaFormatKindOpenAPI,
	"application/vnd.oai.openapi+json":  SchemaFormatKindOpenAPI,
	"application/vnd.oai.openapi+yaml":  SchemaFormatKindOpenAPI,
	"application/raml+yaml":             SchemaFormatKindRAML,
	"application/vnd.google.protobuf":   SchemaFormatKindProtobuf,
}

// ParseSchemaFormat parses a `schemaFormat` value. It never fails: values that are not valid media
// types are reported with SchemaFormatKindInvalid and unrecognised media types as SchemaFormatKindCustom.
func ParseSchemaFormat(s string) SchemaFormat {
	f := SchemaFormat{Raw: s}

	mediaType, params, err := mime.ParseMediaType(s)
	if err != nil {
		f.Kind = SchemaFormatKindInvalid
		return f
	}

	f.MediaType = mediaType
	f.Version = params["version"]

	if kind, ok := knownSchemaMediaTypes[mediaType]; ok {
		f.Kind = kind
	} else {
		f.Kind = SchemaFormatKindCustom
	}

	return f
}

// DefaultSchemaFormat returns the schema format assumed when a schema has no explicit `schemaFormat`,
// which is the native AsyncAPI Schema Object for the document's AsyncAPI version.
func DefaultSchemaFormat(asyncAPIVersion string) string {
	if asyncAPIVersion == "" {
		asyncAPIVersion = Latest
	}
	return "application/vnd.aai.asyncapi+json;version=" + asyncAPIVersion
}

// IsJSONSchemaBased returns true if the format is a JSON Schema based format this module models
// with typed schema objects: the native AsyncAPI Schema Object for a supported AsyncAPI version or
// JSON Schema Draft 07.
func (f SchemaFormat) IsJSONSchemaBased() bool {
	switch f.Kind {
	case SchemaFormatKindAsyncAPI:
		return IsSupported(f.Version)
	case SchemaFormatKindJSONSchema:
		return f.Version == JSONSchemaDraft07
	default:
		return false
	}
}

// IsDraft07 returns true if the format is explicitly JSON Schema Draft 07 rather than the native AsyncAPI Schema Object.
func (f SchemaFormat) IsDraft07() bool {
	return f.Kind == SchemaFormatKindJSONSchema && f.Version == JSONSchemaDraft07
}

// IsStandardKind returns true if the format names one of the schema languages listed in the
// AsyncAPI schema formats tables. A custom format must not refer to one of these.
func (f SchemaFormat) IsStandardKind() bool {
	switch f.Kind {
	case SchemaFormatKindAsyncAPI, SchemaFormatKindJSONSchema, SchemaFormatKindAvro, SchemaFormatKindOpenAPI, SchemaFormatKindRAML, SchemaFormatKindProtobuf:
		return true
	default:
		return false
	}
}
