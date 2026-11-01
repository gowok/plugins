package translator

import (
	"testing"

	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/fr"
	"github.com/go-playground/locales/id"
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

func TestAddDictionaryUnknownLocaleDoesNotLeakIntoFallback(t *testing.T) {
	Configure(en.New)()

	AddDictionary("xx", map[string]string{"leak": "leaked"})

	if _, err := Get("en").T("leak"); err == nil {
		t.Fatal("message for unknown locale leaked into fallback dictionary")
	}
}
