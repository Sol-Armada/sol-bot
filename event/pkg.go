package event

import (
	"github.com/bwmarrin/discordgo"
	"github.com/rs/xid"
)

type Event struct {
	Id string `json:"id"`

	OriginalMessage *discordgo.Message
	PlayerAttedance map[string]bool
}

var CurrentEvent *Event

func New(message *discordgo.Message, players []*discordgo.VoiceState) {
	event := &Event{
		Id:              xid.New().String(),
		OriginalMessage: message,
	}
	event.AddPlayersToAttendance(players)

	CurrentEvent = event
}

func (e *Event) AddPlayersToAttendance(players []*discordgo.VoiceState) {
	for _, player := range players {
		if _, ok := e.PlayerAttedance[player.Member.User.ID]; !ok {
			e.PlayerAttedance[player.Member.User.ID] = true
		}
	}

	e.UpdateMessage()
}

func (e *Event) UpdateMessage() {

}
