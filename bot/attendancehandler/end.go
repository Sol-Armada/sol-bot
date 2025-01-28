package attendancehandler

import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/attendance"
)

func endEventButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	recordId := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	record, err := attendance.Get(recordId)
	if err != nil {
		return err
	}

	if err := record.Record(); err != nil {
		return err
	}

	recordMessage := record.ToDiscordMessage()

	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    record.ChannelId,
		ID:         record.MessageId,
		Components: &recordMessage.Components,
		Embed:      recordMessage.Embed,
	}); err != nil {
		return err
	}

	fromStartMembers := make([]discordgo.SelectMenuOption, 0, len(record.FromStart))

	for _, member := range record.FromStart {
		fromStartMembers = append(fromStartMembers, discordgo.SelectMenuOption{
			Label: member.Name,
			Value: member.Id,
		})
	}

	if len(fromStartMembers) >= 0 {
		ch, err := s.ThreadStartComplex(record.ChannelId, &discordgo.ThreadStart{
			Name:                "Attendance for " + record.Name,
			Type:                discordgo.ChannelTypeGuildPrivateThread,
			Invitable:           true,
			AutoArchiveDuration: 1440,
		})
		if err != nil {
			return err
		}

		if err := s.ThreadMemberAdd(ch.ID, i.Interaction.Member.User.ID); err != nil {
			return err
		}

		if _, err := s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
			Content: "Please select the members that stayed for the event",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							CustomID:    "attendance:stayed:" + record.Id,
							Options:     fromStartMembers,
							Placeholder: "",
							MaxValues:   len(fromStartMembers),
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{

						discordgo.Button{
							Label:    "Submit",
							Style:    discordgo.SuccessButton,
							CustomID: "attendance:stayed_submit:" + record.Id,
						},
					},
				},
			},
		}); err != nil {
			return err
		}
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
		Data: &discordgo.InteractionResponseData{},
	})
}
