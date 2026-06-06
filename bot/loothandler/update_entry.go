package loothandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/rolls"
	"github.com/sol-armada/sol-bot/utils"
)

func updateEntry(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("loot update_entry handler")

	split := strings.Split(i.MessageComponentData().CustomID, ":")
	if len(split) < 5 {
		return customerrors.InvalidButton
	}

	rollEventId := split[2]
	rollItemId := split[3]
	choice := rolls.Choice(split[4])

	member := utils.GetMemberFromContext(ctx).(*members.Member)

	rollEvent, err := rolls.GetEvent(rollEventId)
	if err != nil {
		return err
	}
	if rollEvent == nil {
		customerrors.ErrorResponse(s, i.Interaction, "Roll event not found", nil)
		return nil
	}

	allowed, err := rollEvent.CanParticipate(member.Id)
	if err != nil {
		return err
	}

	if !allowed {
		customerrors.ErrorResponse(s, i.Interaction, "You did not attend this event! You don't qualify for this roll.", nil)
		return nil
	}

	if rollEvent.Ended {
		customerrors.ErrorResponse(s, i.Interaction, "This roll has already ended.", nil)
		return nil
	}

	entry, err := rolls.NewEntry(rollEventId, rollItemId, member.Id, choice)
	if err != nil {
		return err
	}

	if err := entry.Save(); err != nil {
		logger.Error("failed to save roll entry", "error", err)
		return err
	}

	items, err := rollEvent.Items()
	if err != nil {
		return err
	}

	var item *rolls.RollItem
	for _, it := range items {
		if it.Id == rollItemId {
			item = it
			break
		}
	}
	if item == nil {
		return customerrors.InvalidButton
	}

	entries, err := rolls.ListEntriesByItem(rollItemId)
	if err != nil {
		return err
	}

	counts := itemChoiceCounts{}
	for _, itemEntry := range entries {
		switch itemEntry.Choice {
		case rolls.ChoiceNeed:
			counts.Need++
		case rolls.ChoiceGreed:
			counts.Greed++
		}
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    fmt.Sprintf("<@%s> selected **%s**", member.Id, strings.ToUpper(string(choice))),
			Embeds:     []*discordgo.MessageEmbed{buildItemEmbed(rollEvent, item)},
			Components: buildItemComponents(rollEventId, rollItemId, rollEvent.Ended, counts),
		},
	})
}
