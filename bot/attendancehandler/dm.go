package attendancehandler

import (
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/attendance"
	"golang.org/x/sync/errgroup"
)

func directMessageAttendees(s *discordgo.Session, logger *slog.Logger, attendance *attendance.Attendance) {
	semaphore := make(chan struct{}, 5)
	errGroup := new(errgroup.Group)

	for _, member := range attendance.Members {
		semaphore <- struct{}{}

		errGroup.Go(func() error {
			defer func() { <-semaphore }()

			message := fmt.Sprintf("Hello <@%s>,\n\nYou have been marked as attending the event. If you have any questions, please contact the event manager.", member.Id)

			dm, err := s.UserChannelCreate(member.Id)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("creating DM channel for member %s", member.Id))
			}

			if _, err = s.ChannelMessageSend(dm.ID, message); err != nil {
				return errors.Wrap(err, fmt.Sprintf("sending DM to member %s", member.Id))
			}
			return nil
		})
	}
	if err := errGroup.Wait(); err != nil {
		logger.Error("error sending DMs", slog.Any("error", err))
	}
}
