// Package openapi generates Mailpit's OpenAPI specification from the API route table (apiv1.Routes) and the Go
// types and doc comments which it refers to.
//
// It is a development tool only: nothing in the Mailpit binary imports it. Run `go run ./internal/openapi/gen` to update the
// committed specification (server/ui/api/v1/openapi.json), a test fails if it is out of date.
package openapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"reflect"
	"strconv"
	"strings"

	"github.com/axllent/mailpit/server/apiv1"
	"github.com/swaggest/jsonschema-go"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi31"
)

// Generate returns the OpenAPI specification (JSON) of the API.
func Generate() ([]byte, error) {
	return generate(apiv1.Routes)
}

// The descriptions of the error responses, which are the same for every route.
const (
	errorDescription     = "Server error will return with a 400 status code\nwith the error message in the body"
	jsonErrorDescription = "JSON error response"
	notFoundDescription  = "Not found error will return a 404 status code"
)

// jsonError is the body of JSON error responses.
//
// openapi:model JSONError
type jsonError struct {
	// Error message
	Error string `json:"Error" example:"invalid format"`
}

// binary is the schema of a binary response body.
type binary string

// JSONSchema implements jsonschema.Exposer.
func (binary) JSONSchema() (jsonschema.Schema, error) {
	s := jsonschema.Schema{}
	s.WithType(jsonschema.String.Type())
	s.WithFormat("binary")

	return s, nil
}

// integerFormats are the formats of the Go integer types.
var integerFormats = map[reflect.Kind]string{
	reflect.Int: "int64", reflect.Int8: "int8", reflect.Int16: "int16", reflect.Int32: "int32", reflect.Int64: "int64",
	reflect.Uint: "uint64", reflect.Uint8: "uint8", reflect.Uint16: "uint16", reflect.Uint32: "uint32", reflect.Uint64: "uint64",
}

type generator struct {
	docs  *docs
	names map[string]reflect.Type // component names in use
	errs  []error
}

func (g *generator) fail(format string, a ...any) {
	g.errs = append(g.errs, fmt.Errorf(format, a...))
}

// defName names a component schema: the `openapi:model` name of the type, else its default name.
// Two different types may never share a name.
func (g *generator) defName(t reflect.Type, def string) string {
	name := def

	if t == reflect.TypeOf(mail.Address{}) {
		name = "Address" // the one non-module type which is a component
	} else if t == reflect.TypeOf(binary("")) {
		name = "BinaryData"
	} else if sd, err := g.docs.structFor(t); err != nil {
		g.errs = append(g.errs, err)
	} else if sd != nil && sd.model != "" {
		name = sd.model
	}

	if prev, ok := g.names[name]; ok && prev != t {
		g.fail("schema name %q is used by both %s and %s", name, prev, t)
	}

	g.names[name] = t

	return name
}

// intercept applies the doc comments to the schema of every struct once it has been reflected.
func (g *generator) intercept(p jsonschema.InterceptSchemaParams) (bool, error) {
	if !p.Processed || !p.Value.IsValid() {
		return false, nil
	}

	// the format of an integer is its Go type (as go-swagger did), `int` being int64
	if f, ok := integerFormats[p.Value.Kind()]; ok && p.Schema.Format == nil {
		p.Schema.WithFormat(f)
	}

	t := p.Value.Type()
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	sd, err := g.docs.structFor(t)
	if err != nil {
		return false, err
	}

	if sd == nil {
		return false, nil
	}

	if p.Schema.Description == nil && sd.description != "" {
		p.Schema.WithDescription(sd.description)
	}

	return false, apply(p.Schema, t, sd, g.docs)
}

func generate(routes []apiv1.Route) ([]byte, error) {
	d, err := newDocs()
	if err != nil {
		return nil, err
	}

	g := &generator{docs: d, names: map[string]reflect.Type{}}

	r := openapi31.NewReflector()
	// Mailpit's JSON uses the Go field names (no json tags), so untagged fields are included
	r.DefaultOptions = append(r.DefaultOptions,
		jsonschema.ProcessWithoutTags,
		jsonschema.InterceptDefName(g.defName),
		jsonschema.InterceptSchema(g.intercept),
		// a nil slice or map is encoded as null by Go, but it is not documented as a type of its own, as before
		jsonschema.InterceptNullability(func(p jsonschema.InterceptNullabilityParams) {
			if p.NullAdded {
				p.Schema.RemoveType(jsonschema.Null)
			}
		}),
	)
	r.SpecEns().WithInfo(openapi31.Info{
		Title:       "Mailpit API",
		Description: ptr("OpenAPI 3.1 documentation for [Mailpit](https://github.com/axllent/mailpit)."),
		Version:     "v1",
		Contact:     &openapi31.Contact{Name: ptr("GitHub"), URL: ptr("https://github.com/axllent/mailpit")},
		License: &openapi31.License{
			Name: "MIT license",
			URL:  ptr("https://github.com/axllent/mailpit/blob/develop/LICENSE"),
		},
	})

	for _, rt := range routes {
		if err := g.addRoute(r, rt); err != nil {
			g.errs = append(g.errs, fmt.Errorf("%s %s: %w", rt.Method, rt.Path, err))
		}
	}

	if len(g.errs) > 0 {
		return nil, errors.Join(g.errs...)
	}

	pruneUnused(r.Spec)

	raw, err := r.Spec.MarshalJSON()
	if err != nil {
		return nil, err
	}

	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}

	if err := validate(doc, len(routes)); err != nil {
		return nil, err
	}

	var out bytes.Buffer
	if err := json.Indent(&out, raw, "", "  "); err != nil {
		return nil, err
	}

	out.WriteByte('\n')

	return out.Bytes(), nil
}

func (g *generator) addRoute(r *openapi31.Reflector, rt apiv1.Route) error {
	if rt.ID == "" || rt.Tag == "" || rt.Summary == "" || rt.Description == "" {
		return errors.New("ID, Tag, Summary and Description are required")
	}

	if rt.ResponseExample != nil && reflect.TypeOf(rt.ResponseExample) != reflect.TypeOf(rt.Response) {
		return fmt.Errorf("ResponseExample is a %T, it must be a value of the Response type (%T)", rt.ResponseExample, rt.Response)
	}

	if rt.Response == nil || len(rt.ContentTypes) == 0 || rt.ResponseDescription == "" {
		return errors.New("Response, ResponseDescription and ContentTypes are required")
	}

	oc, err := r.NewOperationContext(rt.Method, rt.Path)
	if err != nil {
		return err
	}

	oc.SetID(rt.ID)
	oc.SetTags(rt.Tag)
	oc.SetSummary(rt.Summary)
	oc.SetDescription(rt.Description)

	params := map[string]string{}

	if rt.Params != nil {
		if err := paramDescriptions(reflect.TypeOf(rt.Params), g.docs, params); err != nil {
			return err
		}

		oc.AddReqStructure(rt.Params)
	}

	if rt.Body != nil {
		oc.AddReqStructure(rt.Body, openapi.WithContentType("application/json"))
	}

	resp := rt.Response
	if _, ok := resp.(apiv1.BinaryBody); ok {
		resp = binary("")
	}

	for _, ct := range rt.ContentTypes {
		oc.AddRespStructure(resp, openapi.WithHTTPStatus(http.StatusOK), openapi.WithContentType(ct))
	}

	var errBody any = apiv1.TextBody("")

	errType := "text/plain"
	if rt.JSONErrors {
		errBody, errType = jsonError{}, "application/json"
	}

	oc.AddRespStructure(errBody, openapi.WithHTTPStatus(http.StatusBadRequest), openapi.WithContentType(errType))

	for _, code := range rt.ExtraErrors {
		oc.AddRespStructure(errBody, openapi.WithHTTPStatus(code), openapi.WithContentType(errType))
	}

	if rt.NotFound {
		oc.AddRespStructure(apiv1.TextBody(""), openapi.WithHTTPStatus(http.StatusNotFound), openapi.WithContentType("text/plain"))
	}

	if err := r.AddOperation(oc); err != nil {
		return err
	}

	// parameter descriptions come from the doc comments
	item := r.Spec.Paths.MapOfPathItemValues[rt.Path]

	op := operationOf(&item, rt.Method)
	if op == nil {
		return fmt.Errorf("operation not found in the specification")
	}

	for _, p := range op.Parameters {
		if p.Parameter == nil {
			continue
		}

		d, ok := params[p.Parameter.Name]
		if !ok {
			return fmt.Errorf("parameter %q is not described by Params", p.Parameter.Name)
		}

		p.Parameter.Description = &d
	}

	// response descriptions
	setResponse := func(code int, description string) error {
		if op.Responses == nil {
			return errors.New("no responses in the specification")
		}

		r, ok := op.Responses.MapOfResponseOrReferenceValues[strconv.Itoa(code)]
		if !ok || r.Response == nil {
			return fmt.Errorf("response %d not found in the specification", code)
		}

		r.Response.Description = description

		return nil
	}

	errDescription := errorDescription
	if rt.JSONErrors {
		errDescription = jsonErrorDescription
	}

	if err := setResponse(http.StatusOK, rt.ResponseDescription); err != nil {
		return err
	}

	if rt.ResponseExample != nil {
		ok := op.Responses.MapOfResponseOrReferenceValues[strconv.Itoa(http.StatusOK)].Response

		for _, ct := range rt.ContentTypes {
			media, found := ok.Content[ct]
			if !found {
				return fmt.Errorf("response content type %s not found in the specification", ct)
			}

			// a named example, as RapiDoc shows a (single) media type `example` of a text response wrapped in an object
			value := rt.ResponseExample

			media.Examples = map[string]openapi31.ExampleOrReference{"Example": {Example: &openapi31.Example{Value: &value}}}
			ok.Content[ct] = media
		}
	}

	if err := setResponse(http.StatusBadRequest, errDescription); err != nil {
		return err
	}

	for _, code := range rt.ExtraErrors {
		if err := setResponse(code, errDescription); err != nil {
			return err
		}
	}

	if rt.NotFound {
		if err := setResponse(http.StatusNotFound, notFoundDescription); err != nil {
			return err
		}
	}

	return nil
}

// pruneUnused removes component schemas which nothing refers to (the reflection of a parameters struct
// leaves an empty one behind).
func pruneUnused(spec *openapi31.Spec) {
	if spec.Components == nil {
		return
	}

	schemas := spec.Components.Schemas

	// the references within the schemas themselves are followed from the paths
	used := map[string]bool{}

	var mark func(v any)

	mark = func(v any) {
		b, _ := json.Marshal(v)

		var doc any
		_ = json.Unmarshal(b, &doc)

		var walk func(any)

		walk = func(x any) {
			switch x := x.(type) {
			case map[string]any:
				if ref, ok := x["$ref"].(string); ok {
					name := strings.TrimPrefix(ref, "#/components/schemas/")
					if !used[name] {
						used[name] = true

						if s, ok := schemas[name]; ok {
							mark(s)
						}
					}
				}

				for _, c := range x {
					walk(c)
				}
			case []any:
				for _, c := range x {
					walk(c)
				}
			}
		}

		walk(doc)
	}

	mark(spec.Paths)

	for name := range schemas {
		if !used[name] {
			delete(schemas, name)
		}
	}
}

// operationOf returns the operation of a path item for the HTTP method.
func operationOf(item *openapi31.PathItem, method string) *openapi31.Operation {
	switch method {
	case http.MethodGet:
		return item.Get
	case http.MethodPut:
		return item.Put
	case http.MethodPost:
		return item.Post
	case http.MethodDelete:
		return item.Delete
	case http.MethodPatch:
		return item.Patch
	case http.MethodHead:
		return item.Head
	case http.MethodOptions:
		return item.Options
	case http.MethodTrace:
		return item.Trace
	}

	return nil
}

func ptr[T any](v T) *T { return &v }
