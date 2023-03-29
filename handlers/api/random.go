package api

import (
	"net/http"

	"github.com/apex/log"
	"github.com/labstack/echo/v4"
	"github.com/sol-armada/admin/stores"
	"github.com/sol-armada/admin/user"
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

	users := []user.User{}
	cur, err := stores.Storage.GetRandomUsers(req.Max, req.RankLimit)
	if err != nil {
		logger.WithError(err).Error("getting random users")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}
	if err := cur.All(c.Request().Context(), &users); err != nil {
		logger.WithError(err).Error("getting random users from collection")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, usersResponse{Users: users})
}

func (r *randomUsersRequest) bind(c echo.Context) error {
	if err := c.Bind(r); err != nil {
		return err
	}
	return nil
}
