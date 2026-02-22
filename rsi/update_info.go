package rsi

import (
	"errors"
	"log/slog"

	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/utils"
)

// updateMemberRSIInfo updates RSI information with retry logic
func UpdateMemberRSIInfo(member *members.Member, rsiBackoff *utils.ExponentialBackoff, logger *slog.Logger) error {
	logger.Debug("updating RSI info")
	err := rsiBackoff.Execute(func() error {
		return UpdateRsiInfo(member)
	})

	if errors.Is(err, ErrUserNotFound) {
		logger.Debug("rsi user not found", "error", err)
		member.RSIMember = false
		return nil
	}

	logger.Warn("failed to update RSI info after retries", "error", err)
	return err
}
