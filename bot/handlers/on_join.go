package handlers

import (
	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/admin/members"
	"github.com/sol-armada/admin/rsi"
)

func OnJoinHandler(s *discordgo.Session, i *discordgo.GuildMemberAdd) {
	logger := log.WithFields(log.Fields{
		"guild":   i.GuildID,
		"user":    i.User.ID,
		"handler": "OnJoinHandler",
	})

	logger.Info("User joined")

	member := members.New(i.Member)

	var err error
	member, err = rsi.UpdateRsiInfo(member)
	if err != nil {
		logger.WithError(err).Error("updating rsi info")
		return
	}

	if err := member.Save(); err != nil {
		logger.WithError(err).Error("saving member")
		return
	}
}
