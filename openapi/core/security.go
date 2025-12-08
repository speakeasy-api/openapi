package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"gopkg.in/yaml.v3"
)

type SecurityScheme struct {
	marshaller.CoreModel `model:"securityScheme"`

	Type             marshaller.Node[string]      `key:"type"`
	Description      marshaller.Node[*string]     `key:"description"`
	Name             marshaller.Node[*string]     `key:"name"`
	In               marshaller.Node[*string]     `key:"in"`
	Scheme           marshaller.Node[*string]     `key:"scheme"`
	BearerFormat     marshaller.Node[*string]     `key:"bearerFormat"`
	Flows            marshaller.Node[*OAuthFlows] `key:"flows"`
	OpenIdConnectUrl marshaller.Node[*string]     `key:"openIdConnectUrl"`
	Extensions       core.Extensions              `key:"extensions"`
}

type SecurityRequirement struct {
	marshaller.CoreModel `model:"securityRequirement"`
	*sequencedmap.Map[string, marshaller.Node[[]marshaller.Node[string]]]
}

func NewSecurityRequirement() *SecurityRequirement {
	return &SecurityRequirement{
		Map: sequencedmap.New[string, marshaller.Node[[]marshaller.Node[string]]](),
	}
}

func (s *SecurityRequirement) GetMapKeyNodeOrRoot(key string, rootNode *yaml.Node) *yaml.Node {
	if !s.IsInitialized() {
		return rootNode
	}

	if s.RootNode == nil {
		return rootNode
	}

	for i := 0; i < len(s.RootNode.Content); i += 2 {
		if s.RootNode.Content[i].Value == key {
			return s.RootNode.Content[i]
		}
	}

	return rootNode
}

func (s *SecurityRequirement) GetMapKeyNodeOrRootLine(key string, rootNode *yaml.Node) int {
	node := s.GetMapKeyNodeOrRoot(key, rootNode)
	if node == nil {
		return -1
	}
	return node.Line
}

type OAuthFlows struct {
	marshaller.CoreModel `model:"oAuthFlows"`

	Implicit          marshaller.Node[*OAuthFlow] `key:"implicit"`
	Password          marshaller.Node[*OAuthFlow] `key:"password"`
	ClientCredentials marshaller.Node[*OAuthFlow] `key:"clientCredentials"`
	AuthorizationCode marshaller.Node[*OAuthFlow] `key:"authorizationCode"`
	Extensions        core.Extensions             `key:"extensions"`
}

type OAuthFlow struct {
	marshaller.CoreModel `model:"oAuthFlow"`

	AuthorizationURL marshaller.Node[*string]                           `key:"authorizationUrl"`
	TokenURL         marshaller.Node[*string]                           `key:"tokenUrl"`
	RefreshURL       marshaller.Node[*string]                           `key:"refreshUrl"`
	Scopes           marshaller.Node[*sequencedmap.Map[string, string]] `key:"scopes"`
	Extensions       core.Extensions                                    `key:"extensions"`
}
