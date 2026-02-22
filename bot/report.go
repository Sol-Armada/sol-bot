package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) WeeklyReport() (*discordgo.MessageEmbed, error) {
	commands, err := b.store.CountsByName()
	if err != nil {
		return nil, err
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Weekly Command Usage Report",
		Description: "Here are the counts of commands used in the past week:",
		Fields:      []*discordgo.MessageEmbedField{},
	}

	for name, count := range commands {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   name,
			Value:  fmt.Sprint(count),
			Inline: true,
		})
	}
	return embed, nil
}
