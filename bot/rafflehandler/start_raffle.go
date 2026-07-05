package rafflehandler

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/raffles"
	"github.com/sol-armada/sol-bot/utils"
)

func start(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("raffle start command")

	data := i.ApplicationCommandData()

	name := data.GetOption("name").Value.(string)

	attendanceRecordId := data.Options[0].Value.(string)
	prize := data.Options[1].Value.(string)
	test := false
	if len(data.Options) > 2 {
		test = data.Options[2].Value.(bool)
	}

	a, _ := attendance.Get(attendanceRecordId)
	if a != nil {
		name = a.Name
	} else {
		attendanceRecordId = ""
	}

	raffle, err := raffles.New(name, attendanceRecordId, prize, test)
	if err != nil {
		return err
	}

	embed, err := raffle.GetEmbed()
	if err != nil {
		return err
	}

	message, err := s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						CustomID: fmt.Sprintf("raffle:add_entries:%d", raffle.Id),
						Label:    "Edit Entries",
						Style:    discordgo.PrimaryButton,
					},
					discordgo.Button{
						CustomID: fmt.Sprintf("raffle:back_out:%d", raffle.Id),
						Label:    "Back Out",
						Style:    discordgo.SecondaryButton,
					},
					discordgo.Button{
						CustomID: fmt.Sprintf("raffle:entries:%d", raffle.Id),
						Label:    "Entries",
						Style:    discordgo.SecondaryButton,
					},
					discordgo.Button{
						CustomID: fmt.Sprintf("raffle:end:%d", raffle.Id),
						Label:    "End",
						Style:    discordgo.SecondaryButton,
					},
					discordgo.Button{
						CustomID: fmt.Sprintf("raffle:cancel:%d", raffle.Id),
						Label:    "Cancel",
						Style:    discordgo.DangerButton,
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	raffle.ChannelId = message.ChannelID
	raffle.MessageId = message.ID

	if err := raffle.Save(ctx); err != nil {
		return err
	}

	return s.InteractionResponseDelete(i.Interaction)
}
