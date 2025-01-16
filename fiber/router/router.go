package router

import (
	"github.com/gofiber/fiber/v2"
)

func Group(path string, handlers ...fiber.Handler) fiber.Router {
	return App().Group(path, handlers...)
}

func HandleFunc(method, path string, handlerFunc ...fiber.Handler) {
	App().Add(method, path, handlerFunc...)
}

func Get(path string, handlerFunc ...fiber.Handler) {
	App().Get(path, handlerFunc...)
}

func Post(path string, handlerFunc ...fiber.Handler) {
	App().Get(path, handlerFunc...)
}

func Patch(path string, handlerFunc ...fiber.Handler) {
	App().Patch(path, handlerFunc...)
}

func Put(path string, handlerFunc ...fiber.Handler) {
	App().Put(path, handlerFunc...)
}

func Delete(path string, handlerFunc ...fiber.Handler) {
	App().Delete(path, handlerFunc...)
}

func Use(middleware ...any) {
	App().Use(middleware...)
}
