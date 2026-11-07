package router

import (
	"github.com/gofiber/fiber/v2"
)

func Group(path string, handlers ...fiber.Handler) fiber.Router {
	app := App()
	if app == nil {
		return nil
	}
	return app.Group(path, handlers...)
}

func HandleFunc(method, path string, handlerFunc ...fiber.Handler) {
	if app := App(); app != nil {
		app.Add(method, path, handlerFunc...)
	}
}

func Get(path string, handlerFunc ...fiber.Handler) {
	if app := App(); app != nil {
		app.Get(path, handlerFunc...)
	}
}

func Post(path string, handlerFunc ...fiber.Handler) {
	if app := App(); app != nil {
		app.Post(path, handlerFunc...)
	}
}

func Patch(path string, handlerFunc ...fiber.Handler) {
	if app := App(); app != nil {
		app.Patch(path, handlerFunc...)
	}
}

func Put(path string, handlerFunc ...fiber.Handler) {
	if app := App(); app != nil {
		app.Put(path, handlerFunc...)
	}
}

func Delete(path string, handlerFunc ...fiber.Handler) {
	if app := App(); app != nil {
		app.Delete(path, handlerFunc...)
	}
}

func Use(middleware ...any) {
	if app := App(); app != nil {
		app.Use(middleware...)
	}
}

func Routes() []fiber.Route {
	app := App()
	if app == nil {
		return []fiber.Route{}
	}
	return app.GetRoutes()
}
