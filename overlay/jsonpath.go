package overlay

import (
	"fmt"
	"unsafe"

	"github.com/speakeasy-api/jsonpath/pkg/jsonpath"
	"github.com/speakeasy-api/jsonpath/pkg/jsonpath/config"
	"github.com/speakeasy-api/openapi/internal/version"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"go.yaml.in/yaml/v4"
	yamlv3 "gopkg.in/yaml.v3"
)

// Queryable is an interface for querying YAML nodes using JSONPath expressions.
type Queryable interface {
	Query(root *yaml.Node) []*yaml.Node
}

// nodeV4toV3 converts a *yaml.v4.Node to a *yaml.v3.Node via unsafe pointer cast.
// This is safe because v4.Node is a strict superset of v3.Node — the first 12 fields
// have identical types, sizes, and offsets. The v3 code never accesses the extra v4 fields.
func nodeV4toV3(n *yaml.Node) *yamlv3.Node {
	return (*yamlv3.Node)(unsafe.Pointer(n)) //nolint:gosec // v4.Node is a strict superset of v3.Node (verified via reflect)
}

// nodesV3toV4 converts a []*yaml.v3.Node slice to []*yaml.v4.Node.
// The underlying pointer types have identical memory layouts so the slice header
// can be reinterpreted directly.
func nodesV3toV4(nodes []*yamlv3.Node) []*yaml.Node {
	return *(*[]*yaml.Node)(unsafe.Pointer(&nodes)) //nolint:gosec // pointer types have identical memory layouts
}

type yamlPathQueryable struct {
	path *yamlpath.Path
}

func (y yamlPathQueryable) Query(root *yaml.Node) []*yaml.Node {
	if y.path == nil {
		return []*yaml.Node{}
	}
	// errors aren't actually possible from yamlpath.
	result, _ := y.path.Find(nodeV4toV3(root))
	return nodesV3toV4(result)
}

// rfcJSONPathQueryable wraps a jsonpath.JSONPath to implement Queryable with v4 nodes.
type rfcJSONPathQueryable struct {
	path *jsonpath.JSONPath
}

func (r rfcJSONPathQueryable) Query(root *yaml.Node) []*yaml.Node {
	return nodesV3toV4(r.path.Query(nodeV4toV3(root)))
}

// NewPath creates a new JSONPath queryable from the given target expression.
// The implementation used depends on the overlay version and JSONPathVersion setting:
// - For version 1.0.0: Legacy yamlpath by default, opt-IN to RFC 9535 via "rfc9535"
// - For version 1.1.0+: RFC 9535 by default, opt-OUT to legacy via "legacy"
func (o *Overlay) NewPath(target string, warnings *[]string) (Queryable, error) {
	rfcJSONPath, rfcJSONPathErr := jsonpath.NewPath(target, config.WithPropertyNameExtension())
	if o.UsesRFC9535() {
		if rfcJSONPathErr != nil {
			return nil, rfcJSONPathErr
		}
		return rfcJSONPathQueryable{path: rfcJSONPath}, nil
	}

	// For version < 1.1.0 without explicit rfc9535, warn about future incompatibility
	if rfcJSONPathErr != nil && warnings != nil {
		*warnings = append(*warnings, fmt.Sprintf(
			"invalid rfc9535 jsonpath %s: %s\n"+
				"This will be treated as an error in Overlay 1.1.0+. "+
				"Please fix and opt into the new implementation with `\"x-speakeasy-jsonpath\": rfc9535` "+
				"in the root of your overlay, or upgrade to overlay version 1.1.0. "+
				"See overlay.speakeasy.com for an implementation playground.",
			target, rfcJSONPathErr.Error()))
	}

	path, err := yamlpath.NewPath(target)
	return mustExecute(path), err
}

// UsesRFC9535 determines if the overlay should use RFC 9535 JSONPath implementation.
//
// The behavior depends on the overlay version:
//   - For version 1.0.x: RFC 9535 is opt-IN (default is legacy)
//   - Set JSONPathVersion to "rfc9535" to enable RFC 9535
//   - For version 1.1.0+: RFC 9535 is the DEFAULT (opt-OUT available)
//   - Set JSONPathVersion to "legacy" to use legacy implementation
//
// Explicit settings always take precedence over version-based defaults.
func (o *Overlay) UsesRFC9535() bool {
	// Explicit opt-in always works (for both 1.0.0 and 1.1.0)
	if o.JSONPathVersion == JSONPathRFC9535 {
		return true
	}

	// Explicit opt-out always works (for both versions)
	if o.JSONPathVersion == JSONPathLegacy {
		return false
	}

	// No explicit setting - determine based on version
	// For version 1.1.0+, RFC 9535 is the default
	overlayVersion, err := version.Parse(o.Version)
	if err != nil {
		return false // Invalid version, use legacy behavior for safety
	}

	v110 := version.MustParse("1.1.0")
	return !overlayVersion.LessThan(*v110)
}

func mustExecute(path *yamlpath.Path) yamlPathQueryable {
	return yamlPathQueryable{path}
}
