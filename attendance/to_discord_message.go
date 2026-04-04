package attendance

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/ranks"
	"github.com/sol-armada/sol-bot/tokens"
)

func (a *Attendance) ToDiscordMessage() (*discordgo.MessageSend, error) {
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

		tokenRecords, err := tokens.GetByAttendanceId(a.Id)
		if err != nil {
			return nil, err
		}

		tokenRecordsMap := make(map[string][]tokens.TokenRecord)
		for _, r := range tokenRecords {
			tokenRecordsMap[r.MemberId] = append(tokenRecordsMap[r.MemberId], r)
		}

		var row strings.Builder
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
			row.WriteString("<@" + member.Id + ">")

			if tokenRecords, ok := tokenRecordsMap[member.Id]; ok {
				for i, r := range tokenRecords {
					if i == 0 {
						row.WriteString(" (")
					}
					if r.Amount <= 0 {
						continue
					}
					row.WriteString("" + strconv.Itoa(r.Amount))
					if i != len(tokenRecords)-1 {
						row.WriteString(" + ")
					}
					if i == len(tokenRecords)-1 {
						row.WriteString(")")
					}
				}
			}

			// if not the 10th, add a new line
			if i%10 != 9 {
				row.WriteString("\n")
			}
			fields = append(fields[:len(fields)-1], &discordgo.MessageEmbedField{
				Name:   field.Name,
				Value:  row.String(),
				Inline: true,
			})
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

	embeds := []*discordgo.MessageEmbed{
		{
			Title:       a.Name,
			Description: a.Id,
			Timestamp:   a.DateCreated.Format(time.RFC3339),
			Fields:      fields,
		},
	}

	components := []discordgo.MessageComponent{}

	if !a.Recorded {
		buttons := discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				func() discordgo.Button {
					if a.Tokenable && a.Status == AttendanceStatusCreated {
						return discordgo.Button{
							Label: "Start Event",
							Style: discordgo.SecondaryButton,
							Emoji: &discordgo.ComponentEmoji{
								Name: "▶️",
							},
							CustomID: "attendance:start:" + a.Id,
						}
					}

					return discordgo.Button{
						Label: "Finish Event",
						Style: discordgo.SuccessButton,
						Emoji: &discordgo.ComponentEmoji{
							Name: "🛑",
						},
						CustomID: "attendance:end:" + a.Id,
					}
				}(),
			},
		}
		if a.Tokenable {
			buttons.Components = append(buttons.Components,
				func() discordgo.Button {
					if a.Successful {
						return discordgo.Button{
							Label: "Unsuccessful",
							Style: discordgo.SecondaryButton,
							Emoji: &discordgo.ComponentEmoji{
								Name: "❌",
							},
							CustomID: "attendance:unsuccessful:" + a.Id,
						}
					}
					return discordgo.Button{
						Label: "Successful",
						Style: discordgo.SuccessButton,
						Emoji: &discordgo.ComponentEmoji{
							Name: "✅",
						},
						CustomID: "attendance:successful:" + a.Id,
					}
				}(),
			)
		}
		components = append(components, buttons)

		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
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
			},
		})
	} else {
		buttons := []discordgo.MessageComponent{
			discordgo.Button{
				Label: "Revert",
				Style: discordgo.PrimaryButton,
				Emoji: &discordgo.ComponentEmoji{
					Name: "↩️",
				},
				CustomID: "attendance:revert:" + a.Id,
			},
			discordgo.Button{
				Label: "Export",
				Style: discordgo.PrimaryButton,
				Emoji: &discordgo.ComponentEmoji{
					Name: "📥",
				},
				CustomID: "attendance:export:" + a.Id,
			},
		}

		if a.Tokenable {
			buttons = append(buttons, discordgo.Button{
				Label: "Distribute Tokens",
				Style: discordgo.SuccessButton,
				Emoji: &discordgo.ComponentEmoji{
					Name: "🪙",
				},
				CustomID: "attendance:distribute:" + a.Id,
			})
		}

		components = append(components, discordgo.ActionsRow{
			Components: buttons,
		})
	}

	return &discordgo.MessageSend{
		Embeds:     embeds,
		Components: components,
	}, nil
}
