package utils

import (
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/settings"
)

func Allowed(discordMember *discordgo.Member, feature string) bool {
	return StringSliceContainsOneOf(discordMember.Roles, settings.GetStringSlice("FEATURES."+feature+".ALLOWED_ROLES"))
}
