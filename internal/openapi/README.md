# OpenAPI specification

Mailpit's API documentation, [`server/ui/api/v1/openapi.json`](../../server/ui/api/v1/openapi.json) (OpenAPI 3.1), is
**generated** from the Go source. Do not edit it by hand.

- The API is defined once, in the `Routes` table in [`server/apiv1/routes.go`](../../server/apiv1/routes.go). The server
  registers its handlers from that table, and the specification is generated from it, so a registered route cannot be
  missing from the documentation.
- This package is a development tool. Nothing in the Mailpit binary imports it, and its dependencies are not built into
  Mailpit.
- No external tools or Docker are needed, only Go.

## Regenerating the specification

After changing a route, or a type (or the doc comments of a type) which a route uses, run this from the repository root:

```shell
go run ./internal/openapi/gen
```

Commit the updated `openapi.json` with your change. `go test ./internal/openapi` fails if the committed file is out of
date, and if the generator finds anything missing or inconsistent it fails with an error rather than writing an
incomplete specification.

## Adding a route

1. **Write the handler** in `server/apiv1` as usual.
2. **Use named types** for the request body and the response. A struct which appears in the specification must be a
   named type (an anonymous struct is an error) with a doc comment, and an `openapi:model Name` line to name its schema.
   The doc comments of the struct and of its fields are the descriptions. A request body type should be the same type
   which the handler decodes into.
3. **Describe path and query parameters** with a small struct in `routes.go`, one field per parameter. The `path:"name"` or
   `query:"name"` tag (with `json:"-"`) names it, and the field's doc comment is the description, which is required.
4. **Add a `Route{...}` entry** to `Routes` in `routes.go`. `Method`, `Path` (with `{id}` style parameters), `Handler`,
   `ID` (unique), `Tag`, `Summary`, `Description`, `Response`, `ResponseDescription` and `ContentTypes` are required. See
   the `Route` type for the optional fields (`Params`, `Body`, `ResponseExample` (a value of the `Response` type) for a response which is not a struct with `example:` fields, `NotFound`, `SendAPI`, `Enabled`, `JSONErrors` and `ExtraErrors`).
5. **Regenerate** (see above) and run `go test ./internal/openapi ./server/...`.

There is no need to register the route in `server/server.go`, or to do anything for a webroot (it is added to the
specification when it is served).

### Example

```go
// MessageSize is the size of a message
//
// openapi:model MessageSize
type MessageSize struct {
	// Size in bytes
	Bytes uint64
}

type sizeParams struct {
	// Message database ID or "latest"
	ID string `path:"id" json:"-"`
}

// in Routes:
{
	Method: http.MethodGet, Path: "/api/v1/message/{id}/size",
	Handler: MessageSizeHandler,
	ID:      "MessageSizeParams", Tag: "message",
	Summary:             "Get message size",
	Description:         "Returns the size of a message in bytes.",
	Params:              sizeParams{},
	Response:            MessageSize{},
	ResponseDescription: "Message size",
	ContentTypes:        []string{jsonType},
	NotFound:            true,
},
```

Note that the parameters struct is documentation only: the handler still reads its path and query parameters itself, and
nothing checks that the two agree. If you change a parameter in a handler, change its parameters struct too.

## Doc comments

The descriptions come from the Go doc comments of the structs and their fields. These directives are understood within
them, each on a line of its own:

| Directive             | Effect                                                         |
| --------------------- | -------------------------------------------------------------- |
| `openapi:model Name`  | The component name of a type (on the struct)                   |
| `openapi:ignore`      | The field is not documented                                    |
| `required: true`      | The field is required                                          |
| `example: <json>`     | An example value (JSON, otherwise a string)                    |
| `default: <json>`     | A default value (JSON, otherwise a string)                     |

The parameters struct fields also accept the `default:"..."` and `required:"true"` tags.

A `DELETE` request body is not part of the OpenAPI standard (editors may warn about it), so its type needs a
`func (T) ForceRequestBody() {}` marker method. See `DeleteMessagesRequest`.

## What is checked

The generator fails (and so `go run ./internal/openapi/gen` and the tests do) for:

- a route without an `ID`, `Tag`, `Summary`, `Description`, `Response` or `ResponseDescription`, or a duplicate `ID` or route
- a path parameter which is not documented by `Params` (or documented but not in the path), or one without a description
- a type whose source cannot be found, an anonymous struct, or two different types with the same schema name
- a reference to a schema which does not exist

`go test ./server` also checks that every documented route is actually registered (with and without a webroot).

The generator reads the source files of the module, so it only works from a source checkout (for example `go run` and
`go test`, but not a binary which is moved elsewhere, or a build using `-trimpath`).
