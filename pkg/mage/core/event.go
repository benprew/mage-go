package core

import "github.com/google/uuid"

//go:generate enumer -type=EventType -trimprefix=Evt -output=event_enumer.go

// EventType identifies game events.
type EventType int

const (
	EvtZoneChange EventType = iota
	EvtDeclaredAttacker
	EvtDeclaredBlocker // fired once per (blocker, attacker) pair when a blocker is declared. Flag=true on the first firing for each blocker in a combat to mark the once-per-combat "Whenever ~ blocks" trigger (CR 509.3a); Flag=false on subsequent fires for the same blocker.
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
	EvtTapped                 // fired when a permanent becomes tapped
	EvtLandPlayed             // fired when a land is played from hand
	EvtBlockersDecl           // fired once after all blockers are declared
	EvtEndOfCombat            // fired at the end of combat step, before combat groups reset
	EvtBecameUntapped         // fired when a permanent becomes untapped
	EvtMainPhase              // fired at the beginning of a main phase
	EvtCleanup                // fired at the beginning of the cleanup step (CR 514)
	EvtScry                   // Amount records cards examined, including zero (CR 701.22d).
	EvtDiscard                // fired when a player discards a card; PlayerID = discarding player, SourceID = card ID
	EvtSacrifice              // fired when a permanent is sacrificed (named action per CR 701.16); PlayerID = controller, SourceID = sacrificed permanent ID
	EvtBecomesTarget          // fired when a permanent or player becomes the target of a spell or ability (CR 603.6c, 119.5). SourceID = the spell/ability source (card or permanent), TargetID = the targeted object (permanent or player), PlayerID = the controller of the spell/ability, Flag = true if the source is an activated ability, false if a spell.
	EvtAttackersDeclared      // fired once after all attackers are declared (CR 506.4 / 603.6e). PlayerID = active player, Amount = number of attackers declared. Used by once-per-combat triggers like Duelist's Heritage and "whenever one or more creatures attack" aggregations.
	EvtCombatDamageDealt      // fired once per (controller, recipient-player) pair after combat damage is assigned in a damage step (CR 510.2). PlayerID = controller of the damaging creatures, TargetID = player who took combat damage, Amount = total combat damage dealt to that player by creatures controlled by PlayerID in the step. Used by "whenever one or more creatures you control deal combat damage to a player" aggregations.
	EvtCreatureBlocks         // fired once per blocking creature per combat (CR 509.3a)
	EvtEntersAttacking        // fired when a creature is put onto the battlefield attacking (CR 508.4)
	EvtEntersBlocking         // fired when a creature is put onto the battlefield blocking (CR 509.4)
	EvtFight                  // fired once per fight resolution (CR 701.14). SourceID = first fighter, TargetID = second fighter, PlayerID = controller of the fight effect (zero if unknown). Used by "whenever ~ fights" triggers like Neyith of the Dire Hunt.
	EvtCardsLeftGraveyard     // fired once per "burst" when one or more cards leave a player's graveyard for any other zone (exile via cost, return-to-hand, reanimate, etc.). PlayerID = graveyard owner whose cards left, Amount = number of cards that left in this burst. Used by "whenever one or more cards leave your graveyard" triggers (CR 603.10 — multiple cards moving from one zone to another via the same effect form a single zone-change "event group").
	EvtTappedForMana          // fired after a mana ability with {T} in its cost resolves and produces mana (CR 106.12a). SourceID = tapped permanent, PlayerID = activating player, Amount = mana produced by that ability.
	EvtSurveil                // Amount records cards examined, including zero (CR 701.25d).
	EvtCreatureBecomesBlocked // SourceID is the attacker when it first becomes blocked (CR 509.3c).
	EvtCreatureBlockedBy      // SourceID is the attacker and TargetID is one new blocker (CR 509.3d).
	EvtBlockerGroup           // SourceID is the creature. RelatedIDs are the new creatures that it blocks or that block it. Flag is true when the source blocks.
	EvtAttacksUnblocked       // SourceID is an attacker that is unblocked after blockers are declared. TargetID is its defending player.
	EvtCombatPair             // SourceID is one creature in a new pair; TargetID is the other. Flag is true when SourceID blocks.
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
	// SourceControllerID keeps the damage source controller after the source leaves.
	SourceControllerID uuid.UUID
	TargetID           uuid.UUID
	ActorID            uuid.UUID
	RelatedIDs         []uuid.UUID
	PlayerID           uuid.UUID
	Amount             int
	Flag               bool // context-dependent
	FromZone           Zone
	ToZone             Zone
}
