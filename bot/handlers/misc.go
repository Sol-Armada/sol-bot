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
)

func ProfileCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	member, err := members.Get(i.Member.User.ID)
	if err != nil {
		log.WithError(err).Error("getting user from storage")
		return
	}

	if member.Name == "" {
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
			Value:  member.Name,
			Inline: false,
		},
		{
			Name:   "Rank",
			Value:  member.Rank.String(),
			Inline: true,
		},
		{
			Name:   "Event Attendance Count",
			Value:  fmt.Sprintf("%d", member.Events),
			Inline: true,
		},
	}

	if member.RSIMember {
		rsiFields := []*discordgo.MessageEmbedField{
			{
				Name:   "RSI Profile URL",
				Value:  fmt.Sprintf("https://robertsspaceindustries.com/citizens/%s", member.Name),
				Inline: false,
			},
			{
				Name:   "RSI Validated (/validate)",
				Value:  strconv.FormatBool(member.Validated),
				Inline: false,
			},
		}
		emFields = append(emFields, rsiFields...)
	}

	if len(issues(member)) > 0 {
		emFields = append(emFields, &discordgo.MessageEmbedField{
			Name:   "Restrictions to Promotion",
			Value:  strings.Join(issues(member), ", "),
			Inline: false,
		})
	}

	em := &discordgo.MessageEmbed{
		Title:       "Profile",
		Description: "Information about you in Sol Armada",
		Color:       0x00FFFF,
		Fields:      emFields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Last updated <t:%d:R>", member.Updated.Unix()),
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
