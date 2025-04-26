package giveawayhandler

import (
	"context"
	"strconv"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/xid"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/giveaway"
	"github.com/sol-armada/sol-bot/utils"
)

func start(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("giveaway start command")

	if !utils.Allowed(i.Member, "GIVEAWAYS") {
		return customerrors.InvalidPermissions
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		return err
	}

	data := i.ApplicationCommandData()

	items := make([]*giveaway.Item, 0)
	attendanceId := ""
	timer := 0
	for _, opt := range data.Options {
		if opt.Name == "event" {
			attendanceId = opt.StringValue()
			continue
		}

		if opt.Name == "timer" {
			timer = int(opt.FloatValue())
			if timer <= 0 {
				customerrors.ErrorResponse(s, i.Interaction, "Invalid giveaway timer. Please use whole numbers only and greater than 0", nil)
				return nil
			}
			continue
		}

		itemValue := strings.Split(opt.StringValue(), ":")
		amount := 1
		if len(itemValue) == 2 {
			a, err := strconv.Atoi(itemValue[1])
			if err != nil {
				customerrors.ErrorResponse(s, i.Interaction, "Invalid giveaway item count. Please use whole numbers only", nil)
				return nil
			}
			amount = a
		}

		item := &giveaway.Item{
			Id:     xid.New().String(),
			Name:   itemValue[0],
			Amount: amount,
		}

		if item.Name == "NONE" {
			continue
		}

		items = append(items, item)
	}

	g, err := giveaway.NewGiveaway(attendanceId, items)
	if err != nil {
		return err
	}
	if g == nil {
		customerrors.ErrorResponse(s, i.Interaction, "Invalid giveaway", nil)
		return nil
	}
	g.SetTimer(timer)
	g.ChannelId = i.ChannelID

	msg, err := s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{g.GetEmbed()},
	})
	if err != nil {
		return err
	}
	g.EmbedMessageId = msg.ID

	msg, err = s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
		Components: g.GetComponents(),
	})
	if err != nil {
		return err
	}
	g.InputMessageId = msg.ID

	g.Save()

	if err := s.ChannelMessagePin(i.ChannelID, msg.ID); err != nil {
		return err
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: utils.ToPointer("Giveaway started!"),
	}); err != nil {
		return err
	}

	g.StartTimer(s)

	return nil
}
