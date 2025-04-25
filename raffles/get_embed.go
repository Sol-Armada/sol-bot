package raffles

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/tokens"
)

func (r *Raffle) GetEmbed() (*discordgo.MessageEmbed, error) {
	attendanceRecord, err := attendance.Get(r.AttedanceId)
	if err != nil {
		return nil, err
	}

	feilds := []*discordgo.MessageEmbedField{
		{
			Name:   "Event",
			Value:  attendanceRecord.Name,
			Inline: true,
		},
		{
			Name:   "Prize",
			Value:  r.Prize,
			Inline: true,
		},
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

		if r.WinnerId != "" {
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

	if r.Ended {
		winner, err := members.Get(r.WinnerId)
		if err != nil {
			return nil, err
		}

		feilds = append(feilds, &discordgo.MessageEmbedField{
			Name:   "🥳 Winner 🎉",
			Value:  fmt.Sprintf("<@%s>", winner.Id),
			Inline: false,
		})
	}

	return &discordgo.MessageEmbed{
		Title:  "Raffle",
		Fields: feilds,
	}, nil
}
