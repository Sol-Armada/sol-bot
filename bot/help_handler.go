package bot

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/utils"
)

func helpCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	member := utils.GetMemberFromContext(ctx).(*members.Member)

	memberComamnds := `# Member Commands
## profile
` + "```/profile```" + `
Gets your Sol Armada profile that includes things like if you are validated, your current rank and how many events you have attended.
───────────────────────────────────────────`

	officerCommands := `# Officer Commands
## attendance
Manage attendance for events
` + "```/attendance new {event name} {optional: members[]}```" + `
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
