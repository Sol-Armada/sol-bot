package jobs

import (
	"context"

	"github.com/bwmarrin/discordgo"
)

type JobMonitor interface {
	Update(message string)
	Done()
}

type nopMonitor struct{}

func (nopMonitor) Update(string) {}
func (nopMonitor) Done()         {}

func NewNopMonitor() JobMonitor {
	return nopMonitor{}
}

type job struct {
	Name string
	Cron string
	Run  func(context.Context, *discordgo.Session, JobMonitor) error
}

var Jobs = []job{
	{
		Name: "Promotions Report",
		Cron: "0 0 * * *",
		Run:  promotionsReport,
	},
	{
		Name: "Member Monitor",
		Cron: "*/2 * * * *",
		Run:  MemberMonitor,
	},
}
