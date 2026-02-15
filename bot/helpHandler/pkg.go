package helphandler

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/bot/internal/command"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/utils"
)

type HelpCommand struct{}

var _ command.ApplicationCommand = (*HelpCommand)(nil)

func New() command.ApplicationCommand {
	return &HelpCommand{}
}

// AutocompleteHandler implements [command.ApplicationCommand].
func (h *HelpCommand) AutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// ButtonHandler implements [command.ApplicationCommand].
func (h *HelpCommand) ButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// CommandHandler implements [command.ApplicationCommand].
func (h *HelpCommand) CommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	member := utils.GetMemberFromContext(ctx).(*members.Member)

	memberComamnds := `# Member Commands
## profile
` + "```/profile```" + `
Gets your Sol Armada profile that includes things like if you are validated, your current rank and how many events you have attended.
───────────────────────────────────────────`

	officerCommands := `# Officer Commands
## profile
` + "```/profile {member} {force}```" + `
Gets the Sol Armada profile for the given member. If ` + "`force`" + ` is given, it will force a refresh of the profile.
───────────────────────────────────────────
## attendance
Manage attendance for events
` + "```/attendance create {event name} {optional: members[]}```" + `
Creates a new event attendance with the give name. If a list of members are given, they will be added to the attendance. Alternatively, you can use ` + "`/attendance add`" + ` (see below) to add more members to the attendance.
───────────────────────────────────────────
` + "```/attendance add {event name} {member} {optional: members[]}```" + `
Adds the given member or list of members to the given event attendance.
───────────────────────────────────────────
` + "```/attendance remove {event name} {member} {optional: members[]}```" + `
Removes the given member or list of members from the given event attendance.
───────────────────────────────────────────
` + "```/attendance revert {event name}```" + `
Reverts the given event attendance to "not recorded".
───────────────────────────────────────────
` + "```/attendance refresh```" + `
Refreshes the last ten event attendances' messages in the channel. This is incase anything gets manually updated or you think that they might not be correct. This does _not_ effect members with issues. Use the ` + "`Recheck issues`" + ` button on the attendance before submitting for that action.
───────────────────────────────────────────
` + "```/attendance add_event_name {new event name}```" + `
Adds an event name to the list.
───────────────────────────────────────────
` + "```/attendance remove_event_name {event name}```" + `
Removes an event name from the list.
───────────────────────────────────────────
## rankups
Lists members who need to be ranked up.
` + "```/rankups```" + `
## merits
` + "```/merit {member} {reason}```" + `
Adds a merit to a member.
───────────────────────────────────────────
` + "```/demerit {member} {reason}```" + `
Adds a demerit to a member.
`

	embeds := []*discordgo.MessageEmbed{
		{
			Title:       "Sol-Bot Commands",
			Description: memberComamnds,
		},
	}

	if member.IsOfficer() {
		embeds = append(embeds, &discordgo.MessageEmbed{
			Title:       "Sol-Bot Commands (continued)",
			Description: officerCommands,
		})
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:  discordgo.MessageFlagsEphemeral,
			Embeds: embeds,
		},
	})
}

// ModalHandler implements [command.ApplicationCommand].
func (h *HelpCommand) ModalHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// Name implements [command.ApplicationCommand].
func (h *HelpCommand) Name() string {
	return "help"
}

// OnAfter implements [command.ApplicationCommand].
func (h *HelpCommand) OnAfter(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// OnBefore implements [command.ApplicationCommand].
func (h *HelpCommand) OnBefore(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// OnError implements [command.ApplicationCommand].
func (h *HelpCommand) OnError(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, err error) {
}

// SelectMenuHandler implements [command.ApplicationCommand].
func (h *HelpCommand) SelectMenuHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// Setup implements [command.ApplicationCommand].
func (h *HelpCommand) Setup() (*discordgo.ApplicationCommand, error) {
	return &discordgo.ApplicationCommand{
		Name:        "help",
		Description: "View help",
		Type:        discordgo.ChatApplicationCommand,
	}, nil
}
