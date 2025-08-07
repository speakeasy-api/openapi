package oas3_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockVirtualFS implements fs.FS for testing
type MockVirtualFS struct {
	files map[string]string
}

func NewMockVirtualFS() *MockVirtualFS {
	return &MockVirtualFS{
		files: make(map[string]string),
	}
}

func (m *MockVirtualFS) AddFile(path, content string) {
	// Normalize path separators for cross-platform compatibility
	normalizedPath := filepath.ToSlash(path)
	m.files[normalizedPath] = content
}

func (m *MockVirtualFS) Open(name string) (fs.File, error) {
	// Normalize path separators for cross-platform compatibility
	normalizedName := filepath.ToSlash(name)
	content, exists := m.files[normalizedName]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", name)
	}
	return &MockFile{content: content}, nil
}

// MockFile implements fs.File for testing
type MockFile struct {
	content string
	pos     int
}

func (m *MockFile) Read(p []byte) (n int, err error) {
	if m.pos >= len(m.content) {
		return 0, io.EOF
	}
	n = copy(p, m.content[m.pos:])
	m.pos += n
	return n, nil
}

func (m *MockFile) Close() error {
	return nil
}

func (m *MockFile) Stat() (fs.FileInfo, error) {
	return nil, fmt.Errorf("not implemented")
}

func TestInline_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		input         string
		externalFiles map[string]string
		expected      string
	}{
		{
			name: "simple reference inlining with unused defs removal",
			input: `{
				"type": "object",
				"properties": {
					"user": {
						"$ref": "#/$defs/User"
					}
				},
				"$defs": {
					"User": {
						"type": "object",
						"properties": {
							"name": {
								"type": "string"
							}
						}
					},
					"UnusedDef": {
						"type": "string"
					}
				}
			}`,
			expected: `{
				"type": "object",
				"properties": {
					"user": {
						"type": "object",
						"properties": {
							"name": {
								"type": "string"
							}
						}
					}
				}
			}`,
		},
		{
			name: "nested reference inlining",
			input: `{
				"type": "object",
				"properties": {
					"data": {
						"$ref": "#/$defs/Container"
					}
				},
				"$defs": {
					"Container": {
						"type": "object",
						"properties": {
							"user": {
								"$ref": "#/$defs/User"
							}
						}
					},
					"User": {
						"type": "object",
						"properties": {
							"name": {
								"type": "string"
							}
						}
					}
				}
			}`,
			expected: `{
				"type": "object",
				"properties": {
					"data": {
						"type": "object",
						"properties": {
							"user": {
								"type": "object",
								"properties": {
									"name": {
										"type": "string"
									}
								}
							}
						}
					}
				}
			}`,
		},
		{
			name: "boolean schema reference",
			input: `{
				"type": "object",
				"properties": {
					"any": {
						"$ref": "#/$defs/AnyValue"
					}
				},
				"$defs": {
					"AnyValue": true
				}
			}`,
			expected: `{
				"type": "object",
				"properties": {
					"any": true
				}
			}`,
		},
		{
			name: "array items reference",
			input: `{
				"type": "object",
				"properties": {
					"users": {
						"type": "array",
						"items": {
							"$ref": "#/$defs/User"
						}
					}
				},
				"$defs": {
					"User": {
						"type": "object",
						"properties": {
							"id": {
								"type": "string"
							}
						}
					}
				}
			}`,
			expected: `{
				"type": "object",
				"properties": {
					"users": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"id": {
									"type": "string"
								}
							}
						}
					}
				}
			}`,
		},
		{
			name: "no reference",
			input: `{
				"type": "object",
				"properties": {
					"name": {
						"type": "string"
					}
				}
			}`,
			expected: `{
				"type": "object",
				"properties": {
					"name": {
						"type": "string"
					}
				}
			}`,
		},
		{
			name: "reference to nested property within a schema",
			input: `{
				"type": "object",
				"properties": {
					"address": {
						"$ref": "#/$defs/Person/properties/address"
					}
				},
				"$defs": {
					"Person": {
						"type": "object",
						"properties": {
							"name": {
								"type": "string"
							},
							"address": {
								"type": "object",
								"properties": {
									"street": {
										"type": "string"
									},
									"city": {
										"type": "string"
									}
								}
							}
						}
					}
				}
			}`,
			expected: `{
				"type": "object",
				"properties": {
					"address": {
						"type": "object",
						"properties": {
							"street": {
								"type": "string"
							},
							"city": {
								"type": "string"
							}
						}
					}
				}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			// Parse input JSON into schema
			schema, err := parseJSONToSchema(tt.input)
			require.NoError(t, err, "failed to parse input JSON")

			// Create resolve options with the schema as the root document
			opts := oas3.InlineOptions{
				ResolveOptions: oas3.ResolveOptions{
					TargetLocation: "schema.json",
					RootDocument:   schema,
				},
				RemoveUnusedDefs: true,
			}

			// Inline the schema
			inlined, err := oas3.Inline(ctx, schema, opts)
			require.NoError(t, err, "inlining should succeed")

			// Convert result back to JSON and compare
			actualJSON, err := schemaToJSON(ctx, inlined)
			require.NoError(t, err, "failed to convert result to JSON")

			assert.Equal(t, formatJSON(tt.expected), formatJSON(actualJSON), "inlined schema should match expected result")
		})
	}
}

func TestInline_Error(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name: "unresolvable reference",
			input: `{
				"type": "object",
				"properties": {
					"user": {
						"$ref": "#/$defs/NonExistent"
					}
				}
			}`,
			expectedError: "failed to resolve schema",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			schema, err := parseJSONToSchema(tt.input)
			require.NoError(t, err, "failed to parse input JSON")

			opts := oas3.InlineOptions{
				ResolveOptions: oas3.ResolveOptions{
					TargetLocation: "test://schema",
					RootDocument:   schema,
				},
			}

			_, err = oas3.Inline(ctx, schema, opts)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestInline_NilSchema(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := oas3.InlineOptions{}

	_, err := oas3.Inline(ctx, nil, opts)
	assert.NoError(t, err, "inlining nil schema should not error")
}

// Helper functions for JSON parsing and conversion

func parseJSONToSchema(jsonStr string) (*oas3.JSONSchema[oas3.Referenceable], error) {
	reader := strings.NewReader(jsonStr)
	schema := &oas3.JSONSchema[oas3.Referenceable]{}

	_, err := marshaller.Unmarshal(context.Background(), reader, schema)
	if err != nil {
		return nil, err
	}

	return schema, nil
}

func schemaToJSON(ctx context.Context, schema *oas3.JSONSchema[oas3.Referenceable]) (string, error) {
	var buffer bytes.Buffer

	ctx = yml.ContextWithConfig(ctx, &yml.Config{
		OutputFormat: yml.OutputFormatJSON,
		Indentation:  2,
	})

	if err := marshaller.Marshal(ctx, schema, &buffer); err != nil {
		return "", err
	}

	return buffer.String(), nil
}

func formatJSON(s string) string {
	var out bytes.Buffer
	if err := json.Indent(&out, []byte(s), "", "  "); err != nil {
		// If indentation fails, return the original string
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(out.String())
}

func TestInline_CircularReferences_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		input         string
		externalFiles map[string]string
		expected      string
	}{
		{
			name: "valid circular reference through optional property",
			input: `{
				"type": "object",
				"properties": {
					"name": {
						"type": "string"
					},
					"parent": {
						"$ref": "#/$defs/Node"
					}
				},
				"$defs": {
					"Node": {
						"type": "object",
						"properties": {
							"name": {
								"type": "string"
							},
							"parent": {
								"$ref": "#/$defs/Node"
							}
						}
					}
				}
			}`,
			expected: `{
				"type": "object",
				"properties": {
					"name": {
						"type": "string"
					},
					"parent": {
						"$ref": "#/$defs/Node"
					}
				},
				"$defs": {
					"Node": {
						"type": "object",
						"properties": {
							"name": {
								"type": "string"
							},
							"parent": {
								"$ref": "#/$defs/Node"
							}
						}
					}
				}
			}`,
		},
		{
			name: "valid circular reference through array without minItems",
			input: `{
				"type": "object",
				"properties": {
					"name": {
						"type": "string"
					},
					"children": {
						"type": "array",
						"items": {
							"$ref": "#/$defs/TreeNode"
						}
					}
				},
				"$defs": {
					"TreeNode": {
						"type": "object",
						"properties": {
							"name": {
								"type": "string"
							},
							"children": {
								"type": "array",
								"items": {
									"$ref": "#/$defs/TreeNode"
								}
							}
						}
					}
				}
			}`,
			expected: `{
				"type": "object",
				"properties": {
					"name": {
						"type": "string"
					},
					"children": {
						"type": "array",
						"items": {
							"$ref": "#/$defs/TreeNode"
						}
					}
				},
				"$defs": {
					"TreeNode": {
						"type": "object",
						"properties": {
							"name": {
								"type": "string"
							},
							"children": {
								"type": "array",
								"items": {
									"$ref": "#/$defs/TreeNode"
								}
							}
						}
					}
				}
			}`,
		},
		{
			name: "valid circular reference through oneOf",
			input: `{
				"type": "object",
				"properties": {
					"value": {
						"oneOf": [
							{
								"type": "string"
							},
							{
								"$ref": "#/$defs/RecursiveValue"
							}
						]
					}
				},
				"$defs": {
					"RecursiveValue": {
						"type": "object",
						"properties": {
							"nested": {
								"oneOf": [
									{
										"type": "string"
									},
									{
										"$ref": "#/$defs/RecursiveValue"
									}
								]
							}
						}
					}
				}
			}`,
			expected: `{
				"type": "object",
				"properties": {
					"value": {
						"oneOf": [
							{
								"type": "string"
							},
							{
								"$ref": "#/$defs/RecursiveValue"
							}
						]
					}
				},
				"$defs": {
					"RecursiveValue": {
						"type": "object",
						"properties": {
							"nested": {
								"oneOf": [
									{
										"type": "string"
									},
									{
										"$ref": "#/$defs/RecursiveValue"
									}
								]
							}
						}
					}
				}
			}`,
		},
		{
			name: "valid circular reference with mixed inlining",
			input: `{
				"type": "object",
				"properties": {
					"user": {
						"$ref": "#/$defs/User"
					},
					"manager": {
						"$ref": "#/$defs/Manager"
					}
				},
				"$defs": {
					"User": {
						"type": "object",
						"properties": {
							"name": {
								"type": "string"
							},
							"manager": {
								"$ref": "#/$defs/Manager"
							}
						}
					},
					"Manager": {
						"type": "object",
						"properties": {
							"name": {
								"type": "string"
							},
							"reports": {
								"type": "array",
								"items": {
									"$ref": "#/$defs/User"
								}
							}
						}
					},
					"SimpleType": {
						"type": "string"
					}
				}
			}`,
			expected: `{
				"type": "object",
				"properties": {
					"user": {
						"$ref": "#/$defs/User"
					},
					"manager": {
						"$ref": "#/$defs/Manager"
					}
				},
				"$defs": {
					"User": {
						"type": "object",
						"properties": {
							"name": {
								"type": "string"
							},
							"manager": {
								"$ref": "#/$defs/Manager"
							}
						}
					},
					"Manager": {
						"type": "object",
						"properties": {
							"name": {
								"type": "string"
							},
							"reports": {
								"type": "array",
								"items": {
									"$ref": "#/$defs/User"
								}
							}
						}
					}
				}
			}`,
		},
		{
			name: "external reference to another JSON schema file",
			input: `{
				"type": "object",
				"properties": {
					"address": {
						"$ref": "external.json#/$defs/Address"
					}
				}
			}`,
			externalFiles: map[string]string{
				"external.json": `{
					"type": "object",
					"$defs": {
						"Address": {
							"type": "object",
							"properties": {
								"street": {
									"type": "string"
								},
								"city": {
									"type": "string"
								}
							}
						}
					}
				}`,
			},
			expected: `{
				"type": "object",
				"properties": {
					"address": {
						"type": "object",
						"properties": {
							"street": {
								"type": "string"
							},
							"city": {
								"type": "string"
							}
						}
					}
				}
			}`,
		},
		{
			name: "external reference to non-standard JSON document",
			input: `{
				"type": "object",
				"properties": {
					"user": {
						"$ref": "config.json#/schemas/User"
					}
				}
			}`,
			externalFiles: map[string]string{
				"config.json": `{
					"metadata": {
						"version": "1.0.0"
					},
					"schemas": {
						"User": {
							"type": "object",
							"properties": {
								"id": {
									"type": "integer"
								},
								"email": {
									"type": "string",
									"format": "email"
								}
							}
						}
					}
				}`,
			},
			expected: `{
				"type": "object",
				"properties": {
					"user": {
						"type": "object",
						"properties": {
							"id": {
								"type": "integer"
							},
							"email": {
								"type": "string",
								"format": "email"
							}
						}
					}
				}
			}`,
		},
		{
			name: "external reference with circular dependency",
			input: `{
				"type": "object",
				"properties": {
					"node": {
						"$ref": "tree.json#/$defs/TreeNode"
					}
				}
			}`,
			externalFiles: map[string]string{
				"tree.json": `{
					"$defs": {
						"TreeNode": {
							"type": "object",
							"properties": {
								"value": {
									"type": "string"
								},
								"children": {
									"type": "array",
									"items": {
										"$ref": "#/$defs/TreeNode"
									}
								}
							}
						}
					}
				}`,
			},
			expected: `{
				"type": "object",
				"properties": {
					"node": {
						"$ref": "#/$defs/TreeNode"
					}
				},
				"$defs": {
					"TreeNode": {
						"type": "object",
						"properties": {
							"value": {
								"type": "string"
							},
							"children": {
								"type": "array",
								"items": {
									"$ref": "#/$defs/TreeNode"
								}
							}
						}
					}
				}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			// Parse input JSON into schema
			schema, err := parseJSONToSchema(tt.input)
			require.NoError(t, err, "failed to parse input JSON")

			// Create resolve options
			opts := oas3.InlineOptions{
				ResolveOptions: oas3.ResolveOptions{
					TargetLocation: "schema.json",
					RootDocument:   schema,
				},
				RemoveUnusedDefs: true,
			}

			// If we have external files, set up a custom resolver
			if len(tt.externalFiles) > 0 {
				mockFS := NewMockVirtualFS()
				for filename, content := range tt.externalFiles {
					mockFS.AddFile(filename, content)
				}
				opts.ResolveOptions.VirtualFS = mockFS
			}

			// Inline the schema
			inlined, err := oas3.Inline(ctx, schema, opts)
			require.NoError(t, err, "inlining should succeed for valid circular references")

			// Convert result back to JSON and compare
			actualJSON, err := schemaToJSON(ctx, inlined)
			require.NoError(t, err, "failed to convert result to JSON")

			assert.Equal(t, formatJSON(tt.expected), formatJSON(actualJSON), "inlined schema should match expected result")
		})
	}
}

func TestInline_CircularReferences_Error(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name: "invalid circular reference through required property",
			input: `{
				"type": "object",
				"required": ["child"],
				"properties": {
					"name": {
						"type": "string"
					},
					"child": {
						"$ref": "#/$defs/Node"
					}
				},
				"$defs": {
					"Node": {
						"type": "object",
						"required": ["child"],
						"properties": {
							"name": {
								"type": "string"
							},
							"child": {
								"$ref": "#/$defs/Node"
							}
						}
					}
				}
			}`,
			expectedError: "invalid circular reference",
		},
		{
			name: "invalid circular reference through array with minItems",
			input: `{
				"type": "object",
				"properties": {
					"items": {
						"type": "array",
						"minItems": 1,
						"items": {
							"$ref": "#/$defs/RecursiveItem"
						}
					}
				},
				"$defs": {
					"RecursiveItem": {
						"type": "object",
						"required": ["nested"],
						"properties": {
							"nested": {
								"type": "array",
								"minItems": 1,
								"items": {
									"$ref": "#/$defs/RecursiveItem"
								}
							}
						}
					}
				}
			}`,
			expectedError: "invalid circular reference",
		},
		{
			name: "invalid circular reference through allOf",
			input: `{
				"type": "object",
				"properties": {
					"value": {
						"allOf": [
							{
								"type": "object"
							},
							{
								"$ref": "#/$defs/RecursiveValue"
							}
						]
					}
				},
				"$defs": {
					"RecursiveValue": {
						"allOf": [
							{
								"type": "object"
							},
							{
								"$ref": "#/$defs/RecursiveValue"
							}
						]
					}
				}
			}`,
			expectedError: "invalid circular reference",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			schema, err := parseJSONToSchema(tt.input)
			require.NoError(t, err, "failed to parse input JSON")

			opts := oas3.InlineOptions{
				ResolveOptions: oas3.ResolveOptions{
					TargetLocation: "test://schema",
					RootDocument:   schema,
				},
			}

			_, err = oas3.Inline(ctx, schema, opts)
			require.Error(t, err, "inlining should fail for invalid circular references")
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestInline_OpenAPIComponentReferences_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		openAPIDoc     string
		schemaPointer  string
		expectedSchema string
	}{
		{
			name: "OpenAPI component reference with valid circular reference",
			openAPIDoc: `{
				"openapi": "3.1.1",
				"info": {
					"title": "Test API",
					"version": "1.0.0"
				},
				"paths": {},
				"components": {
					"schemas": {
						"User": {
							"type": "object",
							"properties": {
								"name": {
									"type": "string"
								},
								"manager": {
									"$ref": "#/components/schemas/Manager"
								}
							}
						},
						"Manager": {
							"type": "object",
							"properties": {
								"name": {
									"type": "string"
								},
								"reports": {
									"type": "array",
									"items": {
										"$ref": "#/components/schemas/User"
									}
								}
							}
						},
						"SimpleType": {
							"type": "string"
						}
					}
				}
			}`,
			schemaPointer: "/components/schemas/User",
			expectedSchema: `{
				"$ref": "#/$defs/User",
				"$defs": {
					"User": {
						"type": "object",
						"properties": {
							"name": {
								"type": "string"
							},
							"manager": {
								"type": "object",
								"properties": {
									"name": {
										"type": "string"
									},
									"reports": {
										"type": "array",
										"items": {
											"$ref": "#/$defs/User"
										}
									}
								}
							}
						}
					}
				}
			}`,
		},
		{
			name: "OpenAPI reference to operation response schema",
			openAPIDoc: `{
				"openapi": "3.1.1",
				"info": {
					"title": "Test API",
					"version": "1.0.0"
				},
				"paths": {
					"/users": {
						"get": {
							"responses": {
								"200": {
									"description": "Success",
									"content": {
										"application/json": {
											"schema": {
												"type": "object",
												"properties": {
													"users": {
														"type": "array",
														"items": {
															"$ref": "#/components/schemas/User"
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				},
				"components": {
					"schemas": {
						"User": {
							"type": "object",
							"properties": {
								"id": {
									"type": "integer"
								},
								"name": {
									"type": "string"
								}
							}
						},
						"Container": {
							"type": "object",
							"properties": {
								"data": {
									"$ref": "#/paths/~1users/get/responses/200/content/application~1json/schema"
								}
							}
						}
					}
				}
			}`,
			schemaPointer: "/components/schemas/Container",
			expectedSchema: `{
				"type": "object",
				"properties": {
					"data": {
						"type": "object",
						"properties": {
							"users": {
								"type": "array",
								"items": {
									"type": "object",
									"properties": {
										"id": {
											"type": "integer"
										},
										"name": {
											"type": "string"
										}
									}
								}
							}
						}
					}
				}
			}`,
		},
		{
			name: "OpenAPI component reference with mixed inlining and rewriting",
			openAPIDoc: `{
				"openapi": "3.1.1",
				"info": {
					"title": "Test API",
					"version": "1.0.0"
				},
				"paths": {},
				"components": {
					"schemas": {
						"Container": {
							"type": "object",
							"properties": {
								"value": {
									"$ref": "#/components/schemas/SimpleValue"
								},
								"node": {
									"$ref": "#/components/schemas/TreeNode"
								}
							}
						},
						"SimpleValue": {
							"type": "string"
						},
						"TreeNode": {
							"type": "object",
							"properties": {
								"name": {
									"type": "string"
								},
								"children": {
									"type": "array",
									"items": {
										"$ref": "#/components/schemas/TreeNode"
									}
								}
							}
						}
					}
				}
			}`,
			schemaPointer: "/components/schemas/Container",
			expectedSchema: `{
				"type": "object",
				"properties": {
					"value": {
						"type": "string"
					},
					"node": {
						"$ref": "#/$defs/TreeNode"
					}
				},
				"$defs": {
					"TreeNode": {
						"type": "object",
						"properties": {
							"name": {
								"type": "string"
							},
							"children": {
								"type": "array",
								"items": {
									"$ref": "#/$defs/TreeNode"
								}
							}
						}
					}
				}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			// Parse OpenAPI document
			openAPIDoc, err := parseJSONToOpenAPI(tt.openAPIDoc)
			require.NoError(t, err, "failed to parse OpenAPI document")

			// Extract schema using JSON pointer
			schema, err := extractSchemaFromOpenAPI(openAPIDoc, tt.schemaPointer)
			require.NoError(t, err, "failed to extract schema from OpenAPI document")

			// Create resolve options with the OpenAPI document as the root document
			opts := oas3.InlineOptions{
				ResolveOptions: oas3.ResolveOptions{
					TargetLocation: "openapi.json",
					RootDocument:   openAPIDoc,
				},
				RemoveUnusedDefs: true,
			}

			// Inline the schema
			inlined, err := oas3.Inline(ctx, schema, opts)
			require.NoError(t, err, "inlining should succeed for OpenAPI component references")

			// Convert result back to JSON and compare
			actualJSON, err := schemaToJSON(ctx, inlined)
			require.NoError(t, err, "failed to convert result to JSON")

			assert.Equal(t, formatJSON(tt.expectedSchema), formatJSON(actualJSON), "inlined schema should match expected result")
		})
	}
}

func TestInline_OpenAPIComponentReferences_Error(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		openAPIDoc    string
		schemaPointer string
		expectedError string
	}{
		{
			name: "OpenAPI component reference with invalid circular reference",
			openAPIDoc: `{
				"openapi": "3.1.1",
				"info": {
					"title": "Test API",
					"version": "1.0.0"
				},
				"paths": {},
				"components": {
					"schemas": {
						"User": {
							"type": "object",
							"required": ["manager"],
							"properties": {
								"name": {
									"type": "string"
								},
								"manager": {
									"$ref": "#/components/schemas/Manager"
								}
							}
						},
						"Manager": {
							"type": "object",
							"required": ["user"],
							"properties": {
								"name": {
									"type": "string"
								},
								"user": {
									"$ref": "#/components/schemas/User"
								}
							}
						}
					}
				}
			}`,
			schemaPointer: "/components/schemas/User",
			expectedError: "invalid circular reference",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			// Parse OpenAPI document
			openAPIDoc, err := parseJSONToOpenAPI(tt.openAPIDoc)
			require.NoError(t, err, "failed to parse OpenAPI document")

			// Extract schema using JSON pointer
			schema, err := extractSchemaFromOpenAPI(openAPIDoc, tt.schemaPointer)
			require.NoError(t, err, "failed to extract schema from OpenAPI document")

			opts := oas3.InlineOptions{
				ResolveOptions: oas3.ResolveOptions{
					TargetLocation: "test://openapi",
					RootDocument:   openAPIDoc,
				},
			}

			_, err = oas3.Inline(ctx, schema, opts)
			require.Error(t, err, "inlining should fail for invalid circular references in OpenAPI components")
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

// Helper functions for OpenAPI parsing and schema extraction

func parseJSONToOpenAPI(jsonStr string) (*openapi.OpenAPI, error) {
	reader := strings.NewReader(jsonStr)

	doc, _, err := openapi.Unmarshal(context.Background(), reader)
	if err != nil {
		return nil, err
	}

	return doc, nil
}

func extractSchemaFromOpenAPI(openAPIDoc *openapi.OpenAPI, pointer string) (*oas3.JSONSchema[oas3.Referenceable], error) {
	// Use JSON pointer to extract the schema
	target, err := jsonpointer.GetTarget(openAPIDoc, jsonpointer.JSONPointer(pointer))
	if err != nil {
		return nil, err
	}

	// The target should already be a JSONSchema, so we can cast it directly
	schema, ok := target.(*oas3.JSONSchema[oas3.Referenceable])
	if !ok {
		panic("target is not a JSONSchema")
	}

	return schema, nil
}
