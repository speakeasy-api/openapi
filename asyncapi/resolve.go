package asyncapi

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/speakeasy-api/openapi/asyncapi/core"
	"github.com/speakeasy-api/openapi/internal/asyncapiversion"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/internal/utils"
	"github.com/speakeasy-api/openapi/jsonpointer"
	jsonschema "github.com/speakeasy-api/openapi/jsonschema/asyncapi"
	schemacore "github.com/speakeasy-api/openapi/jsonschema/asyncapi/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/system"
	"github.com/speakeasy-api/openapi/validation"
	"go.yaml.in/yaml/v3"
)

// resolutionMu guards the resolution state ResolveAllReferences stores on models. One lock covers every document:
// each piece of state is written once per call, so contention is negligible, and models stay safe to copy.
var resolutionMu sync.RWMutex

// ErrExternalReferencesDisabled is wrapped by the diagnostics reported for references to other documents when
// ResolveAllOptions.DisableExternalRefs is set, so callers can tell references that weren't followed apart from
// broken ones with errors.Is.
var ErrExternalReferencesDisabled = errors.New("external references are disabled")

// ResolveAllOptions are the options available when resolving all references in an AsyncAPI document.
type ResolveAllOptions struct {
	// AsyncAPILocation is the absolute file path or URL of the document. Relative references to other documents are
	// resolved against it. Without it, only references within the document and absolute references can be resolved.
	AsyncAPILocation string
	// DisableExternalRefs reports references to other documents as errors instead of loading them. The errors wrap
	// ErrExternalReferencesDisabled.
	DisableExternalRefs bool
	// VirtualFS is used for file based references. If not provided the local file system is used.
	VirtualFS system.VirtualFS
	// HTTPClient is used for URL based references. If not provided http.DefaultClient is used. It is not a sandbox:
	// callers that need to restrict which hosts are contacted must provide a client that enforces it.
	HTTPClient system.Client
}

// ResolvedSchema is a schema Reference Object resolved by ResolveAllReferences.
type ResolvedSchema struct {
	// Schema is the resolved schema for native AsyncAPI and JSON Schema based formats, and for references within the
	// document.
	Schema *jsonschema.SchemaUnion
	// Raw is the resolved resource for other formats, such as an Avro schema file, preserved unchanged.
	Raw *yaml.Node
	// SchemaFormat is the format the resource was interpreted with.
	SchemaFormat string
	// Reference is the absolute location the schema was resolved from.
	Reference references.Reference
}

type resolutionState struct {
	resets       []func()
	schemas      map[*jsonschema.SchemaUnion]*ResolvedSchema
	externalDocs map[*ExternalDocumentation]*ExternalDocumentation
}

// GetResolvedSchema returns the resolved target of a schema Reference Object, either a native schema `$ref` or a
// Multi Format Schema Object whose `schema` is a Reference Object, from the last call to ResolveAllReferences.
// Returns nil if the schema isn't a reference or hasn't been resolved.
func (a *AsyncAPI) GetResolvedSchema(schema *jsonschema.SchemaUnion) *ResolvedSchema {
	if a == nil || schema == nil {
		return nil
	}
	resolutionMu.RLock()
	defer resolutionMu.RUnlock()
	if a.resolution == nil {
		return nil
	}
	return a.resolution.schemas[schema]
}

// GetResolvedExternalDocs returns the external documentation itself, or for a Reference Object the documentation it
// resolved to in the last call to ResolveAllReferences. Returns nil for an unresolved Reference Object.
func (a *AsyncAPI) GetResolvedExternalDocs(docs *ExternalDocumentation) *ExternalDocumentation {
	if a == nil || docs == nil {
		return nil
	}
	if !docs.IsReference() {
		return docs
	}
	resolutionMu.RLock()
	defer resolutionMu.RUnlock()
	if a.resolution == nil {
		return nil
	}
	return a.resolution.externalDocs[docs]
}

// ResolveAllReferences resolves every reference in the document, applies traits and runs the checks that need
// referenced objects. It is the only part of this package that reads files or makes network requests, and only
// does so for references the document contains, using the provided options.
//
// Each call rebuilds the resolution state from the current document, so edits made since an earlier call are
// reflected and no stale targets are returned. Resolution doesn't change the authored document: references stay as
// written and are not bundled.
//
// Problems with references, such as missing or wrong-kind targets, cycles, references to other documents that aren't
// allowed and diagnostics from loaded documents, are returned as validation errors. The returned error is reserved
// for failures that stop resolution, such as a cancelled context.
func (a *AsyncAPI) ResolveAllReferences(ctx context.Context, opts ResolveAllOptions) ([]error, error) {
	if a == nil {
		return nil, nil
	}
	if err := a.Sync(ctx); err != nil {
		return nil, fmt.Errorf("failed to sync AsyncAPI document: %w", err)
	}

	rc := &resolveCall{
		doc:  a,
		opts: opts,
		root: location{uri: opts.AsyncAPILocation, doc: a},
		state: &resolutionState{
			schemas:      map[*jsonschema.SchemaUnion]*ResolvedSchema{},
			externalDocs: map[*ExternalDocumentation]*ExternalDocumentation{},
		},
		cache: resolveCache{
			objects:  map[string]any{},
			docs:     map[string][]byte{},
			external: map[string]any{},
		},
		validationOpts: a.validationOptions(),
		queued:         map[any]bool{},
		errs:           []error{},
	}

	resolutionMu.Lock()
	previous := a.resolution
	a.resolution = rc.state
	resolutionMu.Unlock()
	if previous != nil {
		for _, reset := range previous.resets {
			reset()
		}
	}

	rc.run(ctx)

	if err := ctx.Err(); err != nil {
		return rc.errs, err
	}

	validation.SortValidationErrors(rc.errs)

	return rc.errs, nil
}

// validationOptions returns the options used to validate objects of this document outside of Validate.
func (a *AsyncAPI) validationOptions() []validation.Option {
	v, err := asyncapiversion.Parse(a.AsyncAPI)
	if err != nil {
		return nil
	}
	return []validation.Option{jsonschema.WithSchemaFormat(asyncapiversion.DefaultSchemaFormat(a.AsyncAPI)), withDocumentVersion(v)}
}

// location identifies the document references are resolved against.
type location struct {
	// uri is the absolute location of the document, empty for the root document when its location isn't known.
	uri string
	// doc is the *AsyncAPI root document or the *yaml.Node of another document.
	doc any
}

// walkTarget is an object graph to resolve: the root document, or an object loaded from another document.
type walkTarget struct {
	items iter.Seq[WalkItem]
	base  location
	// effective is set for effective views produced by applying traits, which are resolved but not re-checked.
	effective bool
}

type resolveCall struct {
	doc            *AsyncAPI
	opts           ResolveAllOptions
	root           location
	state          *resolutionState
	cache          resolveCache
	validationOpts []validation.Option
	targets        []walkTarget
	queued         map[any]bool
	errs           []error
}

// run resolves the root document and everything loaded from it. Reference wrappers are resolved first so that JSON
// pointers can navigate through them when Reference Objects, tags, external documentation and schemas are resolved
// next. Traits and cross-object checks need all of those, so they run last, and may add effective views to resolve.
func (rc *resolveCall) run(ctx context.Context) {
	rc.targets = append(rc.targets, walkTarget{items: Walk(ctx, rc.doc), base: rc.root})

	wrappers, others, checked := 0, 0, 0
	for checked < len(rc.targets) || others < len(rc.targets) {
		for others < len(rc.targets) {
			for wrappers < len(rc.targets) {
				rc.resolveWrappers(ctx, rc.targets[wrappers])
				wrappers++
			}
			rc.resolveReferences(ctx, rc.targets[others])
			others++
		}
		for checked < len(rc.targets) {
			if !rc.targets[checked].effective {
				rc.applyTraitsAndCheck(ctx, rc.targets[checked])
			}
			checked++
		}
		if ctx.Err() != nil {
			return
		}
	}
}

// quietEffective drops diagnostics recorded while resolving an effective view, as the same references are resolved
// and reported where they were authored.
func (rc *resolveCall) quietEffective(t walkTarget) func() {
	n := len(rc.errs)
	return func() {
		if t.effective {
			rc.errs = rc.errs[:n]
		}
	}
}

func (rc *resolveCall) resolveWrappers(ctx context.Context, t walkTarget) {
	defer rc.quietEffective(t)()

	for item := range t.items {
		_ = item.Match(Matcher{
			ReferencedServer: func(r *ReferencedServer) error {
				resolveReference(ctx, rc, r, "server", t.base)
				return nil
			},
			ReferencedServerVariable: func(r *ReferencedServerVariable) error {
				resolveReference(ctx, rc, r, "server variable", t.base)
				return nil
			},
			ReferencedChannel: func(r *ReferencedChannel) error {
				resolveReference(ctx, rc, r, "channel", t.base)
				return nil
			},
			ReferencedParameter: func(r *ReferencedParameter) error {
				resolveReference(ctx, rc, r, "parameter", t.base)
				return nil
			},
			ReferencedOperation: func(r *ReferencedOperation) error {
				resolveReference(ctx, rc, r, "operation", t.base)
				return nil
			},
			ReferencedOperationTrait: func(r *ReferencedOperationTrait) error {
				resolveReference(ctx, rc, r, "operation trait", t.base)
				return nil
			},
			ReferencedOperationReply: func(r *ReferencedOperationReply) error {
				resolveReference(ctx, rc, r, "operation reply", t.base)
				return nil
			},
			ReferencedOperationReplyAddress: func(r *ReferencedOperationReplyAddress) error {
				resolveReference(ctx, rc, r, "operation reply address", t.base)
				return nil
			},
			ReferencedMessage: func(r *ReferencedMessage) error {
				resolveReference(ctx, rc, r, "message", t.base)
				return nil
			},
			ReferencedMessageTrait: func(r *ReferencedMessageTrait) error {
				resolveReference(ctx, rc, r, "message trait", t.base)
				return nil
			},
			ReferencedCorrelationID: func(r *ReferencedCorrelationID) error {
				resolveReference(ctx, rc, r, "correlation ID", t.base)
				return nil
			},
			ReferencedSecurityScheme: func(r *ReferencedSecurityScheme) error {
				resolveReference(ctx, rc, r, "security scheme", t.base)
				return nil
			},
			ReferencedBindings: func(r *ReferencedBindings) error {
				resolveReference(ctx, rc, r, "bindings", t.base)
				return nil
			},
		})
	}
}

func (rc *resolveCall) resolveReferences(ctx context.Context, t walkTarget) {
	defer rc.quietEffective(t)()

	for item := range t.items {
		_ = item.Match(Matcher{
			Channel: func(c *Channel) error {
				for _, server := range c.Servers {
					resolveReferenceObject[Server, *Server, *core.Server](ctx, rc, server, "server", t.base)
				}
				return nil
			},
			Operation: func(o *Operation) error {
				resolveReferenceObject[Channel, *Channel, *core.Channel](ctx, rc, &o.Channel, "channel", t.base)
				for _, message := range o.Messages {
					resolveReferenceObject[Message, *Message, *core.Message](ctx, rc, message, "message", t.base)
				}
				return nil
			},
			OperationReply: func(r *OperationReply) error {
				resolveReferenceObject[Channel, *Channel, *core.Channel](ctx, rc, r.Channel, "channel", t.base)
				for _, message := range r.Messages {
					resolveReferenceObject[Message, *Message, *core.Message](ctx, rc, message, "message", t.base)
				}
				return nil
			},
			Tag: func(tag *Tag) error {
				rc.resolveTag(ctx, tag, t.base)
				return nil
			},
			ExternalDocumentation: func(docs *ExternalDocumentation) error {
				rc.resolveExternalDocs(ctx, docs, t.base)
				return nil
			},
			SchemaUnion: func(schema *jsonschema.SchemaUnion) error {
				rc.resolveSchema(ctx, schema, t.base)
				return nil
			},
		})
	}
}

func (rc *resolveCall) applyTraitsAndCheck(ctx context.Context, t walkTarget) {
	for item := range t.items {
		pointer := locationParts(item.Location)
		_ = item.Match(Matcher{
			Operation: func(o *Operation) error {
				rc.applyOperationTraits(ctx, o, t.base)
				rc.checkOperation(o, pointer, t.base)
				return nil
			},
			OperationReply: func(r *OperationReply) error {
				rc.checkOperationReply(r, pointer, t.base)
				return nil
			},
			Message: func(m *Message) error {
				rc.applyMessageTraits(ctx, m, t.base)
				return nil
			},
			Channel: func(c *Channel) error {
				rc.checkChannel(c, pointer, t.base)
				return nil
			},
		})
	}
}

// track records how to clear state set on a model, so the next call starts from a clean slate.
func (rc *resolveCall) track(reset func()) {
	rc.state.resets = append(rc.state.resets, reset)
}

func (rc *resolveCall) isRoot(base location) bool {
	return base.doc == rc.root.doc
}

// loaded validates and queues an object loaded from another document so its own references are resolved against
// that document. Objects in the root document are already covered by Validate and the root walk.
func (rc *resolveCall) loaded(ctx context.Context, obj any, base location) {
	if rc.isRoot(base) || rc.queued[obj] {
		return
	}
	rc.queued[obj] = true

	if v, ok := obj.(interface {
		Validate(ctx context.Context, opts ...validation.Option) []error
	}); ok {
		rc.addErrors(v.Validate(ctx, rc.validationOpts...), base)
	}

	rc.targets = append(rc.targets, walkTarget{items: walkObject(ctx, rc.doc, obj), base: base})
}

// addErrors records diagnostics, attributing those found in other documents to their document.
func (rc *resolveCall) addErrors(errs []error, base location) {
	for _, err := range errs {
		var vErr *validation.Error
		if !rc.isRoot(base) && errors.As(err, &vErr) && vErr.DocumentLocation == "" {
			vErr.DocumentLocation = base.uri
		}
		rc.errs = append(rc.errs, err)
	}
}

// referenceError records a failure to resolve a reference at the referring `$ref`.
func (rc *resolveCall) referenceError(err error, base location, c validation.CoreModeler, node validation.ValueNodeGetter) {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return
	}
	rc.addErrors([]error{validation.NewValueError(validation.SeverityError, referenceErrorRule(err), err, c, node)}, base)
}

func referenceErrorRule(err error) string {
	var circular *circularReferenceError
	if errors.As(err, &circular) {
		return validation.RuleValidationCircularReference
	}
	return validation.RuleValidationInvalidReference
}

type circularReferenceError struct {
	chain []string
}

func (e *circularReferenceError) Error() string {
	return "circular reference: " + strings.Join(e.chain, " -> ")
}

// resolveCache holds what one call to ResolveAllReferences has loaded. Fetched documents are shared, while resolved
// objects are namespaced by the kind of object they were interpreted as, so a location is never reused as a
// different type or schema format.
type resolveCache struct {
	objects  map[string]any
	docs     map[string][]byte
	external map[string]any
}

type kindTarget struct {
	cache *resolveCache
	kind  string
}

var _ references.ResolutionTarget = (*kindTarget)(nil)

func (t *kindTarget) InitCache() {}

func (t *kindTarget) GetCachedReferencedObject(key string) (any, bool) {
	obj, ok := t.cache.objects[t.kind+" "+key]
	return obj, ok
}

func (t *kindTarget) StoreReferencedObjectInCache(key string, obj any) {
	t.cache.objects[t.kind+" "+key] = obj
}

func (t *kindTarget) GetCachedReferenceDocument(key string) ([]byte, bool) {
	doc, ok := t.cache.docs[key]
	return doc, ok
}

func (t *kindTarget) StoreReferenceDocumentInCache(key string, doc []byte) {
	t.cache.docs[key] = doc
}

func (t *kindTarget) GetCachedExternalDocument(key string) (any, bool) {
	doc, ok := t.cache.external[key]
	return doc, ok
}

func (t *kindTarget) StoreExternalDocumentInCache(key string, doc any) {
	t.cache.external[key] = doc
}

type chainResult[T any] struct {
	obj  *T
	ref  references.Reference
	base location
}

// resolveChain resolves a reference, following further references until it reaches an object. Each reference is
// resolved against the document containing it, and a reference seen twice in the same chain is reported as a cycle.
func resolveChain[T any](ctx context.Context, rc *resolveCall, kind string, ref references.Reference, base location, unmarshal references.Unmarshal[T], next func(*T) references.Reference) (chainResult[T], error) {
	var chain []string
	for {
		if err := ctx.Err(); err != nil {
			return chainResult[T]{}, err
		}
		if err := rc.checkAccess(ref, base); err != nil {
			return chainResult[T]{}, err
		}

		abs, err := absoluteReference(ref, base.uri)
		if err != nil {
			return chainResult[T]{}, fmt.Errorf("%s reference `%s` is invalid: %w", kind, ref, err)
		}
		if slices.Contains(chain, abs) {
			return chainResult[T]{}, &circularReferenceError{chain: append(chain, abs)}
		}
		chain = append(chain, abs)

		if root, ok := base.doc.(*AsyncAPI); ok && ref.GetURI() == "" {
			if err := checkLocalTarget[T](root, ref, kind); err != nil {
				return chainResult[T]{}, err
			}
		}

		res, vErrs, err := references.Resolve(ctx, ref, unmarshal, references.ResolveOptions{
			RootDocument:        &kindTarget{cache: &rc.cache, kind: kind},
			TargetLocation:      base.uri,
			TargetDocument:      base.doc,
			DisableExternalRefs: rc.opts.DisableExternalRefs,
			VirtualFS:           rc.opts.VirtualFS,
			HTTPClient:          rc.opts.HTTPClient,
		})
		if err != nil {
			return chainResult[T]{}, fmt.Errorf("failed to resolve %s reference `%s`: %w", kind, ref, err)
		}
		if res == nil || res.Object == nil {
			return chainResult[T]{}, fmt.Errorf("%s reference `%s` resolved to nothing", kind, ref)
		}

		nextBase := base
		if ref.GetURI() != "" {
			nextBase = location{uri: res.AbsoluteDocumentPath, doc: res.ResolvedDocument}
			// The shared resolver can report a cached document as its raw bytes, so use the parsed document instead.
			if doc, ok := rc.cache.external[res.AbsoluteDocumentPath]; ok {
				nextBase.doc = doc
			}
		}
		rc.addErrors(vErrs, nextBase)

		if n := next(res.Object); n != "" {
			ref = n
			base = nextBase
			continue
		}

		return chainResult[T]{obj: res.Object, ref: references.Reference(abs), base: nextBase}, nil
	}
}

// checkAccess refuses references the options don't allow, before anything is loaded. A relative reference to another
// document needs the containing document's location; it is never resolved against the working directory.
func (rc *resolveCall) checkAccess(ref references.Reference, base location) error {
	uri := ref.GetURI()
	if uri == "" {
		return nil
	}
	if rc.opts.DisableExternalRefs {
		return fmt.Errorf("reference `%s` is to another document and %w", ref, ErrExternalReferencesDisabled)
	}
	if base.uri == "" && !isAbsoluteLocation(uri) {
		return fmt.Errorf("relative reference `%s` can't be resolved because the document's location is unknown, set ResolveAllOptions.AsyncAPILocation", ref)
	}
	return nil
}

func isAbsoluteLocation(uri string) bool {
	if filepath.IsAbs(uri) {
		return true
	}
	u, err := url.Parse(uri)
	return err == nil && u != nil && (u.Scheme == "file" || (u.Scheme != "" && u.Host != ""))
}

// checkLocalTarget checks a reference within the root document points to the expected kind of object, so wrong-kind
// targets are reported clearly rather than as a type conversion failure.
func checkLocalTarget[T any](doc *AsyncAPI, ref references.Reference, kind string) error {
	target, err := jsonpointer.GetTarget(doc, ref.GetJSONPointer(), jsonpointer.WithStructTags("key"))
	if err != nil {
		return fmt.Errorf("%s reference `%s` target not found: %w", kind, ref, err)
	}
	switch target.(type) {
	case *T, *yaml.Node:
		return nil
	}
	return fmt.Errorf("%s reference `%s` points to %s, not a %s", kind, ref, describeObject(target), kind)
}

func describeObject(obj any) string {
	switch obj.(type) {
	case *AsyncAPI:
		return "the whole document"
	case *ReferencedServer, *Server:
		return "a server"
	case *ReferencedServerVariable, *ServerVariable:
		return "a server variable"
	case *ReferencedChannel, *Channel:
		return "a channel"
	case *ReferencedParameter, *Parameter:
		return "a parameter"
	case *ReferencedOperation, *Operation:
		return "an operation"
	case *ReferencedOperationTrait, *OperationTrait:
		return "an operation trait"
	case *ReferencedOperationReply, *OperationReply:
		return "an operation reply"
	case *ReferencedOperationReplyAddress, *OperationReplyAddress:
		return "an operation reply address"
	case *ReferencedMessage, *Message:
		return "a message"
	case *ReferencedMessageTrait, *MessageTrait:
		return "a message trait"
	case *ReferencedCorrelationID, *CorrelationID:
		return "a correlation ID"
	case *ReferencedSecurityScheme, *SecurityScheme:
		return "a security scheme"
	case *ReferencedBindings, *Bindings:
		return "bindings"
	case *Tag:
		return "a tag"
	case *ExternalDocumentation:
		return "external documentation"
	case *jsonschema.SchemaUnion:
		return "a schema"
	case *Components:
		return "the components object"
	default:
		return "an unsupported location"
	}
}

// absoluteReference returns the absolute form of a reference, used to identify it within a call.
func absoluteReference(ref references.Reference, base string) (string, error) {
	jp := string(ref.GetJSONPointer())
	if ref.GetURI() == "" {
		return utils.BuildAbsoluteReference(base, jp), nil
	}
	res, err := references.ResolveAbsoluteReference(ref, base)
	if err != nil {
		return "", err
	}
	return utils.BuildAbsoluteReference(res.AbsoluteReference, jp), nil
}

func unmarshalReference[T any, V interfaces.Validator[T], C marshaller.CoreModeler](ctx context.Context, node *yaml.Node, skipValidation bool) (*Reference[T, V, C], []error, error) {
	return unmarshalModel[Reference[T, V, C], core.Reference[C]]("reference")(ctx, node, skipValidation)
}

// nextReference continues a chain through a Reference Object, unless it was already resolved. A wrapper from another
// document is resolved against that document when it is loaded, so its result is reused rather than resolving its
// relative reference again against the wrong document.
func nextReference[T any, V interfaces.Validator[T], C marshaller.CoreModeler](r *Reference[T, V, C]) references.Reference {
	if !r.IsReference() || r.IsResolved() {
		return ""
	}
	return r.GetReference()
}

// resolveWrapperChain resolves a reference to an object of type T, which may be reached through other references.
func resolveWrapperChain[T any, V interfaces.Validator[T], C marshaller.CoreModeler](ctx context.Context, rc *resolveCall, kind string, ref references.Reference, base location) (chainResult[T], error) {
	res, err := resolveChain(ctx, rc, kind, ref, base, unmarshalReference[T, V, C], nextReference[T, V, C])
	if err != nil {
		return chainResult[T]{}, err
	}
	obj := res.obj.GetObject()
	if obj == nil {
		return chainResult[T]{}, fmt.Errorf("%s reference `%s` doesn't point to a %s", kind, ref, kind)
	}
	resolvedRef := res.ref
	if res.obj.IsReference() {
		resolvedRef = res.obj.GetResolvedReference()
	}
	return chainResult[T]{obj: obj, ref: resolvedRef, base: res.base}, nil
}

func resolveReference[T any, V interfaces.Validator[T], C marshaller.CoreModeler](ctx context.Context, rc *resolveCall, r *Reference[T, V, C], kind string, base location) {
	if r == nil || !r.IsReference() {
		return
	}

	c := r.GetCore()
	res, err := resolveWrapperChain[T, V, C](ctx, rc, kind, r.GetReference(), base)
	if err != nil {
		rc.referenceError(err, base, c, c.Reference)
		return
	}

	r.setResolved(res.obj, res.ref)
	rc.track(func() { r.setResolved(nil, "") })
	rc.loaded(ctx, res.obj, res.base)
}

func resolveReferenceObject[T any, V interfaces.Validator[T], C marshaller.CoreModeler](ctx context.Context, rc *resolveCall, r *ReferenceObject, kind string, base location) {
	if r == nil || r.Reference == "" {
		return
	}

	c := r.GetCore()
	res, err := resolveWrapperChain[T, V, C](ctx, rc, kind, r.Reference, base)
	if err != nil {
		rc.referenceError(err, base, c, c.Reference)
		return
	}

	r.setResolved(res.obj, res.ref)
	rc.track(func() { r.setResolved(nil, "") })
	rc.loaded(ctx, res.obj, res.base)
}

// unmarshalModel returns an unmarshaller for a model that is its own Reference Object, such as a tag.
func unmarshalModel[T any, C any, PT interface {
	*T
	marshaller.CoreAccessor[C]
}](parentName string) references.Unmarshal[T] {
	return func(ctx context.Context, node *yaml.Node, skipValidation bool) (*T, []error, error) {
		var model T
		validationErrs, err := marshaller.UnmarshalNode[C](ctx, parentName, node, PT(&model))
		if skipValidation {
			validationErrs = nil
		}
		return &model, validationErrs, err
	}
}

func (rc *resolveCall) resolveTag(ctx context.Context, tag *Tag, base location) {
	if !tag.IsReference() {
		return
	}

	next := func(t *Tag) references.Reference {
		if t.IsReference() {
			return *t.Ref
		}
		return ""
	}

	c := tag.GetCore()
	res, err := resolveChain(ctx, rc, "tag", *tag.Ref, base, unmarshalModel[Tag, core.Tag]("tag"), next)
	if err != nil {
		rc.referenceError(err, base, c, c.Ref)
		return
	}

	tag.setResolved(res.obj)
	rc.track(func() { tag.setResolved(nil) })
	rc.loaded(ctx, res.obj, res.base)
}

func (rc *resolveCall) resolveExternalDocs(ctx context.Context, docs *ExternalDocumentation, base location) {
	if !docs.IsReference() {
		return
	}

	next := func(d *ExternalDocumentation) references.Reference {
		if d.IsReference() {
			return *d.Ref
		}
		return ""
	}

	c := docs.GetCore()
	res, err := resolveChain(ctx, rc, "external documentation", *docs.Ref, base, unmarshalModel[ExternalDocumentation, schemacore.ExternalDocumentation]("externalDocs"), next)
	if err != nil {
		rc.referenceError(err, base, c, c.Ref)
		return
	}

	resolutionMu.Lock()
	rc.state.externalDocs[docs] = res.obj
	resolutionMu.Unlock()
	rc.loaded(ctx, res.obj, res.base)
}

func unmarshalSchemaUnion(ctx context.Context, node *yaml.Node, skipValidation bool) (*jsonschema.SchemaUnion, []error, error) {
	var c schemacore.SchemaUnion
	validationErrs, err := marshaller.UnmarshalCore(ctx, "schema", node, &c)
	if err != nil {
		return nil, validationErrs, err
	}

	if skipValidation {
		validationErrs = nil
	}
	var schema jsonschema.SchemaUnion
	return &schema, validationErrs, schema.Populate(c)
}

func nextSchemaReference(schema *jsonschema.SchemaUnion) references.Reference {
	return schema.GetRef()
}

// unmarshalRaw keeps a resource in a format this package doesn't interpret unchanged.
func unmarshalRaw(_ context.Context, node *yaml.Node, _ bool) (*yaml.Node, []error, error) {
	return node, nil, nil
}

// resolveSchema resolves a schema Reference Object according to its schema format. Native and JSON Schema based
// formats resolve to typed schemas. A Reference Object that is a foreign format's whole `schema` is an outer resource
// reference: its target is loaded and kept unchanged, without interpreting any references inside it. References
// nested within schemas are not resolved.
func (rc *resolveCall) resolveSchema(ctx context.Context, schema *jsonschema.SchemaUnion, base location) {
	switch {
	case schema.IsReference():
		res, err := resolveChain(ctx, rc, "schema", schema.GetRef(), base, unmarshalSchemaUnion, nextSchemaReference)
		if err != nil {
			rc.schemaError(err, base, schema.GetRootNode())
			return
		}
		rc.setSchema(schema, &ResolvedSchema{Schema: res.obj, SchemaFormat: res.obj.GetSchemaFormat(rc.doc.AsyncAPI), Reference: res.ref})

	case schema.IsMultiFormat() && schema.GetMultiFormat().GetSchema().IsReference():
		format := schema.GetMultiFormat().GetSchemaFormat()
		definition := schema.GetMultiFormat().GetSchema()
		ref := definition.GetReference()

		if definition.GetSchema() != nil || (ref.GetURI() == "" && rc.isRoot(base)) {
			res, err := resolveChain(ctx, rc, "schema "+format, ref, base, unmarshalSchemaUnion, nextSchemaReference)
			if err != nil {
				rc.schemaError(err, base, definition.GetRootNode())
				return
			}
			if declared := res.obj.GetMultiFormat().GetSchemaFormat(); declared != "" && !strings.EqualFold(declared, format) {
				rc.schemaError(fmt.Errorf("schema reference `%s` is declared as `%s` but its target is `%s`", ref, format, declared), base, definition.GetRootNode())
				return
			}
			rc.setSchema(schema, &ResolvedSchema{Schema: res.obj, SchemaFormat: format, Reference: res.ref})
			return
		}

		if isTextualSchemaFormat(format) {
			rc.addErrors([]error{validation.NewValidationError(validation.SeverityWarning, validation.RuleValidationUnsupportedFormat, fmt.Errorf("schema reference `%s` to a `%s` resource is not resolved, its content is not checked", ref, format), definition.GetRootNode())}, base)
			return
		}

		res, err := resolveChain(ctx, rc, "raw "+format, ref, base, unmarshalRaw, func(*yaml.Node) references.Reference { return "" })
		if err != nil {
			rc.schemaError(err, base, definition.GetRootNode())
			return
		}
		if declared := mappingScalar(res.obj, "schemaFormat"); declared != "" && !strings.EqualFold(declared, format) {
			rc.schemaError(fmt.Errorf("schema reference `%s` is declared as `%s` but its target is `%s`", ref, format, declared), base, definition.GetRootNode())
			return
		}
		rc.setSchema(schema, &ResolvedSchema{Raw: res.obj, SchemaFormat: format, Reference: res.ref})
	}
}

func (rc *resolveCall) setSchema(schema *jsonschema.SchemaUnion, resolved *ResolvedSchema) {
	resolutionMu.Lock()
	defer resolutionMu.Unlock()
	rc.state.schemas[schema] = resolved
}

func (rc *resolveCall) schemaError(err error, base location, node *yaml.Node) {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return
	}
	rc.addErrors([]error{validation.NewValidationError(validation.SeverityError, referenceErrorRule(err), err, node)}, base)
}

// isTextualSchemaFormat reports whether resources in the format are plain text rather than JSON or YAML, and so can't
// be loaded by the YAML based resolver.
func isTextualSchemaFormat(format string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(format)), "application/vnd.google.protobuf")
}

func mappingScalar(node *yaml.Node, key string) string {
	node = resolveNode(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return ""
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			if value := resolveNode(node.Content[i+1]); value != nil && value.Kind == yaml.ScalarNode {
				return value.Value
			}
		}
	}
	return ""
}

func resolveNode(node *yaml.Node) *yaml.Node {
	for node != nil && (node.Kind == yaml.DocumentNode || node.Kind == yaml.AliasNode) {
		if node.Kind == yaml.AliasNode {
			node = node.Alias
			continue
		}
		if len(node.Content) == 0 {
			return nil
		}
		node = node.Content[0]
	}
	return node
}

// walkObject walks an object loaded from another document, as Walk does for the root document.
func walkObject(ctx context.Context, doc *AsyncAPI, obj any) iter.Seq[WalkItem] {
	return func(yield func(WalkItem) bool) {
		w := walker{doc: doc, yield: yield, cancelled: func() bool { return ctx.Err() != nil }}
		switch o := obj.(type) {
		case *Server:
			w.walkServer(o, nil)
		case *ServerVariable:
			w.walkReferencedServerVariable(&ReferencedServerVariable{Object: o}, nil)
		case *Channel:
			w.walkChannel(o, nil)
		case *Parameter:
			w.walkReferencedParameter(&ReferencedParameter{Object: o}, nil)
		case *Operation:
			w.walkOperation(o, nil)
		case *OperationTrait:
			w.walkReferencedOperationTrait(&ReferencedOperationTrait{Object: o}, nil)
		case *OperationReply:
			w.walkReferencedOperationReply(&ReferencedOperationReply{Object: o}, nil)
		case *OperationReplyAddress:
			w.walkReferencedOperationReplyAddress(&ReferencedOperationReplyAddress{Object: o}, nil)
		case *Message:
			w.walkMessage(o, nil)
		case *MessageTrait:
			w.walkReferencedMessageTrait(&ReferencedMessageTrait{Object: o}, nil)
		case *CorrelationID:
			w.walkReferencedCorrelationID(&ReferencedCorrelationID{Object: o}, nil)
		case *SecurityScheme:
			w.walkReferencedSecurityScheme(&ReferencedSecurityScheme{Object: o}, nil)
		case *Bindings:
			w.walkReferencedBindings(&ReferencedBindings{Object: o}, nil)
		case *Tag:
			w.walkTag(o, nil)
		case *ExternalDocumentation:
			w.walkExternalDocs(o, nil)
		}
	}
}

// locationParts returns the JSON pointer segments of a walk location, unescaped.
func locationParts(loc Locations) []string {
	pointer := strings.TrimPrefix(string(loc.ToJSONPointer()), "/")
	if pointer == "" {
		return nil
	}
	parts := strings.Split(pointer, "/")
	for i, part := range parts {
		parts[i] = unescapePointer(part)
	}
	return parts
}

func unescapePointer(part string) string {
	return strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
}
