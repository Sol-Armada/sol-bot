package users

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
	Ally
)
