package router

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestRouterHelpersFollowPluginState(t *testing.T) {
	// Disabled: an unconfigured plugin must not panic on route registration.
	if Enabled() {
		t.Fatal("plugin should be disabled without configuration")
	}
	if app := App(); app != nil {
		t.Fatal("App() should be nil when the plugin is disabled")
	}

	noop := func(*fiber.Ctx) error { return nil }
	Get("/ping", noop)
	Post("/ping", noop)
	Put("/ping", noop)
	Patch("/ping", noop)
	Delete("/ping", noop)
	HandleFunc(http.MethodHead, "/ping", noop)
	Use(noop)
	if group := Group("/api"); group != nil {
		t.Fatal("Group should be nil when the plugin is disabled")
	}

	// Enabled: override the lazy singleton and check the routes really land.
	app := fiber.New()
	fiber_(app)
	t.Cleanup(func() { fiber_(nil) })

	if !Enabled() {
		t.Fatal("plugin should be enabled once an application is available")
	}

	Get("/ping", func(c *fiber.Ctx) error { return c.SendString("pong") })
	Post("/echo", func(c *fiber.Ctx) error { return c.SendString("echo") })
	Put("/echo", func(c *fiber.Ctx) error { return c.SendString("echo") })
	Patch("/echo", func(c *fiber.Ctx) error { return c.SendString("echo") })
	Delete("/echo", func(c *fiber.Ctx) error { return c.SendString("echo") })
	HandleFunc(http.MethodHead, "/probe", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})
	Group("/api").Get("/health", func(c *fiber.Ctx) error { return c.SendString("ok") })

	cases := []struct {
		method string
		path   string
		want   string
	}{
		{http.MethodGet, "/ping", "pong"},
		{http.MethodPost, "/echo", "echo"},
		{http.MethodPut, "/echo", "echo"},
		{http.MethodPatch, "/echo", "echo"},
		{http.MethodDelete, "/echo", "echo"},
		{http.MethodGet, "/api/health", "ok"},
	}

	for _, tc := range cases {
		res, err := app.Test(httptest.NewRequest(tc.method, tc.path, nil), -1)
		if err != nil {
			t.Fatalf("%s %s: %v", tc.method, tc.path, err)
		}

		body, err := io.ReadAll(res.Body)
		_ = res.Body.Close()
		if err != nil {
			t.Fatalf("%s %s: failed to read body: %v", tc.method, tc.path, err)
		}

		if res.StatusCode != http.StatusOK {
			t.Fatalf("%s %s: status = %d, want %d", tc.method, tc.path, res.StatusCode, http.StatusOK)
		}
		if string(body) != tc.want {
			t.Fatalf("%s %s: body = %q, want %q", tc.method, tc.path, body, tc.want)
		}
	}
}
