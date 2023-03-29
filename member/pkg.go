package member

import "github.com/bwmarrin/discordgo"

type Member struct {
	EventCount int `json:"event_count"`

	*discordgo.Member
}
