package rafflehandler

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/raffles"
	"github.com/sol-armada/sol-bot/utils"
)

func start(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("raffle start command")

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		return err
	}

	data := i.ApplicationCommandData()

	attendanceRecordId := data.Options[0].Value.(string)
	prize := data.Options[1].Value.(string)

	name := attendanceRecordId
	a, _ := attendance.Get(attendanceRecordId)
	if a != nil {
		name = a.Name + " Raffle"
	} else {
		attendanceRecordId = ""
	}

	raffle := raffles.New(name, attendanceRecordId, prize)

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
						CustomID: "raffle:add_entries:" + raffle.Id,
						Label:    "Add Entries",
						Style:    discordgo.PrimaryButton,
					},
					discordgo.Button{
						CustomID: "raffle:back_out:" + raffle.Id,
						Label:    "Back Out",
						Style:    discordgo.SecondaryButton,
					},
					discordgo.Button{
						CustomID: "raffle:end:" + raffle.Id,
						Label:    "End",
						Style:    discordgo.SecondaryButton,
					},
					discordgo.Button{
						CustomID: "raffle:cancel:" + raffle.Id,
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

	if err := raffle.SetMessage(message).Save(); err != nil {
		return err
	}

	if _, err := s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Content: "Raffle started!",
		Flags:   discordgo.MessageFlagsEphemeral,
	}); err != nil {
		return err
	}

	return nil
}
