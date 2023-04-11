package event

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/kyokomi/emoji/v2"
	"github.com/pkg/errors"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/events"
)

func updateEventMessage(s *discordgo.Session, event *events.Event) error {
	// get the event message
	message, err := s.ChannelMessage(config.GetString("DISCORD.CHANNELS.ANNOUNCEMENTS"), event.MessageId)
	if err != nil {
		return errors.Wrap(err, "getting event message")
	}

	// update the message
	embedFeilds := message.Embeds[0].Fields[1:]
	for _, ef := range embedFeilds {
		// iterate over the positions in the event
		for _, position := range event.Positions {
			positionEmoji := emoji.CodeMap()[":"+strings.ToLower(position.Emoji)+":"]
			if strings.Contains(ef.Name, positionEmoji) {
				ef.Name = fmt.Sprintf("%s %s (%d/%d)", positionEmoji, position.Name, len(position.Members), position.Max)

				names := ""
				for _, memberId := range position.Members {
					// get the member
					member, err := s.GuildMember(config.GetString("DISCORD.GUILD_ID"), memberId)
					if err != nil {
						return errors.Wrap(err, "getting member")
					}

					if member.Nick != "" {
						names += member.Nick
						continue
					}

					// at the member to the list
					names += member.User.Username + "\n"
				}

				// if we have no names, change it to '-'
				if names == "" {
					names = "-"
				}

				// update the field value with the names
				ef.Value = names

				continue
			}
		}
	}

	if _, err := s.ChannelMessageEditEmbeds(message.ChannelID, message.ID, message.Embeds); err != nil {
		return errors.Wrap(err, "updating embeds")
	}

	return nil
}
