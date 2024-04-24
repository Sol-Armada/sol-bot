package api

import (
	"net/http"

	"github.com/apex/log"
	"github.com/labstack/echo/v4"
	"github.com/sol-armada/sol-bot/users"
)

type randomUsersRequest struct {
	Max       int `query:"max"`
	RankLimit int `query:"rank_limit"`
}

func GetRandomUsers(c echo.Context) error {
	logger := log.WithFields(log.Fields{
		"endpoint": "GetRandomUsers",
	})
	logger.Debug("getting random users")

	req := &randomUsersRequest{}
	if err := req.bind(c); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	randomUsers, err := users.GetRandom(req.Max, req.RankLimit)
	if err != nil {
		logger.WithError(err).Error("getting random users")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, usersResponse{Users: randomUsers})
}

func (r *randomUsersRequest) bind(c echo.Context) error {
	if err := c.Bind(r); err != nil {
		return err
	}
	return nil
}
