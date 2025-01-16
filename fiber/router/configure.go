package router

import (
	"log/slog"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gowok/gowok"
	"github.com/gowok/gowok/maps"
	"github.com/gowok/gowok/singleton"
)

var plugin = "fiber"
var config_ = singleton.New(func() config {
	project := gowok.Get()
	var c config
	err := maps.ToStruct(maps.Get[map[string]any](project.ConfigMap, "fiber"), &c)
	if err != nil {
		slog.Warn("failed to load configuration", "plugin", plugin, "error", err)
		return c
	}

	return c
})
var fiber_ = singleton.New(func() *fiber.App {
	config := config_()
	if !config.Enabled {
		return nil
	}

	project := gowok.Get()
	app := fiber.New()
	project.Hooks.OnStarting(func() {
		go func() {
			err := app.Listen(config.Host)
			if err != nil {
				slog.Warn("failed to listen", "plugin", plugin, "error", err)
				return
			}
		}()
	})

	project.Hooks.OnStopped(func() {
		err := app.ShutdownWithTimeout(10 * time.Second)
		if err != nil {
			slog.Warn("problem happened on shutdown", "plugin", plugin, "error", err)
			return
		}
	})
	return app
})

var appLogOnce sync.Once

func App() *fiber.App {
	a := fiber_()
	if a == nil {
		appLogOnce.Do(func() {
			slog.Warn("failed to get", "plugin", plugin, "error", "disabled")
		})
		return nil
	} else if *a == nil {
		appLogOnce.Do(func() {
			slog.Warn("failed to get", "plugin", plugin, "error", "disabled")
		})
		return nil
	}

	return *a
}

func Configure(project *gowok.Project) {
	_ = App()
}

func Config() config {
	return *config_()
}
