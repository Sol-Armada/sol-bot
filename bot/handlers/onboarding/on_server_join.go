package onboarding

import (
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/admin/config"
)

func JoinServerHandler(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	if config.GetBoolWithDefault("FEATURES.ONBOARDING", false) {
		onboarding(s, m.Member)
	}
}
