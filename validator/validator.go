package validator

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	govalidator "github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/gowok/gowok/singleton"
	"github.com/gowok/gowok/some"
	"github.com/ngamux/ngamux"
)

type validator struct {
	validate *govalidator.Validate
	trans    ut.Translator
}

type ValidationError struct {
	errors       govalidator.ValidationErrors
	translations govalidator.ValidationErrorsTranslations

	errorMessage string
	errorJSON    map[string]string
}

func newValidationError(errs govalidator.ValidationErrors, translations ut.Translator, trans map[string]string) ValidationError {
	validationErrorMessages := errs.Translate(translations)
	errorMessage := ""
	errorJSON := map[string]string{}
	for _, err := range errs {
		namespace := err.Namespace()
		errorJSON[namespace] = validationErrorMessages[namespace]

		customTag := namespace + "." + err.Tag()
		if _, ok := trans[customTag]; ok {
			t, e := translations.T(customTag, err.Field())
			if e == nil {
				errorJSON[namespace] = t
			}
			continue
		}

		customTag = "*." + err.Tag()
		if _, ok := trans[err.Tag()]; ok {
			t, e := translations.T(customTag, err.Field())
			if e == nil {
				errorJSON[namespace] = t
			}
		}
	}

	return ValidationError{errs, validationErrorMessages, errorMessage, errorJSON}
}

func (err ValidationError) Error() string {
	errorMessage := ""
	for field, msg := range err.errorJSON {
		errorMessage += field + ": " + msg + "; "
	}
	return errorMessage
}

func (err ValidationError) MarshalJSON() ([]byte, error) {
	return json.Marshal(err.errorJSON)
}

var vv = singleton.New(func() *validator {
	validate := govalidator.New()
	v := &validator{
		validate: validate,
		trans:    nil,
	}

	en := en.New()
	uni := ut.New(en, en)
	trans, ok := uni.GetTranslator("en")
	if !ok {
		return nil
	}

	err := en_translations.RegisterDefaultTranslations(v.validate, trans)
	if err != nil {
		return nil
	}
	v.trans = trans
	return v
})

func SetTranslator(trans ut.Translator, localeFunc func(*govalidator.Validate, ut.Translator) error) {
	v := *vv()
	v.trans = trans
	localeFunc(v.validate, trans)
	vv(v)
}

func (v *validator) registerTranslationTag(tag, message string, override bool) error {
	err := v.validate.RegisterTranslation(tag, v.trans, func(ut ut.Translator) error {
		return ut.Add(tag, message, override)
	}, func(ut ut.Translator, fe govalidator.FieldError) string {
		t, _ := ut.T(tag, fe.Field())
		return t
	})
	return err
}

func ValidateStruct(input any, trans map[string]string) ValidationError {
	v := *vv()
	for tag, message := range trans {
		if !strings.Contains(tag, ".") {
			tag = "*." + tag
		}
		err := v.registerTranslationTag(tag, message, false)
		if err != nil {
			return ValidationError{
				errorJSON: map[string]string{"*": err.Error()},
			}
		}
	}

	err := v.validate.Struct(input)

	result := ValidationError{}
	some.Of(err).IfPresent(func(err error) {
		switch e := err.(type) {
		case govalidator.ValidationErrors:
			result = newValidationError(e, v.trans, trans)
		}
	})

	return result
}

func ValidateJSON[T any](r *http.Request, schema T, trans map[string]string) (*T, *ValidationError) {
	err := ngamux.Req(r).JSON(&schema)
	if err != nil {
		return new(T), &ValidationError{
			errorMessage: err.Error(),
			errorJSON: map[string]string{
				"*": err.Error(),
			},
		}
	}

	errs := ValidateStruct(schema, trans)
	if len(errs.Error()) > 0 {
		return new(T), &errs
	}

	return &schema, nil
}
