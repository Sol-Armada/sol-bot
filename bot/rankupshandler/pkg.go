package rankupshandler

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/bot/internal/command"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/members"
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

	embed, err := members.GetPromotionsEmbed()
	if err != nil {
		if customerrors.Is(err, customerrors.NoPromotions) {
			_, err = s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "No promotions found",
			})
		}
		return err
	}

	_, err = s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Flags:  discordgo.MessageFlagsEphemeral,
		Embeds: []*discordgo.MessageEmbed{embed},
	})
	return err
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
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
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
