package translator

import (
	"github.com/go-playground/locales"
	ut "github.com/go-playground/universal-translator"
	"github.com/gowok/gowok"
)

type Map map[string]func() locales.Translator

var plugin = "translator"
var defaults string
var translator *ut.UniversalTranslator

func Configure(
	fallback func() locales.Translator,
	translators ...func() locales.Translator,
) func(*gowok.Project) {
	return func(project *gowok.Project) {
		ff := fallback()
		_translators := []locales.Translator{ff}
		for i, t := range translators {
			_translators[i] = t()
		}
		universalTraslator := ut.New(ff, _translators...)
		translator = universalTraslator
	}
}

func Get(name ...string) ut.Translator {
	n := defaults
	if len(name) > 0 {
		n = name[0]
	}

	tt, _ := translator.GetTranslator(n)
	return tt
}

func SetDefault(name string) {
	_, ok := translator.GetTranslator(name)
	if !ok {
		return
	}
	defaults = name
}
