package openapi

import (
	"fmt"
	"reflect"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	walkpkg "github.com/speakeasy-api/openapi/walk"
)

// Matcher is a struct that can be used to match specific nodes in the OpenAPI document.
type Matcher struct {
	OpenAPI                  func(*OpenAPI) error
	Info                     func(*Info) error
	Contact                  func(*Contact) error
	License                  func(*License) error
	ExternalDocs             func(*oas3.ExternalDocumentation) error
	Tag                      func(*Tag) error
	Server                   func(*Server) error
	ServerVariable           func(*ServerVariable) error
	Security                 func(*SecurityRequirement) error
	Paths                    func(*Paths) error
	ReferencedPathItem       func(*ReferencedPathItem) error
	ReferencedParameter      func(*ReferencedParameter) error
	Schema                   func(*oas3.JSONSchema[oas3.Referenceable]) error
	Discriminator            func(*oas3.Discriminator) error
	XML                      func(*oas3.XML) error
	MediaType                func(*MediaType) error
	Encoding                 func(*Encoding) error
	ReferencedHeader         func(*ReferencedHeader) error
	ReferencedExample        func(*ReferencedExample) error
	Operation                func(*Operation) error
	ReferencedRequestBody    func(*ReferencedRequestBody) error
	Responses                func(*Responses) error
	ReferencedResponse       func(*ReferencedResponse) error
	ReferencedLink           func(*ReferencedLink) error
	ReferencedCallback       func(*ReferencedCallback) error
	Components               func(*Components) error
	ReferencedSecurityScheme func(*ReferencedSecurityScheme) error
	OAuthFlows               func(*OAuthFlows) error
	OAuthFlow                func(*OAuthFlow) error
	Extensions               func(*extensions.Extensions) error
	Any                      func(any) error // Any will be called along with the other functions above on a match of a model
}

// MatchFunc represents a particular model in the OpenAPI document that can be matched.
// Pass it a Matcher with the appropriate functions populated to match the model type(s) you are interested in.
type MatchFunc func(Matcher) error

// Use the shared walking infrastructure
type LocationContext = walkpkg.LocationContext[MatchFunc]
type Locations = walkpkg.Locations[MatchFunc]

type matchHandler[T any] struct {
	GetSpecific func(m Matcher) func(*T) error
}

var matchRegistry = map[reflect.Type]any{
	reflect.TypeOf((*OpenAPI)(nil)): matchHandler[OpenAPI]{
		GetSpecific: func(m Matcher) func(*OpenAPI) error { return m.OpenAPI },
	},
	reflect.TypeOf((*Info)(nil)): matchHandler[Info]{
		GetSpecific: func(m Matcher) func(*Info) error { return m.Info },
	},
	reflect.TypeOf((*Contact)(nil)): matchHandler[Contact]{
		GetSpecific: func(m Matcher) func(*Contact) error { return m.Contact },
	},
	reflect.TypeOf((*License)(nil)): matchHandler[License]{
		GetSpecific: func(m Matcher) func(*License) error { return m.License },
	},
	reflect.TypeOf((*oas3.ExternalDocumentation)(nil)): matchHandler[oas3.ExternalDocumentation]{
		GetSpecific: func(m Matcher) func(*oas3.ExternalDocumentation) error { return m.ExternalDocs },
	},
	reflect.TypeOf((*Tag)(nil)): matchHandler[Tag]{
		GetSpecific: func(m Matcher) func(*Tag) error { return m.Tag },
	},
	reflect.TypeOf((*Server)(nil)): matchHandler[Server]{
		GetSpecific: func(m Matcher) func(*Server) error { return m.Server },
	},
	reflect.TypeOf((*ServerVariable)(nil)): matchHandler[ServerVariable]{
		GetSpecific: func(m Matcher) func(*ServerVariable) error { return m.ServerVariable },
	},
	reflect.TypeOf((*SecurityRequirement)(nil)): matchHandler[SecurityRequirement]{
		GetSpecific: func(m Matcher) func(*SecurityRequirement) error { return m.Security },
	},
	reflect.TypeOf((*Paths)(nil)): matchHandler[Paths]{
		GetSpecific: func(m Matcher) func(*Paths) error { return m.Paths },
	},
	reflect.TypeOf((*ReferencedPathItem)(nil)): matchHandler[ReferencedPathItem]{
		GetSpecific: func(m Matcher) func(*ReferencedPathItem) error { return m.ReferencedPathItem },
	},
	reflect.TypeOf((*ReferencedParameter)(nil)): matchHandler[ReferencedParameter]{
		GetSpecific: func(m Matcher) func(*ReferencedParameter) error { return m.ReferencedParameter },
	},
	reflect.TypeOf((*oas3.JSONSchema[oas3.Referenceable])(nil)): matchHandler[oas3.JSONSchema[oas3.Referenceable]]{
		GetSpecific: func(m Matcher) func(*oas3.JSONSchema[oas3.Referenceable]) error { return m.Schema },
	},
	reflect.TypeOf((*oas3.Discriminator)(nil)): matchHandler[oas3.Discriminator]{
		GetSpecific: func(m Matcher) func(*oas3.Discriminator) error { return m.Discriminator },
	},
	reflect.TypeOf((*oas3.XML)(nil)): matchHandler[oas3.XML]{
		GetSpecific: func(m Matcher) func(*oas3.XML) error { return m.XML },
	},
	reflect.TypeOf((*MediaType)(nil)): matchHandler[MediaType]{
		GetSpecific: func(m Matcher) func(*MediaType) error { return m.MediaType },
	},
	reflect.TypeOf((*Encoding)(nil)): matchHandler[Encoding]{
		GetSpecific: func(m Matcher) func(*Encoding) error { return m.Encoding },
	},
	reflect.TypeOf((*ReferencedHeader)(nil)): matchHandler[ReferencedHeader]{
		GetSpecific: func(m Matcher) func(*ReferencedHeader) error { return m.ReferencedHeader },
	},
	reflect.TypeOf((*ReferencedExample)(nil)): matchHandler[ReferencedExample]{
		GetSpecific: func(m Matcher) func(*ReferencedExample) error { return m.ReferencedExample },
	},
	reflect.TypeOf((*Operation)(nil)): matchHandler[Operation]{
		GetSpecific: func(m Matcher) func(*Operation) error { return m.Operation },
	},
	reflect.TypeOf((*ReferencedRequestBody)(nil)): matchHandler[ReferencedRequestBody]{
		GetSpecific: func(m Matcher) func(*ReferencedRequestBody) error { return m.ReferencedRequestBody },
	},
	reflect.TypeOf((*Responses)(nil)): matchHandler[Responses]{
		GetSpecific: func(m Matcher) func(*Responses) error { return m.Responses },
	},
	reflect.TypeOf((*ReferencedResponse)(nil)): matchHandler[ReferencedResponse]{
		GetSpecific: func(m Matcher) func(*ReferencedResponse) error { return m.ReferencedResponse },
	},
	reflect.TypeOf((*ReferencedLink)(nil)): matchHandler[ReferencedLink]{
		GetSpecific: func(m Matcher) func(*ReferencedLink) error { return m.ReferencedLink },
	},
	reflect.TypeOf((*ReferencedCallback)(nil)): matchHandler[ReferencedCallback]{
		GetSpecific: func(m Matcher) func(*ReferencedCallback) error { return m.ReferencedCallback },
	},
	reflect.TypeOf((*Components)(nil)): matchHandler[Components]{
		GetSpecific: func(m Matcher) func(*Components) error { return m.Components },
	},
	reflect.TypeOf((*ReferencedSecurityScheme)(nil)): matchHandler[ReferencedSecurityScheme]{
		GetSpecific: func(m Matcher) func(*ReferencedSecurityScheme) error { return m.ReferencedSecurityScheme },
	},
	reflect.TypeOf((*OAuthFlows)(nil)): matchHandler[OAuthFlows]{
		GetSpecific: func(m Matcher) func(*OAuthFlows) error { return m.OAuthFlows },
	},
	reflect.TypeOf((*OAuthFlow)(nil)): matchHandler[OAuthFlow]{
		GetSpecific: func(m Matcher) func(*OAuthFlow) error { return m.OAuthFlow },
	},
	reflect.TypeOf((*extensions.Extensions)(nil)): matchHandler[extensions.Extensions]{
		GetSpecific: func(m Matcher) func(*extensions.Extensions) error { return m.Extensions },
	},
}

func getMatchFunc[T any](target *T) MatchFunc {
	t := reflect.TypeOf(target)

	h, ok := matchRegistry[t]
	if !ok {
		panic(fmt.Sprintf("no match handler registered for type %v", t))
	}

	handler := h.(matchHandler[T])
	return func(m Matcher) error {
		if m.Any != nil {
			if err := m.Any(target); err != nil {
				return err
			}
		}
		if specific := handler.GetSpecific(m); specific != nil {
			return specific(target)
		}
		return nil
	}
}
