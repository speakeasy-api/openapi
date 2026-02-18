package overlay_test

import (
	"fmt"
	"os"
	"strings"

	"github.com/speakeasy-api/openapi/overlay"
	"github.com/speakeasy-api/openapi/overlay/loader"
	"go.yaml.in/yaml/v4"
)

// Example_applying demonstrates how to apply an overlay to an OpenAPI document.
// Shows loading an overlay specification and applying it to transform an OpenAPI document.
func Example_applying() {
	// Create temporary files for this example
	overlayContent := `overlay: 1.0.0
info:
  title: Pet Store Enhancement Overlay
  version: 1.0.0
actions:
  - target: $.info.description
    update: Enhanced pet store API with additional features`

	openAPIContent := `openapi: 3.1.0
info:
  title: Pet Store API
  version: 1.0.0
  description: A simple pet store API
paths:
  /pets:
    get:
      summary: List pets
      responses:
        '200':
          description: A list of pets`

	// Write temporary files
	overlayFile := "temp_overlay.yaml"
	openAPIFile := "temp_openapi.yaml"
	if err := os.WriteFile(overlayFile, []byte(overlayContent), 0644); err != nil {
		panic(err)
	}
	if err := os.WriteFile(openAPIFile, []byte(openAPIContent), 0644); err != nil {
		panic(err)
	}
	defer os.Remove(overlayFile)
	defer os.Remove(openAPIFile)

	// Parse the overlay
	overlayDoc, err := overlay.Parse(overlayFile)
	if err != nil {
		panic(err)
	}

	// Load the OpenAPI document
	openAPINode, err := loader.LoadSpecification(openAPIFile)
	if err != nil {
		panic(err)
	}

	// Apply the overlay to the OpenAPI document
	err = overlayDoc.ApplyTo(openAPINode)
	if err != nil {
		panic(err)
	}

	// Convert back to YAML string
	var buf strings.Builder
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	err = encoder.Encode(openAPINode)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Transformed document:\n%s", buf.String())
	// Output:
	// Transformed document:
	// openapi: 3.1.0
	// info:
	//   title: Pet Store API
	//   version: 1.0.0
	//   description: Enhanced pet store API with additional features
	// paths:
	//   /pets:
	//     get:
	//       summary: List pets
	//       responses:
	//         '200':
	//           description: A list of pets
}

// Example_creating demonstrates how to create an overlay specification programmatically.
// Shows building an overlay specification with update and remove actions.
func Example_creating() {
	// Create update value as yaml.Node
	var updateNode yaml.Node
	updateNode.SetString("Enhanced API with additional features")

	// Create an overlay with update and remove actions
	overlayDoc := &overlay.Overlay{
		Version: "1.0.0",
		Info: overlay.Info{
			Title:   "API Enhancement Overlay",
			Version: "1.0.0",
		},
		Actions: []overlay.Action{
			{
				Target: "$.info.description",
				Update: updateNode,
			},
			{
				Target: "$.paths['/deprecated-endpoint']",
				Remove: true,
			},
		},
	}

	result, err := overlayDoc.ToString()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Overlay specification:\n%s", result)
	// Output:
	// Overlay specification:
	// overlay: 1.0.0
	// info:
	//   title: API Enhancement Overlay
	//   version: 1.0.0
	// actions:
	//   - target: $.info.description
	//     update: Enhanced API with additional features
	//   - target: $.paths['/deprecated-endpoint']
	//     remove: true
}

// Example_parsing demonstrates how to parse an overlay specification from a file.
// Shows loading an overlay file and accessing its properties.
func Example_parsing() {
	overlayContent := `overlay: 1.0.0
info:
  title: API Modification Overlay
  version: 1.0.0
actions:
  - target: $.info.title
    update: Enhanced Pet Store API
  - target: $.info.version
    update: 2.0.0`

	// Write temporary file
	overlayFile := "temp_overlay.yaml"
	if err := os.WriteFile(overlayFile, []byte(overlayContent), 0644); err != nil {
		panic(err)
	}
	defer func() { _ = os.Remove(overlayFile) }()

	overlayDoc, err := overlay.Parse(overlayFile)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Overlay Version: %s\n", overlayDoc.Version)
	fmt.Printf("Title: %s\n", overlayDoc.Info.Title)
	fmt.Printf("Number of Actions: %d\n", len(overlayDoc.Actions))

	for i, action := range overlayDoc.Actions {
		fmt.Printf("Action %d Target: %s\n", i+1, action.Target)
	}
	// Output:
	// Overlay Version: 1.0.0
	// Title: API Modification Overlay
	// Number of Actions: 2
	// Action 1 Target: $.info.title
	// Action 2 Target: $.info.version
}

// Example_validating demonstrates how to validate an overlay specification.
// Shows loading and validating an overlay specification for correctness.
func Example_validating() {
	// Invalid overlay specification (missing required fields)
	invalidOverlay := `overlay: 1.0.0
info:
  title: Invalid Overlay
actions:
  - target: $.info.title
    description: Missing update or remove`

	// Write temporary file
	overlayFile := "temp_invalid_overlay.yaml"
	if err := os.WriteFile(overlayFile, []byte(invalidOverlay), 0644); err != nil {
		panic(err)
	}
	defer func() { _ = os.Remove(overlayFile) }()

	overlayDoc, err := overlay.Parse(overlayFile)
	if err != nil {
		fmt.Printf("Parse error: %s\n", err.Error())
		return
	}

	validationErr := overlayDoc.Validate()
	if validationErr != nil {
		fmt.Println("Validation errors:")
		fmt.Printf("  %s\n", validationErr.Error())
	} else {
		fmt.Println("Overlay specification is valid!")
	}
	// Output:
	// Validation errors:
	//   overlay info version must be defined
}

// Example_removing demonstrates how to use remove actions in overlays.
// Shows removing specific paths and properties from an OpenAPI document.
func Example_removing() {
	// Sample OpenAPI document with endpoints to remove
	openAPIContent := `openapi: 3.1.0
info:
  title: API
  version: 1.0.0
paths:
  /users:
    get:
      summary: List users
  /users/{id}:
    get:
      summary: Get user
  /admin:
    get:
      summary: Admin endpoint
      deprecated: true`

	// Overlay to remove deprecated endpoints
	overlayContent := `overlay: 1.0.0
info:
  title: Cleanup Overlay
  version: 1.0.0
actions:
  - target: $.paths['/admin']
    remove: true`

	// Write temporary files
	openAPIFile := "temp_openapi.yaml"
	overlayFile := "temp_overlay.yaml"
	if err := os.WriteFile(openAPIFile, []byte(openAPIContent), 0644); err != nil {
		panic(err)
	}
	if err := os.WriteFile(overlayFile, []byte(overlayContent), 0644); err != nil {
		panic(err)
	}
	defer func() { _ = os.Remove(openAPIFile) }()
	defer func() { _ = os.Remove(overlayFile) }()

	overlayDoc, err := overlay.Parse(overlayFile)
	if err != nil {
		panic(err)
	}

	openAPINode, err := loader.LoadSpecification(openAPIFile)
	if err != nil {
		panic(err)
	}

	err = overlayDoc.ApplyTo(openAPINode)
	if err != nil {
		panic(err)
	}

	var buf strings.Builder
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	err = encoder.Encode(openAPINode)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Document after removing deprecated endpoint:\n%s", buf.String())
	// Output:
	// Document after removing deprecated endpoint:
	// openapi: 3.1.0
	// info:
	//   title: API
	//   version: 1.0.0
	// paths:
	//   /users:
	//     get:
	//       summary: List users
	//   /users/{id}:
	//     get:
	//       summary: Get user
}
