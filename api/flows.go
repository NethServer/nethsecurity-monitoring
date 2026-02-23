package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nethserver/nethsecurity-monitoring/flows"
)

type FlowsResponse struct {
	Data []flows.FlowEvent `json:"flows"`
}

type FlowApi struct {
	accessor flows.FlowAccessor
}

func NewFlowApi(accessor flows.FlowAccessor) *FlowApi {
	return &FlowApi{accessor: accessor}
}

func (f *FlowApi) Setup(app *fiber.App) {
	app.Get("flows", func(c *fiber.Ctx) error {
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
}
