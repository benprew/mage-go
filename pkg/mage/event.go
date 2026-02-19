package mage

import "github.com/google/uuid"

//go:generate enumer -type=EventType -trimprefix=Evt

// EventType identifies game events.
type EventType int

const (
	EvtZoneChange EventType = iota
	EvtCreatureDied
	EvtEntersBattlefield
	EvtLeavesBattlefield
	EvtDeclaredAttacker
	EvtDeclaredBlocker
	EvtDamageDealt
	EvtLifeGained
	EvtLifeLost
	EvtCardDrawn
	EvtSpellCast
	EvtAbilityActivated
	EvtAttach
	EvtDetach
	EvtPutIntoGraveyardFromBattlefield
	EvtUpkeep
	EvtDrawStep
	EvtEndStep
	EvtTapped     // fired when a permanent becomes tapped
	EvtLandPlayed // fired when a land is played from hand
)

// GameEvent carries data about a game event.
type GameEvent struct {
	Type     EventType
	SourceID uuid.UUID
	TargetID uuid.UUID
	PlayerID uuid.UUID
	Amount   int
	Flag     bool // context-dependent
}
