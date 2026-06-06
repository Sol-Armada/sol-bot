package loothandler

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/rolls"
	"github.com/sol-armada/sol-bot/utils"
)

func start(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("loot start command")

	data := i.ApplicationCommandData()

	items := make([]*rolls.RollItem, 0)
	timer := 0
	for _, opt := range data.Options {
		if opt.Name == "name" {
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

		rawItemValue := opt.StringValue()
		if rawItemValue == "NONE" || rawItemValue == "" {
			continue
		}

		itemName := rawItemValue
		itemAmount := 1

		split := strings.SplitN(rawItemValue, ":", 2)
		if len(split) > 0 {
			itemName = strings.TrimSpace(split[0])
		}
		if len(split) > 1 {
			amount, err := strconv.Atoi(strings.TrimSpace(split[1]))
			if err != nil || amount <= 0 {
				customerrors.ErrorResponse(s, i.Interaction, "Invalid loot item count. Please use whole numbers only", nil)
				return nil
			}
			itemAmount = amount
		}

		if itemName == "" {
			continue
		}

		items = append(items, rolls.NewItem("", itemName, itemAmount, len(items)))
	}

	if len(items) == 0 {
		customerrors.ErrorResponse(s, i.Interaction, "Please provide at least one item", nil)
		return nil
	}

	name := data.GetOption("name").StringValue()
	var attendanceId *string
	a, err := attendance.Get(name)
	if err == nil {
		attendanceId = &a.Id
		name = a.Name
	}

	endTime := time.Now().UTC().Add(time.Duration(timer) * time.Minute)
	event := rolls.NewEvent(name, attendanceId, &endTime)
	if event == nil {
		customerrors.ErrorResponse(s, i.Interaction, "Invalid roll event", nil)
		return nil
	}

	event.ChannelId = i.ChannelID
	if err := event.Save(); err != nil {
		return err
	}

	primaryMessage, err := s.ChannelMessageSendComplex(i.ChannelID, buildEventPrimaryMessage(event, items))
	if err != nil {
		return err
	}

	event.ChannelId = primaryMessage.ChannelID
	event.EmbedMessageId = primaryMessage.ID
	if err := event.Save(); err != nil {
		return err
	}

	// Create roll items and send one message per item with need/greed/pass buttons.
	for idx, item := range items {
		item.RollEventId = event.Id
		item.SortOrder = idx
		if err := item.Save(); err != nil {
			logger.Error("failed to save roll item", "error", err)
			return err
		}

		message := buildItemMessage(event, item, itemChoiceCounts{})
		message.Reference = &discordgo.MessageReference{
			ChannelID: i.ChannelID,
			MessageID: primaryMessage.ID,
		}
		itemMessage, err := s.ChannelMessageSendComplex(i.ChannelID, message)
		if err != nil {
			return err
		}
		item.ChannelId = itemMessage.ChannelID
		item.MessageId = itemMessage.ID
		if err := item.Save(); err != nil {
			logger.Error("failed to update roll item with message info", "error", err)
			return err
		}
	}

	return s.InteractionResponseDelete(i.Interaction)
}
