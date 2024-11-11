package attendancehandler

import (
	"context"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/utils"
)

var attendanceSubCommands = map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error{
	"new":     NewCommandHandler,
	"add":     AddMembersAttendanceCommandHandler,
	"remove":  RemoveMembersAttendanceCommandHandler,
	"refresh": RefreshAttendanceCommandHandler,
	"revert":  RevertAttendanceCommandHandler,
}

var lastRefreshTime time.Time

func AttendanceCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("attendance command handler")

	if !allowed(i.Member, "ATTENDANCE") {
		return customerrors.InvalidPermissions
	}

	data := i.Interaction.ApplicationCommandData()
	handler, ok := attendanceSubCommands[data.Options[0].Name]
	if !ok {
		return customerrors.InvalidSubcommand
	}

	return handler(ctx, s, i)
}

func allowed(discordMember *discordgo.Member, feature string) bool {
	return utils.StringSliceContainsOneOf(discordMember.Roles, settings.GetStringSlice("FEATURES."+feature+".ALLOWED_ROLES"))
}
