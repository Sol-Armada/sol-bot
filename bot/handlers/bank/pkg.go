package bank

import (
	"fmt"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/xid"
	h "github.com/sol-armada/admin/bot/handlers"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/transaction"
	"github.com/sol-armada/admin/user"
	"golang.org/x/exp/slices"
)

var handlers = map[string]func(*discordgo.Session, *discordgo.InteractionCreate){
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
			h.ErrorResponse(s, i.Interaction, "backend error responding to the interaction")
		}
		return
	}

	commandOption := i.ApplicationCommandData().Options[0].Name
	handlers[commandOption](s, i)
}

func balanceHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger := log.WithField("func", "balanceHandler")
	logger.Debug("getting the bank balance")

	transactions, err := transaction.List()
	if err != nil {
		logger.WithError(err).Error("getting transactions for balance command")
		h.ErrorResponse(s, i.Interaction, "backend error getting transactions")
		return
	}

	balance := int32(0)
	for _, transaction := range transactions {
		balance += transaction.Amount
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: fmt.Sprintf("%d", balance),
		},
	}); err != nil {
		h.ErrorResponse(s, i.Interaction, "backend error responding to the interaction")
	}
}

func addHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger := log.WithField("func", "addHandler")
	logger.Debug("adding to the bank")

	fromId := i.ApplicationCommandData().Options[0].Options[0].Value.(string)
	amount := int32(i.ApplicationCommandData().Options[0].Options[1].Value.(float64))
	notes := ""
	if len(i.ApplicationCommandData().Options[0].Options) == 3 {
		notes = i.ApplicationCommandData().Options[0].Options[3].Value.(string)
	}

	if amount <= 0 {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "Amount must be above zero",
			},
		}); err != nil {
			h.ErrorResponse(s, i.Interaction, "backend issue")
		}
	}

	from, err := user.Get(fromId)
	from.Discord = nil
	if err != nil {
		logger.WithError(err).Error("getting from user for bank add")
		h.ErrorResponse(s, i.Interaction, "backend error getting the from user")
		return
	}

	holder, err := user.Get(i.Member.User.ID)
	holder.Discord = nil
	if err != nil {
		logger.WithError(err).Error("getting holder user for bank add")
		h.ErrorResponse(s, i.Interaction, "backend error getting the holding user (you)")
		return
	}

	transaction := &transaction.Transaction{
		Id:     xid.New().String(),
		Amount: amount,
		From:   from,
		Holder: holder,
		Notes:  notes,
	}

	if err := transaction.Save(); err != nil {
		h.ErrorResponse(s, i.Interaction, "backend error saving the transaction")
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: fmt.Sprintf("Added %d From %s", amount, from.Name),
		},
	}); err != nil {
		h.ErrorResponse(s, i.Interaction, "backend issue")
	}
}

func removeHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger := log.WithField("func", "removeHandler")
	logger.Debug("removing from the bank")

	toId := i.ApplicationCommandData().Options[0].Options[0].Value.(string)
	amount := int32(i.ApplicationCommandData().Options[0].Options[1].Value.(float64))
	notes := ""
	if len(i.ApplicationCommandData().Options[0].Options) == 3 {
		notes = i.ApplicationCommandData().Options[0].Options[3].Value.(string)
	}

	if amount <= 0 {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "Amount must be above zero",
			},
		}); err != nil {
			h.ErrorResponse(s, i.Interaction, "backend issue")
		}
	}

	to, err := user.Get(toId)
	to.Discord = nil
	if err != nil {
		logger.WithError(err).Error("getting to user for bank remove")
		h.ErrorResponse(s, i.Interaction, "backend error getting the to user")
		return
	}

	holder, err := user.Get(i.Member.User.ID)
	holder.Discord = nil
	if err != nil {
		logger.WithError(err).Error("getting holder user for bank remove")
		h.ErrorResponse(s, i.Interaction, "backend error getting the holding user (you)")
		return
	}

	transaction := &transaction.Transaction{
		Id:     xid.New().String(),
		Amount: amount * -1,
		To:     to,
		Holder: holder,
		Notes:  notes,
	}

	if err := transaction.Save(); err != nil {
		h.ErrorResponse(s, i.Interaction, "backend error saving the transaction")
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: fmt.Sprintf("Removing %d to be for %s", amount, to.Name),
		},
	}); err != nil {
		h.ErrorResponse(s, i.Interaction, "backend issue")
	}
}

func spendHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger := log.WithField("func", "spendHandler")
	logger.Debug("spending from the bank")

	spendReason := i.ApplicationCommandData().Options[0].Options[0].Value.(string)
	amount := int32(i.ApplicationCommandData().Options[0].Options[1].Value.(float64))
	notes := ""
	if len(i.ApplicationCommandData().Options[0].Options) == 3 {
		notes = i.ApplicationCommandData().Options[0].Options[3].Value.(string)
	}

	if amount <= 0 {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "Amount must be above zero",
			},
		}); err != nil {
			h.ErrorResponse(s, i.Interaction, "backend issue")
		}
	}

	holder, err := user.Get(i.Member.User.ID)
	holder.Discord = nil
	if err != nil {
		logger.WithError(err).Error("getting holder user for bank remove")
		h.ErrorResponse(s, i.Interaction, "backend error getting the holding user (you)")
		return
	}

	transaction := &transaction.Transaction{
		Id:     xid.New().String(),
		Amount: amount * -1,
		For:    spendReason,
		Holder: holder,
		Notes:  notes,
	}

	if err := transaction.Save(); err != nil {
		h.ErrorResponse(s, i.Interaction, "backend error saving the transaction")
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: fmt.Sprintf("Spending %d for \"%s\"", amount, spendReason),
		},
	}); err != nil {
		h.ErrorResponse(s, i.Interaction, "backend issue")
	}
}
