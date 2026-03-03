package utils

import (
	"os"

	"github.com/sol-armada/sol-bot/settings"
)

// NeedsSetup checks if initial setup is required
func NeedsSetup() bool {
	// Check if setup was completed
	if settings.GetBoolWithDefault("setup_completed", false) {
		return false
	}

	// Check critical Discord settings
	criticalSettings := []string{
		"ATTENDANCE_CHANNEL_ID",
		"AFK_CHANNEL_ID",
		"ONBOARDING_CHANNEL_ID",
		"ONBOARDING_REPORT_CHANNEL_ID",
		"ERROR_CHANNEL_ID",
	}

	for _, setting := range criticalSettings {
		value := os.Getenv(setting)
		if value == "" {
			return true // Missing critical setting
		}
	}

	return false
}
