package members

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type GameplayType string

const (
	Unknown        GameplayType = "unknown"
	BountyHunting  GameplayType = "bounty_hunting"
	Engineering    GameplayType = "engineering"
	Exporation     GameplayType = "exporation"
	FpsCombat      GameplayType = "fps_combat"
	Hauling        GameplayType = "hauling"
	Medical        GameplayType = "medical"
	Mining         GameplayType = "mining"
	Reconnaissance GameplayType = "reconnaissance"
	Racing         GameplayType = "racing"
	Scrapping      GameplayType = "scrapping"
	ShipCrew       GameplayType = "ship_crew"
	ShipCombat     GameplayType = "ship_combat"
	Trading        GameplayType = "trading"
)

func ToGameplayType(s string) GameplayType {
	switch strings.ToLower(s) {
	case "bounty_hunting":
		return BountyHunting
	case "engineering":
		return Engineering
	case "exporation":
		return Exporation
	case "fps_combat":
		return FpsCombat
	case "hauling":
		return Hauling
	case "medical":
		return Medical
	case "mining":
		return Mining
	case "reconnaissance":
		return Reconnaissance
	case "racing":
		return Racing
	case "scrapping":
		return Scrapping
	case "ship_crew":
		return ShipCrew
	case "ship_combat":
		return ShipCombat
	case "trading":
		return Trading
	}

	return Unknown
}

func (g GameplayType) String() string {
	return cases.Title(language.English).String(strings.ReplaceAll(string(g), "_", " "))
}
