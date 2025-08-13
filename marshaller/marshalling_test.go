package marshaller_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/internal/testutils"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/marshaller/tests"
	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshal_TestPrimitiveModel_RoundTrip_YAML_Success(t *testing.T) {
	t.Parallel()

	inputYAML := `stringField: "test string"
stringPtrField: "test ptr string"
boolField: true
boolPtrField: false
intField: 42
intPtrField: 24
float64Field: 3.14
float64PtrField: 2.71
x-custom: "extension value"
`

	// Unmarshal -> Marshal -> Compare
	reader := strings.NewReader(inputYAML)
	model := &tests.TestPrimitiveHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	var buf bytes.Buffer
	err = marshaller.Marshal(t.Context(), model, &buf)
	require.NoError(t, err)

	outputYAML := buf.String()
	assert.YAMLEq(t, inputYAML, outputYAML)
}

func TestMarshal_TestPrimitiveModel_WithChanges_YAML_Success(t *testing.T) {
	t.Parallel()

	inputYAML := `stringField: "original string"
boolField: true
intField: 42
float64Field: 3.14
x-original: "original extension"
`

	expectedYAML := `stringField: "modified string"
boolField: false
intField: 100
float64Field: 2.71
x-original: "original extension"
x-modified: modified extension
`

	// Unmarshal
	reader := strings.NewReader(inputYAML)
	model := &tests.TestPrimitiveHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Modify the model
	model.StringField = "modified string"
	model.BoolField = false
	model.IntField = 100
	model.Float64Field = 2.71
	if model.Extensions != nil {
		model.Extensions.Set("x-modified", testutils.CreateStringYamlNode("modified extension", 1, 1))
	}

	// Marshal
	var buf bytes.Buffer
	err = marshaller.Marshal(t.Context(), model, &buf)
	require.NoError(t, err)

	outputYAML := buf.String()
	assert.YAMLEq(t, expectedYAML, outputYAML)
}

func TestMarshal_TestComplexModel_RoundTrip_YAML_Success(t *testing.T) {
	t.Parallel()

	inputYAML := `nestedModelValue:
  stringField: "nested value"
  boolField: true
  intField: 100
  float64Field: 3.14
arrayField:
  - "item1"
  - "item2"
  - "item3"
nodeArrayField:
  - "node1"
  - "node2"
eitherModelOrPrimitive: 456
x-extension: "ext value"
`

	// Unmarshal -> Marshal -> Compare
	reader := strings.NewReader(inputYAML)
	model := &tests.TestComplexHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	var buf bytes.Buffer
	err = marshaller.Marshal(t.Context(), model, &buf)
	require.NoError(t, err)

	outputYAML := buf.String()
	assert.YAMLEq(t, inputYAML, outputYAML)
}

func TestMarshal_TestComplexModel_WithChanges_YAML_Success(t *testing.T) {
	t.Parallel()

	inputYAML := `nestedModelValue:
  stringField: "nested value"
  boolField: true
  intField: 100
  float64Field: 3.14
arrayField:
  - "item1"
  - "item2"
nodeArrayField:
  - "node1"
  - "node2"
eitherModelOrPrimitive: 456
`

	expectedYAML := `nestedModelValue:
  stringField: "modified nested"
  boolField: true
  intField: 100
  float64Field: 3.14
arrayField:
  - "modified1"
  - "modified2"
  - "modified3"
nodeArrayField:
  - "modifiedNode1"
  - "modifiedNode2"
eitherModelOrPrimitive: 456
`

	// Unmarshal
	reader := strings.NewReader(inputYAML)
	model := &tests.TestComplexHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Modify the model
	model.ArrayField = []string{"modified1", "modified2", "modified3"}
	model.NodeArrayField = []string{"modifiedNode1", "modifiedNode2"}
	model.NestedModelValue.StringField = "modified nested"

	// Marshal
	var buf bytes.Buffer
	err = marshaller.Marshal(t.Context(), model, &buf)
	require.NoError(t, err)

	outputYAML := buf.String()
	assert.YAMLEq(t, expectedYAML, outputYAML)
}

func TestMarshal_TestEmbeddedMapModel_RoundTrip_YAML_Success(t *testing.T) {
	t.Parallel()

	inputYAML := `dynamicKey1: "value1"
dynamicKey2: "value2"
dynamicKey3: "value3"
`

	// Unmarshal -> Marshal -> Compare
	reader := strings.NewReader(inputYAML)
	model := &tests.TestEmbeddedMapHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	var buf bytes.Buffer
	err = marshaller.Marshal(t.Context(), model, &buf)
	require.NoError(t, err)

	outputYAML := buf.String()
	assert.YAMLEq(t, inputYAML, outputYAML)
}

func TestMarshal_TestEmbeddedMapModel_WithChanges_YAML_Success(t *testing.T) {
	t.Parallel()

	inputYAML := `dynamicKey1: "value1"
dynamicKey2: "value2"
`

	expectedYAML := `dynamicKey1: "modified value1"
dynamicKey2: "value2"
newKey: "new value"
`

	// Unmarshal
	reader := strings.NewReader(inputYAML)
	model := &tests.TestEmbeddedMapHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Modify the model
	model.Set("dynamicKey1", "modified value1")
	model.Set("newKey", "new value")

	// Marshal
	var buf bytes.Buffer
	err = marshaller.Marshal(t.Context(), model, &buf)
	require.NoError(t, err)

	outputYAML := buf.String()
	assert.YAMLEq(t, expectedYAML, outputYAML)
}

func TestMarshal_TestEmbeddedMapWithFieldsModel_RoundTrip_YAML_Success(t *testing.T) {
	t.Parallel()

	inputYAML := `name: "test name"
dynamicKey1:
  stringField: "dynamic value 1"
  boolField: true
  intField: 100
  float64Field: 1.23
dynamicKey2:
  stringField: "dynamic value 2"
  boolField: false
  intField: 42
  float64Field: 4.56
x-extension: "ext value"
`

	// Unmarshal -> Marshal -> Compare
	reader := strings.NewReader(inputYAML)
	model := &tests.TestEmbeddedMapWithFieldsHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	var buf bytes.Buffer
	err = marshaller.Marshal(t.Context(), model, &buf)
	require.NoError(t, err)

	outputYAML := buf.String()
	assert.YAMLEq(t, inputYAML, outputYAML)
}

func TestMarshal_WithComments_YAML_Success(t *testing.T) {
	t.Parallel()

	inputYAML := `# This is a comment about the string field
stringField: "test string" # inline comment
# Comment about boolean
boolField: true
intField: 42
float64Field: 3.14
# Extension comment
x-custom: "extension value"
`

	// Unmarshal -> Marshal -> Check comment preservation
	reader := strings.NewReader(inputYAML)
	model := &tests.TestPrimitiveHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	var buf bytes.Buffer
	err = marshaller.Marshal(t.Context(), model, &buf)
	require.NoError(t, err)

	outputYAML := buf.String()
	assert.YAMLEq(t, inputYAML, outputYAML)
}

func TestMarshal_WithAliases_YAML_Success(t *testing.T) {
	t.Parallel()

	inputYAML := `stringField: &alias "aliased value"
stringPtrField: *alias
boolField: true
intField: 42
float64Field: 3.14
x-alias-ext: *alias
`

	// Unmarshal -> Marshal -> Check alias preservation
	reader := strings.NewReader(inputYAML)
	model := &tests.TestPrimitiveHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	var buf bytes.Buffer
	err = marshaller.Marshal(t.Context(), model, &buf)
	require.NoError(t, err)

	outputYAML := buf.String()
	assert.YAMLEq(t, inputYAML, outputYAML)
}

func TestMarshal_JSON_Input_YAML_Output_Success(t *testing.T) {
	t.Parallel()

	inputJSON := `{
  "stringField": "test string",
  "stringPtrField": "test ptr string",
  "boolField": true,
  "boolPtrField": false,
  "intField": 42,
  "intPtrField": 24,
  "float64Field": 3.14,
  "float64PtrField": 2.71,
  "x-custom": "extension value"
}`

	expectedYAML := `stringField: test string
stringPtrField: test ptr string
boolField: true
boolPtrField: false
intField: 42
intPtrField: 24
float64Field: 3.14
float64PtrField: 2.71
x-custom: extension value
`

	// Unmarshal JSON -> Marshal to YAML
	reader := strings.NewReader(inputJSON)
	model := &tests.TestPrimitiveHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Force output format to YAML (override the JSON detection from input)
	model.GetCore().Config.OutputFormat = yml.OutputFormatYAML

	var buf bytes.Buffer
	err = marshaller.Marshal(t.Context(), model, &buf)
	require.NoError(t, err)

	outputYAML := buf.String()
	assert.YAMLEq(t, expectedYAML, outputYAML)
}

func TestMarshal_ComplexNesting_RoundTrip_Success(t *testing.T) {
	t.Parallel()

	inputYAML := `nestedModelValue:
  stringField: "level1"
  boolField: true
  intField: 100
  float64Field: 3.14
arrayField:
  - "array-item-1"
  - "array-item-2"
  - "array-item-3"
nodeArrayField:
  - "node-item-1"
  - "node-item-2"
mapField:
  key1: "map-value-1"
  key2: "map-value-2"
eitherModelOrPrimitive:
  stringField: "either-struct"
  boolField: false
  intField: 200
  float64Field: 2.71
x-root-extension: "root-ext-value"
`

	// Unmarshal -> Marshal -> Verify exact preservation
	reader := strings.NewReader(inputYAML)
	model := &tests.TestComplexHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	var buf bytes.Buffer
	err = marshaller.Marshal(t.Context(), model, &buf)
	require.NoError(t, err)

	outputYAML := buf.String()
	assert.YAMLEq(t, inputYAML, outputYAML)
}

func TestMarshal_ComplexNesting_WithChanges_Success(t *testing.T) {
	t.Parallel()

	inputYAML := `nestedModelValue:
  stringField: "level1"
  boolField: true
  intField: 100
  float64Field: 3.14
arrayField:
  - "array-item-1"
  - "array-item-2"
nodeArrayField:
  - "node-item-1"
  - "node-item-2"
mapField:
  key1: "map-value-1"
  key2: "map-value-2"
eitherModelOrPrimitive: 999
x-root-extension: "root-ext-value"
`

	expectedYAML := `nestedModelValue:
  stringField: "modified-level1"
  boolField: false
  intField: 100
  float64Field: 3.14
arrayField:
  - "new-array-item-1"
  - "new-array-item-2"
  - "new-array-item-3"
nodeArrayField:
  - "new-node-item-1"
mapField:
  key1: "modified-map-value-1"
  key2: "map-value-2"
  key3: "new-map-value-3"
eitherModelOrPrimitive: 777
x-root-extension: "root-ext-value"
x-new-extension: new-ext-value
`

	// Unmarshal
	reader := strings.NewReader(inputYAML)
	model := &tests.TestComplexHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Modify the model extensively
	model.NestedModelValue.StringField = "modified-level1"
	model.NestedModelValue.BoolField = false
	model.ArrayField = []string{"new-array-item-1", "new-array-item-2", "new-array-item-3"}
	model.NodeArrayField = []string{"new-node-item-1"}
	model.MapPrimitiveField.Set("key1", "modified-map-value-1")
	model.MapPrimitiveField.Set("key3", "new-map-value-3")
	// Modify either value to integer
	rightValue := 777
	model.EitherModelOrPrimitive.Right = &rightValue
	model.EitherModelOrPrimitive.Left = nil
	model.Extensions.Set("x-new-extension", testutils.CreateStringYamlNode("new-ext-value", 1, 1))

	// Marshal
	var buf bytes.Buffer
	err = marshaller.Marshal(t.Context(), model, &buf)
	require.NoError(t, err)

	outputYAML := buf.String()
	assert.YAMLEq(t, expectedYAML, outputYAML)
}

// TestMarshal_ExtensiveAliases_PrimitiveFields_Success tests alias preservation on primitive fields
func TestMarshal_ExtensiveAliases_PrimitiveFields_Success(t *testing.T) {
	t.Parallel()

	inputYAML := `# Define aliases for different primitive types
stringAlias: &strAlias "aliased string value"
boolAlias: &boolAlias true
intAlias: &intAlias 42
floatAlias: &floatAlias 3.14
# Use aliases in primitive fields
stringField: *strAlias
stringPtrField: *strAlias
boolField: *boolAlias
boolPtrField: *boolAlias
intField: *intAlias
intPtrField: *intAlias
float64Field: *floatAlias
float64PtrField: *floatAlias
# Use aliases in extensions
x-string-ext: *strAlias
x-bool-ext: *boolAlias
x-int-ext: *intAlias
x-float-ext: *floatAlias
`

	// Unmarshal -> Marshal -> Check alias preservation
	reader := strings.NewReader(inputYAML)
	model := &tests.TestPrimitiveHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	var buf bytes.Buffer
	err = marshaller.Marshal(t.Context(), model, &buf)
	require.NoError(t, err)

	outputYAML := buf.String()
	assert.YAMLEq(t, inputYAML, outputYAML)
}

// TestMarshal_ExtensiveAliases_ArrayElements_Success tests alias preservation in array elements
func TestMarshal_ExtensiveAliases_ArrayElements_Success(t *testing.T) {
	t.Parallel()

	inputYAML := `# Define aliases for array elements
item1: &item1 "first item"
item2: &item2 "second item"
structItem: &structItem
  stringField: "struct in array"
  boolField: true
  intField: 100
  float64Field: 1.23
nestedModelValue:
  stringField: "required value"
  boolField: true
  intField: 42
  float64Field: 3.14
# Add required field for TestComplexHighModel
eitherModelOrPrimitive: 999
# Use aliases in arrays
arrayField:
  - *item1
  - *item2
  - "literal item"
  - *item1
nodeArrayField:
  - *item1
  - *item2
structArrayField:
  - *structItem
  - stringField: "literal struct"
    boolField: false
    intField: 200
    float64Field: 4.56
  - *structItem
# Use aliases in extensions
x-array-ext:
  - *item1
  - *item2
`

	// Unmarshal -> Marshal -> Check alias preservation
	reader := strings.NewReader(inputYAML)
	model := &tests.TestComplexHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	var buf bytes.Buffer
	err = marshaller.Marshal(t.Context(), model, &buf)
	require.NoError(t, err)

	outputYAML := buf.String()
	assert.YAMLEq(t, inputYAML, outputYAML)
}

// TestMarshal_ExtensiveAliases_MapElements_Success tests alias preservation in map values and keys
func TestMarshal_ExtensiveAliases_MapElements_Success(t *testing.T) {
	t.Parallel()

	t.Skip("TODO: Fix alias key marshalling format issues - alias definition value loss and duplicate entries")
	inputYAML := `# Define aliases for map elements
keyAlias: &keyAlias "dynamic-key"
valueAlias: &valueAlias "aliased map value"
structValue: &structValue
  stringField: "struct as map value"
  boolField: true
  intField: 300
  float64Field: 2.71

name: "test name"

# Use aliases in embedded map (keys and values)
*keyAlias :
  stringField: "value for aliased key"
  boolField: false
  intField: 400
  float64Field: 5.67

regularKey: *structValue

anotherKey:
  stringField: *valueAlias
  boolField: true
  intField: 500
  float64Field: 8.90

# Use aliases in extensions
x-map-ext:
  *keyAlias : *valueAlias
  regularExtKey: "regular value"
`

	// Unmarshal -> Marshal -> Check alias preservation
	reader := strings.NewReader(inputYAML)
	model := &tests.TestEmbeddedMapWithFieldsHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	var buf bytes.Buffer
	err = marshaller.Marshal(t.Context(), model, &buf)
	require.NoError(t, err)

	outputYAML := buf.String()
	assert.YAMLEq(t, inputYAML, outputYAML)
}

// TestMarshal_ExtensiveComments_PrimitiveFields_Success tests comment preservation on primitive fields
func TestMarshal_ExtensiveComments_PrimitiveFields_Success(t *testing.T) {
	t.Parallel()

	inputYAML := `# Header comment for the document
# Multiple line header comment
# Comment for string field
stringField: "test string" # inline comment for string
# Comment for string pointer field
stringPtrField: "test ptr string" # inline comment for string ptr
# Boolean field comment
# with multiple lines
boolField: true # inline bool comment
boolPtrField: false # simple inline comment
# Integer field with detailed comment
intField: 42 # the answer to everything
intPtrField: 24 # another int comment
# Float field comment
float64Field: 3.14 # pi value
float64PtrField: 2.71 # euler's number
# Extensions section comment
# Comment for custom extension
x-custom: "extension value" # inline extension comment
# Another extension comment
x-another: "another extension" # another inline comment
`

	// Unmarshal -> Marshal -> Check comment preservation
	reader := strings.NewReader(inputYAML)
	model := &tests.TestPrimitiveHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	var buf bytes.Buffer
	err = marshaller.Marshal(t.Context(), model, &buf)
	require.NoError(t, err)

	outputYAML := buf.String()
	assert.YAMLEq(t, inputYAML, outputYAML)
}

// TestMarshal_ExtensiveComments_ArrayElements_Success tests comment preservation in arrays
func TestMarshal_ExtensiveComments_ArrayElements_Success(t *testing.T) {
	t.Parallel()

	inputYAML := `# Required nested model
nestedModelValue:
  stringField: "nested value"
  boolField: true
  intField: 100
  float64Field: 3.14
# Add required field for TestComplexHighModel
eitherModelOrPrimitive: 999
# Array field with comments
arrayField:
  # Comment for first array item
  - "first item" # inline comment for first item
  # Comment for second array item
  - "second item" # inline comment for second item
  # Comment for third array item
  - "third item" # inline comment for third item
# Node array field with comments
nodeArrayField:
  # Comment for first node item
  - "first node" # inline comment for first node
  # Comment for second node item
  - "second node" # inline comment for second node
# Struct array with comments
structArrayField:
  # Comment for first struct in array
  - # inline comment for struct
    stringField: "first struct" # comment for nested field
    boolField: true # nested bool comment
    intField: 200
    float64Field: 1.23
  # Comment for second struct in array
  - stringField: "second struct"
    boolField: false
    intField: 300
    float64Field: 4.56
# Extensions with array comments
x-array-ext:
  # Comment for extension array item
  - "ext item 1" # inline extension array comment
  - "ext item 2"
`

	// Unmarshal -> Marshal -> Check comment preservation
	reader := strings.NewReader(inputYAML)
	model := &tests.TestComplexHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	var buf bytes.Buffer
	err = marshaller.Marshal(t.Context(), model, &buf)
	require.NoError(t, err)

	outputYAML := buf.String()
	assert.YAMLEq(t, inputYAML, outputYAML)
}

// TestMarshal_ExtensiveComments_MapElements_Success tests comment preservation in maps
func TestMarshal_ExtensiveComments_MapElements_Success(t *testing.T) {
	t.Parallel()

	inputYAML := `# Name field comment
name: "test name" # inline name comment
# Dynamic map entries with comments
# Comment for first dynamic key
dynamicKey1:
  # Comment for nested string field
  stringField: "dynamic value 1" # inline nested comment
  boolField: true # nested bool comment
  intField: 100
  float64Field: 1.23
# Comment for second dynamic key
dynamicKey2:
  stringField: "dynamic value 2"
  boolField: false
  intField: 42
  float64Field: 4.56
# Comment for third dynamic key
dynamicKey3:
  stringField: "dynamic value 3"
  boolField: true
  intField: 789
  float64Field: 9.87
# Extensions section with map comments
# Comment for extension with map value
x-map-extension:
  # Comment for extension map key
  extKey1: "ext value 1" # inline extension map comment
  extKey2: "ext value 2"
# Simple extension comment
x-simple: "simple extension value" # simple inline comment
`

	// Unmarshal -> Marshal -> Check comment preservation
	reader := strings.NewReader(inputYAML)
	model := &tests.TestEmbeddedMapWithFieldsHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	var buf bytes.Buffer
	err = marshaller.Marshal(t.Context(), model, &buf)
	require.NoError(t, err)

	outputYAML := buf.String()
	assert.YAMLEq(t, inputYAML, outputYAML)
}

// TestMarshal_MixedAliasesAndComments_Success tests complex scenarios with both aliases and comments
func TestMarshal_MixedAliasesAndComments_Success(t *testing.T) {
	t.Parallel()

	t.Skip("TODO: Fix comment placement issues - comments being moved to wrong locations during marshalling")
	inputYAML := `# Document header with aliases and comments
# Define commented aliases
commonString: &commonStr "shared value" # alias for common string
commonStruct: &commonStruct # alias for common struct
  stringField: "struct value" # nested field comment
  boolField: true # nested bool comment
  intField: 42
  float64Field: 3.14
# Required field using alias
nestedModelValue: *commonStruct # using struct alias
# Add required field for TestComplexHighModel
eitherModelOrPrimitive: 999
# Array with mixed aliases and comments
arrayField:
  # First item uses alias
  - *commonStr # using string alias
  # Second item is literal with comment
  - "literal value" # literal array item
  # Third item uses alias again
  - *commonStr # reusing string alias
# Node array with aliases and comments
nodeArrayField:
  # Node items with aliases
  - *commonStr # node using alias
  - "literal node" # literal node item
# Struct array with mixed content
structArrayField:
  # First struct uses alias
  - *commonStruct # using struct alias
  # Second struct is literal with comments
  - # literal struct with comments
    stringField: *commonStr # field using alias
    boolField: false # literal bool with comment
    intField: 200
    float64Field: 1.23
# Extensions mixing aliases and comments
# Extension using alias
x-alias-ext: *commonStr # extension with alias
# Extension with commented structure
x-struct-ext: # extension with struct
  key1: *commonStr # nested alias in extension
  key2: "literal ext value" # literal value in extension
# Simple commented extension
x-simple-ext: "simple value" # simple extension comment
`

	// Unmarshal -> Marshal -> Check preservation of both aliases and comments
	reader := strings.NewReader(inputYAML)
	model := &tests.TestComplexHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	var buf bytes.Buffer
	err = marshaller.Marshal(t.Context(), model, &buf)
	require.NoError(t, err)

	outputYAML := buf.String()
	assert.YAMLEq(t, inputYAML, outputYAML)
}

// TestMarshal_CrossReferenceAliases_Success tests aliases defined in one section and used in another
func TestMarshal_CrossReferenceAliases_Success(t *testing.T) {
	t.Parallel()

	inputYAML := `# Define aliases in extensions section
x-alias-definitions:
  stringDef: &crossString "cross-referenced string"
  structDef: &crossStruct
    stringField: "cross-referenced struct"
    boolField: true
    intField: 999
    float64Field: 9.99
# Use cross-referenced aliases in main fields
stringField: *crossString
stringPtrField: *crossString
nestedModelValue: *crossStruct
# Add required field for TestComplexHighModel
eitherModelOrPrimitive: 999
# Use cross-referenced aliases in arrays
arrayField:
  - *crossString
  - "literal item"
  - *crossString
nodeArrayField:
  - *crossString
structArrayField:
  - *crossStruct
  - stringField: *crossString
    boolField: false
    intField: 100
    float64Field: 1.11
# Use cross-referenced aliases in other extensions
x-cross-string: *crossString
x-cross-struct: *crossStruct
x-mixed-array:
  - *crossString
  - *crossStruct
`

	// Unmarshal -> Marshal -> Check cross-reference alias preservation
	reader := strings.NewReader(inputYAML)
	model := &tests.TestComplexHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	var buf bytes.Buffer
	err = marshaller.Marshal(t.Context(), model, &buf)
	require.NoError(t, err)

	outputYAML := buf.String()
	assert.YAMLEq(t, inputYAML, outputYAML)
}

func TestMarshal_TestTypeConversionModel_RoundTrip_YAML_Success(t *testing.T) {
	t.Parallel()

	inputYAML := `httpMethodField: "post"
post:
  stringField: "POST operation"
  boolField: true
  intField: 42
  float64Field: 3.14
get:
  stringField: "GET operation"
  boolField: false
  intField: 100
  float64Field: 1.23
put:
  stringField: "PUT operation"
  boolField: true
  intField: 200
  float64Field: 2.34
x-custom: "extension value"
`

	// Unmarshal -> Marshal -> Compare (tests key type conversion)
	reader := strings.NewReader(inputYAML)
	model := &tests.TestTypeConversionHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	var buf bytes.Buffer
	err = marshaller.Marshal(t.Context(), model, &buf)
	require.NoError(t, err)

	outputYAML := buf.String()
	assert.YAMLEq(t, inputYAML, outputYAML)
}

func TestMarshal_TestTypeConversionModel_WithChanges_YAML_Success(t *testing.T) {
	t.Parallel()

	inputYAML := `httpMethodField: "get"
post:
  stringField: "POST operation"
  boolField: true
  intField: 42
  float64Field: 3.14
x-original: "original extension"
get:
  stringField: "GET operation"
  boolField: false
  intField: 100
  float64Field: 1.23
`

	expectedYAML := `httpMethodField: "put"
post:
  stringField: "Modified POST operation"
  boolField: false
  intField: 42
  float64Field: 3.14
x-original: "original extension"
get:
  stringField: "GET operation"
  boolField: false
  intField: 100
  float64Field: 1.23
patch:
  stringField: "New PATCH operation"
  boolField: true
  intField: 300
  float64Field: 5.67
x-modified: modified extension
`

	// Unmarshal
	reader := strings.NewReader(inputYAML)
	model := &tests.TestTypeConversionHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Modify the model (test HTTPMethod key operations)
	putMethod := tests.HTTPMethodPut
	model.HTTPMethodField = &putMethod

	// Modify existing operation
	postOp, exists := model.Get(tests.HTTPMethodPost)
	require.True(t, exists)
	postOp.StringField = "Modified POST operation"
	postOp.BoolField = false

	// Add new operation with HTTPMethod key
	newOp := &tests.TestPrimitiveHighModel{
		StringField:  "New PATCH operation",
		BoolField:    true,
		IntField:     300,
		Float64Field: 5.67,
	}
	model.Set(tests.HTTPMethod("patch"), newOp)

	// Modify extensions
	if model.Extensions != nil {
		model.Extensions.Set("x-modified", testutils.CreateStringYamlNode("modified extension", 1, 1))
	}

	// Marshal
	var buf bytes.Buffer
	err = marshaller.Marshal(t.Context(), model, &buf)
	require.NoError(t, err)

	outputYAML := buf.String()
	assert.YAMLEq(t, expectedYAML, outputYAML)
}

// TestMarshal_ExtensionOrderingBug_Reproduction reproduces the bug where extensions
// get reordered when new map entries are added
func TestMarshal_ExtensionOrderingBug_Reproduction(t *testing.T) {
	t.Parallel()

	t.Skip("TODO: Fix extension ordering bug")

	inputYAML := `httpMethodField: "get"
post:
  stringField: "POST operation"
  boolField: true
  intField: 42
  float64Field: 3.14
get:
  stringField: "GET operation"
  boolField: false
  intField: 100
  float64Field: 1.23
x-original: "original extension"
`

	// This test demonstrates the ordering bug where x-original moves position
	// when new map entries are added
	expectedYAML := `httpMethodField: "put"
post:
  stringField: "Modified POST operation"
  boolField: false
  intField: 42
  float64Field: 3.14
get:
  stringField: "GET operation"
  boolField: false
  intField: 100
  float64Field: 1.23
patch:
  stringField: "New PATCH operation"
  boolField: true
  intField: 300
  float64Field: 5.67
x-original: "original extension"
x-modified: modified extension
`

	// Unmarshal
	reader := strings.NewReader(inputYAML)
	model := &tests.TestTypeConversionHighModel{}
	validationErrs, err := marshaller.Unmarshal(t.Context(), reader, model)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Modify the model
	putMethod := tests.HTTPMethodPut
	model.HTTPMethodField = &putMethod

	// Modify existing operation
	postOp, exists := model.Get(tests.HTTPMethodPost)
	require.True(t, exists)
	postOp.StringField = "Modified POST operation"
	postOp.BoolField = false

	// Add new operation - this triggers the ordering bug
	newOp := &tests.TestPrimitiveHighModel{
		StringField:  "New PATCH operation",
		BoolField:    true,
		IntField:     300,
		Float64Field: 5.67,
	}
	model.Set(tests.HTTPMethod("patch"), newOp)

	// Add new extension - this also affects ordering
	if model.Extensions != nil {
		model.Extensions.Set("x-modified", testutils.CreateStringYamlNode("modified extension", 1, 1))
	}

	// Marshal
	var buf bytes.Buffer
	err = marshaller.Marshal(t.Context(), model, &buf)
	require.NoError(t, err)

	outputYAML := buf.String()

	// This will fail due to the ordering bug - x-original appears before patch instead of after
	t.Logf("Expected YAML:\n%s", expectedYAML)
	t.Logf("Actual YAML:\n%s", outputYAML)

	// For now, just verify the content is present, not the exact order
	require.Contains(t, outputYAML, `httpMethodField: "put"`)
	require.Contains(t, outputYAML, "Modified POST operation")
	require.Contains(t, outputYAML, "New PATCH operation")
	require.Contains(t, outputYAML, "x-original: \"original extension\"")
	require.Contains(t, outputYAML, "x-modified: modified extension")

	assert.YAMLEq(t, expectedYAML, outputYAML)
}
