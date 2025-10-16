package swagger

import (
	"fmt"
	"reflect"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	walkpkg "github.com/speakeasy-api/openapi/walk"
)

// Matcher is a struct that can be used to match specific nodes in the Swagger document.
type Matcher struct {
	Swagger             func(*Swagger) error
	Info                func(*Info) error
	Contact             func(*Contact) error
	License             func(*License) error
	ExternalDocs        func(*ExternalDocumentation) error
	Tag                 func(*Tag) error
	Paths               func(*Paths) error
	PathItem            func(*PathItem) error
	Operation           func(*Operation) error
	ReferencedParameter func(*ReferencedParameter) error
	Parameter           func(*Parameter) error
	Schema              func(*oas3.JSONSchema[oas3.Concrete]) error
	Discriminator       func(*oas3.Discriminator) error
	XML                 func(*oas3.XML) error
	Responses           func(*Responses) error
	ReferencedResponse  func(*ReferencedResponse) error
	Response            func(*Response) error
	Header              func(*Header) error
	Items               func(*Items) error
	SecurityRequirement func(*SecurityRequirement) error
	SecurityScheme      func(*SecurityScheme) error
	Extensions          func(*extensions.Extensions) error
	Any                 func(any) error // Any will be called along with the other functions above on a match of a model
}

// MatchFunc represents a particular model in the Swagger document that can be matched.
// Pass it a Matcher with the appropriate functions populated to match the model type(s) you are interested in.
type MatchFunc func(Matcher) error

// Use the shared walking infrastructure
type (
	LocationContext = walkpkg.LocationContext[MatchFunc]
	Locations       = walkpkg.Locations[MatchFunc]
)

type matchHandler[T any] struct {
	GetSpecific func(m Matcher) func(*T) error
}

var matchRegistry = map[reflect.Type]any{
	reflect.TypeOf((*Swagger)(nil)): matchHandler[Swagger]{
		GetSpecific: func(m Matcher) func(*Swagger) error { return m.Swagger },
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
	reflect.TypeOf((*ExternalDocumentation)(nil)): matchHandler[ExternalDocumentation]{
		GetSpecific: func(m Matcher) func(*ExternalDocumentation) error { return m.ExternalDocs },
	},
	reflect.TypeOf((*Tag)(nil)): matchHandler[Tag]{
		GetSpecific: func(m Matcher) func(*Tag) error { return m.Tag },
	},
	reflect.TypeOf((*Paths)(nil)): matchHandler[Paths]{
		GetSpecific: func(m Matcher) func(*Paths) error { return m.Paths },
	},
	reflect.TypeOf((*PathItem)(nil)): matchHandler[PathItem]{
		GetSpecific: func(m Matcher) func(*PathItem) error { return m.PathItem },
	},
	reflect.TypeOf((*Operation)(nil)): matchHandler[Operation]{
		GetSpecific: func(m Matcher) func(*Operation) error { return m.Operation },
	},
	reflect.TypeOf((*ReferencedParameter)(nil)): matchHandler[ReferencedParameter]{
		GetSpecific: func(m Matcher) func(*ReferencedParameter) error { return m.ReferencedParameter },
	},
	reflect.TypeOf((*Parameter)(nil)): matchHandler[Parameter]{
		GetSpecific: func(m Matcher) func(*Parameter) error { return m.Parameter },
	},
	reflect.TypeOf((*oas3.JSONSchema[oas3.Concrete])(nil)): matchHandler[oas3.JSONSchema[oas3.Concrete]]{
		GetSpecific: func(m Matcher) func(*oas3.JSONSchema[oas3.Concrete]) error { return m.Schema },
	},
	reflect.TypeOf((*oas3.Discriminator)(nil)): matchHandler[oas3.Discriminator]{
		GetSpecific: func(m Matcher) func(*oas3.Discriminator) error { return m.Discriminator },
	},
	reflect.TypeOf((*oas3.XML)(nil)): matchHandler[oas3.XML]{
		GetSpecific: func(m Matcher) func(*oas3.XML) error { return m.XML },
	},
	reflect.TypeOf((*Responses)(nil)): matchHandler[Responses]{
		GetSpecific: func(m Matcher) func(*Responses) error { return m.Responses },
	},
	reflect.TypeOf((*ReferencedResponse)(nil)): matchHandler[ReferencedResponse]{
		GetSpecific: func(m Matcher) func(*ReferencedResponse) error { return m.ReferencedResponse },
	},
	reflect.TypeOf((*Response)(nil)): matchHandler[Response]{
		GetSpecific: func(m Matcher) func(*Response) error { return m.Response },
	},
	reflect.TypeOf((*Header)(nil)): matchHandler[Header]{
		GetSpecific: func(m Matcher) func(*Header) error { return m.Header },
	},
	reflect.TypeOf((*Items)(nil)): matchHandler[Items]{
		GetSpecific: func(m Matcher) func(*Items) error { return m.Items },
	},
	reflect.TypeOf((*SecurityRequirement)(nil)): matchHandler[SecurityRequirement]{
		GetSpecific: func(m Matcher) func(*SecurityRequirement) error { return m.SecurityRequirement },
	},
	reflect.TypeOf((*SecurityScheme)(nil)): matchHandler[SecurityScheme]{
		GetSpecific: func(m Matcher) func(*SecurityScheme) error { return m.SecurityScheme },
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
