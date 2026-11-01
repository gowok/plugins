package validator

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/locales/en"
	"github.com/go-playground/mold/v4"
	"github.com/go-playground/mold/v4/modifiers"
	ut "github.com/go-playground/universal-translator"
	govalidator "github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
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
	trans  ut.Translator
}

func WithDBUnique(fn func(table, column, value string) bool) func(*config) {
	return func(c *config) {
		c.unique = fn
	}
}

func WithTranslator(trans ut.Translator) func(*config) {
	return func(c *config) {
		c.trans = trans
	}
}

func Configure(opts ...func(*config)) func() {
	config := &config{}
	for _, opt := range opts {
		opt(config)
	}

	return func() {
		v := *vv()

		if config.unique != nil {
			RegisterValidation("db-unique", registerValidationUnique(config.unique))
		}

		if config.trans != nil {
			v.trans = config.trans
		}
	}
}

var Modifiers = singleton.New(func() mold.Transformer {
	return *modifiers.New()
})

func registerValidationUnique(fn func(string, string, string) bool) func(field govalidator.FieldLevel) bool {
	return func(field govalidator.FieldLevel) bool {
		param := strings.TrimSpace(field.Param())
		if param == "" {
			return true
		}

		fp := strings.SplitN(param, ".", 2)
		table := fp[0]
		column := fp[0]
		if len(fp) > 1 {
			column = fp[1]
		}

		value := field.Field().String()
		if field.Field().Kind() != reflect.String {
			value = fmt.Sprint(field.Field().Interface())
		}
		return fn(table, column, value)
	}
}
