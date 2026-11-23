package openapi

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var pathParamRe = regexp.MustCompile(`\{([^}/]+)\}`)

// validate checks the generated specification for the mistakes which the generator cannot rule out itself.
// The generator fails rather than writing a specification which does not pass.
func validate(doc map[string]any, routes int) error {
	var errs []error

	fail := func(format string, a ...any) { errs = append(errs, fmt.Errorf(format, a...)) }

	if v, _ := doc["openapi"].(string); !strings.HasPrefix(v, "3.1.") {
		fail("unexpected openapi version %q", v)
	}

	paths, _ := doc["paths"].(map[string]any)
	ids := map[string]string{}
	operations := 0

	for path, item := range paths {
		for method, o := range item.(map[string]any) {
			op, ok := o.(map[string]any)
			if !ok {
				continue
			}

			operations++
			where := strings.ToUpper(method) + " " + path

			id, _ := op["operationId"].(string)
			if id == "" {
				fail("%s: no operationId", where)
			} else if prev, dup := ids[id]; dup {
				fail("%s: operationId %q is also used by %s", where, id, prev)
			}

			ids[id] = where

			for _, f := range []string{"summary", "description", "tags"} {
				if v, ok := op[f]; !ok || v == "" {
					fail("%s: no %s", where, f)
				}
			}

			// the path parameters in the path and the documented ones are the same
			documented := map[string]bool{}

			params, _ := op["parameters"].([]any)
			for _, p := range params {
				pm := p.(map[string]any)
				if pm["in"] == "path" {
					documented[pm["name"].(string)] = true

					if pm["required"] != true {
						fail("%s: path parameter %q is not required", where, pm["name"])
					}
				}

				if d, _ := pm["description"].(string); d == "" {
					fail("%s: parameter %q has no description", where, pm["name"])
				}
			}

			inPath := map[string]bool{}
			for _, m := range pathParamRe.FindAllStringSubmatch(path, -1) {
				inPath[m[1]] = true
			}

			for name := range inPath {
				if !documented[name] {
					fail("%s: path parameter {%s} is not documented", where, name)
				}
			}

			for name := range documented {
				if !inPath[name] {
					fail("%s: documented path parameter %q is not in the path", where, name)
				}
			}
		}
	}

	if operations != routes {
		fail("%d routes generated %d operations (is a method and path used twice?)", routes, operations)
	}

	// every $ref resolves
	schemas := map[string]any{}
	if c, ok := doc["components"].(map[string]any); ok {
		schemas, _ = c["schemas"].(map[string]any)
	}

	var walk func(v any)

	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			if ref, ok := x["$ref"].(string); ok {
				name := strings.TrimPrefix(ref, "#/components/schemas/")
				if _, found := schemas[name]; !found || name == ref {
					fail("unresolved reference %q", ref)
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

	return errors.Join(errs...)
}
