package openapi

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/swaggest/jsonschema-go"
)

// The OpenAPI descriptions come from the Go doc comments of the structs, so that they sit next to the code.
// Only struct and field doc comments are read, and only this small set of directives within them:
//
//	openapi:model Name   the component name of a (type) schema
//	openapi:ignore       the field is not documented
//	required: true       the field is required
//	example: <json>      an example value (JSON, else a string)
//	default: <json>      a default value (JSON, else a string)
//
// Anything which cannot be found or understood is an error, never silently left out of the specification.

type fieldDoc struct {
	description string
	required    bool
	ignore      bool
	example     string
	defaultVal  string
	model       string
}

type structDoc struct {
	fieldDoc
	fields map[string]*fieldDoc
}

// docs reads and caches the doc comments of the packages within this module.
type docs struct {
	root       string // module root directory
	modulePath string
	pkgs       map[string]map[string]*structDoc // package path => type name => docs
}

func newDocs() (*docs, error) {
	root, err := moduleRoot()
	if err != nil {
		return nil, err
	}

	b, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return nil, err
	}

	first, _, _ := strings.Cut(string(b), "\n")

	modulePath := strings.TrimSpace(strings.TrimPrefix(first, "module"))
	if modulePath == "" || modulePath == strings.TrimSpace(first) {
		return nil, fmt.Errorf("cannot read the module path from %s", filepath.Join(root, "go.mod"))
	}

	return &docs{root: root, modulePath: modulePath, pkgs: map[string]map[string]*structDoc{}}, nil
}

// moduleRoot finds the directory of go.mod from the location of this source file, which is only
// possible (and only needed) when running from a source tree, eg: `go run` or `go test`.
func moduleRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("cannot locate the source tree")
	}

	dir := filepath.Dir(file)

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("cannot find go.mod above %s (are you running from the source tree, without -trimpath?)", file)
		}

		dir = parent
	}
}

// structFor returns the docs of a named struct type of this module, or nil if it is not one of this
// module's structs (stdlib and third party types are not documented from comments).
func (d *docs) structFor(t reflect.Type) (*structDoc, error) {
	if t.Kind() != reflect.Struct || t.Name() == "" || !strings.HasPrefix(t.PkgPath(), d.modulePath+"/") {
		return nil, nil
	}

	types, ok := d.pkgs[t.PkgPath()]
	if !ok {
		var err error

		types, err = parseDir(filepath.Join(d.root, filepath.FromSlash(strings.TrimPrefix(t.PkgPath(), d.modulePath+"/"))))
		if err != nil {
			return nil, err
		}

		d.pkgs[t.PkgPath()] = types
	}

	sd := types[t.Name()]
	if sd == nil {
		return nil, fmt.Errorf("the source of %s.%s was not found", t.PkgPath(), t.Name())
	}

	return sd, nil
}

func parseDir(dir string) (map[string]*structDoc, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	out := map[string]*structDoc{}
	fset := token.NewFileSet()

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}

		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}

			for _, spec := range gd.Specs {
				ts := spec.(*ast.TypeSpec)

				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}

				doc := ts.Doc
				if doc == nil {
					doc = gd.Doc
				}

				sd := parseStruct(st)
				sd.fieldDoc = parseComment(doc)
				out[ts.Name.Name] = sd
			}
		}
	}

	return out, nil
}

func parseStruct(st *ast.StructType) *structDoc {
	sd := &structDoc{fields: map[string]*fieldDoc{}}

	for _, f := range st.Fields.List {
		fd := parseComment(f.Doc)

		for _, n := range f.Names {
			sd.fields[n.Name] = &fd
		}
	}

	return sd
}

// parseComment splits a doc comment into a description and its directives.
func parseComment(cg *ast.CommentGroup) fieldDoc {
	fd := fieldDoc{}
	if cg == nil {
		return fd
	}

	var desc []string

	for _, line := range strings.Split(strings.TrimSpace(cg.Text()), "\n") {
		t := strings.TrimSpace(line)

		switch {
		case t == "openapi:ignore":
			fd.ignore = true
		case strings.HasPrefix(t, "openapi:model "):
			fd.model = strings.TrimSpace(strings.TrimPrefix(t, "openapi:model "))
		case strings.HasPrefix(t, "required:"):
			fd.required = strings.TrimSpace(strings.TrimPrefix(t, "required:")) == "true"
		case strings.HasPrefix(t, "example:"):
			fd.example = strings.TrimSpace(strings.TrimPrefix(t, "example:"))
		case strings.HasPrefix(t, "default:"):
			fd.defaultVal = strings.TrimSpace(strings.TrimPrefix(t, "default:"))
		default:
			desc = append(desc, line)
		}
	}

	fd.description = strings.TrimSpace(strings.Join(desc, "\n"))

	return fd
}

// docValue returns the value of an example or default directive. It is JSON of the type of the field (a string
// field may also be plain text), and anything else is an error rather than a silently wrong example.
func docValue(raw string, t reflect.Type) (any, error) {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	var v any

	parsed := json.Unmarshal([]byte(raw), &v) == nil

	kind := "" // the JSON type which the field is encoded as

	switch {
	case t == reflect.TypeOf(time.Time{}), t.Kind() == reflect.String, t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8:
		if s, ok := v.(string); parsed && ok {
			return s, nil
		}

		return raw, nil // plain text, or something which only looks like another type (eg: 1234 for an ID)
	case t.Kind() == reflect.Bool:
		kind = "boolean"
	case t.Kind() >= reflect.Int && t.Kind() <= reflect.Float64:
		kind = "number"
	case t.Kind() == reflect.Struct || t.Kind() == reflect.Map:
		kind = "object"
	case t.Kind() == reflect.Slice || t.Kind() == reflect.Array:
		kind = "array"
	default:
		return nil, fmt.Errorf("%q cannot be used for a %s field", raw, t.Kind())
	}

	ok := false

	switch v.(type) {
	case bool:
		ok = kind == "boolean"
	case float64:
		ok = kind == "number"
	case map[string]any:
		ok = kind == "object"
	case []any:
		ok = kind == "array"
	}

	if !parsed || !ok {
		return nil, fmt.Errorf("%q is not valid JSON for a %s field", raw, kind)
	}

	return v, nil
}

// apply sets the descriptions and directives of a struct's fields on the properties of its schema.
func apply(s *jsonschema.Schema, t reflect.Type, sd *structDoc, d *docs) error {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		// an embedded struct shares the parent's schema
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			esd, err := d.structFor(f.Type)
			if err != nil {
				return err
			}

			if esd != nil {
				if err := apply(s, f.Type, esd, d); err != nil {
					return err
				}
			}

			continue
		}

		fd := sd.fields[f.Name]
		if fd == nil {
			continue
		}

		name := propertyName(f)

		if fd.ignore {
			delete(s.Properties, name)

			continue
		}

		prop, ok := s.Properties[name]
		if !ok || prop.TypeObject == nil {
			continue
		}

		po := prop.TypeObject

		// the docs of an anonymous struct cannot be found, so it must be a named type (never silently undocumented)
		if inlineStruct(f.Type) {
			return fmt.Errorf("%s.%s is an anonymous struct, give it a named type", t.Name(), f.Name)
		}

		if po.Description == nil && fd.description != "" {
			po.WithDescription(fd.description)
		}

		if fd.example != "" {
			v, err := docValue(fd.example, f.Type)
			if err != nil {
				return fmt.Errorf("%s.%s: example: %w", t.Name(), f.Name, err)
			}

			po.WithExamples(v)
		}

		if fd.defaultVal != "" && po.Default == nil {
			v, err := docValue(fd.defaultVal, f.Type)
			if err != nil {
				return fmt.Errorf("%s.%s: default: %w", t.Name(), f.Name, err)
			}

			po.Default = &v
		}

		if fd.required && !contains(s.Required, name) {
			s.Required = append(s.Required, name)
		}
	}

	return nil
}

// inlineStruct reports whether the type is an anonymous struct, or a pointer, slice, array or map of one.
func inlineStruct(t reflect.Type) bool {
	for {
		switch t.Kind() {
		case reflect.Ptr, reflect.Slice, reflect.Array, reflect.Map:
			t = t.Elem()
		case reflect.Struct:
			return t.Name() == ""
		default:
			return false
		}
	}
}

func propertyName(f reflect.StructField) string {
	for _, k := range []string{"path", "query", "json"} {
		if tag := strings.Split(f.Tag.Get(k), ",")[0]; tag != "" && tag != "-" {
			return tag
		}
	}

	return f.Name
}

func contains(l []string, s string) bool {
	for _, v := range l {
		if v == s {
			return true
		}
	}

	return false
}

// paramDescriptions returns the doc comments of the path and query parameter fields of a params struct.
func paramDescriptions(t reflect.Type, d *docs, out map[string]string) error {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	sd, err := d.structFor(t)
	if err != nil {
		return err
	}

	if sd == nil {
		return fmt.Errorf("parameters type %s is not a struct of this module", t)
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		for _, k := range []string{"path", "query"} {
			if name := f.Tag.Get(k); name != "" {
				fd := sd.fields[f.Name]
				if fd == nil || fd.description == "" {
					return fmt.Errorf("parameter %q of %s has no doc comment", name, t.Name())
				}

				out[name] = fd.description
			}
		}
	}

	return nil
}
