package api

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/nethserver/nethsecurity-monitoring/flows"
)

var validate = validator.New()

type FlowsResponse struct {
	Data        []flows.FlowEvent `json:"flows"`
	PerPage     int               `json:"per_page"`
	Total       int               `json:"total"`
	CurrentPage int               `json:"current_page"`
	LastPage    int               `json:"last_page"`
}

type queryParams struct {
	Page    int          `query:"page"     validate:"min=1"`
	PerPage int          `query:"per_page" validate:"min=1,max=100"`
	SortBy  flows.SortBy `query:"sort_by"  validate:"oneof=duration last_seen_at download_rate upload_rate"`
	Desc    bool         `query:"desc"`
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

		var params queryParams
		if err := c.QueryParser(&params); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid query parameters: " + err.Error(),
			})
		}

		// Apply defaults for absent query params (Fiber v2 cannot infer defaults from struct tags).
		// Use c.Query to distinguish an absent param (empty string) from an explicit zero value.
		if c.Query("page") == "" {
			params.Page = 1
		}
		if c.Query("per_page") == "" {
			params.PerPage = 10
		}
		if c.Query("sort_by") == "" {
			params.SortBy = flows.SortByDownloadRate
		}

		if err := validate.Struct(&params); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid query parameters: " + err.Error(),
			})
		}

		total := len(eventsSlice)

		// Sort the events
		eventsSlice = flows.SortEvents(eventsSlice, flows.SortBy(params.SortBy), params.Desc)

		// Paginate the events
		start := (params.Page - 1) * params.PerPage
		end := start + params.PerPage
		if start > total {
			start = total
		}
		if end > total {
			end = total
		}
		eventsSlice = eventsSlice[start:end]

		// Response
		return c.JSON(FlowsResponse{
			Data:        eventsSlice,
			PerPage:     params.PerPage,
			Total:       total,
			CurrentPage: params.Page,
			LastPage:    (total + params.PerPage - 1) / params.PerPage,
		})
	})
}
