package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/pkg/errors"
	"github.com/sol-armada/admin/config"
)

type UserAccess struct {
	AccessToken  string    `json:"access_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	RefreshToken string    `json:"refresh_token"`
}

func Authenticate(code string) (*UserAccess, error) {
	logger := log.WithField("code", code)
	logger.Info("creating new user access")

	redirectUri := strings.TrimSuffix(config.GetString("DISCORD.REDIRECT_URI"), "/")
	redirectUri = fmt.Sprintf("%s/login", redirectUri)

	data := url.Values{}
	data.Set("client_id", config.GetString("DISCORD.CLIENT_ID"))
	data.Set("client_secret", config.GetString("DISCORD.CLIENT_SECRET"))
	data.Set("redirect_uri", redirectUri)
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)

	logger.WithFields(log.Fields{
		"client_id":     config.GetString("DISCORD.CLIENT_ID"),
		"client_secret": config.GetString("DISCORD.CLIENT_SECRET"),
		"redirect_uri":  redirectUri,
		"grant_type":    "authorization_code",
		"code":          code,
	}).Debug("sending auth request")

	req, err := http.NewRequest("POST", "https://discord.com/api/v10/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	logger.Debug("req for authentication to discord")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		logger.WithField("status code", resp.StatusCode).Error("could not authorize")
		return nil, errors.New("could not authorize")
	}

	if resp.StatusCode == http.StatusBadRequest {
		errorMessage, _ := ioutil.ReadAll(resp.Body)
		type ErrorMessage struct {
			ErrorType   string `json:"error"`
			Description string `json:"error_description"`
		}
		errMsg := ErrorMessage{}
		if err := json.Unmarshal(errorMessage, &errMsg); err != nil {
			return nil, err
		}
		return nil, errors.New(errMsg.ErrorType)
	}

	access := map[string]interface{}{}
	if err := json.NewDecoder(resp.Body).Decode(&access); err != nil {
		return nil, err
	}

	userAccess := &UserAccess{
		AccessToken:  access["access_token"].(string),
		ExpiresAt:    time.Now().Add(time.Duration(access["expires_in"].(float64)) * time.Second),
		RefreshToken: access["refresh_token"].(string),
	}

	logger.WithField("user access", *userAccess).Info("created new user access")

	return userAccess, nil
}
