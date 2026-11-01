package openapi

import (
	"reflect"
	"testing"

	"github.com/go-openapi/spec"
	"github.com/gowok/should"
)

type testRoute struct {
	method, path string
}

func (r testRoute) Method() string { return r.method }
func (r testRoute) Path() string   { return r.path }

func httpDocsInitialize(t *testing.T) *httpDocs {
	should := should.New(t)
	input := &httpDocs{
		Title:          "1",
		Version:        "2",
		Host:           "localhost",
		Description:    "4",
		TermsOfService: "5",
		BasePath:       "6",
		Contact: contact{
			Name:  "7",
			URL:   "8",
			Email: "9",
		},
		License: license{
			Name: "10",
			URL:  "11",
		},
		Schemes:             []string{},
		Consumes:            []string{},
		Produces:            []string{},
		Tags:                []spec.Tag{},
		SecurityDefinitions: map[string]securityDefinition{},
	}

	docs := newHttpDocs(*input)
	should.NotNil(docs)
	should.NotNil(docs.swagger)
	input.swagger = docs.swagger

	should.Equal(input, docs)
	return docs
}

func TestNewHttpDocs(t *testing.T) {
	httpDocsInitialize(t)
}

func TestHttpDocsNew(t *testing.T) {
	docs := httpDocsInitialize(t)
	docs.Add("1", func(o *spec.Operation) {
		should.NotNil(t, o)
	})(testRoute{
		method: "GET",
		path:   "/users",
	})

	should.NotNil(t, docs.swagger.Paths.Paths["/users"].Get)
}

func TestHttpDocsAddReusedForMultipleRoutes(t *testing.T) {
	docs := httpDocsInitialize(t)

	// A single descriptor reused for several routes must not share state.
	add := docs.Add("shared", func(o *spec.Operation) {
		o.AddParam(spec.QueryParam("q"))
	})
	add(testRoute{method: "GET", path: "/a"})
	add(testRoute{method: "POST", path: "/b"})

	getA := docs.swagger.Paths.Paths["/a"].Get
	postB := docs.swagger.Paths.Paths["/b"].Post
	should.NotNil(t, getA)
	should.NotNil(t, postB)

	// Each route must get its own operation, not the same pointer.
	if getA == postB {
		t.Fatal("operations for /a and /b share the same pointer")
	}

	// Registering POST /b must not leak the earlier GET operation into /b.
	should.Nil(t, docs.swagger.Paths.Paths["/b"].Get)
}

func TestSpecSchemaOfReflectTypeInt(t *testing.T) {
	schema := specSchemaOfReflectType(reflect.TypeOf(int(0)))
	should.Equal(t, 1, len(schema.Type))
	should.Equal(t, "integer", string(schema.Type[0]))
	should.Equal(t, "int64", schema.Format)
}

func TestSpecSchemaOfStructSkipsIgnoredField(t *testing.T) {
	type user struct {
		Name   string
		Secret string `json:"-"`
	}

	schema := specSchemaOfStruct(user{})
	should.Equal(t, 1, len(schema.Properties))
	_, ok := schema.Properties["-"]
	should.Equal(t, false, ok)
	_, ok = schema.Properties["Secret"]
	should.Equal(t, false, ok)
	_, ok = schema.Properties["Name"]
	should.Equal(t, true, ok)
}

func TestSpecSchemaOfStructEmptyJSONName(t *testing.T) {
	type user struct {
		Name  string
		Email string `json:",omitempty"`
	}

	schema := specSchemaOfStruct(user{})
	should.Equal(t, 2, len(schema.Properties))
	_, ok := schema.Properties[""]
	should.Equal(t, false, ok)
	_, ok = schema.Properties["Email"]
	should.Equal(t, true, ok)
}

func TestHttpDocsAddTag(t *testing.T) {
	docs := httpDocsInitialize(t)

	docs.AddTag("users", "User management")
	docs.AddTag("orders", "Order management")

	should.Equal(t, 2, len(docs.swagger.Tags))
	should.Equal(t, "users", docs.swagger.Tags[0].Name)
	should.Equal(t, "User management", docs.swagger.Tags[0].Description)
	should.Equal(t, "orders", docs.swagger.Tags[1].Name)
}

func TestHttpDocsSetBasePath(t *testing.T) {
	docs := httpDocsInitialize(t)

	docs.SetBasePath("/api/v1")

	should.Equal(t, "/api/v1", docs.BasePath)
	should.Equal(t, "/api/v1", docs.swagger.BasePath)
}

func TestHttpDocsRemoveTag(t *testing.T) {
	docs := httpDocsInitialize(t)

	docs.AddTag("users", "User management")
	docs.AddTag("orders", "Order management")

	should.Equal(t, true, docs.DeleteTag("users"))
	should.Equal(t, 1, len(docs.swagger.Tags))
	should.Equal(t, "orders", docs.swagger.Tags[0].Name)

	should.Equal(t, false, docs.DeleteTag("missing"))
	should.Equal(t, 1, len(docs.swagger.Tags))
}

func TestHttpDocsAddDefinition(t *testing.T) {
	docs := httpDocsInitialize(t)
	type user struct{ Email string }
	ref := docs.AddDefinition(user{})

	should.Equal(t, ref.Ref.String(), "#/definitions/user")

	type userT struct {
		Email string `json:"email"`
	}
	ref = docs.AddDefinition(userT{})
	should.Equal(t, ref.Ref.String(), "#/definitions/userT")
}

func TestHttpDocsHasDefinition(t *testing.T) {
	docs := httpDocsInitialize(t)

	type user struct{ Email string }
	should.Equal(t, false, docs.HasDefinition("user"))

	docs.AddDefinition(user{})
	should.Equal(t, true, docs.HasDefinition("user"))
	should.Equal(t, false, docs.HasDefinition("missing"))
}
