package giveawayhandler

import (
	"context"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/xid"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/giveaway"
	"github.com/sol-armada/sol-bot/utils"
)

func start(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
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
	name := ""
	timer := 0
	for _, opt := range data.Options {
		if opt.Name == "name" {
			name = opt.StringValue()
			continue
		}

		if opt.Name == "timer" {
			str := opt.StringValue()

			dur, err := utils.StringToDuration(str)
			if err != nil {
				return err
			}

			timer = int(dur.Minutes())
			if timer < 0 {
				customerrors.ErrorResponse(s, i.Interaction, "Invalid timer format. Please use the format 24h15m10s", nil)
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

	attendanceId := ""
	a, err := attendance.Get(name)
	if err == nil {
		attendanceId = a.Id
		name = a.Name
	}

	g, err := giveaway.NewGiveaway(s, name, attendanceId, items)
	if err != nil {
		return err
	}
	if g == nil {
		customerrors.ErrorResponse(s, i.Interaction, "Invalid giveaway", nil)
		return nil
	}
	g.SetEndTime(timer)
	g.ChannelId = i.ChannelID

	msg, err := s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{g.GetEmbed()},
	})
	if err != nil {
		return err
	}
	g.EmbedMessageId = msg.ID

	if err := s.ChannelMessagePin(i.ChannelID, msg.ID); err != nil {
		return err
	}

	msg, err = s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
		Components: g.GetComponents(),
	})
	if err != nil {
		return err
	}
	g.InputMessageId = msg.ID

	g.Save()

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content:new("Giveaway started!"),
	}); err != nil {
		return err
	}

	return nil
}
