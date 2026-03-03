package auth

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type Access struct {
	Token        string    `json:"access_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	RefreshToken string    `json:"refresh_token"`
}

var redirectUri, clientId, clientSecret string

func init() {
	redirectUri = os.Getenv("DISCORD_REDIRECT_URI")
	if redirectUri == "" {
		panic("DISCORD_REDIRECT_URI environment variable is required")
	}
	clientId = os.Getenv("DISCORD_CLIENT_ID")
	if clientId == "" {
		panic("DISCORD_CLIENT_ID environment variable is required")
	}
	clientSecret = os.Getenv("DISCORD_CLIENT_SECRET")
	if clientSecret == "" {
		panic("DISCORD_CLIENT_SECRET environment variable is required")
	}
}

func Authenticate(code string) (*Access, error) {
	logger := slog.Default().With("code", code)
	logger.Info("creating new member access")

	logger.Debug("sending auth request",
		"client_id", clientId,
		"client_secret", clientSecret,
		"redirect_uri", redirectUri,
		"grant_type", "authorization_code",
		"code", code,
	)

	data := url.Values{}
	data.Set("client_id", clientId)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", redirectUri)
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)

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
		logger.Error("could not authorize", "status code", resp.StatusCode)
		return nil, errors.New("could not authorize")
	}

	if resp.StatusCode == http.StatusBadRequest {
		errorMessage, _ := io.ReadAll(resp.Body)
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

	accessMap := map[string]any{}
	if err := json.NewDecoder(resp.Body).Decode(&accessMap); err != nil {
		return nil, err
	}

	access := &Access{
		Token:        accessMap["access_token"].(string),
		ExpiresAt:    time.Now().Add(time.Duration(accessMap["expires_in"].(float64)) * time.Second),
		RefreshToken: accessMap["refresh_token"].(string),
	}

	logger.Debug("created new member access", "member access", *access)

	return access, nil
}
