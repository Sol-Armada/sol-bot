package yourelatehandler

import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/bot/internal/command"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/utils"
)

type YoureLateCommand struct{}

var _ command.ApplicationCommand = (*YoureLateCommand)(nil)

func New() command.ApplicationCommand {
	return &YoureLateCommand{}
}

// AutocompleteHandler implements [command.ApplicationCommand].
func (y *YoureLateCommand) AutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// ButtonHandler implements [command.ApplicationCommand].
func (y *YoureLateCommand) ButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// CommandHandler implements [command.ApplicationCommand].
func (y *YoureLateCommand) CommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("yourelate command handler")

	if !utils.Allowed(i.Member, "TOKENS") {
		return customerrors.InvalidPermissions
	}

	var lateArrival *discordgo.User
	var poc *discordgo.User

	data := i.ApplicationCommandData()
	for _, opt := range data.Options {
		if opt.Name == "member" {
			lateArrival = opt.UserValue(s)
		}

		if opt.Name == "poc" {
			pocMember := opt.UserValue(s)
			if pocMember != nil {
				poc = pocMember
			}
		}
	}

	if poc == nil {
		poc = i.Member.User
	}

	sb := strings.Builder{}
	sb.WriteString("Welcome to the event " + lateArrival.Mention() + "!\n\n")

	sb.WriteString("How To Join If You’re Late:\n")
	sb.WriteString("Read the description ")

	isEventThread := false
	ch, err := s.Channel(i.ChannelID)
	if err != nil {
		logger.Error("failed to get channel", "error", err)
		return err
	}
	messages, err := s.ChannelMessages(ch.ID, 100, "", "", "")
	if err != nil {
		logger.Error("failed to get channel messages", "error", err)
		return err
	}
	message := messages[len(messages)-1].ReferencedMessage
	if message != nil && len(messages) > 0 && message.Author.Username == "sesh" {
		isEventThread = true
	}

	if isEventThread {
		sb.WriteString("here: <#" + message.ID + ">\n")
	} else {
		sb.WriteString("in the appropriate event post in <#" + settings.GetString("DISCORD.CHANNELS.EVENT_SIGNUP") + ">!\n")
	}

	sb.WriteString("Complete any event prep required to participate.\n")
	sb.WriteString("Head to our current planetary region. Land at the LEO station there.\n")
	sb.WriteString("Ask for a party invite. Please DM " + poc.Mention() + " with any questions!\n")

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: sb.String(),
		},
	})
}

// ModalHandler implements [command.ApplicationCommand].
func (y *YoureLateCommand) ModalHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// Name implements [command.ApplicationCommand].
func (y *YoureLateCommand) Name() string {
	return "you_are_late"
}

// OnAfter implements [command.ApplicationCommand].
func (y *YoureLateCommand) OnAfter(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// OnBefore implements [command.ApplicationCommand].
func (y *YoureLateCommand) OnBefore(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// OnError implements [command.ApplicationCommand].
func (y *YoureLateCommand) OnError(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, err error) {
}

// SelectMenuHandler implements [command.ApplicationCommand].
func (y *YoureLateCommand) SelectMenuHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// Setup implements [command.ApplicationCommand].
func (y *YoureLateCommand) Setup() (*discordgo.ApplicationCommand, error) {
	return &discordgo.ApplicationCommand{
		Name:        "you_are_late",
		Description: "Let a member know what to do when they are late",
		Type:        discordgo.ChatApplicationCommand,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:         "member",
				Description:  "The member to notify",
				Type:         discordgo.ApplicationCommandOptionUser,
				Required:     true,
				Autocomplete: true,
			},
			{
				Name:        "poc",
				Description: "The point of contect to use in the message",
				Type:        discordgo.ApplicationCommandOptionUser,
				Required:    false,
			},
		},
	}, nil
}
