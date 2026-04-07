package api

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/nethserver/nethsecurity-monitoring/stats"
)

type StatsApi struct {
	receiver *stats.Receiver
}

func NewStatsApi(receiver *stats.Receiver) *StatsApi {
	return &StatsApi{receiver: receiver}
}

func (s *StatsApi) Setup(app *fiber.App) {
	app.Post("/stats", func(c *fiber.Ctx) error {
		var payload stats.Payload
		if err := c.BodyParser(&payload); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid stats payload: " + err.Error(),
			})
		}

		if err := s.receiver.Handle(context.Background(), payload); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to persist stats payload: " + err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).Send(nil)
	})
}
