package sos

import (
	"errors"
	"time"

	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/stores"
)

type CombatLevel string

const (
	HEAVY  CombatLevel = "HEAVY"
	MEDIUM CombatLevel = "MEDIUM"
	LIGHT  CombatLevel = "LIGHT"
	SAFE   CombatLevel = "SAFE"
)

type Ticket struct {
	Id            string          `json:"id"`
	Reporter      *members.Member `json:"reporter"`
	Responder     *members.Member `json:"responder"`
	Location      string          `json:"location"`
	InCombat      bool            `json:"in_combat"`
	PVP           bool            `json:"pvp,omitempty"`
	CombatLevel   CombatLevel     `json:"combat_level,omitempty"`
	Incapacitated bool            `json:"incapacitated"`
	Resolved      bool            `json:"resolved"`
	Cancelled     bool            `json:"cancelled"`
	ResolvedAt    *time.Time      `json:"resolved_at"`
	ResolvedBy    *members.Member `json:"resolved_by"`
}

var store *stores.SOSStore

func Setup() error {
	storesClient := stores.Get()
	sosStore, ok := storesClient.GetSOSStore()
	if !ok {
		return errors.New("sos store not found")
	}
	store = sosStore

	return nil
}
