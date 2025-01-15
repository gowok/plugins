package openapi

import (
	"log/slog"

	"github.com/gowok/gowok"
	"github.com/gowok/gowok/maps"
	"github.com/gowok/gowok/singleton"
)

var plugin = "openapi"

var docs = singleton.New(func() *httpDocs {
	hd := httpDocs{}
	err := maps.ToStruct(maps.Get[map[string]any](gowok.Get().ConfigMap, "openapi"), &hd)
	if err != nil {
		slog.Warn("can not load configuration", "plugin", plugin, "error", err)
		return newHttpDocs(hd)
	}

	return newHttpDocs(hd)
})

func Docs() *httpDocs {
	d := docs()
	return *d
}
