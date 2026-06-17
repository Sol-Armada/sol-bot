package jobs

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/settings"
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

func GetJobs() []job {
	return []job{
		{
			Name: "Promotions Report",
			Cron: settings.GetStringWithDefault("PROMOTIONS_REPORT_CRON", "0 0 * * *"),
			Run:  promotionsReport,
		},
		{
			Name: "Member Monitor",
			Cron: settings.GetStringWithDefault("MEMBER_MONITOR_CRON", "0 * * * *"),
			Run:  MemberMonitor,
		},
	}
}
