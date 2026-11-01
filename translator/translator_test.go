package translator

import (
	"testing"

	"github.com/go-playground/locales"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/fr"
	"github.com/go-playground/locales/id"
	"github.com/go-playground/locales/ru"
)

func TestConfigureRegistersAllTranslators(t *testing.T) {
	Configure(en.New, id.New, fr.New)()

	for _, locale := range []string{"en", "id", "fr"} {
		tr := Get(locale)
		if tr == nil {
			t.Fatalf("no translator for %q", locale)
		}
		if got := tr.Locale(); got != locale {
			t.Fatalf("Get(%q) returned translator for %q", locale, got)
		}
	}
}

func TestGetUnknownLocaleReturnsNil(t *testing.T) {
	Configure(en.New, id.New)()

	if tr := Get("fr"); tr != nil {
		t.Fatalf("Get(\"fr\") returned a translator for %q, want nil", tr.Locale())
	}

	if tr := Get("id"); tr == nil || tr.Locale() != "id" {
		t.Fatal("Get(\"id\") should return the Indonesian translator")
	}

	if tr := Get(); tr == nil || tr.Locale() != "en" {
		t.Fatal("Get() should return the fallback translator for the default locale")
	}
}

func TestAddDictionaryUnknownLocaleDoesNotLeakIntoFallback(t *testing.T) {
	Configure(en.New)()

	AddDictionary("xx", map[string]string{"leak": "leaked"})

	if _, err := Get("en").T("leak"); err == nil {
		t.Fatal("message for unknown locale leaked into fallback dictionary")
	}
}

func TestRForwardsRangeArgumentsInOrder(t *testing.T) {
	Configure(ru.New)()

	tr := Get("ru")
	for rule, text := range map[locales.PluralRule]string{
		locales.PluralRuleOne:   "{0}-{1} ONE",
		locales.PluralRuleFew:   "{0}-{1} FEW",
		locales.PluralRuleMany:  "{0}-{1} MANY",
		locales.PluralRuleOther: "{0}-{1} OTHER",
	} {
		if err := tr.AddRange("k", text, rule, true); err != nil {
			t.Fatal(err)
		}
	}

	got, err := R("k", 1, 0, 1, 0, "a", "b")
	if err != nil {
		t.Fatal(err)
	}
	if got != "a-b ONE" {
		t.Fatalf("R() = %q, want %q", got, "a-b ONE")
	}
}
