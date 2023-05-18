package api

import (
	"fmt"
	"net/http"

	"github.com/apex/log"
	"github.com/gorilla/mux"
	"github.com/labstack/echo/v4"
	"github.com/sol-armada/admin/stores"
	"github.com/sol-armada/admin/user"
)

type loginRequest struct {
	Code string `json:"code"`
}

type loginResponse struct {
	User *user.User `json:"user"`
}

func (r *loginRequest) bind(c echo.Context) error {
	if err := c.Bind(r); err != nil {
		return err
	}
	// if err := c.Validate(r); err != nil {
	// 	return err
	// }

	return nil
}

func Login(c echo.Context) error {
	logger := log.WithFields(log.Fields{
		"endpoint": "Login",
	})
	logger.Debug("logging in")

	req := &loginRequest{}
	if err := req.bind(c); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	// create the user
	u := &user.User{}
	if err := u.Login(req.Code); err != nil {
		if err.Error() == "invalid_grant" {
			return c.JSON(http.StatusBadRequest, err.Error())
		}

		log.WithError(err).Error("authenicating user")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	// check the user is allowed
	if !u.IsAdmin() {
		return c.JSON(http.StatusUnauthorized, "unauthorized")
	}

	return c.JSON(http.StatusOK, loginResponse{
		User: u,
	})
}

func CheckLogin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	storedUser := &user.User{}
	if err := stores.Storage.GetUser(id).Decode(storedUser); err != nil {
		log.WithError(err).Error("check login return")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if _, err := fmt.Fprint(w, storedUser.StillLoggedIn()); err != nil {
		log.WithError(err).Error("check login return")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func EncryptAccess(w http.ResponseWriter, r *http.Request) {}

func DecryptAccess(w http.ResponseWriter, r *http.Request) {}
