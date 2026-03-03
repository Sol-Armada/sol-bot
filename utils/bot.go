package utils

import (
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/settings"
)

func Allowed(discordMember *discordgo.Member, feature string) bool {
	return StringSliceContainsOneOf(discordMember.Roles, settings.GetConfigSlice("FEATURES_"+feature+"_ALLOWED_ROLES"))
}
