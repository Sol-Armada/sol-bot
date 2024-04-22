package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/members"
	"github.com/sol-armada/admin/ranks"
	"github.com/sol-armada/admin/stores"
)

func AttendanceCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	storedUser := &members.Member{}
	if err := stores.Users.Get(i.Member.User.ID).Decode(&storedUser); err != nil {
		log.WithError(err).Error("getting user from storage")
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("This is a depricated command and will be removed in the future, please use `/profile` instead.\n\n%d events", storedUser.Events),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		log.WithError(err).Error("responding to attendance command interaction")
	}
}

func ProfileCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user, err := members.Get(i.Member.User.ID)
	if err != nil {
		log.WithError(err).Error("getting user from storage")
		return
	}

	if user.Name == "" {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You have not been onboarded yet! Contact an @Officer for some help!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		}); err != nil {
			log.WithError(err).Error("responding to attendance command interaction")
		}
		return
	}

	emFields := []*discordgo.MessageEmbedField{
		{
			Name:   "RSI Handle",
			Value:  user.Name,
			Inline: false,
		},
		{
			Name:   "Rank",
			Value:  user.Rank.String(),
			Inline: true,
		},
		{
			Name:   "Event Attendance Count",
			Value:  fmt.Sprintf("%d", user.Events),
			Inline: true,
		},
	}

	if user.RSIMember {
		rsiFields := []*discordgo.MessageEmbedField{
			{
				Name:   "RSI Profile URL",
				Value:  fmt.Sprintf("https://robertsspaceindustries.com/citizens/%s", user.Name),
				Inline: false,
			},
			{
				Name:   "RSI Validated (/validate)",
				Value:  strconv.FormatBool(user.Validated),
				Inline: false,
			},
		}
		emFields = append(emFields, rsiFields...)
	}

	if len(user.Issues()) > 0 {
		emFields = append(emFields, &discordgo.MessageEmbedField{
			Name:   "Restrictions to Promotion",
			Value:  strings.Join(user.Issues(), ", "),
			Inline: false,
		})
	}

	em := &discordgo.MessageEmbed{
		Title:       "Profile",
		Description: "Information about you in Sol Armada",
		Color:       0x00FFFF,
		Fields:      emFields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Last updated %s UTC", user.Updated.Format("2006-01-02 15:04:05")),
		},
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
			Embeds: []*discordgo.MessageEmbed{
				em,
			},
		},
	}); err != nil {
		log.WithError(err).Error("responding to attendance command interaction")
	}
}

func GiveMeritCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user, err := members.Get(i.Member.User.ID)
	if err != nil {
		log.WithError(err).Error("getting user")
		return
	}

	if user.Rank > ranks.GetRankByName(config.GetStringWithDefault("FEATURES.ATTENDANCE.MIN_RANK", "admiral")) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "You do not have permission to use this command",
			},
		}); err != nil {
			log.WithError(err).Error("responding to onboarding command")
			return
		}
		return
	}

	data := i.ApplicationCommandData()

	receivingDiscordUser := data.Options[0].UserValue(s)

	receivingUser, err := members.Get(receivingDiscordUser.ID)
	if err != nil {
		log.WithError(err).Error("getting receiving user")
		return
	}

	if err := receivingUser.GiveMerit(data.Options[1].StringValue(), user); err != nil {
		log.WithError(err).Error("giving user merit")
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Gave %s the merit!", receivingDiscordUser.Username),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		log.WithError(err).Error("responding to onboarding command")
	}
}

func GiveDemeritCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user, err := members.Get(i.Member.User.ID)
	if err != nil {
		log.WithError(err).Error("getting user")
		return
	}

	if user.Rank > ranks.GetRankByName(config.GetStringWithDefault("FEATURES.ATTENDANCE.MIN_RANK", "admiral")) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "You do not have permission to use this command",
			},
		}); err != nil {
			log.WithError(err).Error("responding to onboarding command")
			return
		}
		return
	}

	data := i.ApplicationCommandData()

	receivingDiscordUser := data.Options[0].UserValue(s)

	receivingUser, err := members.Get(receivingDiscordUser.ID)
	if err != nil {
		log.WithError(err).Error("getting receiving user")
		return
	}

	if err := receivingUser.GiveDemerit(data.Options[1].StringValue(), user); err != nil {
		log.WithError(err).Error("giving user merit")
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Gave %s the demerit!", receivingDiscordUser.Username),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		log.WithError(err).Error("responding to onboarding command")
	}
}
