package bot

import (
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/members"
)

func onLeaveHandler(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
	log.Debug("member left")

	if m.User.Bot {
		return
	}

	member, err := members.Get(m.User.ID)
	if err != nil && !errors.Is(err, members.MemberNotFound) {
		log.WithError(err).Error("getting member")
		return
	}

	now := time.Now().UTC()
	member.LeftAt = &now

	_ = member.Save()

	memberMessage := member.GetOnboardingMessage()
	if _, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: member.ChannelId,
		ID:      member.MessageId,
		Content: &memberMessage.Content,
		Embeds:  &memberMessage.Embeds,
	}); err != nil {
		log.WithError(err).Error("editing member message on leave")
	}
}
