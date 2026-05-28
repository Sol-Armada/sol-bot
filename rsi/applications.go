package rsi

import (
	"context"
	"fmt"
)

// Application represents an organization application
type Application struct {
	Handle string
	Note   string
	Date   string
	Avatar string
}

// GetApplications retrieves pending applications for the organization
// Note: This feature requires authenticated API access which is not provided by the GoScrapeRSI module.
// URL: https://robertsspaceindustries.com/orgs/SOLARMADA/admin/applications?page=1&pagesize=100
func (client *RSIClient) GetApplications(ctx context.Context, page int, pageSize int) ([]Application, error) {
	if client.orgSID == "" {
		return nil, fmt.Errorf("%w: organization SID is required", ErrInvalidConfig)
	}

	// This endpoint requires authentication which is not supported by the current module
	return nil, fmt.Errorf("%w: organization applications endpoint not supported by GoScrapeRSI module", ErrInvalidConfig)
}

// GetAllApplications retrieves all pending applications by paginating through all pages
func (client *RSIClient) GetAllApplications(ctx context.Context) ([]Application, error) {
	if client.orgSID == "" {
		return nil, fmt.Errorf("%w: organization SID is required", ErrInvalidConfig)
	}

	// This endpoint requires authentication which is not supported by the current module
	return nil, fmt.Errorf("%w: organization applications endpoint not supported by GoScrapeRSI module", ErrInvalidConfig)
}
