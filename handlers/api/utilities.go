package api

import (
	"net/http"

	"github.com/apex/log"
	"github.com/labstack/echo/v4"
	"github.com/rs/xid"
)

func GenerateXID(c echo.Context) error {
	logger := log.WithFields(log.Fields{
		"endpoint": "GenerateXID",
	})
	logger.Debug("generating a new xid")
	
	return c.HTML(http.StatusOK, xid.New().String())
}
