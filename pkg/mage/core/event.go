package core

import "github.com/google/uuid"

//go:generate enumer -type=EventType -trimprefix=Evt -output=event_enumer.go

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
	EvtBeginCombat
	EvtEndStep
	EvtTapped             // fired when a permanent becomes tapped
	EvtLandPlayed         // fired when a land is played from hand
	EvtBlockersDecl       // fired once after all blockers are declared
	EvtEndOfCombat        // fired at the end of combat step, before combat groups reset
	EvtBecameUntapped     // fired when a permanent becomes untapped
	EvtMainPhase          // fired at the beginning of a main phase
	EvtCleanup            // fired at the beginning of the cleanup step (CR 514)
	EvtCreatureBlocks     // fired once per blocking creature per combat (CR 509.3a)
	EvtEntersAttacking    // fired when a creature is put onto the battlefield attacking (CR 508.4)
	EvtEntersBlocking     // fired when a creature is put onto the battlefield blocking (CR 509.4)
	EvtScry               // fired when a player scries; Amount is the number of cards scried (CR 701.18)
	EvtDiscard            // fired when a player discards a card; PlayerID = discarding player, SourceID = card ID
	EvtSacrifice          // fired when a permanent is sacrificed; PlayerID = controller, SourceID = sacrificed permanent ID
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
