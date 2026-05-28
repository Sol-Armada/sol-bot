package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/ranks"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/utils"
)

type job struct {
	Name string
	Cron string
	Run  func(context.Context, *discordgo.Session) error
}

var Jobs = []job{
	{
		Name: "Promotions Report",
		Cron: "0 0 * * *",
		Run:  promotionsReport,
	},
}

func promotionsReport(ctx context.Context, s *discordgo.Session) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("promotions report job")

	channelId := settings.GetString("PROMOTIONS_CHANNEL_ID")
	if channelId == "" {
		return fmt.Errorf("promotions channel id not set")
	}

	channelMessages, err := s.ChannelMessages(channelId, 100, "", "", "")
	if err != nil {
		return err
	}

	for _, message := range channelMessages {
		t, err := discordgo.SnowflakeTimestamp(message.ID)
		if err != nil {
			logger.Error("failed to parse message timestamp", "messageId", message.ID, "error", err)
			continue
		}

		// check if message was send since midnight of today
		if t.Before(time.Now().Truncate(24 * time.Hour)) {
			continue
		}

		if message.Author.ID == s.State.User.ID {
			return nil
		}
	}

	// get promotions
	promotions, err := members.ListPromotions()
	if err != nil {
		return err
	}

	if len(promotions) == 0 {
		_, err = s.ChannelMessageSend(channelId, "No promotion actions needed today")
		return err
	}

	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "Members to Rank Up",
			Value:  "",
			Inline: true,
		},
	}
	embed := &discordgo.MessageEmbed{
		Title:  "",
		Fields: fields,
	}

	ind := 0
	for _, promotion := range promotions {
		if promotion.NextRank == 0 {
			continue
		}

		if ind%10 == 0 && ind != 0 {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "Members to Rank Up (continued)",
				Value:  "",
				Inline: true,
			})
		}

		field := fields[len(fields)-1]
		field.Value += fmt.Sprintf("<@%s> to %s (%d Events)", promotion.ID, ranks.Rank(promotion.NextRank).String(), promotion.AttendanceCount)

		// if not the 10th member, add a newline
		if ind%10 != 9 {
			field.Value += "\n"
		}

		ind++
	}

	_, err = s.ChannelMessageSendEmbed(channelId, embed)
	return err
}
