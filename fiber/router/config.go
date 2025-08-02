package router

import "github.com/gofiber/fiber/v2"

type config struct {
	fiber.Config
	Enabled bool   `json:"enabled"`
	Host    string `json:"host"`
}
