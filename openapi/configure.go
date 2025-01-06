package openapi

import (
	"log/slog"

	"github.com/gowok/gowok"
	"github.com/gowok/gowok/maps"
)

var docs = gowok.Singleton(func() *httpDocs {
	return newHttpDocs(httpDocs{})
})

func Configure(project *gowok.Project) {
	confMap, ok := project.ConfigMap["openapi"]
	if !ok {
		slog.Warn("no configuration", "plugin", "openapi")
		return
	}

	hd := httpDocs{}
	err := maps.MapToStruct(confMap, &hd)
	if err != nil {
		slog.Warn("openapi", "error", err)
		return
	}

	docs(newHttpDocs(hd))
}

func Docs() *httpDocs {
	d := docs()
	return *d
}
