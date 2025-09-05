package attendance

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/ranks"
)

func (a *Attendance) ToDiscordMessage() *discordgo.MessageSend {
	name := fmt.Sprintf("Attendees (%d)", len(a.Members))
	fields := []*discordgo.MessageEmbedField{
		{
			Name:  "Submitted By",
			Value: "<@" + a.SubmittedBy.Id + ">",
		},
		{
			Name:   name,
			Value:  "No Attendees",
			Inline: true,
		},
	}

	if len(a.Members) > 0 {
		sort.Slice(a.Members, func(i, j int) bool {
			if a.Members[i].IsGuest {
				return false
			}
			if a.Members[i].IsAffiliate {
				return false
			}

			if a.Members[i].IsAlly {
				return false
			}

			if a.Members[j].IsGuest {
				return true
			}

			if a.Members[j].IsAffiliate {
				return true
			}

			if a.Members[j].IsAlly {
				return true
			}

			if a.Members[i].Rank < a.Members[j].Rank {
				return true
			}

			if a.Members[i].Rank != ranks.None && a.Members[i].Rank == a.Members[j].Rank {
				return a.Members[i].Name < a.Members[j].Name
			}

			return false
		})

		fields[1].Value = ""

		for i, member := range a.Members {
			// for every 10 members, make a new field
			if i%10 == 0 && i != 0 {
				fields = append(fields, &discordgo.MessageEmbedField{
					Name:   "Attendees (continued)",
					Value:  "",
					Inline: true,
				})
			}

			field := fields[len(fields)-1]
			field.Value += "<@" + member.Id + ">"

			if a.IsFromStart(member) && a.Tokenable {
				if a.TheyStayed(member) {
					field.Value += " 🌟"
				} else {
					field.Value += " ⭐"
				}
			}

			// if not the 10th, add a new line
			if i%10 != 9 {
				field.Value += "\n"
			}
		}
	}

	if len(a.WithIssues) > 0 {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Attendees with Issues",
			Value:  "",
			Inline: true,
		})

		for i, member := range a.WithIssues {
			if member == nil {
				continue
			}

			field := fields[len(fields)-1]

			field.Value += "<@" + member.Id + "> - " + strings.Join(Issues(member), ", ")

			// if not the 10th, add a new line
			if i%10 != 9 {
				field.Value += "\n"
			}

			// for every 10 members, make a new field
			if i%10 == 0 && i != 0 {
				fields = append(fields, &discordgo.MessageEmbedField{
					Name:   "Attendees with Issues (continued)",
					Value:  "",
					Inline: true,
				})
			}
		}
	}

	if a.Payouts != nil {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Payouts",
			Value:  "Total: " + strconv.Itoa(int(a.Payouts.Total)) + "\nPer Member: " + strconv.Itoa(int(a.Payouts.PerMember)) + "\nOrg Take: " + strconv.Itoa(int(a.Payouts.OrgTake)),
			Inline: false,
		})
	}

	var footer *discordgo.MessageEmbedFooter
	if a.Tokenable {
		footer = &discordgo.MessageEmbedFooter{
			Text: "⭐ joined from the start | 🌟 stayed entire event",
		}
	}

	embeds := []*discordgo.MessageEmbed{
		{
			Title:       a.Name,
			Description: a.Id,
			Timestamp:   a.DateCreated.Format(time.RFC3339),
			Fields:      fields,
			Footer:      footer,
		},
	}

	buttons := []discordgo.MessageComponent{}
	if !a.Recorded {
		startButton := discordgo.Button{
			Label: "Start Event",
			Style: discordgo.SuccessButton,
			Emoji: &discordgo.ComponentEmoji{
				Name: "▶️",
			},
			CustomID: "attendance:start:" + a.Id,
		}

		if a.Active {
			startButton.Label = "End Event"
			startButton.Emoji.Name = "🛑"
			startButton.CustomID = "attendance:end:" + a.Id
		}
		buttons = append(buttons, startButton)

		if a.Active && a.Tokenable {
			successButton := discordgo.Button{
				Label: "Successful",
				Style: discordgo.SuccessButton,
				Emoji: &discordgo.ComponentEmoji{
					Name: "✅",
				},
				CustomID: "attendance:successful:" + a.Id,
			}

			if a.Successful {
				successButton.Label = "Unsuccessful"
				successButton.Style = discordgo.DangerButton
				successButton.Emoji.Name = "➖"
				successButton.CustomID = "attendance:unsuccessful:" + a.Id
			}

			buttons = append(buttons, successButton)
		}

		buttons = append(buttons,
			discordgo.Button{
				Label:    "Delete",
				Style:    discordgo.DangerButton,
				Disabled: a.Recorded,
				Emoji: &discordgo.ComponentEmoji{
					Name: "🗑️",
				},
				CustomID: "attendance:delete:" + a.Id,
			},
			discordgo.Button{
				Label:    "Recheck Issues",
				Style:    discordgo.PrimaryButton,
				Disabled: a.Recorded,
				Emoji: &discordgo.ComponentEmoji{
					Name: "🔁",
				},
				CustomID: "attendance:recheck:" + a.Id,
			},
		)

		payoutButton := discordgo.Button{
			Label:    "Add Payout",
			Style:    discordgo.PrimaryButton,
			Disabled: a.Recorded,
			Emoji: &discordgo.ComponentEmoji{
				Name: "💰",
			},
			CustomID: "attendance:payout:" + a.Id,
		}
		if a.Payouts != nil {
			payoutButton.Label = "Edit Payout"
			payoutButton.CustomID = "attendance:payout:" + a.Id
		}
		buttons = append(buttons, payoutButton)
	}

	components := []discordgo.MessageComponent{}

	if !a.Recorded {
		components = append(components, discordgo.ActionsRow{
			Components: buttons,
		})
	}

	components = append(components, discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label: "Export",
				Style: discordgo.PrimaryButton,
				Emoji: &discordgo.ComponentEmoji{
					Name: "📥",
				},
				CustomID: "attendance:export:" + a.Id,
			},
		},
	})

	return &discordgo.MessageSend{
		Embeds:     embeds,
		Components: components,
	}
}
