package giveaway

import (
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/utils"
)

func (g *Giveaway) GetEmbed() *discordgo.MessageEmbed {
	feilds := []*discordgo.MessageEmbedField{}

	a, err := attendance.Get(g.AttendanceId)
	if err != nil {
		a = &attendance.Attendance{
			Name: "",
		}
	}

	if g.Ended {
		for _, item := range g.Items {
			if len(item.Members) == 0 {
				continue
			}

			members := item.Members
			membersStr := ""
			for _, member := range members {
				membersStr += fmt.Sprintf("<@%s>\n", member)
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
			Title:       a.Name,
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
	r := max(time.Until(g.EndTime), 0)
	days := int(r.Hours() / 24)
	hours := int(r.Hours()) % 24
	minutes := int(r.Minutes()) % 60
	seconds := int(r.Seconds()) % 60

	timeRemainingParts := []string{}

	if days > 0 {
		dayStr := fmt.Sprintf("%d Day", days)
		if days > 1 {
			dayStr += "s"
		}
		timeRemainingParts = append(timeRemainingParts, dayStr)
	}
	if hours > 0 {
		hourStr := fmt.Sprintf("%d Hour", hours)
		if hours > 1 {
			hourStr += "s"
		}
		timeRemainingParts = append(timeRemainingParts, hourStr)
	}
	if minutes > 0 && hours == 0 && days == 0 {
		minuteStr := fmt.Sprintf("%d Minute", minutes)
		if minutes > 1 {
			minuteStr += "s"
		}
		timeRemainingParts = append(timeRemainingParts, minuteStr)
	}
	if seconds > 0 && minutes < 2 && hours == 0 && days == 0 {
		secondStr := fmt.Sprintf("%d Second", seconds)
		if seconds > 1 {
			secondStr += "s"
		}
		timeRemainingParts = append(timeRemainingParts, secondStr)
	}

	timeRemaining := strings.Join(timeRemainingParts, " ")
	if timeRemaining == "" {
		timeRemaining = "0 Seconds"
	}

	footer := &discordgo.MessageEmbedFooter{
		// Text: fmt.Sprintf("Time Remaining: %s", timeRemaining),
		Text: fmt.Sprintf("Giveaway ends in %s", timeRemaining),
	}

	return &discordgo.MessageEmbed{
		Title:       a.Name,
		Description: "### 🎁  Ongoing Giveaways  🎁\nLooking to earn yourself something nice? Select the items you want from the dropdown below for a chance to win!\n\n_Tokens not required. Must be part of the associated event._",
		Fields:      feilds,
		Color:       0x000000 + rand.Intn(0xffffff),
		Footer:      footer,
		Timestamp:   g.EndTime.Format(time.RFC3339),
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
