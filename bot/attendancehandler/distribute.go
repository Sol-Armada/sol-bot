package attendancehandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/attendance"
)

func distributeButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	attendanceId := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	a, err := attendance.Get(attendanceId)
	if err != nil {
		return err
	}

	participants, err := a.Participants()
	if err != nil {
		return err
	}

	fromStartParticipants, err := a.PraticipantsFromStart()
	if err != nil {
		return err
	}

	components := []discordgo.MessageComponent{
		discordgo.Label{
			Label: "Select members who were event managers",
			Component: discordgo.SelectMenu{
				CustomID: "managers",
				Options: func() []discordgo.SelectMenuOption {
					opts := make([]discordgo.SelectMenuOption, len(participants))
					for i, participant := range participants {
						opts[i] = discordgo.SelectMenuOption{
							Label: participant.Member.Name,
							Value: participant.Member.Id,
						}
					}
					return opts
				}(),
				MaxValues: len(participants),
			},
		},
	}

	if len(fromStartParticipants) > 0 {
		components = append(components,
			discordgo.Label{
				Label: "Select members who were at the start",
				Component: discordgo.SelectMenu{
					CustomID: "from_start",
					Options: func() []discordgo.SelectMenuOption {
						opts := make([]discordgo.SelectMenuOption, len(fromStartParticipants))
						for i, participant := range fromStartParticipants {
							opts[i] = discordgo.SelectMenuOption{
								Label: participant.Member.Name,
								Value: participant.Member.Id,
							}
						}
						return opts
					}(),
					MaxValues: len(fromStartParticipants),
				},
			},
		)
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   fmt.Sprintf("attendance:distribute:%s", attendanceId),
			Title:      "Distribute Tokens",
			Components: components,
		},
	})
}

func distributeModalHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	attendanceId := strings.Split(i.ModalSubmitData().CustomID, ":")[2]

	a, err := attendance.Get(attendanceId)
	if err != nil {
		return err
	}

	data := i.ModalSubmitData()

	var managersSelect, fromStartSelect *discordgo.SelectMenu

	for _, component := range data.Components {
		if label, ok := component.(*discordgo.Label); ok {
			if selectComponent, ok := label.Component.(*discordgo.SelectMenu); ok {
				switch selectComponent.CustomID {
				case "managers":
					managersSelect = selectComponent
				case "from_start":
					fromStartSelect = selectComponent
				}
			}
		}
	}

	selectedManagers := make([]string, 0, len(managersSelect.Values))
	selectedManagers = append(selectedManagers, managersSelect.Values...)

	for _, managerId := range selectedManagers {
		if err := a.SetParticipantManager(managerId); err != nil {
			return err
		}
	}

	selectedMembers := make([]string, 0)
	if fromStartSelect != nil {
		selectedMembers = append(selectedMembers, fromStartSelect.Values...)
	}

	for _, memberId := range selectedMembers {
		if err := a.SetParticipantStayedUntilEnd(memberId); err != nil {
			return err
		}
	}

	distributedTo, err := a.DistributeTokens()
	if err != nil {
		return err
	}

	attendanceMessage, err := a.ToDiscordMessage()
	if err != nil {
		return err
	}

	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    a.ChannelId,
		ID:         a.MessageId,
		Components: &attendanceMessage.Components,
		Embeds:     &attendanceMessage.Embeds,
	}); err != nil {
		return err
	}

	if distributedTo == "" {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
		})
	}

	msg, err := s.ChannelMessage(a.ChannelId, a.MessageId)
	if err != nil {
		return err
	}

	ch := msg.Thread
	if ch == nil {
		ch, err = s.MessageThreadStartComplex(a.ChannelId, a.MessageId, &discordgo.ThreadStart{
			Name:                "Thread for " + a.Name + " (" + a.Id + ")",
			Type:                discordgo.ChannelTypeGuildPublicThread,
			Invitable:           true,
			AutoArchiveDuration: 1440,
		})
		if err != nil {
			return err
		}
	}

	if _, err := s.ChannelMessageSend(ch.ID, distributedTo); err != nil {
		return err
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
	})
}
