package giveaway

import "github.com/bwmarrin/discordgo"

func (g *Giveaway) GetComponents() []discordgo.MessageComponent {
	if g.Ended {
		return []discordgo.MessageComponent{}
	}

	items := make([]discordgo.SelectMenuOption, 0, len(g.Items))
	for _, item := range g.Items {
		if item == nil {
			continue
		}
		items = append(items, discordgo.SelectMenuOption{
			Label: item.Name,
			Value: item.Id,
		})
	}

	btns := []discordgo.MessageComponent{
		discordgo.Button{
			CustomID: "giveaway:view_entries:" + g.Id,
			Label:    "View My Entries",
			Style:    discordgo.SecondaryButton,
		},
	}

	if len(items) == 1 {
		btns = append(btns,
			discordgo.Button{
				CustomID: "giveaway:exit:" + g.Id + ":" + items[0].Value,
				Label:    "Remove Entries",
				Style:    discordgo.SecondaryButton,
			},
		)
	}

	btns = append(btns,
		discordgo.Button{
			CustomID: "giveaway:end:" + g.Id,
			Label:    "End",
			Style:    discordgo.DangerButton,
		},
	)

	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    "giveaway:update_entry:" + g.Id,
					Options:     items,
					Placeholder: "Select which items you want",
					MaxValues:   len(items),
				},
			},
		},
		discordgo.ActionsRow{
			Components: btns,
		},
	}
}
