package router

import (
	"net/http"

	"github.com/labstack/echo/v4"
	h "github.com/sol-armada/sol-bot/health"
)

func health(c echo.Context) error {
	// check dynamodb connection
	if !h.IsHealthy() {
		return c.JSON(http.StatusInternalServerError, "{\"status\": \"error\", \"message\": \"internal server error\"}")
	}

	return c.String(http.StatusOK, "OK")
}
