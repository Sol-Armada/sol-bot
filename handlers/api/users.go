package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"

	"github.com/apex/log"
	"github.com/labstack/echo/v4"
	"github.com/sol-armada/admin/ranks"
	"github.com/sol-armada/admin/request"
	"github.com/sol-armada/admin/stores"
	"github.com/sol-armada/admin/user"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type getUsersRequest struct {
	Rank string `json:"rank"`
}

func (r *getUsersRequest) bind(c echo.Context) error {
	if err := c.Bind(r); err != nil {
		return err
	}
	return nil
}

type usersResponse struct {
	Users []user.User `json:"users"`
}

func GetUsers(c echo.Context) error {
	logger := log.WithFields(log.Fields{
		"endpoint": "GetUsers",
	})
	logger.Debug("getting users")

	req := &getUsersRequest{}
	if err := req.bind(c); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	users := []user.User{}
	cur, err := stores.Storage.GetUsers(bson.D{})
	if err != nil {
		if !errors.Is(err, mongo.ErrNilDocument) {
			logger.WithError(err).Error("getting users")
			return c.JSON(http.StatusInternalServerError, "internal server error")
		}

		goto RETURN
	}
	if err := cur.All(c.Request().Context(), &users); err != nil {
		logger.WithError(err).Error("getting users from collection")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}
RETURN:
	return c.JSON(http.StatusOK, usersResponse{Users: users})
}

type getUserResponse struct {
	User *user.User `json:"user"`
}

func GetUser(c echo.Context) error {
	logger := log.WithFields(log.Fields{
		"endpoint": "GetUser",
	})

	storedUser := &user.User{}
	if err := stores.Storage.GetUser(c.Param("id")).Decode(&storedUser); err != nil {
		logger.WithError(err).Error("getting user")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, getUserResponse{
		User: storedUser,
	})
}

func SetRank(w http.ResponseWriter, r *http.Request) {
	logger := log.WithFields(log.Fields{
		"endpoint": "SetRank",
	})

	// make sure we are only getting put
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	u, err := request.GetUser(r)
	if err != nil {
		logger.WithError(err).Error("getting user")
	}

	params, err := request.GetBody(r)
	if err != nil {
		logger.WithError(err).Error("getting body")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if params["rank"] == nil {
		http.Error(w, "Invalid Request", http.StatusBadRequest)
		return
	}

	var rid int
	switch reflect.TypeOf(params["rank"]).Kind() {
	case reflect.String:
		rid, err = strconv.Atoi(params["rank"].(string))
		if err != nil {
			logger.WithError(err).Error("converting the rank id")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	case reflect.Float64:
		rid = int(params["rank"].(float64))
	default:
		http.Error(w, "Invalid Request", http.StatusBadRequest)
		return
	}

	u.Rank = ranks.Rank(rid)

	if err := u.Save(); err != nil {
		logger.WithError(err).Error("updating user")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if _, err := fmt.Fprint(w, http.StatusOK); err != nil {
		logger.WithError(err).Error("returning status")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

type updateUserRequest struct {
	User map[string]interface{} `json:"user"`
}

func (r *updateUserRequest) bind(c echo.Context) error {
	if err := c.Bind(r); err != nil {
		return err
	}
	return nil
}

func UpdateUser(c echo.Context) error {
	logger := log.WithFields(log.Fields{
		"endpoint": "UpdateUser",
	})
	logger.Debug("updating user")

	req := &updateUserRequest{}
	if err := req.bind(c); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	mu, err := json.Marshal(req.User)
	if err != nil {
		logger.WithError(err).Error("marshal user from request")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	u := &user.User{}
	if err := json.Unmarshal(mu, u); err != nil {
		logger.WithError(err).Error("unmarshal user from request")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	if u.Events < 0 {
		u.Events = 0
	}

	if err := u.Save(); err != nil {
		logger.WithError(err).Error("returning status")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	return c.NoContent(http.StatusOK)
}

func IncrementEvent(c echo.Context) error {
	logger := log.WithFields(log.Fields{
		"endpoint": "IncrementEvent",
	})
	logger.Debug("incrementing event count")

	u, err := user.Get(c.Param("id"))
	if err != nil {
		logger.WithError(err).Error("returning status")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	u.Events++

	if err := u.Save(); err != nil {
		logger.WithError(err).Error("returning status")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, getUserResponse{
		User: u,
	})
}

func DecrementEvent(c echo.Context) error {
	logger := log.WithFields(log.Fields{
		"endpoint": "DecrementEvent",
	})
	logger.Debug("decrementing event count")

	u, err := user.Get(c.Param("id"))
	if err != nil {
		logger.WithError(err).Error("returning status")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	u.Events--

	if u.Events < 0 {
		u.Events = 0
	}

	if err := u.Save(); err != nil {
		logger.WithError(err).Error("returning status")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	// return c.NoContent(http.StatusOK)
	return c.JSON(http.StatusOK, getUserResponse{
		User: u,
	})
}
