package bot

import (
	"log/slog"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/activity"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/settings"
)

func onVoiceUpdate(_ *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	if v.Member.User.Bot {
		return
	}

	member, err := members.Get(v.Member.User.ID)
	if err != nil {
		if !errors.Is(err, members.MemberNotFound) {
			slog.Error("getting member", "error", err)
			return
		}

		member = members.New(v.Member)
	}

	what := activity.Unknown
	var where *string
	switch {
	case v.BeforeUpdate == nil && v.VoiceState.ChannelID != "":
		what = activity.VoiceJoin
		where = &v.VoiceState.ChannelID
	case v.VoiceState.ChannelID == "":
		what = activity.VoiceLeave
	case v.BeforeUpdate != nil && settings.GetString("FEATURES.ACTIVITY_TRACKING.AFK_CHANNEL_ID") == v.VoiceState.ChannelID:
		what = activity.VoiceAFK
		where = &v.VoiceState.ChannelID
	case v.BeforeUpdate != nil && v.VoiceState.ChannelID != "":
		what = activity.VoiceSwitch
		where = &v.VoiceState.ChannelID
	}

	newActivity := activity.Activity{
		Who:  member,
		When: time.Now().UTC(),
		Meta: activity.Meta{
			What:  what,
			Where: where,
		},
	}
	if err = newActivity.Save(); err != nil {
		log.WithError(err).Error("saving activity")
	}
}
