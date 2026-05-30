package attendance

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/tokens"
)

func (a *Attendance) ToDiscordMessage() (*discordgo.MessageSend, error) {
	participants, err := a.Participants()
	if err != nil {
		return nil, err
	}

	name := fmt.Sprintf("Attendees (%d)", len(participants))
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

	if len(participants) > 0 {
		sort.Slice(participants, func(i, j int) bool {
			memberA := participants[i].Member
			memberB := participants[j].Member
			switch {
			case memberA.Rank != memberB.Rank:
				return memberA.Rank < memberB.Rank
			default:
				return memberA.Name < memberB.Name
			}
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
		for i, participant := range participants {
			if len(Issues(participant.Member)) > 0 {
				continue
			}

			// for every 10 members, make a new field
			if i%10 == 0 && i != 0 {
				fields = append(fields, &discordgo.MessageEmbedField{
					Name:   "Attendees (continued)",
					Value:  "",
					Inline: true,
				})
				row.Reset()
			}

			field := fields[len(fields)-1]
			row.WriteString("<@")
			row.WriteString(participant.Member.Id)
			row.WriteString(">")

			if a.Status == AttendanceStatusRecorded {
				if tokenRecords, ok := tokenRecordsMap[participant.Member.Id]; ok {
					for i, r := range tokenRecords {
						if i == 0 {
							row.WriteString(" (")
						}
						if r.Amount <= 0 {
							continue
						}
						row.WriteString("")
						row.WriteString(strconv.Itoa(r.Amount))
						if i != len(tokenRecords)-1 {
							row.WriteString(" + ")
						}
						if i == len(tokenRecords)-1 {
							row.WriteString(")")
						}
					}
				}
			} else {
				if participant.JoinedAtStart {
					row.WriteString(" (On Time)")
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

	withIssues, err := a.WithIssues()
	if err != nil {
		return nil, err
	}

	if len(withIssues) > 0 {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Attendees with Issues",
			Value:  "",
			Inline: true,
		})

		for i, member := range withIssues {
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
		buttons.Components = append(buttons.Components,
			discordgo.Button{
				Label: func() string {
					if a.Tokenable {
						return "Remove Tokens"
					}
					return "Make Tokenable"
				}(),
				Style: discordgo.SecondaryButton,
				Emoji: &discordgo.ComponentEmoji{
					Name: "🪙",
				},
				CustomID: "attendance:toggle_tokenable:" + a.Id,
			},
		)
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
					Label:    "Refresh",
					Style:    discordgo.PrimaryButton,
					Disabled: a.Recorded,
					Emoji: &discordgo.ComponentEmoji{
						Name: "🔁",
					},
					CustomID: "attendance:refresh:" + a.Id,
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
