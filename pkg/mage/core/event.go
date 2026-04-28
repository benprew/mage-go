package core

import "github.com/google/uuid"

//go:generate enumer -type=EventType -trimprefix=Evt -output=event_enumer.go

// EventType identifies game events.
type EventType int

const (
	EvtZoneChange EventType = iota
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
	EvtSacrifice          // fired when a permanent is sacrificed (named action per CR 701.16); PlayerID = controller, SourceID = sacrificed permanent ID
	EvtBecomesTarget      // fired when a permanent or player becomes the target of a spell or ability (CR 603.6c, 119.5). SourceID = the spell/ability source (card or permanent), TargetID = the targeted object (permanent or player), PlayerID = the controller of the spell/ability, Flag = true if the source is an activated ability, false if a spell.
	EvtAttackersDeclared  // fired once after all attackers are declared (CR 506.4 / 603.6e). PlayerID = active player, Amount = number of attackers declared. Used by once-per-combat triggers like Duelist's Heritage and "whenever one or more creatures attack" aggregations.
	EvtCombatDamageDealt  // fired once per (controller, recipient-player) pair after combat damage is assigned in a damage step (CR 510.2). PlayerID = controller of the damaging creatures, TargetID = player who took combat damage, Amount = total combat damage dealt to that player by creatures controlled by PlayerID in the step. Used by "whenever one or more creatures you control deal combat damage to a player" aggregations.
)

// GameEvent carries data about a game event.
//
// FromZone and ToZone are populated for EvtZoneChange and identify the
// move's endpoints (CR 603.10). For non-zone events both default to
// ZoneLibrary's zero value and should be ignored — match on the EventType
// first.
type GameEvent struct {
	Type     EventType
	SourceID uuid.UUID
	TargetID uuid.UUID
	PlayerID uuid.UUID
	Amount   int
	Flag     bool // context-dependent
	FromZone Zone
	ToZone   Zone
}
