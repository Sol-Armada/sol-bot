package attendancehandler

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/tokens"
	"golang.org/x/sync/errgroup"
)

func directMessageAttendees(s *discordgo.Session, logger *slog.Logger, a *attendance.Attendance) {
	semaphore := make(chan struct{}, 5)
	errGroup := new(errgroup.Group)

	participants, err := a.Participants()
	if err != nil {
		logger.Error("error getting participants", slog.Any("error", err))
		return
	}

	for _, participant := range participants {
		semaphore <- struct{}{}

		errGroup.Go(func() error {
			defer func() { <-semaphore }()

			member := participant.Member

			if member.DmOptOut {
				return nil
			}

			dm, err := s.UserChannelCreate(member.Id)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("creating DM channel for member %s", member.Id))
			}

			attdncTokenRecords, err := tokens.GetByMemberIdAndAttendanceId(member.Id, a.Id)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("getting tokens for attendance %s", a.Id))
			}

			attdncTokens := 0
			for _, record := range attdncTokenRecords {
				attdncTokens += record.Amount
			}

			totalTokens, err := tokens.GetBalanceByMemberId(member.Id)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("getting total tokens for member %s", member.Id))
			}

			attdncCount, err := attendance.GetMemberAttendanceCount(member.Id)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("getting attendance count for member %s", member.Id))
			}

			var msg strings.Builder
			fmt.Fprintf(&msg, "You have been recorded as attending the event `%s` on %s", a.Name, a.DateCreated.Format("Jan 2, 2006"))

			if attdncTokens > 0 {
				fmt.Fprintf(&msg, " and have earned %d tokens for this event", attdncTokens)
			}

			if _, err = s.ChannelMessageSendComplex(dm.ID, &discordgo.MessageSend{
				Content: msg.String(),
				Embeds: []*discordgo.MessageEmbed{
					{
						Title: "Profile Details",
						Fields: []*discordgo.MessageEmbedField{
							{
								Name:   "Total Events Attended",
								Value:  fmt.Sprintf("%d", attdncCount),
								Inline: true,
							},
							{
								Name:   "Total Tokens",
								Value:  fmt.Sprintf("%d", totalTokens),
								Inline: true,
							},
						},
					},
				},
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.Button{
								Label:    "Opt out of DMs",
								Style:    discordgo.PrimaryButton,
								CustomID: "settings:dm_opt_out",
							},
						},
					},
				},
			}); err != nil {
				return errors.Wrap(err, fmt.Sprintf("sending DM to member %s", member.Id))
			}
			return nil
		})
	}
	if err := errGroup.Wait(); err != nil {
		logger.Error("error sending DMs", slog.Any("error", err))
	}
}
