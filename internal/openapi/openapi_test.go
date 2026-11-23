package openapi

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/axllent/mailpit/server/apiv1"
	"github.com/swaggest/jsonschema-go"
)

type object = map[string]any

func generated(t *testing.T) object {
	t.Helper()

	b, err := Generate()
	if err != nil {
		t.Fatal(err)
	}

	var doc object
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}

	return doc
}

func schema(t *testing.T, doc object, name string) object {
	t.Helper()

	s, ok := doc["components"].(object)["schemas"].(object)[name].(object)
	if !ok {
		t.Fatalf("schema %q not found", name)
	}

	return s
}

// The committed specification is what is generated. If this fails, run `go run ./internal/openapi/gen`.
func TestSpecUpToDate(t *testing.T) {
	root, err := moduleRoot()
	if err != nil {
		t.Fatal(err)
	}

	committed, err := os.ReadFile(filepath.Join(root, "server", "ui", "api", "v1", "openapi.json"))
	if err != nil {
		t.Fatal(err)
	}

	fresh, err := Generate()
	if err != nil {
		t.Fatal(err)
	}

	// a checkout may have converted the line endings
	committed = bytes.ReplaceAll(committed, []byte("\r\n"), []byte("\n"))

	if !bytes.Equal(committed, fresh) {
		t.Fatal("server/ui/api/v1/openapi.json is out of date, run `go run ./internal/openapi/gen` from the repository root")
	}
}

func TestGenerateIsDeterministic(t *testing.T) {
	first, err := Generate()
	if err != nil {
		t.Fatal(err)
	}

	for range 3 {
		again, err := Generate()
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(first, again) {
			t.Fatal("two runs generated different specifications")
		}
	}
}

// Every route is an operation, and the identifying parts of the routes are unique.
func TestEveryRouteIsDocumented(t *testing.T) {
	doc := generated(t)
	paths := doc["paths"].(object)

	seen := map[string]bool{}

	for _, rt := range apiv1.Routes {
		op, ok := paths[rt.Path].(object)[strings.ToLower(rt.Method)].(object)
		if !ok {
			t.Errorf("%s %s is not in the specification", rt.Method, rt.Path)
			continue
		}

		if op["operationId"] != rt.ID {
			t.Errorf("%s %s: operationId %v, want %s", rt.Method, rt.Path, op["operationId"], rt.ID)
		}

		if key := rt.Method + " " + rt.Path; seen[key] {
			t.Errorf("%s is defined twice", key)
		} else {
			seen[key] = true
		}
	}
}

// What the doc comments say (descriptions, required, example, default and ignored fields) reaches the
// specification. Each is paired with a control of the same shape which must not be affected.
func TestDocCommentsAreApplied(t *testing.T) {
	doc := generated(t)

	// openapi:ignore: the legacy fields are not documented, their neighbours are
	summary := schema(t, doc, "MessagesSummary")["properties"].(object)
	if _, ok := summary["count"]; ok {
		t.Error("MessagesSummary.count is ignored by its doc comment but is documented")
	}

	if _, ok := summary["messages_count"]; !ok {
		t.Error("control: MessagesSummary.messages_count should be documented")
	}

	relay := schema(t, doc, "WebUIMessageRelay")["properties"].(object)
	if _, ok := relay["RecipientAllowlist"]; ok {
		t.Error("RecipientAllowlist is ignored by its doc comment but is documented")
	}

	if _, ok := relay["SMTPServer"]; !ok {
		t.Error("control: WebUIMessageRelay.SMTPServer should be documented")
	}

	// required: true
	trigger := schema(t, doc, "ChaosTrigger")
	if got := trigger["required"]; len(got.([]any)) != 2 {
		t.Errorf("ChaosTrigger required = %v, want ErrorCode and Probability", got)
	}

	if _, ok := schema(t, doc, "Attachment")["required"]; ok {
		t.Error("control: Attachment has no required fields")
	}

	// the recipients of a send request share one named type, which has its own docs
	address := schema(t, doc, "SendAddress")
	if d := address["properties"].(object)["Email"].(object)["description"]; d != "Email address" {
		t.Errorf("SendAddress.Email description = %v", d)
	}

	if got := address["required"]; len(got.([]any)) != 1 || got.([]any)[0] != "Email" {
		t.Errorf("SendAddress required = %v, want Email", got)
	}

	// the same type, used by a list and by a single recipient
	req := schema(t, doc, "SendRequest")["properties"].(object)
	if ref := req["To"].(object)["items"].(object)["$ref"]; ref != "#/components/schemas/SendAddress" {
		t.Errorf("SendRequest.To items = %v", ref)
	}

	if ref := req["From"].(object)["$ref"]; ref != "#/components/schemas/SendAddress" {
		t.Errorf("SendRequest.From = %v", ref)
	}

	// control: a field of another shape is not a recipient
	if _, ok := req["Bcc"].(object)["items"].(object)["$ref"]; ok {
		t.Error("control: SendRequest.Bcc is a list of strings, not recipients")
	}

	// example and default
	ids := schema(t, doc, "SetReadStatusRequest")["properties"].(object)["IDs"].(object)
	if ids["default"] == nil || ids["examples"] == nil {
		t.Errorf("SetReadStatusRequest.IDs default and examples are missing: %v", ids)
	}

	// descriptions of a type and of a field
	if d := schema(t, doc, "Message")["properties"].(object)["Subject"].(object)["description"]; d != "Message subject" {
		t.Errorf("Message.Subject description = %v", d)
	}
}

// The format of an integer is its Go type, as the original go-swagger specification had it. Each is paired
// with a control of another type.
func TestIntegerFormats(t *testing.T) {
	props := schema(t, generated(t), "AppInformation")["properties"].(object)

	for field, format := range map[string]string{
		"DatabaseSize": "uint64", // unsigned
		"Messages":     "uint64",
		"RuntimeStats": "", // an object, not an integer
		"Version":      "", // a string
	} {
		got, _ := props[field].(object)["format"].(string)
		if got != format {
			t.Errorf("AppInformation.%s format = %q, want %q", field, got, format)
		}
	}

	// control: a signed integer is not unsigned
	if f := schema(t, generated(t), "MessagesSummary")["properties"].(object)["start"].(object)["format"]; f != "int64" {
		t.Errorf("MessagesSummary.start format = %v, want int64", f)
	}
}

// The descriptions of the responses are the ones written for the route (successful) and the shared ones (errors),
// not the library's defaults ("OK", "Bad Request"). Each is paired with a control of another kind.
func TestResponseDescriptions(t *testing.T) {
	doc := generated(t)
	paths := doc["paths"].(object)

	description := func(path, method, code string) any {
		return paths[path].(object)[method].(object)["responses"].(object)[code].(object)["description"]
	}

	for _, c := range []struct{ path, method, code, want string }{
		{"/view/{id}.html", "get", "200", "HTML response"},
		{"/view/{id}.txt", "get", "200", "Plain text response"}, // control: another kind of text response
		{"/api/v1/message/{id}/part/{partID}", "get", "200", "Binary data response which inherits the attachment's content type."},
		{"/api/v1/tags", "put", "200", "Plain text \"ok\" response"},
		{"/view/{id}.html", "get", "400", errorDescription},
		{"/view/{id}.html", "get", "404", notFoundDescription},
		{"/api/v1/send", "post", "400", jsonErrorDescription}, // control: JSON errors
		{"/api/v1/send", "post", "503", jsonErrorDescription},
	} {
		if got := description(c.path, c.method, c.code); got != c.want {
			t.Errorf("%s %s %s description = %q, want %q", strings.ToUpper(c.method), c.path, c.code, got, c.want)
		}
	}

	// every route's successful response is described by the route, none by the library default
	for _, rt := range apiv1.Routes {
		if got := description(rt.Path, strings.ToLower(rt.Method), "200"); got != rt.ResponseDescription {
			t.Errorf("%s %s: description %q, want %q", rt.Method, rt.Path, got, rt.ResponseDescription)
		}
	}
}

// A route's response example is on its successful response only. Each is paired with a control without one.
func TestResponseExamples(t *testing.T) {
	doc := generated(t)
	paths := doc["paths"].(object)

	content := func(path, method, code string) object {
		return paths[path].(object)[method].(object)["responses"].(object)[code].(object)["content"].(object)
	}

	withExample := 0

	for _, rt := range apiv1.Routes {
		// the example as it is in the JSON specification
		var want any

		if rt.ResponseExample != nil {
			b, err := json.Marshal(rt.ResponseExample)
			if err != nil {
				t.Fatal(err)
			}

			if err := json.Unmarshal(b, &want); err != nil {
				t.Fatal(err)
			}
		}

		for ct, media := range content(rt.Path, strings.ToLower(rt.Method), "200") {
			var got any

			examples, has := media.(object)["examples"].(object)
			if has {
				got = examples["Example"].(object)["value"]
			}

			switch {
			case rt.ResponseExample != nil && !has:
				t.Errorf("%s %s [%s]: the example %v is missing", rt.Method, rt.Path, ct, want)
			case rt.ResponseExample != nil && !reflect.DeepEqual(got, want):
				t.Errorf("%s %s [%s]: example %v, want %v", rt.Method, rt.Path, ct, got, want)
			case rt.ResponseExample == nil && has:
				t.Errorf("%s %s [%s]: has an example %v which the route does not", rt.Method, rt.Path, ct, got)
			}

			if has {
				withExample++
			}
		}
	}

	if withExample == 0 {
		t.Error("no route has a response example")
	}

	// a JSON response has its example as an object
	headers := content("/api/v1/message/{id}/headers", "get", "200")["application/json"].(object)["examples"].(object)["Example"].(object)["value"]
	if _, ok := headers.(object)["Subject"]; !ok {
		t.Errorf("the headers example is not an object of headers: %v", headers)
	}

	// controls: a text response without an example, and the error responses of a route with one
	if _, ok := content("/view/{id}.txt", "get", "200")["text/plain"].(object)["examples"]; ok {
		t.Error("control: GET /view/{id}.txt should have no example")
	}

	if _, ok := content("/api/v1/tags", "put", "400")["text/plain"].(object)["examples"]; ok {
		t.Error("control: the 400 response of a route with an example should have none")
	}
}

// The marker for a DELETE request body is honoured, and bodies of other methods do not need it.
func TestRequestBodies(t *testing.T) {
	doc := generated(t)
	paths := doc["paths"].(object)

	for _, c := range []struct{ path, method string }{
		{"/api/v1/messages", "delete"}, // needs the marker
		{"/api/v1/messages", "put"},
		{"/api/v1/send", "post"},
		{"/api/v1/chaos", "put"},
	} {
		if _, ok := paths[c.path].(object)[c.method].(object)["requestBody"]; !ok {
			t.Errorf("%s %s has no request body", strings.ToUpper(c.method), c.path)
		}
	}

	// control: a GET has none
	if _, ok := paths["/api/v1/messages"].(object)["get"].(object)["requestBody"]; ok {
		t.Error("GET /api/v1/messages should not have a request body")
	}
}

// route returns a copy of the route with the operation ID.
func route(t *testing.T, id string) apiv1.Route {
	t.Helper()

	for _, rt := range apiv1.Routes {
		if rt.ID == id {
			return rt
		}
	}

	t.Fatalf("no route %q", id)

	return apiv1.Route{}
}

// The generator fails, rather than writing an incomplete specification, for these. Every one has a control
// (the valid route which it was made from) which must generate.
func TestInvalidRoutesFail(t *testing.T) {
	plain := route(t, "GetAllTags")      // no parameters
	param := route(t, "DeleteTagParams") // a path parameter
	body := route(t, "SetTagsParams")    // a body

	for _, rt := range []apiv1.Route{plain, param, body} {
		if _, err := generate([]apiv1.Route{rt}); err != nil {
			t.Fatalf("control route %s must be valid: %v", rt.ID, err)
		}
	}

	mutate := func(rt apiv1.Route, f func(r *apiv1.Route)) apiv1.Route {
		f(&rt)

		return rt
	}

	// control: an example of the response type is valid, for JSON as well as text
	withExample := mutate(plain, func(r *apiv1.Route) { r.ResponseExample = []string{"Tag 1", "Tag 2"} })
	if _, err := generate([]apiv1.Route{withExample}); err != nil {
		t.Fatalf("control route with a valid example must be valid: %v", err)
	}

	for name, routes := range map[string][]apiv1.Route{
		"no summary":                             {mutate(plain, func(r *apiv1.Route) { r.Summary = "" })},
		"no description":                         {mutate(plain, func(r *apiv1.Route) { r.Description = "" })},
		"no tag":                                 {mutate(plain, func(r *apiv1.Route) { r.Tag = "" })},
		"no id":                                  {mutate(plain, func(r *apiv1.Route) { r.ID = "" })},
		"no response":                            {mutate(plain, func(r *apiv1.Route) { r.Response = nil })},
		"example of another type":                {mutate(plain, func(r *apiv1.Route) { r.ResponseExample = apiv1.TextBody("ok") })},
		"example which is not the response type": {mutate(body, func(r *apiv1.Route) { r.ResponseExample = "ok" })}, // a string, not a TextBody
		"no response description":                {mutate(plain, func(r *apiv1.Route) { r.ResponseDescription = "" })},
		"no content type":                        {mutate(plain, func(r *apiv1.Route) { r.ContentTypes = nil })},
		"duplicate operation id":                 {plain, mutate(plain, func(r *apiv1.Route) { r.Method = http.MethodDelete })},
		"duplicate route":                        {plain, plain},
		"undocumented path parameter": {
			mutate(param, func(r *apiv1.Route) { r.Params = nil }),
		},
		"documented path parameter not in the path": {
			mutate(param, func(r *apiv1.Route) { r.Path = "/api/v1/tags" }),
		},
		"parameters type with no source": {
			mutate(param, func(r *apiv1.Route) { r.Params = new(noSource) }),
		},
		"body type with no source": {
			mutate(body, func(r *apiv1.Route) { r.Body = new(noSource) }),
		},
	} {
		_, err := generate(routes)
		if err == nil {
			t.Errorf("%s: generated a specification, should have failed", name)
		}

		t.Logf("%s: %v", name, err)
	}
}

// An example or default must be JSON of the type of its field, else it is an error rather than a wrong example (a
// string where an object is expected, as with unquoted keys). Each failure has a control of the same type which is valid.
func TestDocValues(t *testing.T) {
	type object2 struct{ Name string }

	for name, c := range map[string]struct {
		raw     string
		typ     reflect.Type
		want    any
		wantErr bool
	}{
		"string plain text":         {"abc", reflect.TypeOf(""), "abc", false},
		"string JSON":               {`"abc"`, reflect.TypeOf(""), "abc", false},
		"string which is a number":  {"1234", reflect.TypeOf(""), "1234", false}, // an ID, not the number
		"string which is a bool":    {"true", reflect.TypeOf(""), "true", false},
		"time":                      {"1970-01-01T00:00:00Z", reflect.TypeOf(time.Time{}), "1970-01-01T00:00:00Z", false},
		"integer":                   {"451", reflect.TypeOf(uint64(0)), float64(451), false},
		"integer not a number":      {"abc", reflect.TypeOf(0), nil, true},
		"integer as a string":       {`"451"`, reflect.TypeOf(0), nil, true},
		"boolean":                   {"true", reflect.TypeOf(false), true, false},
		"boolean not a boolean":     {"yes", reflect.TypeOf(false), nil, true},
		"array":                     {`["a","b"]`, reflect.TypeOf([]string{}), []any{"a", "b"}, false},
		"array not JSON":            {"a", reflect.TypeOf([]string{}), nil, true},
		"array given an object":     {`{"a":1}`, reflect.TypeOf([]string{}), nil, true},
		"map":                       {`{"a":1}`, reflect.TypeOf(map[string]int{}), map[string]any{"a": float64(1)}, false},
		"map given an array":        {`[1]`, reflect.TypeOf(map[string]int{}), nil, true},
		"object":                    {`{"Name":"John Doe"}`, reflect.TypeOf(object2{}), map[string]any{"Name": "John Doe"}, false},
		"object through a pointer":  {`{"Name":"John Doe"}`, reflect.TypeOf(&object2{}), map[string]any{"Name": "John Doe"}, false},
		"object with unquoted keys": {`{Name: "John Doe"}`, reflect.TypeOf(&object2{}), nil, true},
	} {
		got, err := docValue(c.raw, c.typ)
		if (err != nil) != c.wantErr {
			t.Errorf("%s: error = %v, want error %v", name, err, c.wantErr)
			continue
		}

		if !c.wantErr && !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: value = %#v, want %#v", name, got, c.want)
		}
	}
}

// The docs of an anonymous struct cannot be found, so it is an error rather than being left undocumented.
// Each is paired with the same shape as a named type.
func TestAnonymousStructsAreRejected(t *testing.T) {
	type named struct{ A string }

	for name, c := range map[string]struct {
		typ    reflect.Type
		inline bool
	}{
		"struct":          {reflect.TypeOf(struct{ A string }{}), true},
		"pointer":         {reflect.TypeOf(&struct{ A string }{}), true},
		"slice":           {reflect.TypeOf([]struct{ A string }{}), true},
		"map":             {reflect.TypeOf(map[string]struct{ A string }{}), true},
		"named":           {reflect.TypeOf(named{}), false},
		"named slice":     {reflect.TypeOf([]named{}), false},
		"standard type":   {reflect.TypeOf(time.Time{}), false},
		"string":          {reflect.TypeOf(""), false},
		"string slice":    {reflect.TypeOf([]string{}), false},
		"map of strings":  {reflect.TypeOf(map[string]string{}), false},
		"map of named":    {reflect.TypeOf(map[string]named{}), false},
		"pointer to name": {reflect.TypeOf(&named{}), false},
	} {
		if got := inlineStruct(c.typ); got != c.inline {
			t.Errorf("%s: inlineStruct = %v, want %v", name, got, c.inline)
		}
	}

	// applying the docs to a struct which has one fails, and the same with a named type does not
	type withInline struct{ Inner struct{ A string } }

	type withNamed struct{ Inner named }

	for typ, wantErr := range map[reflect.Type]bool{reflect.TypeOf(withInline{}): true, reflect.TypeOf(withNamed{}): false} {
		s := &jsonschema.Schema{}
		s.Properties = map[string]jsonschema.SchemaOrBool{"Inner": {TypeObject: &jsonschema.Schema{}}}
		sd := &structDoc{fields: map[string]*fieldDoc{"Inner": {}}}

		if err := apply(s, typ, sd, nil); (err != nil) != wantErr {
			t.Errorf("%s: apply error = %v, want error %v", typ, err, wantErr)
		}
	}
}

// noSource is defined in a test, and so not found in the sources which the generator reads.
type noSource struct{ Name string }

func TestParseComment(t *testing.T) {
	for name, c := range map[string]struct {
		comment                   string
		desc, example, def, model string
		required, ignore          bool
	}{
		"plain":     {comment: "// Hello\n// world", desc: "Hello\nworld"},
		"directive": {comment: "// A field\n//\n// required: true\n// example: [\"a\"]\n// default: []", desc: "A field", required: true, example: `["a"]`, def: "[]"},
		"ignore":    {comment: "// Legacy\n// openapi:ignore", desc: "Legacy", ignore: true},
		"model":     {comment: "// A type\n//\n// openapi:model Thing", desc: "A type", model: "Thing"},
		"not":       {comment: "// required: false", desc: ""},
		"similar":   {comment: "// Mentions openapi:ignore and required: true inside text", desc: "Mentions openapi:ignore and required: true inside text"},
	} {
		fd := commentOf(t, c.comment)

		if fd.description != c.desc || fd.example != c.example || fd.defaultVal != c.def || fd.model != c.model ||
			fd.required != c.required || fd.ignore != c.ignore {
			t.Errorf("%s: parsed %+v", name, fd)
		}
	}
}

// commentOf parses a doc comment as the generator does.
func commentOf(t *testing.T, comment string) fieldDoc {
	t.Helper()

	f, err := parser.ParseFile(token.NewFileSet(), "x.go", "package x\n\n"+comment+"\ntype T struct{}\n", parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	return parseComment(f.Decls[0].(*ast.GenDecl).Doc)
}
