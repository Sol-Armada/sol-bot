package jobs

import (
	"context"

	"github.com/bwmarrin/discordgo"
)

type job struct {
	Name string
	Cron string
	Run  func(context.Context, *discordgo.Session) error
}

var Jobs = []job{
	{
		Name: "Promotions Report",
		Cron: "0 0 * * *",
		Run:  promotionsReport,
	},
}
