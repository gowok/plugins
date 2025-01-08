package openapi

import (
	"log/slog"

	"github.com/gowok/gowok"
	"github.com/gowok/gowok/maps"
)

var plugin = "openapi"

var docs = gowok.Singleton(func() *httpDocs {
	hd := httpDocs{}
	confMap, ok := gowok.Get().ConfigMap["openapi"]
	if !ok {
		slog.Warn("no configuration", "plugin", plugin)
		return newHttpDocs(hd)
	}

	err := maps.MapToStruct(confMap, &hd)
	if err != nil {
		slog.Warn("openapi", "error", err)
		return newHttpDocs(hd)
	}

	return newHttpDocs(hd)
})

func Docs() *httpDocs {
	d := docs()
	return *d
}
