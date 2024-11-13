package attendancehandler

import (
	"context"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/config"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/utils"
)

var subCommands = map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error{
	"create":       NewCommandHandler,
	"add":          AddMembersCommandHandler,
	"remove":       RemoveMembersCommandHandler,
	"refresh":      RefreshCommandHandler,
	"revert":       RevertCommandHandler,
	"addEventName": AddNameCommandHandler,
}

var lastRefreshTime time.Time

func Setup() (*discordgo.ApplicationCommand, error) {
	tags := []string{
		"FPS",
		"SALVAGE",
		"MINING",
		"FREIGHT",
		"RACING",
		"COMBAT",
		"EXPLORATION",
		"MISSIONS",
		"TRADING",
		"MERCENARY",
		"OTHER",
	}

	if err := config.SetConfig("attendance_tags", tags); err != nil {
		return nil, errors.Wrap(err, "setting attendance tags")
	}

	subCommands := []*discordgo.ApplicationCommandOption{}

	// new attedance record
	newAttendanceOptions := []*discordgo.ApplicationCommandOption{
		{
			Name:         "name",
			Description:  "Name of the event",
			Type:         discordgo.ApplicationCommandOptionString,
			Required:     true,
			Autocomplete: true,
		},
	}
	for i := 0; i < 10; i++ {
		o := &discordgo.ApplicationCommandOption{
			Name:         fmt.Sprintf("member-%d", i+1),
			Description:  "The member to take attendance for",
			Type:         discordgo.ApplicationCommandOptionUser,
			Autocomplete: true,
		}
		newAttendanceOptions = append(newAttendanceOptions, o)
	}

	subCommands = append(subCommands, &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionSubCommand,
		Name:        "create",
		Description: "Create a new attendance record",
		Options:     newAttendanceOptions,
	})
	// end new attendance record

	// add member to attendance record
	addToAttendanceOptions := []*discordgo.ApplicationCommandOption{
		{
			Name:         "event",
			Description:  "The event to add the member to",
			Type:         discordgo.ApplicationCommandOptionString,
			Required:     true,
			Autocomplete: true,
		},
	}
	for i := 0; i < 10; i++ {
		o := &discordgo.ApplicationCommandOption{
			Name:         fmt.Sprintf("member-%d", i+1),
			Description:  "The member to take attendance for",
			Type:         discordgo.ApplicationCommandOptionUser,
			Autocomplete: true,
		}
		addToAttendanceOptions = append(addToAttendanceOptions, o)
	}
	subCommands = append(subCommands, &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionSubCommand,
		Name:        "add",
		Description: "Add a member to an attendance record",
		Options:     addToAttendanceOptions,
	})
	// end add member to attendance record

	// remove member from attendance record
	removeFromAttendanceOptions := []*discordgo.ApplicationCommandOption{
		{
			Name:         "event",
			Description:  "The event to remove the member from",
			Type:         discordgo.ApplicationCommandOptionString,
			Required:     true,
			Autocomplete: true,
		},
	}
	for i := 0; i < 10; i++ {
		o := &discordgo.ApplicationCommandOption{
			Name:         fmt.Sprintf("member-%d", i+1),
			Description:  "The member to remove from attendance",
			Type:         discordgo.ApplicationCommandOptionUser,
			Autocomplete: true,
		}
		removeFromAttendanceOptions = append(removeFromAttendanceOptions, o)
	}
	subCommands = append(subCommands, &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionSubCommand,
		Name:        "remove",
		Description: "remove a member from an attendance record",
		Options:     removeFromAttendanceOptions,
	})
	// end remove member from attendance record

	// revert attendance record
	subCommands = append(subCommands, &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionSubCommand,
		Name:        "revert",
		Description: "revert an attendance record",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:         "event",
				Description:  "The event to revert",
				Type:         discordgo.ApplicationCommandOptionString,
				Required:     true,
				Autocomplete: true,
			},
		},
	})
	// end revert attendance record

	// refresh attendance records
	subCommands = append(subCommands, &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionSubCommand,
		Name:        "refresh",
		Description: "refresh the last 10 attendance records",
	})
	// end refresh attendance records

	// add event name
	subCommands = append(subCommands, &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionSubCommand,
		Name:        "addEventName",
		Description: "add an event name to the available list",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "name",
				Description: "The name of the event",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
		},
	})
	// end add event name

	// add event name
	subCommands = append(subCommands, &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionSubCommand,
		Name:        "removeEventName",
		Description: "remove an event name to the available list",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "name",
				Description: "The name of the event",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
		},
	})
	// end add event name

	return &discordgo.ApplicationCommand{
		Name:        "attendance",
		Description: "Manage attendance records",
		Type:        discordgo.ChatApplicationCommand,
		Options:     subCommands,
	}, nil
}

func CommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("attendance command handler")

	if !allowed(i.Member, "ATTENDANCE") {
		return customerrors.InvalidPermissions
	}

	data := i.Interaction.ApplicationCommandData()
	handler, ok := subCommands[data.Options[0].Name]
	if !ok {
		return customerrors.InvalidSubcommand
	}

	return handler(ctx, s, i)
}

func allowed(discordMember *discordgo.Member, feature string) bool {
	return utils.StringSliceContainsOneOf(discordMember.Roles, settings.GetStringSlice("FEATURES."+feature+".ALLOWED_ROLES"))
}
