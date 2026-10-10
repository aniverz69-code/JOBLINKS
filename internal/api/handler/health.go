package handler

import (
	"github.com/gofiber/fiber/v2"
)

// Health reports service status and database availability.
func (a *API) Health(c *fiber.Ctx) error {
	if err := a.pool.Ping(c.Context()); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status":   "degraded",
			"database": "down",
		})
	}
	return c.JSON(fiber.Map{
		"status":   "ok",
		"database": "up",
	})
}
