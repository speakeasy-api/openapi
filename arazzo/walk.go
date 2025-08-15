package arazzo

import (
	"context"
	"iter"
	"reflect"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
	walkpkg "github.com/speakeasy-api/openapi/walk"
)

// WalkItem represents a single item yielded by the Walk iterator.
type WalkItem struct {
	Match    MatchFunc
	Location Locations
	Arazzo   *Arazzo
}

// MatchFunc represents a particular model in the Arazzo document that can be matched.
// Pass it a Matcher with the appropriate functions populated to match the model type(s) you are interested in.
type MatchFunc func(Matcher) error

// Use the shared walking infrastructure
type LocationContext = walkpkg.LocationContext[MatchFunc]
type Locations = walkpkg.Locations[MatchFunc]

// Matcher is a struct that can be used to match specific nodes in the Arazzo document.
type Matcher struct {
	Arazzo                func(*Arazzo) error
	Info                  func(*Info) error
	SourceDescription     func(*SourceDescription) error
	Workflow              func(*Workflow) error
	ReusableParameter     func(*ReusableParameter) error
	JSONSchema            func(*oas3.JSONSchema[oas3.Referenceable]) error
	Step                  func(*Step) error
	ReusableSuccessAction func(*ReusableSuccessAction) error
	ReusableFailureAction func(*ReusableFailureAction) error
	Components            func(*Components) error
	Parameter             func(*Parameter) error
	SuccessAction         func(*SuccessAction) error
	FailureAction         func(*FailureAction) error
	Extensions            func(*extensions.Extensions) error
	Any                   func(any) error // Any will be called along with the other functions above on a match of a model
}

// Walk returns an iterator that yields MatchFunc items for each model in the Arazzo document.
// Users can iterate over the results using a for loop and break out at any time.
func Walk(ctx context.Context, arazzo *Arazzo) iter.Seq[WalkItem] {
	return func(yield func(WalkItem) bool) {
		if arazzo == nil {
			return
		}
		walk(ctx, arazzo, yield)
	}
}

func walk(ctx context.Context, arazzo *Arazzo, yield func(WalkItem) bool) {
	arazzoMatchFunc := getMatchFunc(arazzo)

	// Visit the root Arazzo document first, location nil to specify the root
	if !yield(WalkItem{Match: arazzoMatchFunc, Location: nil, Arazzo: arazzo}) {
		return
	}

	// Visit each of the top level fields in turn populating their location context with field and any key/index information
	loc := Locations{}

	if !walkInfo(ctx, &arazzo.Info, append(loc, LocationContext{Parent: arazzoMatchFunc, ParentField: "info"}), arazzo, yield) {
		return
	}

	if !walkSourceDescriptions(ctx, arazzo.SourceDescriptions, append(loc, LocationContext{Parent: arazzoMatchFunc, ParentField: "sourceDescriptions"}), arazzo, yield) {
		return
	}

	if !walkWorkflows(ctx, arazzo.Workflows, append(loc, LocationContext{Parent: arazzoMatchFunc, ParentField: "workflows"}), arazzo, yield) {
		return
	}

	if !walkComponents(ctx, arazzo.Components, append(loc, LocationContext{Parent: arazzoMatchFunc, ParentField: "components"}), arazzo, yield) {
		return
	}

	// Visit Arazzo Extensions
	yield(WalkItem{Match: getMatchFunc(arazzo.Extensions), Location: append(loc, LocationContext{Parent: arazzoMatchFunc, ParentField: ""}), Arazzo: arazzo})
}

func walkInfo(_ context.Context, info *Info, loc Locations, arazzo *Arazzo, yield func(WalkItem) bool) bool {
	if info == nil {
		return true
	}

	infoMatchFunc := getMatchFunc(info)

	if !yield(WalkItem{Match: infoMatchFunc, Location: loc, Arazzo: arazzo}) {
		return false
	}

	// Visit Info Extensions
	return yield(WalkItem{Match: getMatchFunc(info.Extensions), Location: append(loc, LocationContext{Parent: infoMatchFunc, ParentField: ""}), Arazzo: arazzo})
}

func walkSourceDescriptions(ctx context.Context, sourceDescriptions []*SourceDescription, loc Locations, arazzo *Arazzo, yield func(WalkItem) bool) bool {
	if len(sourceDescriptions) == 0 {
		return true
	}

	// Get the last loc so we can set the parent index
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for i, sd := range sourceDescriptions {
		parentLoc.ParentIndex = pointer.From(i)

		if !walkSourceDescription(ctx, sd, append(loc, parentLoc), arazzo, yield) {
			return false
		}
	}
	return true
}

func walkSourceDescription(_ context.Context, sd *SourceDescription, loc Locations, arazzo *Arazzo, yield func(WalkItem) bool) bool {
	if sd == nil {
		return true
	}

	sdMatchFunc := getMatchFunc(sd)

	if !yield(WalkItem{Match: sdMatchFunc, Location: loc, Arazzo: arazzo}) {
		return false
	}

	// Visit SourceDescription Extensions
	return yield(WalkItem{Match: getMatchFunc(sd.Extensions), Location: append(loc, LocationContext{Parent: sdMatchFunc, ParentField: ""}), Arazzo: arazzo})
}

func walkWorkflows(ctx context.Context, workflows []*Workflow, loc Locations, arazzo *Arazzo, yield func(WalkItem) bool) bool {
	if len(workflows) == 0 {
		return true
	}

	// Get the last loc so we can set the parent index
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for i, workflow := range workflows {
		parentLoc.ParentIndex = pointer.From(i)

		if !walkWorkflow(ctx, workflow, append(loc, parentLoc), arazzo, yield) {
			return false
		}
	}
	return true
}

func walkWorkflow(ctx context.Context, workflow *Workflow, loc Locations, arazzo *Arazzo, yield func(WalkItem) bool) bool {
	if workflow == nil {
		return true
	}

	workflowMatchFunc := getMatchFunc(workflow)

	if !yield(WalkItem{Match: workflowMatchFunc, Location: loc, Arazzo: arazzo}) {
		return false
	}

	// Walk through parameters
	if !walkReusableParameters(ctx, workflow.Parameters, append(loc, LocationContext{Parent: workflowMatchFunc, ParentField: "parameters"}), arazzo, yield) {
		return false
	}

	// Walk through inputs schema using oas3 walking
	if !walkJSONSchema(ctx, workflow.Inputs, append(loc, LocationContext{Parent: workflowMatchFunc, ParentField: "inputs"}), arazzo, yield) {
		return false
	}

	// Walk through steps
	if !walkSteps(ctx, workflow.Steps, append(loc, LocationContext{Parent: workflowMatchFunc, ParentField: "steps"}), arazzo, yield) {
		return false
	}

	// Walk through success actions
	if !walkReusableSuccessActions(ctx, workflow.SuccessActions, append(loc, LocationContext{Parent: workflowMatchFunc, ParentField: "successActions"}), arazzo, yield) {
		return false
	}

	// Walk through failure actions
	if !walkReusableFailureActions(ctx, workflow.FailureActions, append(loc, LocationContext{Parent: workflowMatchFunc, ParentField: "failureActions"}), arazzo, yield) {
		return false
	}

	// Visit Workflow Extensions
	return yield(WalkItem{Match: getMatchFunc(workflow.Extensions), Location: append(loc, LocationContext{Parent: workflowMatchFunc, ParentField: ""}), Arazzo: arazzo})
}

func walkReusableParameters(ctx context.Context, parameters []*ReusableParameter, loc Locations, arazzo *Arazzo, yield func(WalkItem) bool) bool {
	if len(parameters) == 0 {
		return true
	}

	// Get the last loc so we can set the parent index
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for i, parameter := range parameters {
		parentLoc.ParentIndex = pointer.From(i)

		if !walkReusableParameter(ctx, parameter, append(loc, parentLoc), arazzo, yield) {
			return false
		}
	}
	return true
}

func walkReusableParameter(_ context.Context, parameter *ReusableParameter, loc Locations, arazzo *Arazzo, yield func(WalkItem) bool) bool {
	if parameter == nil {
		return true
	}

	parameterMatchFunc := getMatchFunc(parameter)

	if !yield(WalkItem{Match: parameterMatchFunc, Location: loc, Arazzo: arazzo}) {
		return false
	}

	// Visit ReusableParameter Extensions
	// ReusableParameter doesn't have Extensions field, so we skip it
	return true
}

func walkJSONSchema(ctx context.Context, schema *oas3.JSONSchema[oas3.Referenceable], loc Locations, arazzo *Arazzo, yield func(WalkItem) bool) bool {
	if schema == nil {
		return true
	}

	// Use the oas3 package's walking functionality
	for item := range oas3.Walk(ctx, schema) {
		// Convert the oas3 walk item to an arazzo walk item
		arazzoMatchFunc := convertSchemaMatchFunc(item.Match)
		arazzoLocation := convertSchemaLocation(item.Location, loc)

		if !yield(WalkItem{Match: arazzoMatchFunc, Location: arazzoLocation, Arazzo: arazzo}) {
			return false
		}
	}

	return true
}

// convertSchemaMatchFunc converts an oas3.SchemaMatchFunc to an arazzo.MatchFunc
func convertSchemaMatchFunc(schemaMatchFunc oas3.SchemaMatchFunc) MatchFunc {
	return func(m Matcher) error {
		return schemaMatchFunc(oas3.SchemaMatcher{
			Schema:        m.JSONSchema,
			Discriminator: nil, // Arazzo doesn't have discriminator matcher
			XML:           nil, // Arazzo doesn't have XML matcher
			ExternalDocs:  nil, // Arazzo doesn't have external docs matcher
			Extensions:    m.Extensions,
			Any:           m.Any,
		})
	}
}

// convertSchemaLocation converts oas3 schema locations to arazzo locations
func convertSchemaLocation(schemaLoc walkpkg.Locations[oas3.SchemaMatchFunc], baseLoc Locations) Locations {
	// Start with the base location (where the schema is located in the Arazzo document)
	result := make(Locations, len(baseLoc))
	copy(result, baseLoc)

	// Convert each oas3 location context to arazzo location context
	for _, schemaLocCtx := range schemaLoc {
		result = append(result, LocationContext{
			Parent:      convertSchemaMatchFunc(schemaLocCtx.Parent),
			ParentField: schemaLocCtx.ParentField,
			ParentKey:   schemaLocCtx.ParentKey,
			ParentIndex: schemaLocCtx.ParentIndex,
		})
	}

	return result
}

func walkSteps(ctx context.Context, steps []*Step, loc Locations, arazzo *Arazzo, yield func(WalkItem) bool) bool {
	if len(steps) == 0 {
		return true
	}

	// Get the last loc so we can set the parent index
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for i, step := range steps {
		parentLoc.ParentIndex = pointer.From(i)

		if !walkStep(ctx, step, append(loc, parentLoc), arazzo, yield) {
			return false
		}
	}
	return true
}

func walkStep(ctx context.Context, step *Step, loc Locations, arazzo *Arazzo, yield func(WalkItem) bool) bool {
	if step == nil {
		return true
	}

	stepMatchFunc := getMatchFunc(step)

	if !yield(WalkItem{Match: stepMatchFunc, Location: loc, Arazzo: arazzo}) {
		return false
	}

	// Walk through parameters
	if !walkReusableParameters(ctx, step.Parameters, append(loc, LocationContext{Parent: stepMatchFunc, ParentField: "parameters"}), arazzo, yield) {
		return false
	}

	// Walk through success actions
	if !walkReusableSuccessActions(ctx, step.OnSuccess, append(loc, LocationContext{Parent: stepMatchFunc, ParentField: "onSuccess"}), arazzo, yield) {
		return false
	}

	// Walk through failure actions
	if !walkReusableFailureActions(ctx, step.OnFailure, append(loc, LocationContext{Parent: stepMatchFunc, ParentField: "onFailure"}), arazzo, yield) {
		return false
	}

	// Visit Step Extensions
	return yield(WalkItem{Match: getMatchFunc(step.Extensions), Location: append(loc, LocationContext{Parent: stepMatchFunc, ParentField: ""}), Arazzo: arazzo})
}

func walkReusableSuccessActions(ctx context.Context, actions []*ReusableSuccessAction, loc Locations, arazzo *Arazzo, yield func(WalkItem) bool) bool {
	if len(actions) == 0 {
		return true
	}

	// Get the last loc so we can set the parent index
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for i, action := range actions {
		parentLoc.ParentIndex = pointer.From(i)

		if !walkReusableSuccessAction(ctx, action, append(loc, parentLoc), arazzo, yield) {
			return false
		}
	}
	return true
}

func walkReusableSuccessAction(_ context.Context, action *ReusableSuccessAction, loc Locations, arazzo *Arazzo, yield func(WalkItem) bool) bool {
	if action == nil {
		return true
	}

	actionMatchFunc := getMatchFunc(action)

	if !yield(WalkItem{Match: actionMatchFunc, Location: loc, Arazzo: arazzo}) {
		return false
	}

	// Visit ReusableSuccessAction Extensions
	// ReusableSuccessAction doesn't have Extensions field, so we skip it
	return true
}

func walkReusableFailureActions(ctx context.Context, actions []*ReusableFailureAction, loc Locations, arazzo *Arazzo, yield func(WalkItem) bool) bool {
	if len(actions) == 0 {
		return true
	}

	// Get the last loc so we can set the parent index
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for i, action := range actions {
		parentLoc.ParentIndex = pointer.From(i)

		if !walkReusableFailureAction(ctx, action, append(loc, parentLoc), arazzo, yield) {
			return false
		}
	}
	return true
}

func walkReusableFailureAction(_ context.Context, action *ReusableFailureAction, loc Locations, arazzo *Arazzo, yield func(WalkItem) bool) bool {
	if action == nil {
		return true
	}

	actionMatchFunc := getMatchFunc(action)

	if !yield(WalkItem{Match: actionMatchFunc, Location: loc, Arazzo: arazzo}) {
		return false
	}

	// Visit ReusableFailureAction Extensions
	// ReusableFailureAction doesn't have Extensions field, so we skip it
	return true
}

func walkComponents(ctx context.Context, components *Components, loc Locations, arazzo *Arazzo, yield func(WalkItem) bool) bool {
	if components == nil {
		return true
	}

	componentsMatchFunc := getMatchFunc(components)

	if !yield(WalkItem{Match: componentsMatchFunc, Location: loc, Arazzo: arazzo}) {
		return false
	}

	// Walk through inputs
	if !walkComponentInputs(ctx, components.Inputs, append(loc, LocationContext{Parent: componentsMatchFunc, ParentField: "inputs"}), arazzo, yield) {
		return false
	}

	// Walk through parameters
	if !walkComponentParameters(ctx, components.Parameters, append(loc, LocationContext{Parent: componentsMatchFunc, ParentField: "parameters"}), arazzo, yield) {
		return false
	}

	// Walk through success actions
	if !walkComponentSuccessActions(ctx, components.SuccessActions, append(loc, LocationContext{Parent: componentsMatchFunc, ParentField: "successActions"}), arazzo, yield) {
		return false
	}

	// Walk through failure actions
	if !walkComponentFailureActions(ctx, components.FailureActions, append(loc, LocationContext{Parent: componentsMatchFunc, ParentField: "failureActions"}), arazzo, yield) {
		return false
	}

	// Visit Components Extensions
	return yield(WalkItem{Match: getMatchFunc(components.Extensions), Location: append(loc, LocationContext{Parent: componentsMatchFunc, ParentField: ""}), Arazzo: arazzo})
}

func walkComponentInputs(ctx context.Context, inputs *sequencedmap.Map[string, *oas3.JSONSchema[oas3.Referenceable]], loc Locations, arazzo *Arazzo, yield func(WalkItem) bool) bool {
	if inputs == nil || inputs.Len() == 0 {
		return true
	}

	// Get the last loc so we can set the parent key
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for name, schema := range inputs.All() {
		parentLoc.ParentKey = pointer.From(name)

		if !walkJSONSchema(ctx, schema, append(loc, parentLoc), arazzo, yield) {
			return false
		}
	}
	return true
}

func walkComponentParameters(ctx context.Context, parameters *sequencedmap.Map[string, *Parameter], loc Locations, arazzo *Arazzo, yield func(WalkItem) bool) bool {
	if parameters == nil || parameters.Len() == 0 {
		return true
	}

	// Get the last loc so we can set the parent key
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for name, parameter := range parameters.All() {
		parentLoc.ParentKey = pointer.From(name)

		if !walkParameter(ctx, parameter, append(loc, parentLoc), arazzo, yield) {
			return false
		}
	}
	return true
}

func walkParameter(_ context.Context, parameter *Parameter, loc Locations, arazzo *Arazzo, yield func(WalkItem) bool) bool {
	if parameter == nil {
		return true
	}

	parameterMatchFunc := getMatchFunc(parameter)

	if !yield(WalkItem{Match: parameterMatchFunc, Location: loc, Arazzo: arazzo}) {
		return false
	}

	// Visit Parameter Extensions
	return yield(WalkItem{Match: getMatchFunc(parameter.Extensions), Location: append(loc, LocationContext{Parent: parameterMatchFunc, ParentField: ""}), Arazzo: arazzo})
}

func walkComponentSuccessActions(ctx context.Context, actions *sequencedmap.Map[string, *SuccessAction], loc Locations, arazzo *Arazzo, yield func(WalkItem) bool) bool {
	if actions == nil || actions.Len() == 0 {
		return true
	}

	// Get the last loc so we can set the parent key
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for name, action := range actions.All() {
		parentLoc.ParentKey = pointer.From(name)

		if !walkSuccessAction(ctx, action, append(loc, parentLoc), arazzo, yield) {
			return false
		}
	}
	return true
}

func walkSuccessAction(_ context.Context, action *SuccessAction, loc Locations, arazzo *Arazzo, yield func(WalkItem) bool) bool {
	if action == nil {
		return true
	}

	actionMatchFunc := getMatchFunc(action)

	if !yield(WalkItem{Match: actionMatchFunc, Location: loc, Arazzo: arazzo}) {
		return false
	}

	// Visit SuccessAction Extensions
	return yield(WalkItem{Match: getMatchFunc(action.Extensions), Location: append(loc, LocationContext{Parent: actionMatchFunc, ParentField: ""}), Arazzo: arazzo})
}

func walkComponentFailureActions(ctx context.Context, actions *sequencedmap.Map[string, *FailureAction], loc Locations, arazzo *Arazzo, yield func(WalkItem) bool) bool {
	if actions == nil || actions.Len() == 0 {
		return true
	}

	// Get the last loc so we can set the parent key
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for name, action := range actions.All() {
		parentLoc.ParentKey = pointer.From(name)

		if !walkFailureAction(ctx, action, append(loc, parentLoc), arazzo, yield) {
			return false
		}
	}
	return true
}

func walkFailureAction(_ context.Context, action *FailureAction, loc Locations, arazzo *Arazzo, yield func(WalkItem) bool) bool {
	if action == nil {
		return true
	}

	actionMatchFunc := getMatchFunc(action)

	if !yield(WalkItem{Match: actionMatchFunc, Location: loc, Arazzo: arazzo}) {
		return false
	}

	// Visit FailureAction Extensions
	return yield(WalkItem{Match: getMatchFunc(action.Extensions), Location: append(loc, LocationContext{Parent: actionMatchFunc, ParentField: ""}), Arazzo: arazzo})
}

type matchHandler[T any] struct {
	GetSpecific func(m Matcher) func(*T) error
}

var matchRegistry = map[reflect.Type]any{
	reflect.TypeOf((*Arazzo)(nil)): matchHandler[Arazzo]{
		GetSpecific: func(m Matcher) func(*Arazzo) error { return m.Arazzo },
	},
	reflect.TypeOf((*Info)(nil)): matchHandler[Info]{
		GetSpecific: func(m Matcher) func(*Info) error { return m.Info },
	},
	reflect.TypeOf((*SourceDescription)(nil)): matchHandler[SourceDescription]{
		GetSpecific: func(m Matcher) func(*SourceDescription) error { return m.SourceDescription },
	},
	reflect.TypeOf((*Workflow)(nil)): matchHandler[Workflow]{
		GetSpecific: func(m Matcher) func(*Workflow) error { return m.Workflow },
	},
	reflect.TypeOf((*ReusableParameter)(nil)): matchHandler[ReusableParameter]{
		GetSpecific: func(m Matcher) func(*ReusableParameter) error { return m.ReusableParameter },
	},
	reflect.TypeOf((*oas3.JSONSchema[oas3.Referenceable])(nil)): matchHandler[oas3.JSONSchema[oas3.Referenceable]]{
		GetSpecific: func(m Matcher) func(*oas3.JSONSchema[oas3.Referenceable]) error { return m.JSONSchema },
	},
	reflect.TypeOf((*Step)(nil)): matchHandler[Step]{
		GetSpecific: func(m Matcher) func(*Step) error { return m.Step },
	},
	reflect.TypeOf((*ReusableSuccessAction)(nil)): matchHandler[ReusableSuccessAction]{
		GetSpecific: func(m Matcher) func(*ReusableSuccessAction) error { return m.ReusableSuccessAction },
	},
	reflect.TypeOf((*ReusableFailureAction)(nil)): matchHandler[ReusableFailureAction]{
		GetSpecific: func(m Matcher) func(*ReusableFailureAction) error { return m.ReusableFailureAction },
	},
	reflect.TypeOf((*Components)(nil)): matchHandler[Components]{
		GetSpecific: func(m Matcher) func(*Components) error { return m.Components },
	},
	reflect.TypeOf((*Parameter)(nil)): matchHandler[Parameter]{
		GetSpecific: func(m Matcher) func(*Parameter) error { return m.Parameter },
	},
	reflect.TypeOf((*SuccessAction)(nil)): matchHandler[SuccessAction]{
		GetSpecific: func(m Matcher) func(*SuccessAction) error { return m.SuccessAction },
	},
	reflect.TypeOf((*FailureAction)(nil)): matchHandler[FailureAction]{
		GetSpecific: func(m Matcher) func(*FailureAction) error { return m.FailureAction },
	},
	reflect.TypeOf((*extensions.Extensions)(nil)): matchHandler[extensions.Extensions]{
		GetSpecific: func(m Matcher) func(*extensions.Extensions) error { return m.Extensions },
	},
}

func getMatchFunc[T any](target *T) MatchFunc {
	t := reflect.TypeOf(target)

	h, ok := matchRegistry[t]
	if !ok {
		// For unknown types, just use the Any matcher
		return func(m Matcher) error {
			if m.Any != nil {
				return m.Any(target)
			}
			return nil
		}
	}

	handler, ok := h.(matchHandler[T])
	if !ok {
		// For unknown types, just use the Any matcher
		return func(m Matcher) error {
			if m.Any != nil {
				return m.Any(target)
			}
			return nil
		}
	}

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
