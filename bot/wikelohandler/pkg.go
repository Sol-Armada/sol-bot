package wikelohandler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/utils"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type itemProgress struct {
	name      string
	needed    string
	inventory string
	progress  string
}

func Setup() (*discordgo.ApplicationCommand, error) {
	return &discordgo.ApplicationCommand{
		Name:        "wikelo",
		Description: "See Wikelo progress",
		Type:        discordgo.ChatApplicationCommand,
	}, nil
}

var (
	sheetId = "16mFI4wZc6FOmz08JiCkiK69Rx6mg87Aodq-dsfcgZ-8"
)

func CommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("wikelo command handler")

	client, err := sheets.NewService(context.Background(), option.WithAPIKey(settings.GetString("GOOGLE_SHEET_API_KEY")))
	if err != nil {
		return err
	}

	embeds := make([]*discordgo.MessageEmbed, 0)

	readRange := "Overview!B6:G"

	resp, err := client.Spreadsheets.Values.Get(sheetId, readRange).Do()
	if err != nil {
		return err
	}

	if len(resp.Values) == 0 {
		logger.Debug("No data found.")
		return nil
	}

	itemProgresses := make([]itemProgress, 0, len(resp.Values))
	for _, row := range resp.Values {
		if len(row) < 6 || row[0] == nil || row[0] == "Item" || row[0] == "" {
			continue
		}

		if row[0] == "Information" {
			break
		}

		logger.Debugf("Row: %s", row)
		itemProgresses = append(itemProgresses, itemProgress{
			name:      row[0].(string),
			needed:    row[1].(string),
			inventory: row[2].(string),
			progress:  row[5].(string),
		})
	}

	resp, err = client.Spreadsheets.Values.Get(sheetId, "Overview!B3").Do()
	if err != nil {
		logger.WithError(err).Error("Failed to get sheet data")
		return err
	}

	if len(resp.Values) == 0 {
		logger.Debug("No data found.")
		return nil
	}

	overallProgress := resp.Values[0][0]

	resp, err = client.Spreadsheets.Values.Get(sheetId, "Contributions!D3:W").Do()
	if err != nil {
		logger.WithError(err).Error("Failed to get sheet data")
		return nil
	}

	if len(resp.Values) == 0 {
		logger.Debug("No data found.")
		return nil
	}

	fields := make([]*discordgo.MessageEmbedField, 0, len(itemProgresses))
	for _, item := range itemProgresses {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   item.name,
			Value:  item.inventory + "/" + item.needed + " (" + item.progress + ")",
			Inline: true,
		})
	}

	embeds = append(embeds, &discordgo.MessageEmbed{
		Title:       "Wikelo Progress",
		Description: "Overall progress\n" + overallProgress.(string),
		Fields:      fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Inventory/Needed (Progress)",
		},
	})

	items := make([]string, 0)
	contributions := make(map[string]map[string]float64)
	for i, row := range resp.Values {
		if i == 0 { // header
			for j := 1; j < len(row); j++ {
				if row[j] == "" {
					continue
				}
				items = append(items, row[j].(string))
			}
			continue
		}

		who := row[0].(string)
		if _, ok := contributions[who]; !ok {
			contributions[who] = make(map[string]float64)
		}
		for j := 1; j < len(row); j++ {
			if row[j] == "" {
				continue
			}
			item := items[j-1]
			if _, ok := contributions[who][item]; !ok {
				contributions[who][item] = 0
			}
			aString := row[j].(string)
			if aString == "" {
				continue
			}

			f, err := strconv.ParseFloat(aString, 64)
			if err != nil {
				logger.WithError(err).Error("Failed to convert string to float64")
				continue
			}
			contributions[who][item] += f
		}
	}

	fields = make([]*discordgo.MessageEmbedField, 0, len(contributions))
	for who, items := range contributions {
		if !strings.Contains(i.Member.User.Username, who) && !strings.Contains(i.Member.Nick, who) {
			continue
		}

		for item, amount := range items {
			field := &discordgo.MessageEmbedField{
				Name:   item,
				Value:  fmt.Sprintf("%d", amount),
				Inline: true,
			}
			fields = append(fields, field)
		}
	}

	resp, err = client.Spreadsheets.Values.Get(sheetId, "Leaderboard!B4:C").Do()
	if err != nil {
		return err
	}

	print := []string{}
	for r, row := range resp.Values {
		if strings.Contains(i.Member.Nick, row[0].(string)) || strings.Contains(i.Member.User.Username, row[0].(string)) {
			if r != 0 && resp.Values[r-1] != nil {
				print = append(print, fmt.Sprintf("%d) %s", r, resp.Values[r-1][0].(string)))
			}

			print = append(print, fmt.Sprintf("%d) %s", r+1, row[0].(string)))

			if resp.Values[r+1] != nil {
				print = append(print, fmt.Sprintf("%d) %s", r+2, resp.Values[r+1][0].(string)))
			}

			if r == 0 && resp.Values[r+2] != nil {
				print = append(print, fmt.Sprintf("%d) %s", r+3, resp.Values[r+2][0].(string)))
			}

			break
		}
	}
	if len(fields) != 0 {
		embeds = append(embeds, &discordgo.MessageEmbed{
			Title:       "Personal Contributions and Leaderboard Rank",
			Description: "Rankings\n" + strings.Join(print, "\n"),
			Fields:      fields,
		})
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: embeds,
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}
