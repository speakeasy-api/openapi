package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

// SecurityScheme is the backing model for the Security Scheme Object.
type SecurityScheme struct {
	marshaller.CoreModel `model:"securityScheme"`

	Type             marshaller.Node[string]                    `key:"type"`
	Description      marshaller.Node[*string]                   `key:"description"`
	Name             marshaller.Node[*string]                   `key:"name"`
	In               marshaller.Node[*string]                   `key:"in"`
	Scheme           marshaller.Node[*string]                   `key:"scheme"`
	BearerFormat     marshaller.Node[*string]                   `key:"bearerFormat"`
	Flows            marshaller.Node[*OAuthFlows]               `key:"flows"`
	OpenIDConnectURL marshaller.Node[*string]                   `key:"openIdConnectUrl"`
	Scopes           marshaller.Node[[]marshaller.Node[string]] `key:"scopes"`

	Extensions core.Extensions `key:"extensions"`
}

// OAuthFlows is the backing model for the OAuth Flows Object.
type OAuthFlows struct {
	marshaller.CoreModel `model:"oAuthFlows"`

	Implicit          marshaller.Node[*OAuthFlow] `key:"implicit"`
	Password          marshaller.Node[*OAuthFlow] `key:"password"`
	ClientCredentials marshaller.Node[*OAuthFlow] `key:"clientCredentials"`
	AuthorizationCode marshaller.Node[*OAuthFlow] `key:"authorizationCode"`

	Extensions core.Extensions `key:"extensions"`
}

// OAuthFlow is the backing model for the OAuth Flow Object.
type OAuthFlow struct {
	marshaller.CoreModel `model:"oAuthFlow"`

	AuthorizationURL marshaller.Node[*string]                                            `key:"authorizationUrl"`
	TokenURL         marshaller.Node[*string]                                            `key:"tokenUrl"`
	RefreshURL       marshaller.Node[*string]                                            `key:"refreshUrl"`
	AvailableScopes  marshaller.Node[*sequencedmap.Map[string, marshaller.Node[string]]] `key:"availableScopes" required:"true"`

	Extensions core.Extensions `key:"extensions"`
}
