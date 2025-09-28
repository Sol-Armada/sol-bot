package rsi

import (
	"context"
	"fmt"
	"strings"

	"github.com/gocolly/colly/v2"
)

// Application represents an organization application
type Application struct {
	Handle string
	Note   string
	Date   string
	Avatar string
}

// GetApplications retrieves pending applications for the organization
// URL: https://robertsspaceindustries.com/orgs/SOLARMADA/admin/applications?page=1&pagesize=100
func (client *RSIClient) GetApplications(ctx context.Context, page int, pageSize int) ([]Application, error) {
	if client.token == "" || client.orgSID == "" {
		return nil, fmt.Errorf("%w: RSI token and organization SID are required", ErrInvalidConfig)
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 100
	}

	c := client.createCollector()
	var applications []Application
	var err error

	c.OnResponse(func(r *colly.Response) {
		if r.StatusCode == 404 {
			err = ErrUserNotFound
		} else if r.StatusCode != 200 {
			err = fmt.Errorf("%w: status code %d", ErrRequestFailed, r.StatusCode)
		}
	})

	// Parse application entries
	c.OnHTML("table.DataTable tbody tr", func(e *colly.HTMLElement) {
		app := Application{}

		// Extract handle
		handleElement := e.DOM.Find("td:nth-child(1) a")
		app.Handle = strings.TrimSpace(handleElement.Text())

		// Extract avatar
		avatarElement := e.DOM.Find("td:nth-child(1) img")
		if src, exists := avatarElement.Attr("src"); exists {
			app.Avatar = src
		}

		// Extract note
		app.Note = strings.TrimSpace(e.DOM.Find("td:nth-child(2)").Text())

		// Extract date
		app.Date = strings.TrimSpace(e.DOM.Find("td:nth-child(3)").Text())

		if app.Handle != "" {
			applications = append(applications, app)
		}
	})

	url := fmt.Sprintf("https://robertsspaceindustries.com/orgs/%s/admin/applications?page=%d&pagesize=%d",
		client.orgSID, page, pageSize)

	if visitErr := c.Visit(url); visitErr != nil {
		return nil, fmt.Errorf("%w: %v", ErrRequestFailed, visitErr)
	}

	if err != nil {
		return nil, err
	}

	return applications, nil
}

// GetAllApplications retrieves all pending applications by paginating through all pages
func (client *RSIClient) GetAllApplications(ctx context.Context) ([]Application, error) {
	if client.token == "" || client.orgSID == "" {
		return nil, fmt.Errorf("%w: RSI token and organization SID are required", ErrInvalidConfig)
	}

	var allApplications []Application
	page := 1
	pageSize := 100

	for {
		applications, err := client.GetApplications(ctx, page, pageSize)
		if err != nil {
			return nil, err
		}

		if len(applications) == 0 {
			break
		}

		allApplications = append(allApplications, applications...)

		// If we got fewer results than the page size, we've reached the end
		if len(applications) < pageSize {
			break
		}

		page++
	}

	return allApplications, nil
}
