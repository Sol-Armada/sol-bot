package giveaway

import (
	"fmt"
	"math/rand"

	"github.com/bwmarrin/discordgo"
)

func (g *Giveaway) GetEmbed() *discordgo.MessageEmbed {
	feilds := []*discordgo.MessageEmbedField{}

	if g.Ended {
		for _, item := range g.Items {
			members := item.Members
			membersStr := ""
			for _, member := range members {
				membersStr += fmt.Sprintf("<@%s>\n", member.Id)
			}

			field := &discordgo.MessageEmbedField{
				Name:   fmt.Sprintf("Prize: %s", item.Name),
				Value:  membersStr,
				Inline: true,
			}
			feilds = append(feilds, field)
		}

		return &discordgo.MessageEmbed{
			Title:       g.Attendance.Name,
			Description: "### 🎊 Giveaway Winners 🎊\nCongrats on your winnings! Please meet at the OIC's hanger to collect your prizes, if your name is shown below.",
			Fields:      feilds,
			Color:       0x000000 + rand.Intn(0xffffff),
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Giveaway has ended",
			},
		}
	}

	for _, item := range g.Items {
		field := &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%s (%dx Winners)", item.Name, item.Amount),
			Value:  "[Stats](https://google.com/)",
			Inline: true,
		}
		feilds = append(feilds, field)
	}

	footer := &discordgo.MessageEmbedFooter{
		Text: fmt.Sprintf("Time Remaining: %d", g.Timer),
	}

	return &discordgo.MessageEmbed{
		Title:       g.Attendance.Name,
		Description: "### 🎁 Ongoing Giveaways\nLooking to earn yourself something nice? Select the items you want from the dropdown below for a chance to win! _Tokens not required. Must be part of the associated event._",
		Fields:      feilds,
		Color:       0x000000 + rand.Intn(0xffffff),
		Footer:      footer,
	}
}
