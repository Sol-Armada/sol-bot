package users

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/apex/log"
	"github.com/sol-armada/admin/config"
)

type UserAccess struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

func authenticate(code string) (*UserAccess, error) {
	log.WithField("code", code).Info("creating new user access")

	// TODO: repalce these values with config values
	data := url.Values{}
	data.Set("client_id", config.GetString("DISCORD.CLIENT_ID"))
	data.Set("client_secret", config.GetString("DISCORD.CLIENT_SECRET"))
	data.Set("redirect_uri", config.GetString("DISCORD.REDIRECT_URI"))
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)

	log.WithFields(log.Fields{
		"client_id":     config.GetString("DISCORD.CLIENT_ID"),
		"client_secret": config.GetString("DISCORD.CLIENT_SECRET"),
		"redirect_uri":  config.GetString("DISCORD.REDIRECT_URI"),
		"grant_type":    "authorization_code",
		"code":          code,
	}).Debug("sending auth request")

	req, err := http.NewRequest("POST", "https://discord.com/api/v10/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	log.WithField("code", resp.StatusCode).Debug("authentication status")

	if resp.StatusCode == http.StatusUnauthorized {
		log.WithField("status code", resp.StatusCode).Error("could not authorize")
		return nil, errors.New("could not authorize")
	}

	userAccess := &UserAccess{}
	if err := json.NewDecoder(resp.Body).Decode(&userAccess); err != nil {
		return nil, err
	}

	log.WithField("user access", userAccess).Info("created new user access")

	return userAccess, nil
}
