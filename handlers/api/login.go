package api

import (
	"net/http"
	"time"

	"github.com/apex/log"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/sol-armada/sol-bot/config"
	"github.com/sol-armada/sol-bot/users"
)

type loginRequest struct {
	Code string `json:"code"`
}

type loginResponse struct {
	User  *users.User `json:"user"`
	Token string      `json:"token"`
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
	u := &users.User{}
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

	token, err := createToken(u.ID)
	if err != nil {
		log.WithError(err).Error("creating token")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, loginResponse{
		User:  u,
		Token: token,
	})
}

func createToken(id string) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["id"] = id
	claims["exp"] = time.Now().AddDate(0, 0, 7).Unix()
	return token.SignedString([]byte(config.GetString("SERVER.SECRET")))
}
