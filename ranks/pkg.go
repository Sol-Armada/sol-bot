package ranks

import "strings"

type Rank int

const (
	None Rank = iota
	Admiral
	Commander
	Lieutenant
	Specialist
	Technician
	Member
	Recruit
)

var Prefix map[Rank]string = map[Rank]string{
	Admiral:    "[ADM]",
	Commander:  "[CDR]",
	Lieutenant: "[LT]",
	Specialist: "[SPC]",
	Technician: "[TEC]",
}

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
	default:
		return None
	}
}

func GetRankByRSIRankName(name string) Rank {
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
	default:
		return None
	}
}

// String returns the string representation of the rank.
func (r Rank) String() string {
	switch r {
	case Admiral:
		return "Admiral"
	case Commander:
		return "Commander"
	case Lieutenant:
		return "Lieutenant"
	case Specialist:
		return "Specialist"
	case Technician:
		return "Technician"
	case Member:
		return "Member"
	case Recruit:
		return "Recruit"
	}
	return ""
}

func (r Rank) ShortString() string {
	switch r {
	case Admiral:
		return "ADM"
	case Commander:
		return "COM"
	case Lieutenant:
		return "LT"
	case Specialist:
		return "SPC"
	case Technician:
		return "TEC"
	}
	return ""
}
