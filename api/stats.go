package api

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/nethserver/nethsecurity-monitoring/stats"
)

type StatsApi struct {
	saver stats.Saver
}

func NewStatsApi(saver stats.Saver) *StatsApi {
	return &StatsApi{saver: saver}
}

func (s *StatsApi) Setup(app *fiber.App) {
	app.Post("/stats", func(c *fiber.Ctx) error {
		var payload stats.Payload
		if err := c.BodyParser(&payload); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid stats payload: " + err.Error(),
			})
		}

		if err := s.saver.Save(context.Background(), payload); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to persist stats payload: " + err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).Send(nil)
	})
}
