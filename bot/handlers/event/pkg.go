package event

import (
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/admin/bot/handlers"
	"github.com/sol-armada/admin/ranks"
	"github.com/sol-armada/admin/stores"
	"github.com/sol-armada/admin/users"
)

var eventSubCommands = map[string]func(*discordgo.Session, *discordgo.Interaction){
	"attendance": TakeAttendance,
}

func EventCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// get the user
	storage := stores.Storage
	userResault := storage.GetUser(i.Member.User.ID)
	user := &users.User{}
	if err := userResault.Decode(user); err != nil {
		handlers.ErrorResponse(s, i.Interaction, "Internal server error... >_<; Try again later")
		return
	}

	// check for permission
	if user.Rank > ranks.Lieutenant {
		handlers.ErrorResponse(s, i.Interaction, "You do not have permission to use this command")
		return
	}

	// send to the sub command
	if handler, ok := eventSubCommands[i.ApplicationCommandData().Options[0].Name]; ok {
		handler(s, i.Interaction)
		return
	}

	// somehow they used a sub command that doesn't exist
	handlers.ErrorResponse(s, i.Interaction, "That sub command doesn't exist. Not sure how you even got here. Good job.")
}
