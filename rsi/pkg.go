package rsi

import (
	"context"
	"errors"
	"fmt"
	"strings"

	rsimodule "github.com/koo04/GoScrapeRSI"
	"github.com/sol-armada/sol-bot/settings"
)

const RsiBaseURL = "https://robertsspaceindustries.com"

var (
	ErrUserNotFound  = errors.New("rsi user was not found")
	ErrInvalidConfig = errors.New("invalid RSI configuration")
	ErrRequestFailed = errors.New("rsi request failed")
	ErrForbidden     = errors.New("access to rsi is forbidden")
)

// RSIClient handles interactions with the RSI website
type RSIClient struct {
	client *rsimodule.Client
	orgSID string
	allies []string
}

// Config holds the configuration for the RSI client
type Config struct {
	Token  string
	Device string
	OrgSID string
	Allies []string
}

// NewClient creates a new RSI client with the given configuration
func NewClient(config Config) *RSIClient {
	return &RSIClient{
		client: rsimodule.NewClient(),
		orgSID: config.OrgSID,
		allies: config.Allies,
	}
}

// NewDefaultClient creates a new RSI client using settings from the config
func NewDefaultClient() *RSIClient {
	config := Config{
		Token:  settings.GetString("RSI.TOKEN"),
		Device: settings.GetString("RSI.DEVICE"),
		OrgSID: settings.GetString("rsi_org_sid"),
		Allies: settings.GetStringSlice("ALLIES"),
	}

	return NewClient(config)
}

// GetRSIInfo fetches RSI profile information for a handle
func (client *RSIClient) GetRSIInfo(ctx context.Context, handle string) (*rsimodule.UserProfile, error) {
	// Fetch user profile from the module
	profile, err := client.client.GetUser(ctx, handle)
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			return nil, ErrUserNotFound
		}
		if strings.Contains(err.Error(), "403") {
			return nil, ErrForbidden
		}
		return nil, fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}

	// If profile is empty, user not found
	if profile == nil || profile.Info.Handle == "" {
		return nil, ErrUserNotFound
	}

	return profile, nil
}

// ValidHandle checks if an RSI handle exists
func (client *RSIClient) ValidHandle(ctx context.Context, handle string) bool {
	profile, err := client.client.GetUser(ctx, handle)
	if err != nil {
		return false
	}
	return profile != nil && profile.Info.Handle != ""
}

// IsMemberOfOrg checks if a handle is a member of a specific organization
func (client *RSIClient) IsMemberOfOrg(ctx context.Context, handle string, org string) (bool, error) {
	profile, err := client.client.GetUser(ctx, handle)
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			return false, ErrUserNotFound
		}
		if strings.Contains(err.Error(), "403") {
			return false, ErrForbidden
		}
		return false, fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}

	if profile == nil || profile.Info.Handle == "" {
		return false, ErrUserNotFound
	}

	// Check main organization
	if profile.Organization.SID != "" && strings.EqualFold(profile.Organization.SID, org) {
		return true, nil
	}

	// Check affiliations
	for _, aff := range profile.Affiliation {
		if strings.EqualFold(aff.SID, org) {
			return true, nil
		}
	}

	return false, nil
}

// GetBio retrieves the bio for an RSI handle
func (client *RSIClient) GetBio(ctx context.Context, handle string) (string, error) {
	profile, err := client.client.GetUser(ctx, handle)
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			return "", ErrUserNotFound
		}
		if strings.Contains(err.Error(), "403") {
			return "", ErrForbidden
		}
		return "", fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}

	if profile == nil || profile.Info.Handle == "" {
		return "", ErrUserNotFound
	}

	return profile.Info.Bio, nil
}

// Package-level helpers using a default client

var defaultClient *RSIClient

// getDefaultClient initializes the default client if it hasn't been initialized
func getDefaultClient() *RSIClient {
	if defaultClient == nil {
		defaultClient = NewDefaultClient()
	}
	return defaultClient
}

// GetRSIInfo fetches RSI profile information using the default client.
func GetRSIInfo(handle string) (*rsimodule.UserProfile, error) {
	return getDefaultClient().GetRSIInfo(context.Background(), handle)
}

// ValidHandle checks if an RSI handle exists using the default client
// Deprecated: Use RSIClient.ValidHandle with context instead
func ValidHandle(handle string) bool {
	return getDefaultClient().ValidHandle(context.Background(), handle)
}

// IsMemberOfOrg checks if a handle is a member of a specific organization using the default client
// Deprecated: Use RSIClient.IsMemberOfOrg with context instead
func IsMemberOfOrg(handle string, org string) (bool, error) {
	return getDefaultClient().IsMemberOfOrg(context.Background(), handle, org)
}

// GetBio retrieves the bio for an RSI handle using the default client
// Deprecated: Use RSIClient.GetBio with context instead
func GetBio(handle string) (string, error) {
	return getDefaultClient().GetBio(context.Background(), handle)
}

// UserProfileURL returns the RSI profile URL for a given handle
func UserProfileURL(handle string) string {
	return fmt.Sprintf("%s/citizens/%s", RsiBaseURL, handle)
}
