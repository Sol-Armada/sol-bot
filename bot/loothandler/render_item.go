package loothandler

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/rolls"
)

type itemChoiceCounts struct {
	Need  int
	Greed int
	Pass  int
}

func buildEventPrimaryMessage(event *rolls.RollEvent, items []*rolls.RollItem) *discordgo.MessageSend {
	itemLines := make([]string, 0, len(items))
	for _, item := range items {
		itemLines = append(itemLines, fmt.Sprintf("- %s (x%d)", item.Name, item.Amount))
	}

	description := "Need/Greed roll event started."
	if len(itemLines) > 0 {
		description = description + "\n\nItems:\n" + strings.Join(itemLines, "\n")
	}

	if event.EndTime != nil {
		endUnix := event.EndTime.UTC().Unix()
		description = description + fmt.Sprintf("\n\nEnds: <t:%d:F> (<t:%d:R>)", endUnix, endUnix)
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       event.Name,
				Description: description,
			},
		},
	}
}

func buildItemMessage(event *rolls.RollEvent, item *rolls.RollItem, counts itemChoiceCounts) *discordgo.MessageSend {
	return &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{buildItemEmbed(event, item)},
		Components: buildItemComponents(event.Id, item.Id, event.Ended, counts),
	}
}

func buildItemEmbed(event *rolls.RollEvent, item *rolls.RollItem) *discordgo.MessageEmbed {
	footer := &discordgo.MessageEmbedFooter{}
	if event.EndTime != nil {
		footer.Text = fmt.Sprintf("Ends %s", event.EndTime.UTC().Format(time.RFC1123))
	}

	return &discordgo.MessageEmbed{
		Title:       event.Name,
		Description: fmt.Sprintf("Item: **%s** (x%d)", item.Name, item.Amount),
		Footer:      footer,
	}
}

func buildItemComponents(eventId, itemId string, ended bool, counts itemChoiceCounts) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					CustomID: fmt.Sprintf("loot:update_entry:%s:%s:%s", eventId, itemId, rolls.ChoiceNeed),
					Label:    fmt.Sprintf("Need (%d)", counts.Need),
					Style:    discordgo.PrimaryButton,
					Disabled: ended,
				},
				discordgo.Button{
					CustomID: fmt.Sprintf("loot:update_entry:%s:%s:%s", eventId, itemId, rolls.ChoiceGreed),
					Label:    fmt.Sprintf("Greed (%d)", counts.Greed),
					Style:    discordgo.SecondaryButton,
					Disabled: ended,
				},
			},
		},
	}
}
