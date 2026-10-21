package validator

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
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
