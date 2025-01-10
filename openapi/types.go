package openapi

type contact struct {
	Name  string
	URL   string
	Email string
}

type license struct {
	Name string
	URL  string
}

type securityDefinition struct {
	FieldName   string `json:"field_name"`
	ValueSource string `json:"value_source"`

	// Type value is one of:
	// - basic
	// - apiKey
	// - oauth2
	// - implicit
	// - password
	// - application
	// - accessCode
	Type string `json:"type"`
}
