package swagger

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/require"
)

func TestUpgrade_MinimalSwaggerJSON_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	inputSwaggerJSON := `{
  "swagger": "2.0",
  "info": {
    "title": "Minimal API",
    "version": "1.0.0"
  },
  "paths": {
    "/ping": {
      "get": {
        "responses": {
          "200": {
            "description": "ok"
          }
        }
      }
    }
  }
}
`

	// Unmarshal Swagger 2.0 (JSON)
	swDoc, validationErrs, err := Unmarshal(ctx, strings.NewReader(inputSwaggerJSON))
	require.NoError(t, err, "unmarshal should succeed")
	require.Empty(t, validationErrs, "swagger should be valid")

	// Upgrade to OpenAPI 3.0
	oaDoc, err := Upgrade(ctx, swDoc)
	require.NoError(t, err, "upgrade should succeed")
	require.NotNil(t, oaDoc, "openapi document should not be nil")

	// Marshal OpenAPI as JSON (match input format)
	cfg := swDoc.GetCore().GetConfig()
	oaDoc.GetCore().SetConfig(cfg)

	var buf bytes.Buffer
	err = marshaller.Marshal(ctx, oaDoc, &buf)
	require.NoError(t, err, "marshal should succeed")

	actualJSON := buf.String()

	expectedJSON := `{
  "openapi": "3.0.0",
  "info": {
    "title": "Minimal API",
    "version": "1.0.0"
  },
  "paths": {
    "/ping": {
      "get": {
        "responses": {
          "200": {
            "description": "ok"
          }
        }
      }
    }
  }
}
`

	require.Equal(t, expectedJSON, actualJSON, "upgraded OpenAPI JSON should match expected")
}

func TestUpgrade_BodyParameter_To_RequestBody_JSON_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	inputSwaggerJSON := `{
  "swagger": "2.0",
  "info": {
    "title": "Body Param API",
    "version": "1.0.0"
  },
  "paths": {
    "/users": {
      "post": {
        "consumes": ["application/json"],
        "parameters": [
          {
            "in": "body",
            "name": "body",
            "required": true,
            "schema": {
              "type": "object",
              "properties": {
                "name": { "type": "string" }
              }
            }
          }
        ],
        "responses": {
          "201": {
            "description": "created"
          }
        }
      }
    }
  }
}
`

	// Unmarshal Swagger 2.0 (JSON)
	swDoc, validationErrs, err := Unmarshal(ctx, strings.NewReader(inputSwaggerJSON))
	require.NoError(t, err, "unmarshal should succeed")
	require.Empty(t, validationErrs, "swagger should be valid")

	// Upgrade to OpenAPI 3.0
	oaDoc, err := Upgrade(ctx, swDoc)
	require.NoError(t, err, "upgrade should succeed")
	require.NotNil(t, oaDoc, "openapi document should not be nil")

	// Preserve input format (JSON)
	cfg := swDoc.GetCore().GetConfig()
	oaDoc.GetCore().SetConfig(cfg)

	var buf bytes.Buffer
	err = marshaller.Marshal(ctx, oaDoc, &buf)
	require.NoError(t, err, "marshal should succeed")

	actualJSON := buf.String()

	expectedJSON := `{
  "openapi": "3.0.0",
  "info": {
    "title": "Body Param API",
    "version": "1.0.0"
  },
  "paths": {
    "/users": {
      "post": {
        "requestBody": {
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "name": {
                    "type": "string"
                  }
                }
              }
            }
          },
          "required": true
        },
        "responses": {
          "201": {
            "description": "created"
          }
        }
      }
    }
  }
}
`

	require.Equal(t, expectedJSON, actualJSON, "upgraded OpenAPI JSON should map body parameter to requestBody")
}

func TestUpgrade_Servers_FromHostBasePathSchemes_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	inputSwaggerJSON := `{
  "swagger": "2.0",
  "info": {
    "title": "Server API",
    "version": "1.0.0"
  },
  "host": "api.example.com",
  "basePath": "/v1",
  "schemes": ["http", "https"],
  "paths": {
    "/ping": {
      "get": {
        "responses": {
          "200": {
            "description": "ok"
          }
        }
      }
    }
  }
}
`

	swDoc, validationErrs, err := Unmarshal(ctx, strings.NewReader(inputSwaggerJSON))
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	oaDoc, err := Upgrade(ctx, swDoc)
	require.NoError(t, err)
	require.NotNil(t, oaDoc)

	cfg := swDoc.GetCore().GetConfig()
	oaDoc.GetCore().SetConfig(cfg)

	var buf bytes.Buffer
	err = marshaller.Marshal(ctx, oaDoc, &buf)
	require.NoError(t, err)

	actual := buf.String()

	expected := `{
  "openapi": "3.0.0",
  "info": {
    "title": "Server API",
    "version": "1.0.0"
  },
  "servers": [
    {
      "url": "http://api.example.com/v1"
    },
    {
      "url": "https://api.example.com/v1"
    }
  ],
  "paths": {
    "/ping": {
      "get": {
        "responses": {
          "200": {
            "description": "ok"
          }
        }
      }
    }
  }
}
`

	require.Equal(t, expected, actual, "servers should be constructed from host/basePath/schemes")
}

func TestUpgrade_ResponseSchema_To_Content_WithProduces_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	input := `{
  "swagger": "2.0",
  "info": {
    "title": "Produces API",
    "version": "1.0.0"
  },
  "produces": ["application/json", "application/xml"],
  "paths": {
    "/things": {
      "get": {
        "responses": {
          "200": {
            "description": "ok",
            "schema": { "type": "object" }
          }
        }
      }
    }
  }
}
`

	swDoc, validationErrs, err := Unmarshal(ctx, strings.NewReader(input))
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	oaDoc, err := Upgrade(ctx, swDoc)
	require.NoError(t, err)
	require.NotNil(t, oaDoc)

	cfg := swDoc.GetCore().GetConfig()
	oaDoc.GetCore().SetConfig(cfg)

	var buf bytes.Buffer
	err = marshaller.Marshal(ctx, oaDoc, &buf)
	require.NoError(t, err)

	actual := buf.String()

	expected := `{
  "openapi": "3.0.0",
  "info": {
    "title": "Produces API",
    "version": "1.0.0"
  },
  "paths": {
    "/things": {
      "get": {
        "responses": {
          "200": {
            "description": "ok",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object"
                }
              },
              "application/xml": {
                "schema": {
                  "type": "object"
                }
              }
            }
          }
        }
      }
    }
  }
}
`

	require.Equal(t, expected, actual, "response schema should be wrapped under content for each produces type")
}

func TestUpgrade_FormData_To_RequestBody_Multipart_File_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	input := `{
  "swagger": "2.0",
  "info": {
    "title": "Upload API",
    "version": "1.0.0"
  },
  "paths": {
    "/upload": {
      "post": {
        "parameters": [
          { "in": "formData", "name": "file", "type": "file" },
          { "in": "formData", "name": "title", "type": "string", "required": true }
        ],
        "responses": {
          "200": { "description": "ok" }
        }
      }
    }
  }
}
`

	swDoc, validationErrs, err := Unmarshal(ctx, strings.NewReader(input))
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	oaDoc, err := Upgrade(ctx, swDoc)
	require.NoError(t, err)
	require.NotNil(t, oaDoc)

	cfg := swDoc.GetCore().GetConfig()
	oaDoc.GetCore().SetConfig(cfg)

	var buf bytes.Buffer
	err = marshaller.Marshal(ctx, oaDoc, &buf)
	require.NoError(t, err)

	actual := buf.String()

	expected := `{
  "openapi": "3.0.0",
  "info": {
    "title": "Upload API",
    "version": "1.0.0"
  },
  "paths": {
    "/upload": {
      "post": {
        "requestBody": {
          "content": {
            "multipart/form-data": {
              "schema": {
                "type": "object",
                "properties": {
                  "file": {
                    "type": "string",
                    "format": "binary"
                  },
                  "title": {
                    "type": "string"
                  }
                }
              }
            }
          },
          "required": true
        },
        "responses": {
          "200": {
            "description": "ok"
          }
        }
      }
    }
  }
}
`

	require.Equal(t, expected, actual, "formData with file should become multipart/form-data requestBody and set required=true if any field required")
}

func TestUpgrade_GlobalDefinitions_RefRewrite_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	input := `{
  "swagger": "2.0",
  "info": {
    "title": "Defs API",
    "version": "1.0.0"
  },
  "definitions": {
    "MyModel": {
      "type": "object",
      "properties": {
        "id": { "type": "string" }
      }
    }
  },
  "paths": {
    "/x": {
      "get": {
        "produces": ["application/json"],
        "responses": {
          "200": {
            "description": "ok",
            "schema": { "$ref": "#/definitions/MyModel" }
          }
        }
      }
    }
  }
}
`

	swDoc, validationErrs, err := Unmarshal(ctx, strings.NewReader(input))
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	oaDoc, err := Upgrade(ctx, swDoc)
	require.NoError(t, err)
	require.NotNil(t, oaDoc)

	cfg := swDoc.GetCore().GetConfig()
	oaDoc.GetCore().SetConfig(cfg)

	var buf bytes.Buffer
	err = marshaller.Marshal(ctx, oaDoc, &buf)
	require.NoError(t, err)

	actual := buf.String()

	expected := `{
  "openapi": "3.0.0",
  "info": {
    "title": "Defs API",
    "version": "1.0.0"
  },
  "paths": {
    "/x": {
      "get": {
        "responses": {
          "200": {
            "description": "ok",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/MyModel"
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
      "MyModel": {
        "type": "object",
        "properties": {
          "id": {
            "type": "string"
          }
        }
      }
    }
  }
}
`

	require.Equal(t, expected, actual, "definition refs should be rewritten to components.schemas and schemas moved under components")
}

func TestUpgrade_GlobalParameters_And_Responses_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	input := `{
  "swagger": "2.0",
  "info": {
    "title": "Globals API",
    "version": "1.0.0"
  },
  "parameters": {
    "PageParam": { "name": "page", "in": "query", "type": "integer" },
    "BodyParam": {
      "name": "body",
      "in": "body",
      "schema": {
        "type": "object",
        "properties": {
          "n": { "type": "string" }
        }
      }
    }
  },
  "responses": {
    "NotFound": {
      "description": "not found",
      "schema": { "type": "string" },
      "headers": {
        "X-Rate-Limit": { "type": "integer", "format": "int32" }
      }
    }
  },
  "paths": {
    "/x": {
      "get": {
        "parameters": [
          { "$ref": "#/parameters/PageParam" },
          { "$ref": "#/parameters/BodyParam" }
        ],
        "responses": {
          "404": { "$ref": "#/responses/NotFound" }
        }
      }
    }
  }
}
`

	swDoc, validationErrs, err := Unmarshal(ctx, strings.NewReader(input))
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	oaDoc, err := Upgrade(ctx, swDoc)
	require.NoError(t, err)
	require.NotNil(t, oaDoc)

	cfg := swDoc.GetCore().GetConfig()
	oaDoc.GetCore().SetConfig(cfg)

	var buf bytes.Buffer
	err = marshaller.Marshal(ctx, oaDoc, &buf)
	require.NoError(t, err)

	actual := buf.String()

	expected := `{
  "openapi": "3.0.0",
  "info": {
    "title": "Globals API",
    "version": "1.0.0"
  },
  "paths": {
    "/x": {
      "get": {
        "parameters": [
          {
            "$ref": "#/components/parameters/PageParam"
          }
        ],
        "requestBody": {
          "$ref": "#/components/requestBodies/BodyParam"
        },
        "responses": {
          "404": {
            "$ref": "#/components/responses/NotFound"
          }
        }
      }
    }
  },
  "components": {
    "responses": {
      "NotFound": {
        "description": "not found",
        "headers": {
          "X-Rate-Limit": {
            "schema": {
              "type": "integer",
              "format": "int32"
            }
          }
        },
        "content": {
          "application/json": {
            "schema": {
              "type": "string"
            }
          }
        }
      }
    },
    "parameters": {
      "PageParam": {
        "name": "page",
        "in": "query",
        "schema": {
          "type": "integer"
        }
      }
    },
    "requestBodies": {
      "BodyParam": {
        "content": {
          "application/json": {
            "schema": {
              "type": "object",
              "properties": {
                "n": {
                  "type": "string"
                }
              }
            }
          }
        }
      }
    }
  }
}
`

	require.Equal(t, expected, actual, "globals should map to components and references updated accordingly")
}

func TestUpgrade_Parameter_CollectionFormat_To_StyleExplode_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	input := `{
  "swagger": "2.0",
  "info": {
    "title": "CollectionFormat API",
    "version": "1.0.0"
  },
  "paths": {
    "/search": {
      "get": {
        "parameters": [
          {
            "name": "tags",
            "in": "query",
            "type": "array",
            "collectionFormat": "csv",
            "items": { "type": "string" }
          }
        ],
        "responses": {
          "200": { "description": "ok" }
        }
      }
    }
  }
}
`

	swDoc, validationErrs, err := Unmarshal(ctx, strings.NewReader(input))
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	oaDoc, err := Upgrade(ctx, swDoc)
	require.NoError(t, err)
	require.NotNil(t, oaDoc)

	cfg := swDoc.GetCore().GetConfig()
	oaDoc.GetCore().SetConfig(cfg)

	var buf bytes.Buffer
	err = marshaller.Marshal(ctx, oaDoc, &buf)
	require.NoError(t, err)

	actual := buf.String()

	expected := `{
  "openapi": "3.0.0",
  "info": {
    "title": "CollectionFormat API",
    "version": "1.0.0"
  },
  "paths": {
    "/search": {
      "get": {
        "parameters": [
          {
            "name": "tags",
            "in": "query",
            "style": "form",
            "explode": false,
            "schema": {
              "type": "array",
              "items": {
                "type": "string"
              }
            }
          }
        ],
        "responses": {
          "200": {
            "description": "ok"
          }
        }
      }
    }
  }
}
`

	require.Equal(t, expected, actual, "collectionFormat csv should map to style=form, explode=false with array schema")
}

func TestUpgrade_SecurityDefinitions_To_SecuritySchemes_JSON_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	input := `{
  "swagger": "2.0",
  "info": {
    "title": "Security API",
    "version": "1.0.0"
  },
  "securityDefinitions": {
    "basicAuth": {
      "type": "basic"
    },
    "apiKeyHeader": {
      "type": "apiKey",
      "name": "X-API-Key",
      "in": "header"
    },
    "oauth2Auth": {
      "type": "oauth2",
      "flow": "accessCode",
      "authorizationUrl": "https://auth.example.com/authorize",
      "tokenUrl": "https://auth.example.com/token",
      "scopes": {
        "read": "Read access",
        "write": "Write access"
      }
    }
  },
  "paths": {
    "/ping": {
      "get": {
        "responses": {
          "200": { "description": "ok" }
        }
      }
    }
  }
}
`

	swDoc, validationErrs, err := Unmarshal(ctx, strings.NewReader(input))
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	oaDoc, err := Upgrade(ctx, swDoc)
	require.NoError(t, err)
	require.NotNil(t, oaDoc)

	cfg := swDoc.GetCore().GetConfig()
	oaDoc.GetCore().SetConfig(cfg)

	var buf bytes.Buffer
	err = marshaller.Marshal(ctx, oaDoc, &buf)
	require.NoError(t, err)

	actual := buf.String()

	// Note: Only asserting components.securitySchemes mapping and structure; path kept minimal
	expected := `{
  "openapi": "3.0.0",
  "info": {
    "title": "Security API",
    "version": "1.0.0"
  },
  "paths": {
    "/ping": {
      "get": {
        "responses": {
          "200": {
            "description": "ok"
          }
        }
      }
    }
  },
  "components": {
    "securitySchemes": {
      "basicAuth": {
        "type": "http",
        "scheme": "basic"
      },
      "apiKeyHeader": {
        "type": "apiKey",
        "name": "X-API-Key",
        "in": "header"
      },
      "oauth2Auth": {
        "type": "oauth2",
        "flows": {
          "authorizationCode": {
            "authorizationUrl": "https://auth.example.com/authorize",
            "tokenUrl": "https://auth.example.com/token",
            "scopes": {
              "read": "Read access",
              "write": "Write access"
            }
          }
        }
      }
    }
  }
}
`

	require.Equal(t, expected, actual, "securityDefinitions should map to components.securitySchemes with correct types/flows")
}

func TestUpgrade_Produces_Overrides_Global_YAML_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Global produces application/json, but operation overrides to text/plain
	inputYAML := `swagger: "2.0"
info:
  title: Produces Override
  version: "1.0.0"
produces:
  - application/json
paths:
  /data:
    get:
      produces:
        - text/plain
      responses:
        "200":
          description: ok
          schema:
            type: string
`

	swDoc, validationErrs, err := Unmarshal(ctx, strings.NewReader(inputYAML))
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	oaDoc, err := Upgrade(ctx, swDoc)
	require.NoError(t, err)
	require.NotNil(t, oaDoc)

	// Preserve YAML output
	cfg := swDoc.GetCore().GetConfig()
	oaDoc.GetCore().SetConfig(cfg)

	var buf bytes.Buffer
	err = marshaller.Marshal(ctx, oaDoc, &buf)
	require.NoError(t, err)

	actualYAML := buf.String()

	expectedYAML := `openapi: "3.0.0"
info:
  title: "Produces Override"
  version: "1.0.0"
paths:
  /data:
    get:
      responses:
        "200":
          description: "ok"
          content:
            text/plain:
              schema:
                type: "string"
`

	require.Equal(t, expectedYAML, actualYAML, "operation-level produces should override global produces in response content")
}

func TestUpgrade_FormData_UrlEncoded_YAML_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// formData without file -> application/x-www-form-urlencoded and aggregate fields
	inputYAML := `swagger: "2.0"
info:
  title: Submit API
  version: "1.0.0"
paths:
  /submit:
    post:
      parameters:
        - in: formData
          name: a
          type: string
          required: true
        - in: formData
          name: b
          type: integer
      responses:
        "200":
          description: ok
`

	swDoc, validationErrs, err := Unmarshal(ctx, strings.NewReader(inputYAML))
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	oaDoc, err := Upgrade(ctx, swDoc)
	require.NoError(t, err)
	require.NotNil(t, oaDoc)

	cfg := swDoc.GetCore().GetConfig()
	oaDoc.GetCore().SetConfig(cfg)

	var buf bytes.Buffer
	err = marshaller.Marshal(ctx, oaDoc, &buf)
	require.NoError(t, err)

	actualYAML := buf.String()

	expectedYAML := `openapi: "3.0.0"
info:
  title: "Submit API"
  version: "1.0.0"
paths:
  /submit:
    post:
      requestBody:
        content:
          application/x-www-form-urlencoded:
            schema:
              type: "object"
              properties:
                a:
                  type: "string"
                b:
                  type: "integer"
        required: true
      responses:
        "200":
          description: "ok"
`

	require.Equal(t, expectedYAML, actualYAML, "formData without file should map to x-www-form-urlencoded and aggregate fields in object schema")
}

func TestUpgrade_Response_Examples_To_Content_Example_JSON_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	input := `{
  "swagger": "2.0",
  "info": {
    "title": "Examples API",
    "version": "1.0.0"
  },
  "paths": {
    "/ex": {
      "get": {
        "produces": ["application/json"],
        "responses": {
          "200": {
            "description": "ok",
            "schema": { "type": "object", "properties": { "id": { "type": "integer" } } },
            "examples": {
              "application/json": { "id": 123 }
            }
          }
        }
      }
    }
  }
}
`

	swDoc, validationErrs, err := Unmarshal(ctx, strings.NewReader(input))
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	oaDoc, err := Upgrade(ctx, swDoc)
	require.NoError(t, err)
	require.NotNil(t, oaDoc)

	cfg := swDoc.GetCore().GetConfig()
	oaDoc.GetCore().SetConfig(cfg)

	var buf bytes.Buffer
	err = marshaller.Marshal(ctx, oaDoc, &buf)
	require.NoError(t, err)

	actual := buf.String()

	expected := `{
  "openapi": "3.0.0",
  "info": {
    "title": "Examples API",
    "version": "1.0.0"
  },
  "paths": {
    "/ex": {
      "get": {
        "responses": {
          "200": {
            "description": "ok",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "id": {
                      "type": "integer"
                    }
                  }
                },
                "example": {"id": 123}
              }
            }
          }
        }
      }
    }
  }
}
`

	require.Equal(t, expected, actual, "response examples should map to content[mediaType].example")
}

func TestUpgrade_Info_Contact_License_JSON_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	input := `{
  "swagger": "2.0",
  "info": {
    "title": "Info API",
    "description": "API with contact and license",
    "termsOfService": "https://example.com/tos",
    "contact": {
      "name": "Alice",
      "url": "https://example.com",
      "email": "alice@example.com"
    },
    "license": {
      "name": "MIT",
      "url": "https://opensource.org/licenses/MIT"
    },
    "version": "1.0.0"
  },
  "paths": {
    "/ping": {
      "get": {
        "responses": {
          "200": { "description": "ok" }
        }
      }
    }
  }
}
`

	swDoc, validationErrs, err := Unmarshal(ctx, strings.NewReader(input))
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	oaDoc, err := Upgrade(ctx, swDoc)
	require.NoError(t, err)
	require.NotNil(t, oaDoc)

	cfg := swDoc.GetCore().GetConfig()
	oaDoc.GetCore().SetConfig(cfg)

	var buf bytes.Buffer
	err = marshaller.Marshal(ctx, oaDoc, &buf)
	require.NoError(t, err)

	actual := buf.String()

	expected := `{
  "openapi": "3.0.0",
  "info": {
    "title": "Info API",
    "version": "1.0.0",
    "description": "API with contact and license",
    "termsOfService": "https://example.com/tos",
    "contact": {
      "name": "Alice",
      "url": "https://example.com",
      "email": "alice@example.com"
    },
    "license": {
      "name": "MIT",
      "url": "https://opensource.org/licenses/MIT"
    }
  },
  "paths": {
    "/ping": {
      "get": {
        "responses": {
          "200": {
            "description": "ok"
          }
        }
      }
    }
  }
}
`

	require.Equal(t, expected, actual, "info.contact/license/termsOfService should be preserved")
}

func TestUpgrade_Tags_JSON_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	input := `{
  "swagger": "2.0",
  "info": { "title": "Tags API", "version": "1.0.0" },
  "tags": [
    { "name": "users", "description": "User operations" },
    { "name": "admin", "description": "Admin operations" }
  ],
  "paths": {
    "/ping": {
      "get": {
        "responses": {
          "200": { "description": "ok" }
        }
      }
    }
  }
}
`

	swDoc, validationErrs, err := Unmarshal(ctx, strings.NewReader(input))
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	oaDoc, err := Upgrade(ctx, swDoc)
	require.NoError(t, err)
	require.NotNil(t, oaDoc)

	cfg := swDoc.GetCore().GetConfig()
	oaDoc.GetCore().SetConfig(cfg)

	var buf bytes.Buffer
	err = marshaller.Marshal(ctx, oaDoc, &buf)
	require.NoError(t, err)

	actual := buf.String()

	expected := `{
  "openapi": "3.0.0",
  "info": {
    "title": "Tags API",
    "version": "1.0.0"
  },
  "tags": [
    {
      "name": "users",
      "description": "User operations"
    },
    {
      "name": "admin",
      "description": "Admin operations"
    }
  ],
  "paths": {
    "/ping": {
      "get": {
        "responses": {
          "200": {
            "description": "ok"
          }
        }
      }
    }
  }
}
`

	require.Equal(t, expected, actual, "root tags should be preserved")
}

func TestUpgrade_SecurityDefinitions_AllOAuthFlows_JSON_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	input := `{
  "swagger": "2.0",
  "info": { "title": "OAuth Flows API", "version": "1.0.0" },
  "securityDefinitions": {
    "implicitFlow": {
      "type": "oauth2",
      "flow": "implicit",
      "authorizationUrl": "https://auth.example.com/authorize",
      "scopes": { "read": "Read access" }
    },
    "passwordFlow": {
      "type": "oauth2",
      "flow": "password",
      "tokenUrl": "https://auth.example.com/token",
      "scopes": { "write": "Write access" }
    },
    "applicationFlow": {
      "type": "oauth2",
      "flow": "application",
      "tokenUrl": "https://auth.example.com/token",
      "scopes": { "svc": "Service access" }
    },
    "accessCodeFlow": {
      "type": "oauth2",
      "flow": "accessCode",
      "authorizationUrl": "https://auth.example.com/authorize",
      "tokenUrl": "https://auth.example.com/token",
      "scopes": { "all": "All access" }
    }
  },
  "paths": {
    "/ok": {
      "get": {
        "responses": {
          "200": { "description": "ok" }
        }
      }
    }
  }
}
`

	swDoc, validationErrs, err := Unmarshal(ctx, strings.NewReader(input))
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	oaDoc, err := Upgrade(ctx, swDoc)
	require.NoError(t, err)
	require.NotNil(t, oaDoc)

	cfg := swDoc.GetCore().GetConfig()
	oaDoc.GetCore().SetConfig(cfg)

	var buf bytes.Buffer
	err = marshaller.Marshal(ctx, oaDoc, &buf)
	require.NoError(t, err)

	actual := buf.String()

	expected := `{
  "openapi": "3.0.0",
  "info": {
    "title": "OAuth Flows API",
    "version": "1.0.0"
  },
  "paths": {
    "/ok": {
      "get": {
        "responses": {
          "200": {
            "description": "ok"
          }
        }
      }
    }
  },
  "components": {
    "securitySchemes": {
      "implicitFlow": {
        "type": "oauth2",
        "flows": {
          "implicit": {
            "authorizationUrl": "https://auth.example.com/authorize",
            "scopes": {
              "read": "Read access"
            }
          }
        }
      },
      "passwordFlow": {
        "type": "oauth2",
        "flows": {
          "password": {
            "tokenUrl": "https://auth.example.com/token",
            "scopes": {
              "write": "Write access"
            }
          }
        }
      },
      "applicationFlow": {
        "type": "oauth2",
        "flows": {
          "clientCredentials": {
            "tokenUrl": "https://auth.example.com/token",
            "scopes": {
              "svc": "Service access"
            }
          }
        }
      },
      "accessCodeFlow": {
        "type": "oauth2",
        "flows": {
          "authorizationCode": {
            "authorizationUrl": "https://auth.example.com/authorize",
            "tokenUrl": "https://auth.example.com/token",
            "scopes": {
              "all": "All access"
            }
          }
        }
      }
    }
  }
}
`

	require.Equal(t, expected, actual, "all OAuth2 flows should map to OAS3 flows")
}

func TestUpgrade_GlobalSecurityRequirements_JSON_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	input := `{
  "swagger": "2.0",
  "info": { "title": "SecurityReq API", "version": "1.0.0" },
  "securityDefinitions": {
    "apiKeyHeader": {
      "type": "apiKey",
      "name": "X-API-Key",
      "in": "header"
    }
  },
  "security": [
    { "apiKeyHeader": [] }
  ],
  "paths": {
    "/ok": {
      "get": {
        "responses": {
          "200": { "description": "ok" }
        }
      }
    }
  }
}
`

	swDoc, validationErrs, err := Unmarshal(ctx, strings.NewReader(input))
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	oaDoc, err := Upgrade(ctx, swDoc)
	require.NoError(t, err)
	require.NotNil(t, oaDoc)

	cfg := swDoc.GetCore().GetConfig()
	oaDoc.GetCore().SetConfig(cfg)

	var buf bytes.Buffer
	err = marshaller.Marshal(ctx, oaDoc, &buf)
	require.NoError(t, err)

	actual := buf.String()

	expected := `{
  "openapi": "3.0.0",
  "info": {
    "title": "SecurityReq API",
    "version": "1.0.0"
  },
  "security": [
    {
      "apiKeyHeader": []
    }
  ],
  "paths": {
    "/ok": {
      "get": {
        "responses": {
          "200": {
            "description": "ok"
          }
        }
      }
    }
  },
  "components": {
    "securitySchemes": {
      "apiKeyHeader": {
        "type": "apiKey",
        "name": "X-API-Key",
        "in": "header"
      }
    }
  }
}
`

	require.Equal(t, expected, actual, "global security requirements should be preserved")
}

func TestUpgrade_PathLevelParameters_JSON_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	input := `{
  "swagger": "2.0",
  "info": { "title": "Path Params API", "version": "1.0.0" },
  "paths": {
    "/users/{id}": {
      "parameters": [
        { "name": "id", "in": "path", "required": true, "type": "string" },
        { "name": "version", "in": "query", "type": "integer" }
      ],
      "get": {
        "responses": {
          "200": { "description": "ok" }
        }
      }
    }
  }
}
`

	swDoc, validationErrs, err := Unmarshal(ctx, strings.NewReader(input))
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	oaDoc, err := Upgrade(ctx, swDoc)
	require.NoError(t, err)
	require.NotNil(t, oaDoc)

	cfg := swDoc.GetCore().GetConfig()
	oaDoc.GetCore().SetConfig(cfg)

	var buf bytes.Buffer
	err = marshaller.Marshal(ctx, oaDoc, &buf)
	require.NoError(t, err)

	actual := buf.String()

	expected := `{
  "openapi": "3.0.0",
  "info": {
    "title": "Path Params API",
    "version": "1.0.0"
  },
  "paths": {
    "/users/{id}": {
      "get": {
        "responses": {
          "200": {
            "description": "ok"
          }
        }
      },
      "parameters": [
        {
          "name": "id",
          "in": "path",
          "required": true,
          "schema": {
            "type": "string"
          }
        },
        {
          "name": "version",
          "in": "query",
          "schema": {
            "type": "integer"
          }
        }
      ]
    }
  }
}
`

	require.Equal(t, expected, actual, "path-level non-body parameters should be preserved and mapped with schema")
}
