package giveaway

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/members"
)

func (g *Giveaway) GetViewEntriesEmbed(member *members.Member) *discordgo.MessageEmbed {
	entries := g.GetMembersEntries(member)

	entryNames := []string{}
	for _, item := range entries {
		entryNames = append(entryNames, fmt.Sprintf("- [%s](https://google.com/)", item.Name))
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Giveaway for %s", g.Attendance.Name),
		Description: "## 🎟️ You have entered the Giveaway!\nBelow are the entries you submitted.\n\n _If you would like to resubmit for different items, please adjust the items you want in the original message. See pinned messages in this channel if you can't find it._",
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "Your Entries",
				Value: strings.Join(entryNames, "\n"),
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Need to see this message again? Press \"View Entries\" on the giveaway message.",
		},
	}
}
