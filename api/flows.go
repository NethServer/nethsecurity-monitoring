package api

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/nethserver/nethsecurity-monitoring/flows"
)

type FlowsResponse struct {
	Data []flows.FlowEvent `json:"flows"`
}

type FlowApi struct {
	accessor flows.FlowAccessor
	ingestor flows.FlowIngestor
}

func NewFlowApi(accessor flows.FlowAccessor, ingestor flows.FlowIngestor) *FlowApi {
	return &FlowApi{accessor: accessor, ingestor: ingestor}
}

func (f *FlowApi) Setup(app *fiber.App) {
	app.Get("/flows", func(c *fiber.Ctx) error {
		eventsMap := f.accessor.GetEvents()
		eventsSlice := make([]flows.FlowEvent, 0, len(eventsMap))
		for _, ev := range eventsMap {
			eventsSlice = append(eventsSlice, ev)
		}

		// Response
		return c.JSON(FlowsResponse{
			Data: eventsSlice,
		})
	})

	app.Post("/flows", func(c *fiber.Ctx) error {
		var event flows.FlowEvent
		if err := c.BodyParser(&event); err != nil {
			// Check if error is due to unsupported flow type
			if errors.Is(err, flows.ErrUnsupportedFlowType) {
				slog.Debug("Ignoring flow event with unsupported type", "error", err)
				return c.Status(fiber.StatusOK).Send(nil)
			}
			// Malformed JSON
			slog.Error("Invalid flow event payload", "error", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid flow payload: " + err.Error(),
			})
		}

		f.ingestor.Process(event)
		return c.Status(fiber.StatusOK).Send(nil)
	})
}
