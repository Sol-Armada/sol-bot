package bot

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/database/postgresql"
)

func (b *Bot) WeeklyReport() (*discordgo.MessageEmbed, error) {
	pg := postgresql.Get()
	if pg == nil || pg.Queries == nil {
		return nil, fmt.Errorf("postgresql client not initialized")
	}

	counts, err := pg.Queries.GetWeeklyCommandCounts(context.Background())
	if err != nil {
		return nil, err
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Weekly Command Usage Report",
		Description: "Command usage over the last 7 days.",
		Fields:      []*discordgo.MessageEmbedField{},
	}

	if len(counts) == 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Status",
			Value:  "No command usage recorded in the last 7 days.",
			Inline: false,
		})
		return embed, nil
	}

	for _, row := range counts {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   row.Name,
			Value:  fmt.Sprintf("%d", row.Count),
			Inline: true,
		})
	}

	return embed, nil
}
