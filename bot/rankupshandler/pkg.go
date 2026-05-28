package rankupshandler

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/bot/internal/command"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/ranks"
	"github.com/sol-armada/sol-bot/utils"
)

type RankupsCommand struct{}

var _ command.ApplicationCommand = (*RankupsCommand)(nil)

func New() command.ApplicationCommand {
	return &RankupsCommand{}
}

// AutocompleteHandler implements [command.ApplicationCommand].
func (r *RankupsCommand) AutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// ButtonHandler implements [command.ApplicationCommand].
func (r *RankupsCommand) ButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// CommandHandler implements [command.ApplicationCommand].
func (r *RankupsCommand) CommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("rank ups command handler")

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	// // get members
	// membersList, err := members.List(0)
	// if err != nil {
	// 	return err
	// }

	// // check if any members need to rank up
	// type t struct {
	// 	Member   members.Member
	// 	NextRank ranks.Rank
	// 	Count    int
	// }
	// needsRankUp := []t{}
	// for _, member := range membersList {
	// 	if !member.IsRanked() || member.IsGuest || member.IsAlly || member.IsAffiliate {
	// 		continue
	// 	}

	// 	logger.Debug("checking if member needs rank up", "member", member.Id)

	// 	count, err := attendance.GetMemberAttendanceCount(member.Id)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	tt := t{Member: member, Count: count}

	// 	if member.Rank == ranks.Recruit && count >= 3 {
	// 		tt.NextRank = ranks.Member
	// 	}
	// 	if member.Rank == ranks.Member && count >= 10 {
	// 		tt.NextRank = ranks.Technician
	// 	}
	// 	if member.Rank == ranks.Technician && count >= 20 {
	// 		tt.NextRank = ranks.Specialist
	// 	}

	// 	if tt.NextRank != 0 {
	// 		needsRankUp = append(needsRankUp, tt)
	// 	}
	// }

	// get promotions
	promotions, err := members.ListPromotions()
	if err != nil {
		return err
	}

	if len(promotions) == 0 {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "There are no members that need a rank up",
		})
		return nil
	}

	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "Members to Rank Up",
			Value:  "",
			Inline: true,
		},
	}
	embed := &discordgo.MessageEmbed{
		Title:  "",
		Fields: fields,
	}

	ind := 0
	for _, promotion := range promotions {
		if promotion.NextRank == 0 {
			continue
		}

		if ind%10 == 0 && ind != 0 {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "Members to Rank Up (continued)",
				Value:  "",
				Inline: true,
			})
		}

		field := fields[len(fields)-1]
		field.Value += fmt.Sprintf("<@%s> to %s (%d Events)", promotion.ID, ranks.Rank(promotion.NextRank).String(), promotion.AttendanceCount)

		// if not the 10th member, add a newline
		if ind%10 != 9 {
			field.Value += "\n"
		}

		ind++
	}

	// create followup
	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: "These members need a rank up",
		Embeds:  []*discordgo.MessageEmbed{embed},
	})
	if err != nil {
		return err
	}

	return nil
}

// ModalHandler implements [command.ApplicationCommand].
func (r *RankupsCommand) ModalHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// Name implements [command.ApplicationCommand].
func (r *RankupsCommand) Name() string {
	return "rankups"
}

// OnAfter implements [command.ApplicationCommand].
func (r *RankupsCommand) OnAfter(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// OnBefore implements [command.ApplicationCommand].
func (r *RankupsCommand) OnBefore(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// OnError implements [command.ApplicationCommand].
func (r *RankupsCommand) OnError(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, err error) {
}

// SelectMenuHandler implements [command.ApplicationCommand].
func (r *RankupsCommand) SelectMenuHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// Setup implements [command.ApplicationCommand].
func (r *RankupsCommand) Setup() (*discordgo.ApplicationCommand, error) {
	return &discordgo.ApplicationCommand{
		Name:        "rankups",
		Description: "Rank up your RSI profile",
	}, nil
}

func (c *RankupsCommand) SetupAliases() ([]*discordgo.ApplicationCommand, error) {
	return nil, nil
}
