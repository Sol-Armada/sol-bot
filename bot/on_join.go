package bot

import (
	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/admin/members"
	"github.com/sol-armada/admin/rsi"
)

func onJoinHandler(s *discordgo.Session, i *discordgo.GuildMemberAdd) {
	logger := log.WithFields(log.Fields{
		"guild":   i.GuildID,
		"user":    i.User.ID,
		"handler": "OnJoinHandler",
	})

	logger.Info("User joined")

	if i.Member.User.Bot {
		logger.Debug("skipping bot")
		return
	}

	member := members.New(i.Member)

	var err error
	member, err = rsi.UpdateRsiInfo(member)
	if err != nil && !errors.Is(err, rsi.UserNotFound) {
		logger.WithError(err).Error("updating rsi info")
	}

	if err := member.Save(); err != nil {
		logger.WithError(err).Error("saving member")
		return
	}
}
