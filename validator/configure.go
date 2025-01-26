package validator

import (
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	govalidator "github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/gowok/gowok"
	"github.com/gowok/gowok/singleton"
)

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

type config struct {
	unique func(table, column, value string) bool
}

func WithDBUnique(fn func(table, column, value string) bool) func(*config) {
	return func(c *config) {
		c.unique = fn
	}
}

func Configure(opts ...func(*config)) func(*gowok.Project) {
	config := &config{}
	for _, opt := range opts {
		opt(config)
	}

	return func(project *gowok.Project) {
		vv()

		if config.unique != nil {
			RegisterValidation("db-unique", registerValidationUnique(config.unique))
		}
	}
}

func registerValidationUnique(fn func(string, string, string) bool) func(field govalidator.FieldLevel) bool {
	return func(field govalidator.FieldLevel) bool {
		fp := strings.Split(field.Param(), ".")
		table := ""
		column := ""
		if len(fp) <= 0 {
			return true
		}

		table = fp[0]
		if len(fp) == 1 {
			column = fp[0]
		} else {
			column = fp[1]
		}
		return fn(table, column, field.Field().String())
	}
}
