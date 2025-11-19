package openapi

import (
	"log/slog"

	"github.com/gowok/fp/maps"
	"github.com/gowok/gowok"
	"github.com/gowok/gowok/singleton"
)

var plugin = "openapi"

var docs = singleton.New(func() *httpDocs {
	openapiFile := maps.Get[string](gowok.Config.Map(), "openapi")
	if openapiFile != "" {
		return newHttpDocsFromYAMLFile(openapiFile)
	}

	hd := httpDocs{}
	err := maps.ToStruct(maps.Get[map[string]any](gowok.Config.Map(), "openapi"), &hd)
	if err == nil {
		return newHttpDocs(hd)
	}

	slog.Warn("can not load configuration", "plugin", plugin, "error", err)
	return newHttpDocs(hd)
})

func Docs() *httpDocs {
	d := docs()
	return *d
}
