package users

type Rank int

const (
	Bot Rank = iota
	Administrator
	Commander
	Officer
	Specialist
	Technician
	Member
	Recruit
)
