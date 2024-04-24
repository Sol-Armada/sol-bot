package handlers

import (
	"net/http"

	"github.com/apex/log"
	"github.com/labstack/echo/v4"
	"github.com/sol-armada/sol-bot/web"
)

func IndexHander(c echo.Context) error {
	rawFile, _ := web.StaticFiles.ReadFile("dist/index.html")
	if err := c.Render(http.StatusOK, "index", rawFile); err != nil {
		log.WithError(err).Error("render error")
		return err
	}
	return nil
}
