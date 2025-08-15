package openapi_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/require"
)

func TestCallback_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	yml := `
'{$request.body#/webhookUrl}':
  post:
    summary: Webhook notification
    description: Receives webhook notifications from the service
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            properties:
              event:
                type: string
              data:
                type: object
    responses:
      '200':
        description: Webhook received successfully
      '400':
        description: Invalid webhook payload
'{$request.body#/callbackUrl}?event={$request.body#/eventType}':
  put:
    summary: Callback notification
    description: Receives callback notifications with event type
    requestBody:
      content:
        application/json:
          schema:
            type: object
    responses:
      '200':
        description: Callback received
      '404':
        description: Callback endpoint not found
x-custom: value
x-timeout: 30
x-retry-count: 3
`

	var callback openapi.Callback

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &callback)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Verify callback structure
	require.Equal(t, 2, callback.Len())

	// Verify first runtime expression
	webhookPath, exists := callback.Get("{$request.body#/webhookUrl}")
	require.True(t, exists)
	require.NotNil(t, webhookPath.Object)

	// Verify POST operation in webhook path
	postOp := webhookPath.Object.Post()
	require.NotNil(t, postOp)
	require.Equal(t, "Webhook notification", postOp.GetSummary())
	require.Equal(t, "Receives webhook notifications from the service", postOp.GetDescription())
	require.NotNil(t, postOp.RequestBody)
	require.NotNil(t, postOp.Responses)

	// Verify responses in POST operation
	require.Equal(t, 2, postOp.Responses.Len())
	response200, exists := postOp.Responses.Get("200")
	require.True(t, exists)
	require.Equal(t, "Webhook received successfully", response200.Object.GetDescription())

	response400, exists := postOp.Responses.Get("400")
	require.True(t, exists)
	require.Equal(t, "Invalid webhook payload", response400.Object.GetDescription())

	// Verify second runtime expression
	callbackPath, exists := callback.Get("{$request.body#/callbackUrl}?event={$request.body#/eventType}")
	require.True(t, exists)
	require.NotNil(t, callbackPath.Object)

	// Verify PUT operation in callback path
	putOp := callbackPath.Object.Put()
	require.NotNil(t, putOp)
	require.Equal(t, "Callback notification", putOp.GetSummary())
	require.Equal(t, "Receives callback notifications with event type", putOp.GetDescription())
	require.NotNil(t, putOp.RequestBody)
	require.NotNil(t, putOp.Responses)

	// Verify responses in PUT operation
	require.Equal(t, 2, putOp.Responses.Len())
	putResponse200, exists := putOp.Responses.Get("200")
	require.True(t, exists)
	require.Equal(t, "Callback received", putResponse200.Object.GetDescription())

	putResponse404, exists := putOp.Responses.Get("404")
	require.True(t, exists)
	require.Equal(t, "Callback endpoint not found", putResponse404.Object.GetDescription())

	// Verify extensions
	require.NotNil(t, callback.Extensions)
	require.True(t, callback.Extensions.Has("x-custom"))
	require.True(t, callback.Extensions.Has("x-timeout"))
	require.True(t, callback.Extensions.Has("x-retry-count"))
}
