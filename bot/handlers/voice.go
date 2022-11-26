package handlers

import (
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
)

func OnVoiceJoin(s *discordgo.Session, i *discordgo.VoiceStateUpdate) {
	// see if they need to be added to attendance
	if activeEvent != nil && i.VoiceState.ChannelID != "" {
		for index, row := range activeEvent.Components {
			components := row.(*discordgo.ActionsRow).Components
			// if this row is full, just skip
			if len(components) >= 5 {
				continue
			}

			// if they are already in the list, we don't need to do anything
			for _, component := range components {
				if strings.Contains(component.(*discordgo.Button).CustomID, i.UserID) {
					return
				}
			}

			label := i.VoiceState.Member.User.Username
			if i.VoiceState.Member.Nick != "" {
				label = i.VoiceState.Member.Nick
			}
			components = append(components, discordgo.Button{
				Label:    label,
				CustomID: "event:attendance:toggle:" + i.VoiceState.Member.User.ID,
				Style:    discordgo.PrimaryButton,
			})

			activeEvent.Components[index].(*discordgo.ActionsRow).Components = components
			break
		}

		messageEdit := discordgo.NewMessageEdit(activeEvent.ChannelID, activeEvent.ID)
		messageEdit.Components = activeEvent.Components

		if _, err := s.ChannelMessageEditComplex(messageEdit); err != nil {
			log.WithError(err).Error("updating active event message on voice channel join")
		}
	}
}
