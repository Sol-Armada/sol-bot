package ranks

import "strings"

type Rank int

const (
	Bot Rank = iota
	Admiral
	Commander
	Lieutenant
	Specialist
	Technician
	Member
	Recruit
	Guest
	Ally = 99
)

func GetRankByName(name string) Rank {
	switch strings.ToUpper(name) {
	case "ADMIRAL":
		return Admiral
	case "COMMANDER":
		return Commander
	case "LIEUTENANT":
		return Lieutenant
	case "SPECIALIST":
		return Specialist
	case "TECHNICIAN":
		return Technician
	case "MEMBER":
		return Member
	case "RECRUIT":
		return Recruit
	case "GUEST":
		return Guest
	case "BOT":
		return Bot
	default:
		return Recruit
	}
}

func GetRankByRSIRankName(name string) Rank {
	switch strings.ToUpper(name) {
	case "DIRECTOR":
		return Admiral
	case "COMMANDER":
		return Commander
	case "LIEUTENANT":
		return Lieutenant
	case "CHIEF":
		return Specialist
	case "SPECIALIST":
		return Technician
	case "INITIATE":
		return Member
	default:
		return Guest
	}
}
