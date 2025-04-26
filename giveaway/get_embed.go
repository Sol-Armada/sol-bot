package giveaway

import (
	"fmt"
	"math/rand"
	"slices"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/utils"
)

func (g *Giveaway) GetEmbed() *discordgo.MessageEmbed {
	feilds := []*discordgo.MessageEmbedField{}

	if g.Ended {
		for _, item := range g.Items {
			if len(item.Members) == 0 {
				continue
			}

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

		feilds = sortFieldsByName(feilds)

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
		valueStr := "No stats available"

		if slices.Contains(utils.GetItemNames(), item.Name) {
			valueStr = fmt.Sprintf("[Stats](https://uexcorp.space/items/info?name=%s)", strings.ToLower(strings.ReplaceAll(item.Name, " ", "-")))
		}

		valueStr += fmt.Sprintf("\nEntries: %d", len(item.Members))

		field := &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%s (%dx Winners)", item.Name, item.Amount),
			Value:  valueStr,
			Inline: true,
		}
		feilds = append(feilds, field)
	}

	feilds = sortFieldsByName(feilds)

	// get the time remaining
	timeRemainingM := g.TimeRemainingS / 60
	timeRemainingS := g.TimeRemainingS % 60
	footer := &discordgo.MessageEmbedFooter{
		Text: fmt.Sprintf("Time Remaining: %02d:%02d", timeRemainingM, timeRemainingS),
	}

	return &discordgo.MessageEmbed{
		Title:       g.Attendance.Name,
		Description: "### 🎁  Ongoing Giveaways  🎁\nLooking to earn yourself something nice? Select the items you want from the dropdown below for a chance to win!\n\n_Tokens not required. Must be part of the associated event._",
		Fields:      feilds,
		Color:       0x000000 + rand.Intn(0xffffff),
		Footer:      footer,
	}
}

func sortFieldsByName(feilds []*discordgo.MessageEmbedField) []*discordgo.MessageEmbedField {
	slices.SortFunc(feilds, func(a, b *discordgo.MessageEmbedField) int {
		if strings.ToLower(a.Name) < strings.ToLower(b.Name) {
			return -1
		} else if strings.ToLower(a.Name) > strings.ToLower(b.Name) {
			return 1
		} else {
			return 0
		}
	})

	return feilds
}
