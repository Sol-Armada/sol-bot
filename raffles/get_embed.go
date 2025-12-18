package raffles

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/tokens"
)

func (r *Raffle) GetEmbed() (*discordgo.MessageEmbed, error) {
	feilds := []*discordgo.MessageEmbedField{
		{
			Name:   "Prize",
			Value:  r.Prize,
			Inline: true,
		},
	}

	if r.Quantity > 1 {
		feilds = append(feilds, &discordgo.MessageEmbedField{
			Name:   "Quantity",
			Value:  fmt.Sprintf("%d", r.Quantity),
			Inline: true,
		})
	}

	tokenFields := []*discordgo.MessageEmbedField{
		{
			Name:  "Tokens Available",
			Value: "",
		},
	}

	ticketFields := []*discordgo.MessageEmbedField{
		{
			Name:  "Participating",
			Value: "",
		},
	}

	i := 0
	for memberId, ticketAmount := range r.Tickets {
		// for every 10 members, make a new field
		if i%10 == 0 && i != 0 {
			tokenFields = append(tokenFields, &discordgo.MessageEmbedField{
				Name:   "",
				Value:  "",
				Inline: true,
			})

			ticketFields = append(ticketFields, &discordgo.MessageEmbedField{
				Name:   "",
				Value:  "",
				Inline: true,
			})
		}

		tokenAmount, err := tokens.GetBalanceByMemberId(memberId)
		if err != nil {
			return nil, err
		}

		tokenField := tokenFields[len(tokenFields)-1]
		tokenField.Value += fmt.Sprintf("<@%s> | %d\n", memberId, tokenAmount)

		ticketField := ticketFields[len(ticketFields)-1]
		ticketField.Value += fmt.Sprintf("<@%s>", memberId)

		if len(r.Winners) != 0 {
			ticketField.Value += " | " + fmt.Sprintf("%d", ticketAmount)
		}

		// if not the 10th, add a new line
		if i%10 != 9 {
			ticketField.Value += "\n"
		}

		i++
	}

	if !r.Ended {
		feilds = append(feilds, tokenFields...)
	}

	feilds = append(feilds, ticketFields...)

	if r.Ended && len(r.Winners) > 0 {
		name := "🥳 Winner 🎉"
		if r.Quantity > 1 {
			name = "🥳 Winners 🎉"
		}
		var winnerIds strings.Builder
		for i, winnerId := range r.Winners {
			winnerIds.WriteString("<@")
			winnerIds.WriteString(winnerId)
			winnerIds.WriteString("> ")
			if i < len(r.Winners)-2 {
				winnerIds.WriteString(", ")
			}
		}
		feilds = append(feilds, &discordgo.MessageEmbedField{
			Name:   name,
			Value:  winnerIds.String(),
			Inline: false,
		})
	}

	title := r.Name + " Raffle"
	if r.Test {
		title = "[TEST] " + title
	}

	embed := &discordgo.MessageEmbed{
		Title:  title,
		Fields: feilds,
	}

	if r.AttedanceId != "" {
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text: "Attendance ID: " + r.AttedanceId,
		}
	}

	return embed, nil
}
