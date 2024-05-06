package members

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type GameplayTypes string

const (
	BountyHunting  GameplayTypes = "bounty_hunting"
	Engineering    GameplayTypes = "engineering"
	Exporation     GameplayTypes = "exporation"
	FpsCombat      GameplayTypes = "fps_combat"
	Hauling        GameplayTypes = "hauling"
	Medical        GameplayTypes = "medical"
	Mining         GameplayTypes = "mining"
	Reconnaissance GameplayTypes = "reconnaissance"
	Racing         GameplayTypes = "racing"
	Scrapping      GameplayTypes = "scrapping"
	ShipCrew       GameplayTypes = "ship_crew"
	ShipCombat     GameplayTypes = "ship_combat"
	Trading        GameplayTypes = "trading"
)

func (g GameplayTypes) String() string {
	return cases.Title(language.English).String(strings.ReplaceAll(string(g), "_", " "))
}
