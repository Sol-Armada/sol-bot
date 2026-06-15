package bot

import (
	"context"

	"github.com/sol-armada/sol-bot/database/dbnotify"
)

func (b *Bot) StartAttendanceWatch(ctx context.Context) {
	b.dbListener.Run(ctx, func(ctx context.Context, event dbnotify.Event) error {
		switch event.Channel {
		case dbnotify.ChannelAttendance:
			return b.handleAttendanceEvent(ctx, event)
		default:
			return nil
		}
	})
}

func (b *Bot) handleAttendanceEvent(ctx context.Context, event dbnotify.Event) error {
	switch event.Operation {
	case dbnotify.Insert:
		return b.handleAttendanceInsert(ctx, event)
	// case dbnotify.Update:
	// 	return b.handleAttendanceUpdate(ctx, event)
	// case dbnotify.Delete:
	// 	return b.handleAttendanceDelete(ctx, event)
	default:
		return nil
	}
}

func (b *Bot) handleAttendanceInsert(ctx context.Context, event dbnotify.Event) error {
	// attandanceMessage, err := a.ToDiscordMessage()
	// if err != nil {
	// 	return errors.Wrap(err, "creating attendance message")
	// }

	// message, err := s.ChannelMessageSendComplex(a.ChannelId, attandanceMessage)
	// if err != nil {
	// 	return errors.Wrap(err, "sending attendance message")
	// }
	// a.MessageId = message.ID
	return nil
}
