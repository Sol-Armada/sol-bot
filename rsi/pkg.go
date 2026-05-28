package rsi

import (
	"context"
	"errors"
	"fmt"
	"strings"

	rsimodule "github.com/koo04/GoScrapeRSI"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/ranks"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/utils"
)

const RsiBaseURL = "https://robertsspaceindustries.com"

var (
	// ErrUserNotFound is returned when an RSI user is not found
	ErrUserNotFound = errors.New("rsi user was not found")
	// ErrInvalidConfig is returned when RSI configuration is invalid
	ErrInvalidConfig = errors.New("invalid RSI configuration")
	// ErrRequestFailed is returned when an RSI request fails
	ErrRequestFailed = errors.New("rsi request failed")
	// ErrForbidden is returned when access to RSI is forbidden
	ErrForbidden = errors.New("access to rsi is forbidden")

	// Deprecated: Use ErrUserNotFound instead
	RsiUserNotFound = ErrUserNotFound
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
func NewClient(config Config) (*RSIClient, error) {
	return &RSIClient{
		client: rsimodule.NewClient(),
		orgSID: config.OrgSID,
		allies: config.Allies,
	}, nil
}

// NewDefaultClient creates a new RSI client using settings from the config
func NewDefaultClient() (*RSIClient, error) {
	config := Config{
		Token:  settings.GetString("RSI.TOKEN"),
		Device: settings.GetString("RSI.DEVICE"),
		OrgSID: settings.GetString("rsi_org_sid"),
		Allies: settings.GetStringSlice("ALLIES"),
	}

	return NewClient(config)
}

// UpdateRsiInfo updates member information from RSI website
func (client *RSIClient) UpdateRsiInfo(ctx context.Context, member *members.Member) error {
	member.RSIMember = false
	member.IsAlly = false
	member.IsAffiliate = false
	member.IsGuest = true
	member.Rank = ranks.None
	member.PrimaryOrg = ""
	member.Affilations = []string{}

	// Fetch user profile from the module
	profile, err := client.client.GetUser(ctx, member.Name)
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			return ErrUserNotFound
		}
		if strings.Contains(err.Error(), "403") {
			return ErrForbidden
		}
		return fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}

	// If profile is empty, user not found
	if profile == nil || profile.Info.Handle == "" {
		return ErrUserNotFound
	}

	// Set primary organization info
	if profile.Organization.SID != "" {
		member.PrimaryOrg = profile.Organization.SID
		if profile.Organization.SID == client.orgSID {
			member.Rank = ranks.GetRankByRSIRankName(profile.Organization.Rank)
			member.IsGuest = false
		}
	}

	// Process affiliations
	member.Affilations = []string{}
	for _, aff := range profile.Affiliation {
		member.Affilations = append(member.Affilations, aff.SID)
		if aff.SID == client.orgSID {
			member.IsAffiliate = true
			member.Rank = ranks.Member
			member.IsGuest = false
			member.IsAlly = false
		}
	}

	member.RSIMember = true

	if client.isAllyOrg(member.PrimaryOrg) {
		member.IsAlly = true
	}

	return nil
}

// isAllyOrg checks if an organization is in the allies list
func (client *RSIClient) isAllyOrg(org string) bool {
	return utils.StringSliceContains(client.allies, org)
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

// Backward compatibility functions using a default client

var defaultClient *RSIClient

// initDefaultClient initializes the default client if it hasn't been initialized
func initDefaultClient() error {
	if defaultClient == nil {
		var err error
		defaultClient, err = NewDefaultClient()
		if err != nil {
			return fmt.Errorf("failed to initialize default RSI client: %w", err)
		}
	}
	return nil
}

// UpdateRsiInfo updates member information from RSI website using the default client
// Deprecated: Use RSIClient.UpdateRsiInfo with context instead
func UpdateRsiInfo(member *members.Member) error {
	if err := initDefaultClient(); err != nil {
		return err
	}
	return defaultClient.UpdateRsiInfo(context.Background(), member)
}

// ValidHandle checks if an RSI handle exists using the default client
// Deprecated: Use RSIClient.ValidHandle with context instead
func ValidHandle(handle string) bool {
	if err := initDefaultClient(); err != nil {
		return false
	}
	return defaultClient.ValidHandle(context.Background(), handle)
}

// IsMemberOfOrg checks if a handle is a member of a specific organization using the default client
// Deprecated: Use RSIClient.IsMemberOfOrg with context instead
func IsMemberOfOrg(handle string, org string) (bool, error) {
	if err := initDefaultClient(); err != nil {
		return false, err
	}
	return defaultClient.IsMemberOfOrg(context.Background(), handle, org)
}

// GetBio retrieves the bio for an RSI handle using the default client
// Deprecated: Use RSIClient.GetBio with context instead
func GetBio(handle string) (string, error) {
	if err := initDefaultClient(); err != nil {
		return "", err
	}
	return defaultClient.GetBio(context.Background(), handle)
}

// UserProfileURL returns the RSI profile URL for a given handle
func UserProfileURL(handle string) string {
	return fmt.Sprintf("%s/citizens/%s", RsiBaseURL, handle)
}
