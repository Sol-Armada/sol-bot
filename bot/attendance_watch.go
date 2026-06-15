package bot

import (
	"context"
	"slices"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/database/dbnotify"
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
	id := event.PrimaryKey["id"].(string)
	a, err := attendance.Get(id)
	if err != nil {
		return err
	}

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
		return s == "message_id" || s == "updated_at"
	})

	if len(event.ChangedColumns) == 0 {
		return nil
	}

	id := event.PrimaryKey["id"].(string)
	a, err := attendance.Get(id)
	if err != nil {
		return err
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
