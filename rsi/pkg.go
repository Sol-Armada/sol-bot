package rsi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
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
	token   string
	orgSID  string
	allies  []string
	timeout time.Duration
}

// Config holds the configuration for the RSI client
type Config struct {
	Token   string
	Device  string
	OrgSID  string
	Allies  []string
	Timeout time.Duration
}

// NewClient creates a new RSI client with the given configuration
func NewClient(config Config) (*RSIClient, error) {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &RSIClient{
		token:   fmt.Sprintf("Rsi-Token=%s; _rsi_device=%s;", config.Token, config.Device),
		orgSID:  config.OrgSID,
		allies:  config.Allies,
		timeout: config.Timeout,
	}, nil
}

// NewDefaultClient creates a new RSI client using settings from the config
func NewDefaultClient() (*RSIClient, error) {
	config := Config{
		Token:   settings.GetString("RSI.TOKEN"),
		Device:  settings.GetString("RSI.DEVICE"),
		OrgSID:  settings.GetString("rsi_org_sid"),
		Allies:  settings.GetStringSlice("ALLIES"),
		Timeout: 30 * time.Second,
	}

	return NewClient(config)
}

// createCollector creates a new colly collector with common settings
func (c *RSIClient) createCollector() *colly.Collector {
	collector := colly.NewCollector(colly.AllowURLRevisit())
	collector.SetRequestTimeout(c.timeout)

	collector.OnRequest(func(r *colly.Request) {
		r.Headers.Set("cookie", c.token)
	})

	return collector
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

	c := client.createCollector()
	var err error

	c.OnResponse(func(r *colly.Response) {
		switch r.StatusCode {
		case 404:
			err = ErrUserNotFound
		case 403:
			err = ErrForbidden
		case 200:
			// OK, do nothing
		default:
			err = fmt.Errorf("%w: status code %d", ErrRequestFailed, r.StatusCode)
		}
	})

	c.OnXML(`//div[contains(@class, "org main")]//div[@class="info"]//span[contains(text(), "SID")]/following-sibling::strong`, func(e *colly.XMLElement) {
		if e.Text == "" {
			e.Text = "None"
		}
		member.PrimaryOrg = e.Text
	})

	c.OnXML(`//div[contains(@class, "org main")]//div[@class="info"]//span[contains(text(), "rank")]/following-sibling::strong`, func(e *colly.XMLElement) {
		if member.PrimaryOrg == client.orgSID {
			member.Rank = ranks.GetRankByRSIRankName(e.Text)
			member.IsGuest = false
		}
	})

	c.OnXML(`//div[contains(@class, "orgs-content")]`, func(e *colly.XMLElement) {
		member.Affilations = e.ChildTexts(`//div[contains(@class, "org affiliation")]//div[@class="info"]//span[contains(text(), "SID")]/following-sibling::strong`)
		if utils.StringSliceContains(member.Affilations, client.orgSID) {
			member.IsAffiliate = true
			member.Rank = ranks.Member
			member.IsGuest = false
			member.IsAlly = false
		}
	})

	c.OnXML(`//div[contains(@class, "org main")]//div[contains(@class,"member-visibility-restriction")]`, func(e *colly.XMLElement) {
		member.PrimaryOrg = "REDACTED"
		member.IsGuest = true
	})

	url := fmt.Sprintf("%s/citizens/%s/organizations", RsiBaseURL, strings.ReplaceAll(member.Name, ".", ""))
	if visitErr := c.Visit(url); visitErr != nil {
		if strings.Contains(visitErr.Error(), "Not Found") {
			return ErrUserNotFound
		}
		return fmt.Errorf("%w: %v", ErrRequestFailed, visitErr)
	}

	if err != nil {
		return err
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
	c := client.createCollector()
	exists := true

	c.OnResponse(func(r *colly.Response) {
		if r.StatusCode != 200 {
			exists = false
		}
	})

	if err := c.Visit(fmt.Sprintf("%s/citizens/%s/organizations", RsiBaseURL, handle)); err != nil {
		if strings.Contains(err.Error(), "Not Found") {
			exists = false
		}
	}

	return exists
}

// IsMemberOfOrg checks if a handle is a member of a specific organization
func (client *RSIClient) IsMemberOfOrg(ctx context.Context, handle string, org string) (bool, error) {
	c := client.createCollector()

	var orgs []string
	var err error

	c.OnResponse(func(r *colly.Response) {
		if r.StatusCode == 404 {
			err = ErrUserNotFound
		} else if r.StatusCode != 200 {
			err = fmt.Errorf("%w: status code %d", ErrRequestFailed, r.StatusCode)
		}
	})

	// Get primary organization
	c.OnXML(`//div[contains(@class, "org main")]//div[@class="info"]//span[contains(text(), "SID")]/following-sibling::strong`, func(e *colly.XMLElement) {
		if e.Text != "" && e.Text != "None" {
			orgs = append(orgs, e.Text)
		}
	})

	// Get affiliated organizations
	c.OnXML(`//div[contains(@class, "orgs-content")]`, func(e *colly.XMLElement) {
		affiliations := e.ChildTexts(`//div[contains(@class, "org affiliation")]//div[@class="info"]//span[contains(text(), "SID")]/following-sibling::strong`)
		orgs = append(orgs, affiliations...)
	})

	if visitErr := c.Visit(fmt.Sprintf("%s/citizens/%s/organizations", RsiBaseURL, handle)); visitErr != nil {
		if strings.Contains(visitErr.Error(), "Not Found") {
			return false, ErrUserNotFound
		}
		return false, fmt.Errorf("%w: %v", ErrRequestFailed, visitErr)
	}

	if err != nil {
		return false, err
	}

	for _, o := range orgs {
		if strings.EqualFold(o, org) {
			return true, nil
		}
	}

	return false, nil
}

// GetBio retrieves the bio for an RSI handle
func (client *RSIClient) GetBio(ctx context.Context, handle string) (string, error) {
	c := client.createCollector()

	var err error
	bio := ""

	c.OnResponse(func(r *colly.Response) {
		switch r.StatusCode {
		case 404:
			err = ErrUserNotFound
		case 403:
			err = ErrForbidden
		default:
			if r.StatusCode != 200 {
				err = fmt.Errorf("%w: status code %d", ErrRequestFailed, r.StatusCode)
			}
		}
	})

	c.OnXML(`//div[@id="public-profile"]//div[contains(@class, "bio")]/div`, func(e *colly.XMLElement) {
		bio = e.Text
	})

	if visitErr := c.Visit(fmt.Sprintf("%s/citizens/%s", RsiBaseURL, handle)); visitErr != nil {
		if strings.Contains(visitErr.Error(), "Not Found") {
			return "", ErrUserNotFound
		}
		return "", fmt.Errorf("%w: %v", ErrRequestFailed, visitErr)
	}

	if err != nil {
		return "", err
	}

	return bio, nil
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
