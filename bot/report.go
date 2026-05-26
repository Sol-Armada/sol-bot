package bot

import (
	"github.com/bwmarrin/discordgo"
)

func (b *Bot) WeeklyReport() (*discordgo.MessageEmbed, error) {
	embed := &discordgo.MessageEmbed{
		Title:       "Weekly Command Usage Report",
		Description: "Command usage persistence is temporarily disabled during PostgreSQL cutover.",
		Fields:      []*discordgo.MessageEmbedField{},
	}

	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "Status",
		Value:  "Not available",
		Inline: false,
	})
	return embed, nil
}
