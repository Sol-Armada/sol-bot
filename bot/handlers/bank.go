package handlers

import (
	"fmt"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/xid"
	"github.com/sol-armada/admin/config"
	customerrors "github.com/sol-armada/admin/errors"
	"github.com/sol-armada/admin/transactions"
	"github.com/sol-armada/admin/users"
	"golang.org/x/exp/slices"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var bankHandlers = map[string]func(*discordgo.Session, *discordgo.InteractionCreate){
	"balance": balanceHandler,
	"add":     addHandler,
	"remove":  removeHandler,
	"spend":   spendHandler,
}

func BankCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger := log.WithField("func", "BankCommandHandler")
	logger.Debug("bank")

	holders := config.GetStringSlice("BANK.HOLDERS")
	if !slices.Contains(holders, i.Member.User.ID) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "You do not have permission for this command",
			},
		}); err != nil {
			customerrors.ErrorResponse(s, i.Interaction, "backend error responding to the interaction")
		}
		return
	}

	commandOption := i.ApplicationCommandData().Options[0].Name
	bankHandlers[commandOption](s, i)
}

func balanceHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger := log.WithField("func", "balanceHandler")
	logger.Debug("getting the bank balance")

	transactions, err := transactions.List()
	if err != nil {
		logger.WithError(err).Error("getting transactions for balance command")
		customerrors.ErrorResponse(s, i.Interaction, "backend error getting transactions")
		return
	}

	balance := int32(0)
	for _, transaction := range transactions {
		balance += transaction.Amount
	}

	p := message.NewPrinter(language.English)

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: p.Sprintf("%d aUEC", balance),
		},
	}); err != nil {
		customerrors.ErrorResponse(s, i.Interaction, "backend error responding to the interaction")
	}
}

func addHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger := log.WithField("func", "addHandler")
	logger.Debug("adding to the bank")

	fromId := i.ApplicationCommandData().Options[0].Options[0].Value.(string)
	amount := int32(i.ApplicationCommandData().Options[0].Options[1].Value.(float64))
	notes := ""
	if len(i.ApplicationCommandData().Options[0].Options) == 3 {
		notes = i.ApplicationCommandData().Options[0].Options[2].Value.(string)
	}

	if amount <= 0 {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "Amount must be above zero",
			},
		}); err != nil {
			customerrors.ErrorResponse(s, i.Interaction, "backend issue")
			return
		}
	}

	from, err := users.Get(fromId)
	if err != nil {
		logger.WithError(err).Error("getting from user")
		customerrors.ErrorResponse(s, i.Interaction, "backend issue")
		return
	}
	from.Discord = nil
	if err != nil {
		logger.WithError(err).Error("getting from user for bank add")
		customerrors.ErrorResponse(s, i.Interaction, "backend error getting the from user")
		return
	}

	holder, err := users.Get(i.Member.User.ID)
	holder.Discord = nil
	if err != nil {
		logger.WithError(err).Error("getting holder user for bank add")
		customerrors.ErrorResponse(s, i.Interaction, "backend error getting the holding user (you)")
		return
	}

	transaction := &transactions.Transaction{
		Id:     xid.New().String(),
		Amount: amount,
		From:   from,
		Holder: holder,
		Notes:  notes,
	}

	if err := transaction.Save(); err != nil {
		customerrors.ErrorResponse(s, i.Interaction, "backend error saving the transaction")
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: fmt.Sprintf("Added %d From %s", amount, from.Name),
		},
	}); err != nil {
		customerrors.ErrorResponse(s, i.Interaction, "backend issue")
	}
}

func removeHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger := log.WithField("func", "removeHandler")
	logger.Debug("removing from the bank")

	toId := i.ApplicationCommandData().Options[0].Options[0].Value.(string)
	amount := int32(i.ApplicationCommandData().Options[0].Options[1].Value.(float64))
	notes := ""
	if len(i.ApplicationCommandData().Options[0].Options) == 3 {
		notes = i.ApplicationCommandData().Options[0].Options[2].Value.(string)
	}

	if amount <= 0 {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "Amount must be above zero",
			},
		}); err != nil {
			customerrors.ErrorResponse(s, i.Interaction, "backend issue")
		}
	}

	to, err := users.Get(toId)
	to.Discord = nil
	if err != nil {
		logger.WithError(err).Error("getting to user for bank remove")
		customerrors.ErrorResponse(s, i.Interaction, "backend error getting the to user")
		return
	}

	holder, err := users.Get(i.Member.User.ID)
	holder.Discord = nil
	if err != nil {
		logger.WithError(err).Error("getting holder user for bank remove")
		customerrors.ErrorResponse(s, i.Interaction, "backend error getting the holding user (you)")
		return
	}

	transaction := &transactions.Transaction{
		Id:     xid.New().String(),
		Amount: amount * -1,
		To:     to,
		Holder: holder,
		Notes:  notes,
	}

	if err := transaction.Save(); err != nil {
		customerrors.ErrorResponse(s, i.Interaction, "backend error saving the transaction")
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: fmt.Sprintf("Removing %d to be for %s", amount, to.Name),
		},
	}); err != nil {
		customerrors.ErrorResponse(s, i.Interaction, "backend issue")
	}
}

func spendHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger := log.WithField("func", "spendHandler")
	logger.Debug("spending from the bank")

	spendReason := i.ApplicationCommandData().Options[0].Options[0].Value.(string)
	amount := int32(i.ApplicationCommandData().Options[0].Options[1].Value.(float64))
	notes := ""
	if len(i.ApplicationCommandData().Options[0].Options) == 3 {
		notes = i.ApplicationCommandData().Options[0].Options[2].Value.(string)
	}

	if amount <= 0 {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "Amount must be above zero",
			},
		}); err != nil {
			customerrors.ErrorResponse(s, i.Interaction, "backend issue")
		}
	}

	holder, err := users.Get(i.Member.User.ID)
	holder.Discord = nil
	if err != nil {
		logger.WithError(err).Error("getting holder user for bank remove")
		customerrors.ErrorResponse(s, i.Interaction, "backend error getting the holding user (you)")
		return
	}

	transaction := &transactions.Transaction{
		Id:     xid.New().String(),
		Amount: amount * -1,
		For:    spendReason,
		Holder: holder,
		Notes:  notes,
	}

	if err := transaction.Save(); err != nil {
		customerrors.ErrorResponse(s, i.Interaction, "backend error saving the transaction")
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: fmt.Sprintf("Spending %d for \"%s\"", amount, spendReason),
		},
	}); err != nil {
		customerrors.ErrorResponse(s, i.Interaction, "backend issue")
	}
}
