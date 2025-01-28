package attendancehandler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/utils"
)

func addPayoutButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("add payout button handler")

	if !allowed(i.Member, "ATTENDANCE") {
		return customerrors.InvalidPermissions
	}

	attendanceId := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			Title:    "Add Payout",
			CustomID: fmt.Sprintf("attendance:payout:%s", attendanceId),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "total_payout",
							Placeholder: "Ex. 1000000",
							MaxLength:   100,
							Required:    true,
							Label:       "Total Payout",
							Style:       discordgo.TextInputShort,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{

						discordgo.TextInput{
							CustomID:    "per_member",
							Placeholder: "Ex. 50000",
							MaxLength:   100,
							Required:    false,
							Label:       "Per Member",
							Style:       discordgo.TextInputShort,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "org_share",
							Placeholder: "Ex. 20000",
							MaxLength:   100,
							Required:    false,
							Label:       "Org Take",
							Style:       discordgo.TextInputShort,
						},
					},
				},
			},
		},
	}); err != nil {
		return err
	}

	return nil
}

func AddPayoutModalHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("add payout modal handler")

	if !allowed(i.Member, "ATTENDANCE") {
		return customerrors.InvalidPermissions
	}

	attendanceId := strings.Split(i.ModalSubmitData().CustomID, ":")[2]

	attendance, err := attendance.Get(attendanceId)
	if err != nil {
		return err
	}

	totalPayoutS := i.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	totalPayout, err := strconv.ParseInt(strings.ReplaceAll(totalPayoutS, ",", ""), 10, 64)
	if err != nil {
		return errors.New("invalid total payout")
	}

	perMemeberS := i.ModalSubmitData().Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	perMember, err := strconv.ParseInt(strings.ReplaceAll(perMemeberS, ",", ""), 10, 64)
	if err != nil {
		perMember = 0
	}

	orgShareS := i.ModalSubmitData().Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	orgShare, err := strconv.ParseInt(strings.ReplaceAll(orgShareS, ",", ""), 10, 64)
	if err != nil {
		orgShare = 0
	}

	if err := attendance.AddPayout(totalPayout, perMember, orgShare); err != nil {
		return err
	}

	m := attendance.ToDiscordMessage()
	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    i.ChannelID,
		ID:         i.Message.ID,
		Embeds:     &m.Embeds,
		Components: &m.Components,
	}); err != nil {
		return err
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: "Payout added successfully",
		},
	})
}
