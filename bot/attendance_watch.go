package bot

import (
	"context"
	"slices"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/database/dbnotify"
	"github.com/sol-armada/sol-bot/settings"
)

func (b *Bot) StartAttendanceWatch() error {
	return b.dbListener.Run(b.ctx, func(ctx context.Context, event dbnotify.Event) error {
		switch event.Channel {
		case dbnotify.ChannelAttendance:
			return b.handleAttendanceEvent(event)
		default:
			return nil
		}
	})
}

func (b *Bot) handleAttendanceEvent(event dbnotify.Event) error {
	switch event.Operation {
	case dbnotify.Insert:
		return b.handleAttendanceInsert(event)
	case dbnotify.Update:
		return b.handleAttendanceUpdate(event)
	case dbnotify.Delete:
		return b.handleAttendanceDelete(event)
	default:
		return nil
	}
}

func (b *Bot) handleAttendanceInsert(event dbnotify.Event) error {
	id, ok := event.PrimaryKey["id"].(string)
	if !ok {
		return errors.New("attendance record missing primary key")
	}

	a, err := attendance.Get(id)
	if err != nil {
		return err
	}

	a.ChannelId = settings.GetString("FEATURES.ATTENDANCE.CHANNEL_ID")

	attandanceMessage, err := a.ToDiscordMessage()
	if err != nil {
		return errors.Wrap(err, "creating attendance message")
	}

	message, err := b.ChannelMessageSendComplex(a.ChannelId, attandanceMessage)
	if err != nil {
		return errors.Wrap(err, "sending attendance message")
	}
	a.MessageId = message.ID

	return errors.Wrap(a.Save(), "saving inserted attedance record")
}

func (b *Bot) handleAttendanceUpdate(event dbnotify.Event) error {
	event.ChangedColumns = slices.DeleteFunc(event.ChangedColumns, func(s string) bool {
		return s == "updated_at" || s == "date_updated"
	})

	if len(event.ChangedColumns) == 0 || event.PrimaryKey["id"] == nil {
		return nil
	}

	// check if only the message and channel ids were updated, if so, we don't need to update the message
	if len(event.ChangedColumns) == 2 && event.ChangedColumns[0] == "channel_id" && event.ChangedColumns[1] == "message_id" {
		return nil
	}

	id := event.PrimaryKey["id"].(string)
	a, err := attendance.Get(id)
	if err != nil {
		return err
	}

	if a.MessageId == "" || a.ChannelId == "" {
		return nil
	}

	attandanceMessage, err := a.ToDiscordMessage()
	if err != nil {
		return errors.Wrap(err, "creating attendance message")
	}

	if _, err := b.Session.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Embeds:     &attandanceMessage.Embeds,
		Components: &attandanceMessage.Components,
		ID:         a.MessageId,
		Channel:    a.ChannelId,
	}); err != nil {
		return errors.Wrap(err, "editing attendance message")
	}

	return nil
}

func (b *Bot) handleAttendanceDelete(event dbnotify.Event) error {
	id := event.PrimaryKey["id"].(string)
	a, err := attendance.Get(id)
	if err != nil {
		return err
	}

	return errors.Wrap(b.Session.ChannelMessageDelete(a.ChannelId, a.MessageId), "deleting attendance record")
}
