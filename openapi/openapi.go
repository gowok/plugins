package openapi

import (
	"log/slog"
	"net/http"
	"os"
	"reflect"
	"strings"

	"github.com/go-openapi/jsonreference"
	"github.com/go-openapi/spec"
	"github.com/gowok/gowok/maps"
	"github.com/gowok/gowok/some"
	"github.com/ngamux/ngamux"
	"gopkg.in/yaml.v3"
)

type httpDocs struct {
	swagger                           *spec.Swagger
	Title, Version, Host, Description string
	TermsOfService                    string  `json:"terms_of_service"`
	BasePath                          string  `json:"base_path"`
	Contact                           contact `json:"contact"`
	License                           license `json:"license"`
	Schemes, Consumes, Produces       []string
	Tags                              []spec.Tag
	SecurityDefinitions               map[string]securityDefinition `json:"security_definitions"`
}

func newHttpDocs(docs httpDocs) *httpDocs {
	swagger := spec.Swagger{
		VendorExtensible: spec.VendorExtensible{},
		SwaggerProps: spec.SwaggerProps{
			Swagger:  "2.0",
			Consumes: docs.Consumes,
			Produces: docs.Produces,
			Schemes:  docs.Schemes,
			Host:     docs.Host,
			Info: &spec.Info{
				InfoProps: spec.InfoProps{
					Version:        docs.Version,
					Title:          docs.Title,
					Description:    docs.Description,
					TermsOfService: docs.TermsOfService,
					Contact: &spec.ContactInfo{
						ContactInfoProps: spec.ContactInfoProps{
							Name:  docs.Contact.Name,
							URL:   docs.Contact.URL,
							Email: docs.Contact.Email,
						},
					},
					License: &spec.License{
						LicenseProps: spec.LicenseProps{
							Name: docs.License.Name,
							URL:  docs.License.URL,
						},
					},
				},
			},
			SecurityDefinitions: make(map[string]*spec.SecurityScheme),
			Paths: &spec.Paths{
				Paths: make(map[string]spec.PathItem),
			},
			Definitions: spec.Definitions{},
			Tags:        make([]spec.Tag, len(docs.Tags)),
		},
	}

	for k, v := range docs.SecurityDefinitions {
		swagger.SecurityDefinitions[k] = &spec.SecurityScheme{
			SecuritySchemeProps: spec.SecuritySchemeProps{
				Name: v.FieldName,
				In:   v.ValueSource,
				Type: v.Type,
			},
		}
	}

	copy(swagger.Tags, docs.Tags)
	docs.swagger = &swagger

	return &docs
}

func newHttpDocsFromYAMLFile(filePath string) *httpDocs {
	hd := newHttpDocs(httpDocs{})

	file, err := os.Open(filePath)
	if err != nil {
		slog.Warn("can not load configuration", "plugin", plugin, "error", err)
		return hd
	}
	defer file.Close()

	mm := map[string]any{}
	err = yaml.NewDecoder(file).Decode(&mm)
	if err != nil {
		return hd
	}

	swagger := spec.Swagger{}
	err = maps.ToStruct(mm, &swagger)
	if err != nil {
		return hd
	}

	hd.swagger = &swagger
	return hd
}

func (docs *httpDocs) Add(description string, callback func(*spec.Operation)) func(ngamux.Route) {
	operation := spec.NewOperation(description)
	operation.Description = description
	item := spec.PathItemProps{}
	return func(route ngamux.Route) {
		some.Of(callback).OrElse(func(*spec.Operation) {})(operation)

		if docs.swagger.Paths == nil {
			docs.swagger.Paths = &spec.Paths{Paths: map[string]spec.PathItem{}}
		}
		if itemFound, ok := docs.swagger.Paths.Paths[route.Path]; ok {
			item = itemFound.PathItemProps
		}

		switch route.Method {
		case http.MethodGet:
			item.Get = operation
		case http.MethodPost:
			item.Post = operation
		case http.MethodPut:
			item.Put = operation
		case http.MethodHead:
			item.Head = operation
		case http.MethodPatch:
			item.Patch = operation
		case http.MethodDelete:
			item.Delete = operation
		case http.MethodOptions:
			item.Options = operation
		}
		docs.swagger.Paths.Paths[route.Path] = spec.PathItem{
			PathItemProps: item,
		}
	}
}

func parseStructTag(tag string) []string {
	if tag == "" {
		return make([]string, 0)
	}
	return strings.Split(tag, ",")
}

func specSchemaOfReflectType(t reflect.Type) *spec.Schema {
	fieldSchema := &spec.Schema{}
	switch t.Kind() {
	case reflect.String:
		fieldSchema = spec.StringProperty()
	case reflect.Int64:
		fieldSchema = spec.Int64Property()
	case reflect.Int32:
		fieldSchema = spec.Int32Property()
	case reflect.Int16:
		fieldSchema = spec.Int16Property()
	case reflect.Int8:
		fieldSchema = spec.Int8Property()
	case reflect.Float64:
		fieldSchema = spec.Float64Property()
	case reflect.Float32:
		fieldSchema = spec.Float32Property()
	case reflect.Bool:
		fieldSchema = spec.BooleanProperty()
	default:
		fieldSchema = spec.StringProperty()
	}
	fieldSchema.AdditionalProperties = &spec.SchemaOrBool{Allows: false}
	return fieldSchema
}

func (docs *httpDocs) specSchemaOfStruct(v interface{}) *spec.Schema {
	t := reflect.TypeOf(v)
	schema := spec.MapProperty(nil).WithProperties(make(map[string]spec.Schema))
	schema.AdditionalProperties = &spec.SchemaOrBool{Allows: false}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		jsonTagParts := parseStructTag(field.Tag.Get("json"))
		jsonTag := ""
		if len(jsonTagParts) <= 0 {
			jsonTag = field.Name
		} else {
			jsonTag = jsonTagParts[0]
		}

		if field.Type.Kind() == reflect.Struct {
			nestedSchema := docs.specSchemaOfStruct(reflect.New(field.Type).Elem().Interface())
			schema.Properties[jsonTag] = *nestedSchema
			continue
		}

		fieldSchema := specSchemaOfReflectType(field.Type)

		example := field.Tag.Get("example")
		if example != "" {
			fieldSchema.Example = example
		}

		schema.Properties[jsonTag] = *fieldSchema
	}
	return schema
}

func (docs *httpDocs) AddDefinition(schema any) spec.Ref {
	t := reflect.TypeOf(schema)
	ss := &spec.Schema{}
	if t.Kind() == reflect.Struct {
		ss = docs.specSchemaOfStruct(schema)
	} else {
		ss = specSchemaOfReflectType(t)
		ss.Example = schema
	}
	docs.swagger.Definitions[t.Name()] = *ss
	return spec.Ref{Ref: jsonreference.MustCreateRef("#/definitions/" + t.Name())}
}

func (docs httpDocs) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	ngamux.Res(rw).JSON(docs.swagger)
}
