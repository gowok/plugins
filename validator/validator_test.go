package validator

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
)

type testUser struct {
	Name  string `json:"name" mod:"trim,lcase" validate:"required,eq=john"`
	Email string `json:"email" validate:"required,email"`
}

func TestValidateStruct(t *testing.T) {
	tests := []struct {
		name  string
		input any
		opts  []ValidateStructOptFn
	}{
		{"no options", testUser{Name: "john", Email: "john@mail.com"}, nil},
		{
			"with translations",
			testUser{Name: "john", Email: "john@mail.com"},
			[]ValidateStructOptFn{WithTranslations(map[string]string{"required": "{0} is required"})},
		},
		{
			"with per field translations",
			testUser{Name: "john", Email: "john@mail.com"},
			[]ValidateStructOptFn{WithTranslations(map[string]string{"testUser.Email.email": "bad email"})},
		},
		{
			"with modify",
			&testUser{Name: "  JOHN  ", Email: "john@mail.com"},
			[]ValidateStructOptFn{WithModify()},
		},
		{
			"with modify and translations",
			&testUser{Name: "  JOHN  ", Email: "john@mail.com"},
			[]ValidateStructOptFn{WithModify(), WithTranslations(map[string]string{"eq": "{0} must be john"})},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateStruct(test.input, test.opts...)
			if len(err.Error()) > 0 {
				t.Fatalf("expected no error, got %q", err.Error())
			}
		})
	}
}

func TestValidateStruct_repeatedTranslations(t *testing.T) {
	input := testUser{Email: "john@mail.com"}
	opts := WithTranslations(map[string]string{"testUser.Name.required": "{0} must be filled"})

	for i := 0; i < 2; i++ {
		err := ValidateStruct(input, opts)
		if msg := err.Error(); !strings.Contains(msg, "must be filled") || strings.Contains(msg, "conflicting") {
			t.Fatalf("attempt %d: expected custom translation, got %q", i+1, msg)
		}
	}
}

func TestValidateStruct_nilOption(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("ValidateStruct panicked on nil option: %v", r)
		}
	}()

	err := ValidateStruct(testUser{Name: "john", Email: "john@mail.com"}, nil)
	if len(err.Error()) > 0 {
		t.Fatalf("expected no error, got %q", err.Error())
	}
}

func TestValidateStruct_modifyMutatesInput(t *testing.T) {
	input := &testUser{Name: "  JOHN  ", Email: "john@mail.com"}
	err := ValidateStruct(input, WithModify())
	if len(err.Error()) > 0 {
		t.Fatalf("expected no error, got %q", err.Error())
	}
	if input.Name != "john" {
		t.Fatalf("expected Name modified to %q, got %q", "john", input.Name)
	}
}

func TestValidateJSON(t *testing.T) {
	body := `{"name":"john","email":"john@mail.com"}`
	tests := []struct {
		name string
		opts []ValidateStructOptFn
	}{
		{"no options", nil},
		{"with translations", []ValidateStructOptFn{WithTranslations(map[string]string{"email": "{0} bad"})}},
		{"with modify", []ValidateStructOptFn{WithModify()}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", strings.NewReader(body))
			r.Header.Set("Content-Type", "application/json")

			got, err := ValidateJSON(r, testUser{}, test.opts...)
			if err != nil {
				t.Fatalf("expected no error, got %q", err.Error())
			}
			if got.Name != "john" || got.Email != "john@mail.com" {
				t.Fatalf("unexpected result %+v", got)
			}
		})
	}
}

func TestWithTranslations(t *testing.T) {
	trans := map[string]string{"required": "err"}
	opts := ValidateStructOpts{}
	WithTranslations(trans)(&opts)
	if opts.trans["required"] != "err" {
		t.Fatalf("expected translations set, got %v", opts.trans)
	}
}

func TestWithModify(t *testing.T) {
	opts := ValidateStructOpts{}
	WithModify()(&opts)
	if !opts.modify {
		t.Fatal("expected modify true")
	}
}

func TestModifiers(t *testing.T) {
	input := &testUser{Name: "  JOHN  "}
	if err := Modifiers().Struct(context.Background(), input); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if input.Name != "john" {
		t.Fatalf("expected %q, got %q", "john", input.Name)
	}
}

func TestConfigure_appliesTranslator(t *testing.T) {
	orig := *vv()
	defer func() { vv(orig) }()

	other, ok := ut.New(en.New(), en.New()).GetTranslator("en")
	if !ok {
		t.Fatal("failed to create translator")
	}

	Configure(WithTranslator(other))()

	if got := (*vv()).trans; got != other {
		t.Fatal("expected Configure to apply the configured translator, but it was discarded")
	}
}

func TestValidationError_nonTranslatorMessage(t *testing.T) {
	err := ValidateStruct(42, WithModify())
	if err.Error() == "" {
		t.Fatal("expected modifier error to surface via Error(), got empty message")
	}
}

func TestRegisterValidationUnique_emptyParam(t *testing.T) {
	RegisterValidation("db-unique-empty", registerValidationUnique(func(table, column, value string) bool {
		t.Fatalf("expected no callback without a parameter, got (%q, %q, %q)", table, column, value)
		return false
	}))

	type input struct {
		Name string `validate:"db-unique-empty"`
	}

	if err := ValidateStruct(input{Name: "john"}); len(err.Error()) > 0 {
		t.Fatalf("expected no error, got %q", err.Error())
	}
}

func TestRegisterValidationUnique_value(t *testing.T) {
	var gotTable, gotColumn, gotValue string
	RegisterValidation("db-unique-value", registerValidationUnique(func(table, column, value string) bool {
		gotTable, gotColumn, gotValue = table, column, value
		return true
	}))

	type input struct {
		ID int `validate:"db-unique-value=users.id"`
	}

	if err := ValidateStruct(input{ID: 42}); len(err.Error()) > 0 {
		t.Fatalf("expected no error, got %q", err.Error())
	}
	if gotTable != "users" || gotColumn != "id" || gotValue != "42" {
		t.Fatalf("unexpected lookup (%q, %q, %q)", gotTable, gotColumn, gotValue)
	}
}

func TestValidateJSON_modifierError(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader("42"))
	r.Header.Set("Content-Type", "application/json")

	if _, err := ValidateJSON(r, 0, WithModify()); err == nil {
		t.Fatal("expected modifier error to be reported, got nil")
	}
}
