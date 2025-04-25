package bot

import (
	"slices"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/settings"
)

func OnRoleChange(s *discordgo.Session, m *discordgo.GuildMemberUpdate) error {
	memberRoleId := settings.GetString("DISCORD.ROLES.RANKS.MEMBER")
	if !slices.Contains(m.BeforeUpdate.Roles, memberRoleId) && slices.Contains(m.Roles, memberRoleId) {
		member, err := members.Get(m.User.ID)
		if err != nil {
			return err
		}

		member.MemberSince = time.Now().UTC()
		member.IsGuest = false
		member.IsAffiliate = false
		member.IsAlly = false

		return member.Save()
	}

	return nil
}
