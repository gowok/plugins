package translator

import (
	"github.com/go-playground/locales"
	ut "github.com/go-playground/universal-translator"
)

type Map map[string]func() locales.Translator

var plugin = "translator"
var defaults string
var translator *ut.UniversalTranslator

func Configure(
	fallback func() locales.Translator,
	translators ...func() locales.Translator,
) func() {
	return func() {
		ff := fallback()
		_translators := []locales.Translator{ff}
		for _, t := range translators {
			_translators = append(_translators, t())
		}
		universalTraslator := ut.New(ff, _translators...)
		translator = universalTraslator
	}
}

func Get(name ...string) ut.Translator {
	if len(name) > 0 {
		tt, ok := translator.GetTranslator(name[0])
		if !ok {
			return nil
		}
		return tt
	}

	tt, _ := translator.GetTranslator(defaults)
	return tt
}

func SetDefault(name string) {
	_, ok := translator.GetTranslator(name)
	if !ok {
		return
	}
	defaults = name
}
