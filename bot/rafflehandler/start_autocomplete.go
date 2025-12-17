package rafflehandler

import (
	"context"
	"errors"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/utils"
)

func startAutocomplete(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("raffle start autocomplete")

	data := i.ApplicationCommandData()

	choices := []*discordgo.ApplicationCommandOptionChoice{}

	for _, option := range data.Options {
		switch option.Name {
		case "name":
			if option.Focused {
				attendanceRecords, err := attendance.ListActive(5)
				if err != nil {
					return errors.Join(err, errors.New("getting active attendance records"))
				}

				for _, record := range attendanceRecords {
					choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
						Name:  record.Name + " - " + record.DateCreated.Format(time.RFC822),
						Value: record.Id,
					})
				}
			}
		case "prize":
			if option.Focused {
				ships := []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "Anvil F7A Hornet Mk II Executive",
						Value: "anvil_f7a_hornet_mk_ii_executive",
					},
					{
						Name:  "Anvil F8C Lightning Executive",
						Value: "anvil_f8c_lightning_executive",
					},
					{
						Name:  "Drake Corsair Executive",
						Value: "drake_corsair_executive",
					},
					{
						Name:  "Drake Cutlass Black Executive",
						Value: "drake_cutlass_black_executive",
					},
					{
						Name:  "Gatac Syulen Executive",
						Value: "gatac_syulen_executive",
					},
				}
				choices = append(choices, ships...)
			}
		}
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	}); err != nil {
		return errors.Join(err, errors.New("responding to raffle start autocomplete"))
	}

	return nil
}
