package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/admin/stores"
	"github.com/sol-armada/admin/users"
)

func AttendanceCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	storedUser := &users.User{}
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
	user, err := users.Get(i.Member.User.ID)
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
