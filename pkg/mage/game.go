package mage

import (
	"errors"
	"fmt"
	"math/rand"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// Sentinel errors for common failure conditions.
var (
	ErrPlayerNotFound    = errors.New("player not found")
	ErrSourceNotFound    = errors.New("source not found on battlefield")
	ErrPermanentNotFound = errors.New("permanent not found")
	ErrCardNotInHand     = errors.New("card not found in hand")
	ErrSourceTapped      = errors.New("source is already tapped")
	ErrNoCreature        = errors.New("no creature to sacrifice")
	ErrSorcerySpeed      = errors.New("can only activate at sorcery speed")
)

// ExiledCard tracks a card in exile along with metadata about why it was exiled.
type ExiledCard struct {
	Card     Card
	ExiledBy uuid.UUID // ID of the permanent/spell that caused the exile
}

// Game is the central game state and engine.
type Game struct {
	players     []Player
	battlefield []*Permanent
	exile       []ExiledCard // exile zone with metadata
	stack       *Stack
	combat      *Combat
	effects     *EffectManager

	turn         int
	step         PhaseStep
	activePlayer int // index into players

	// Event handling
	pendingTriggers []*pendingTrigger

	// Extra turns
	extraTurns []uuid.UUID // player IDs who get extra turns

	// Per-turn step schedule and pending skips (CR 500.7–500.11).
	schedule *TurnSchedule

	// X value for the currently resolving spell
	currentX int

	// Chosen mode for the currently resolving modal spell (0-indexed)
	currentMode int

	// Amount from the triggering event (e.g. damage dealt) for triggered abilities
	currentEventAmount int
	// SourceID of the triggering event (e.g. the damager on EvtDamageDealt).
	// Read by Game.EventSourceID() during resolution of a triggered ability.
	currentEventSourceID uuid.UUID

	// Card currently being resolved (set during ResolveStackObject)
	resolvingCard Card

	// Interactive play tracking
	landsPlayedThisTurn int

	// Per-player additional-land-play allowance granted this turn ("you may
	// play an additional land this turn"). Reset alongside landsPlayedThisTurn
	// during cleanup. CR 305.2 / Explore-style effects.
	extraLandPlaysThisTurn map[uuid.UUID]int

	// Per-source flag set by OptionalCost: true if the controller chose to
	// pay the optional cost most recently. Resolution-time effects branch on
	// LastCostOptionalPaid(sourceID).
	optionalCostPaid map[uuid.UUID]bool

	// Damage tracking: maps target permanent ID -> set of source permanent IDs that dealt damage this turn
	damageDealtBy map[uuid.UUID]map[uuid.UUID]bool

	// Player damage tracking: maps player ID -> total damage taken this turn
	damageTakenThisTurn map[uuid.UUID]int

	// Artifact damage tracking: maps player ID -> artifact damage taken this turn
	artifactDamageTakenThisTurn map[uuid.UUID]int

	// Per-step combat damage aggregation. Maps controllerID -> recipientPlayerID
	// -> total combat damage dealt this damage step. Reset before each
	// ResolveDamage call; consumed to fire EvtCombatDamageDealt afterward.
	combatDamageThisStep map[uuid.UUID]map[uuid.UUID]int
	// Per-step combat damage breakdown by source permanent. Maps
	// (controllerID, recipientPlayerID) -> sourcePermanentID -> amount.
	// Populated alongside combatDamageThisStep; preserved across the
	// EvtCombatDamageDealt fire so trigger predicates can filter on source
	// attributes (e.g., "non-Human creatures you control"). Cleared at the
	// end of flushCombatDamageAggregator.
	combatDamageSourcesThisStep map[uuid.UUID]map[uuid.UUID]map[uuid.UUID]int

	// Artifact mana restriction: players who have activated artifact-only mana sources
	artifactManaOnly map[uuid.UUID]bool

	// Creature mana restriction: players who have creature-only mana (Metamorphosis)
	creatureManaOnly map[uuid.UUID]bool

	// Creatures that attacked this turn (survives combat reset for end-of-turn checks)
	attackedThisTurn map[uuid.UUID]bool

	// Blockers this turn: key = blocker ID, value = attacker IDs it blocked
	// Survives combat reset for post-combat checks (e.g., Glyph of Reincarnation)
	blockedThisTurn map[uuid.UUID][]uuid.UUID

	// Instant spells cast this turn per player (for Ichneumon Druid, etc.)
	instantsCastThisTurn map[uuid.UUID]int

	// Creature deaths this turn (total count across all players)
	creatureDeathsThisTurn int

	// Times an object (permanent or player) became the target of a spell or
	// activated ability this turn (CR 603.6c). Keyed by object ID. Used by
	// "first time each turn" target triggers like Kira, Great Glass-Spinner.
	timesTargetedThisTurn map[uuid.UUID]int

	// CleanupPriorityRounds counts how many times players have received priority
	// during a cleanup step in this game. Normally no priority is given during
	// cleanup (CR 514.3); it is only granted when a state-based action fires or
	// a triggered ability triggers during cleanup (CR 514.3a). Tests assert on
	// this to distinguish the two cases. Not reset across turns — tests take a
	// snapshot and compare deltas.
	cleanupPriorityRounds int

	// Targets of the spell currently being resolved (for ETB copy effects)
	resolvingTargets []uuid.UUID

	// Permanent currently being put onto the battlefield during PutOnBattlefield,
	// before it is appended to g.battlefield. Looked up by FindPermanent so
	// that counter-placement replacements (ETB additional, doubling) can match
	// the entering permanent during ETB resolution.
	enteringPermanent *Permanent

	// DamageDistribution chosen at cast/activation time for the spell currently
	// being resolved (CR 601.2d, divided damage). Cleared after resolution.
	resolvingDamageDistribution map[uuid.UUID]int

	// LKI snapshot of the most recently sacrificed permanent paid as a cost
	// for the spell or ability currently on the stack. Set by SacrificeSourceCost,
	// SacrificeMatchingCost, and SacrificeCreatureCost; read by effects via
	// LastSacrificed(). Cleared at the start of each cost-pay cycle and after
	// resolution.
	lastSacrificed *SacrificedSnapshot

	// LKI snapshots for permanents that have left the battlefield, keyed by
	// permanent ID. Populated by RemoveFromBattlefield so that death/leave
	// triggers (and their conditions) can read the dying permanent's
	// controller, types, P/T, and token-ness after it has moved zones.
	// Cleared at end-of-turn cleanup.
	lki map[uuid.UUID]*PermanentLKI

	// Mill amount modifiers (CR 614 replacement-style) keyed by source permanent ID.
	// Each entry maps milled-player ID -> proposed amount -> new amount; modifiers
	// stack and are dropped when the source leaves the battlefield (cleared in Apply).
	millModifiers []millModifierEntry

	// Life-gain amount modifiers (CR 614 replacement-style) keyed by source
	// permanent ID. Used by Rhox Faithmender ("If you would gain life, you
	// gain twice that much life instead.") and similar effects.
	lifeGainModifiers []lifeGainModifierEntry

	// Delayed triggers
	delayedTriggers []*DelayedTrigger

	// Coin flip results (for test determinism; popped in order)
	coinFlipResults []bool

	// Priority handler — called when a player receives priority.
	// If nil, the engine drains the stack atomically (legacy behavior).
	onPriority PriorityHandler

	// AfterPriorityAction is called after a non-pass priority action is executed.
	// Used by the interactive layer for logging and display.
	afterPriorityAction func(g *Game, playerIdx int, action PriorityAction)

	// BeforeStackResolve is called before the top of the stack is resolved
	// during a priority round. Used by the interactive layer for logging.
	beforeStackResolve func(g *Game)

	// OnDamageDealt is called after damage is dealt to a player or creature.
	// sourceName is the name of the source card/permanent, targetName is the
	// name of the target player or creature, amount is damage dealt, and
	// isCombat indicates whether it was combat damage.
	onDamageDealt func(sourceName, targetName string, amount int, isCombat bool)

	// Control flags
	stopped bool

	// resolvingCombatDamage is true while combat damage is being resolved.
	// Used by the replacement pipeline to identify combat damage actions.
	resolvingCombatDamage bool

	// castFromExilePermissions records which exiled cards specific players
	// may cast (Gonti, Lord of Luxury and similar effects). The permission
	// persists until the card leaves exile.
	castFromExilePermissions []CastableFromExilePermission

	// exileInsteadCards maps card IDs that should be exiled instead of put
	// into a graveyard this turn (Scholar of the Lost Trove rider). Value
	// is the source ID that granted the rider. Cleared at end of turn.
	exileInsteadCards map[uuid.UUID]uuid.UUID

	// lastCostReveal is the most recent card revealed by a
	// [RevealFromHandCost]. Cleared on each cost-pay cycle by the cost
	// pipeline; clients can read it during effect resolution.
	lastCostReveal Card

	// Per-turn trackers (see per_turn_trackers.go). Reset by
	// resetPerTurnTrackers in the turn-end cleanup pipeline.
	discardCountThisTurn       map[uuid.UUID]int  // playerID -> discards this turn
	lifeGainedThisTurn         map[uuid.UUID]int  // playerID -> life gained this turn
	permDamageReceivedThisTurn map[uuid.UUID]int  // permID/playerID -> damage taken this turn
	attackedOrBlockedThisTurn  map[uuid.UUID]bool // permID -> attacked or blocked this turn
}

func (g *Game) ActivePlayer() int {
	return g.activePlayer
}

// DelayedTrigger represents a one-shot triggered ability that fires when
// a specific event occurs (e.g., "destroy this creature at end of turn").
type DelayedTrigger struct {
	EventType     EventType
	TargetID      uuid.UUID
	Effects       []Effect
	SourceID      uuid.UUID
	Controller    uuid.UUID
	MatchEventID  uuid.UUID // if set, only fire when evt.SourceID matches
	MatchPlayerID uuid.UUID // if set, only fire when evt.PlayerID matches
	MatchTargetID uuid.UUID // if set, only fire when evt.TargetID matches
	Persistent    bool      // if true, trigger is not consumed after firing
}

type pendingTrigger struct {
	ability    TriggeredAbility
	event      *GameEvent
	sourceID   uuid.UUID
	controller uuid.UUID
}

// NewGame creates a new 2-player game.
func NewGame(playerA, playerB Player) *Game {
	return &Game{
		players:                     []Player{playerA, playerB},
		stack:                       NewStack(),
		combat:                      NewCombat(),
		effects:                     NewEffectManager(),
		turn:                        1,
		damageDealtBy:               make(map[uuid.UUID]map[uuid.UUID]bool),
		damageTakenThisTurn:         make(map[uuid.UUID]int),
		artifactDamageTakenThisTurn: make(map[uuid.UUID]int),
		combatDamageThisStep:        make(map[uuid.UUID]map[uuid.UUID]int),
		combatDamageSourcesThisStep: make(map[uuid.UUID]map[uuid.UUID]map[uuid.UUID]int),
		attackedThisTurn:            make(map[uuid.UUID]bool),
		blockedThisTurn:             make(map[uuid.UUID][]uuid.UUID),
		instantsCastThisTurn:        make(map[uuid.UUID]int),
		timesTargetedThisTurn:       make(map[uuid.UUID]int),
		artifactManaOnly:            make(map[uuid.UUID]bool),
		creatureManaOnly:            make(map[uuid.UUID]bool),
		schedule:                    newTurnSchedule(),
		exileInsteadCards:           make(map[uuid.UUID]uuid.UUID),
	}
}

// GetPlayer returns the player with the given ID.
func (g *Game) GetPlayer(id uuid.UUID) Player {
	for _, p := range g.players {
		if p.PlayerID() == id {
			return p
		}
	}
	return nil
}

// GetOpponent returns the other player.
func (g *Game) GetOpponent(id uuid.UUID) Player {
	for _, p := range g.players {
		if p.PlayerID() != id {
			return p
		}
	}
	return nil
}

// ActivePlayerObj returns the currently active player.
func (g *Game) ActivePlayerObj() Player {
	return g.players[g.activePlayer]
}

// NonActivePlayerObj returns the non-active player.
func (g *Game) NonActivePlayerObj() Player {
	return g.players[(g.activePlayer+1)%2]
}

// FindPermanent finds a permanent by ID on the battlefield.
// Phased-out permanents are invisible.
func (g *Game) FindPermanent(id uuid.UUID) *Permanent {
	for _, p := range g.battlefield {
		if p.PhasedOut {
			continue
		}
		if p.ID() == id {
			return p
		}
	}
	if g.enteringPermanent != nil && g.enteringPermanent.ID() == id {
		return g.enteringPermanent
	}
	return nil
}

// FindPermanentIncludingPhased finds a permanent by ID even if phased out.
func (g *Game) FindPermanentIncludingPhased(id uuid.UUID) *Permanent {
	for _, p := range g.battlefield {
		if p.ID() == id {
			return p
		}
	}
	return nil
}

// FindPermanentByName finds a permanent by name on the battlefield (first match).
// Phased-out permanents are invisible.
func (g *Game) FindPermanentByName(name string, controller uuid.UUID) *Permanent {
	for _, p := range g.battlefield {
		if p.PhasedOut {
			continue
		}
		if p.Name() == name && p.Controller == controller {
			return p
		}
	}
	return nil
}

// AnyBattlefield returns true if any permanent on the battlefield matches f.
// Phased-out permanents are invisible.
func (g *Game) AnyBattlefield(f PermanentFilter) bool {
	for _, p := range g.battlefield {
		if p.PhasedOut {
			continue
		}
		if f.Match(p, g) {
			return true
		}
	}
	return false
}

// FilterBattlefield returns all permanents on the battlefield matching f.
// Phased-out permanents are invisible.
func (g *Game) FilterBattlefield(f PermanentFilter) []*Permanent {
	var result []*Permanent
	for _, p := range g.battlefield {
		if p.PhasedOut {
			continue
		}
		if f.Match(p, g) {
			result = append(result, p)
		}
	}
	return result
}

// CountBattlefield returns the number of permanents on the battlefield matching f.
// Phased-out permanents are invisible.
func (g *Game) CountBattlefield(f PermanentFilter) int {
	n := 0
	for _, p := range g.battlefield {
		if p.PhasedOut {
			continue
		}
		if f.Match(p, g) {
			n++
		}
	}
	return n
}

// FindCardAnywhere finds a card by ID anywhere in the game.
// Also checks the currently resolving card (which may be in limbo between
// stack pop and graveyard placement during resolution).
func (g *Game) FindCardAnywhere(id uuid.UUID) Card {
	for _, p := range g.battlefield {
		if p.ID() == id {
			return p.Card
		}
	}
	// Search the stack (spells that have been cast but not yet resolved)
	for _, obj := range g.stack.Objects() {
		if obj.Card != nil && obj.Card.ID() == id {
			return obj.Card
		}
	}
	// Check the currently resolving card (popped from stack, not yet in graveyard)
	if g.resolvingCard != nil && g.resolvingCard.ID() == id {
		return g.resolvingCard
	}
	for _, pl := range g.players {
		for _, c := range pl.Hand() {
			if c.ID() == id {
				return c
			}
		}
		for _, c := range pl.Graveyard() {
			if c.ID() == id {
				return c
			}
		}
	}
	for _, ec := range g.exile {
		if ec.Card.ID() == id {
			return ec.Card
		}
	}
	return nil
}

// findCardForDamageSource finds the Card associated with a damage source ID.
// This looks at permanents, graveyard, and the currently resolving card.
func (g *Game) findCardForDamageSource(sourceID uuid.UUID) Card {
	perm := g.FindPermanent(sourceID)
	if perm != nil {
		return perm.Card
	}
	if g.resolvingCard != nil && g.resolvingCard.ID() == sourceID {
		return g.resolvingCard
	}
	return g.FindCardAnywhere(sourceID)
}

// TryPayCostFromLands attempts to pay a mana cost by tapping untapped lands
// controlled by the player. Returns true if the cost was fully paid.
// FlipCoin simulates a coin flip. Returns true for "win" (heads).
// If CoinFlipResults is non-empty, pops from the front (for test determinism).
func (g *Game) FlipCoin(playerID uuid.UUID) bool {
	if len(g.coinFlipResults) > 0 {
		result := g.coinFlipResults[0]
		g.coinFlipResults = g.coinFlipResults[1:]
		return result
	}
	return rand.Intn(2) == 0
}

func (g *Game) TryPayCostFromLands(playerID uuid.UUID, manaCostStr string) bool {
	cost := ParseManaCost(manaCostStr)

	// Collect untapped lands controlled by the player
	var lands []*Permanent
	for _, p := range g.battlefield {
		if p.Controller == playerID && !p.Tapped && p.HasType(TypeLand) {
			lands = append(lands, p)
		}
	}

	used := make(map[uuid.UUID]bool)

	// Pay colored costs first
	colorCosts := []struct {
		amount  int
		subtype string
	}{
		{cost.White, "Plains"},
		{cost.Blue, "Island"},
		{cost.Black, "Swamp"},
		{cost.Red, "Mountain"},
		{cost.Green, "Forest"},
	}

	for _, cc := range colorCosts {
		for i := 0; i < cc.amount; i++ {
			found := false
			for _, land := range lands {
				if !used[land.ID()] && land.HasSubType(cc.subtype) {
					used[land.ID()] = true
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	// Pay generic cost with any remaining untapped land
	for i := 0; i < cost.Generic; i++ {
		found := false
		for _, land := range lands {
			if !used[land.ID()] {
				used[land.ID()] = true
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Actually tap the selected lands
	for _, land := range lands {
		if used[land.ID()] {
			g.TapPermanent(land)
		}
	}

	return true
}

// PutOnBattlefield puts a card onto the battlefield under the given controller.
func (g *Game) PutOnBattlefield(card Card, controller uuid.UUID) *Permanent {
	perm := NewPermanent(card, controller)
	perm.TurnControlGained = g.turn

	// Set ability sources and controllers
	for _, a := range perm.RuntimeAbilities {
		a.SetSource(perm.ID())
		a.SetController(controller)
	}

	// EntersTapped keyword check — consumed on entry, attr cleared immediately after.
	if perm.HasKeyword(EntersTapped) || g.effects.Rules.ShouldEnterTapped(perm) {
		perm.Tapped = true
		perm.RevokeBaseAttr(EntersTapped)
	}

	// Expose the entering permanent to FindPermanent for the duration of ETB
	// resolution so counter-placement replacements (ETB additional counters,
	// doublers) can match it before it joins g.battlefield.
	g.enteringPermanent = perm
	defer func() { g.enteringPermanent = nil }()

	// Add X counters if configured (replacement effect, not a trigger).
	// Routed through AddCountersWithReplacement so ETB-additional counter
	// effects (Oona's Blackguard) and counter doublers (Branching Evolution)
	// can intercept the placement.
	for _, a := range perm.RuntimeAbilities {
		if xc, ok := a.(*EntersWithXCountersAbility); ok && g.currentX > 0 {
			g.AddCountersWithReplacement(perm, xc.CounterType, g.currentX, perm.ID(), true)
			break
		}
	}

	// Add fixed N counters if configured (replacement effect, not a trigger).
	for _, a := range perm.RuntimeAbilities {
		if nc, ok := a.(*EntersWithNCountersAbility); ok {
			g.AddCountersWithReplacement(perm, nc.CounterType, nc.Count, perm.ID(), true)
		}
	}

	// Copy creature on ETB (Vesuvan Doppelganger): copy target creature's P/T and keywords
	for _, a := range perm.RuntimeAbilities {
		if _, ok := a.(*CopyCreatureOnETBAbility); ok && len(g.resolvingTargets) > 0 {
			target := g.FindPermanent(g.resolvingTargets[0])
			if target != nil {
				g.effects.AddCopyEffect(perm.ID(), target)
			}
			break
		}
	}

	g.battlefield = append(g.battlefield, perm)

	// Register continuous effects from static abilities
	for _, a := range perm.RuntimeAbilities {
		if sa, ok := a.(*StaticAbilityHolder); ok {
			for _, e := range sa.Effects {
				// Set the source ID on the continuous effect
				g.setEffectSource(e, perm.ID())
				g.effects.Add(e)
			}
		}
	}

	// Run unconditional ETB effects (e.g. Primal Clay mode choice)
	for _, a := range perm.RuntimeAbilities {
		if etb, ok := a.(*ETBEffectAbility); ok {
			_ = etb.Effect.Apply(g, perm.ID(), controller, nil)
		}
	}

	// Run ETB-with-targets effects (e.g. Oubliette exile on entry)
	for _, a := range perm.RuntimeAbilities {
		if etb, ok := a.(*ETBWithTargetsAbility); ok && len(g.resolvingTargets) > 0 {
			_ = etb.Effect.Apply(g, perm.ID(), controller, g.resolvingTargets)
			break
		}
	}

	g.effects.Apply(g)

	g.FireEvent(GameEvent{
		Type:     EvtEntersBattlefield,
		SourceID: perm.ID(),
		PlayerID: controller,
		Amount:   g.currentX, // preserve X from resolving spell for ETB triggers
		FromZone: ZoneAny,    // engine doesn't model the precise origin of an ETB
		ToZone:   ZoneBattlefield,
	})
	g.FireEvent(GameEvent{
		Type:     EvtZoneChange,
		SourceID: perm.ID(),
		PlayerID: controller,
		Amount:   g.currentX,
		FromZone: ZoneAny,
		ToZone:   ZoneBattlefield,
	})

	return perm
}

// PutOnBattlefieldAttacking (CR 508.4) puts a creature onto the battlefield and
// marks it as attacking the given defender (player or planeswalker). For the
// purpose of trigger events and effects, such a creature is "attacking" but
// never "attacked" — AttacksTrigger does not fire. Per CR 508.4a, if the
// specified defender is no longer in the game (zero UUID or unknown player),
// the creature still enters but never becomes an attacking creature.
func (g *Game) PutOnBattlefieldAttacking(card Card, controller, defenderID uuid.UUID) *Permanent {
	if card.Owner() == uuid.Nil {
		card.SetOwner(controller)
	}
	perm := g.PutOnBattlefield(card, controller)
	if defenderID == uuid.Nil || !g.isValidDefender(defenderID) {
		return perm
	}
	g.combat.AddAttacker(perm.ID(), defenderID)
	g.FireEvent(GameEvent{
		Type:     EvtEntersAttacking,
		SourceID: perm.ID(),
		TargetID: defenderID,
		PlayerID: controller,
	})
	return perm
}

// PutOnBattlefieldBlocking (CR 509.4) puts a creature onto the battlefield and
// marks it as blocking the given attacker. Per CR 509.4, such a creature is
// "blocking" but never "blocked" — BlocksTrigger does not fire. Per CR 509.4a,
// if the specified attacker is no longer attacking, the creature still enters
// but never becomes a blocking creature.
func (g *Game) PutOnBattlefieldBlocking(card Card, controller, attackerID uuid.UUID) *Permanent {
	if card.Owner() == uuid.Nil {
		card.SetOwner(controller)
	}
	perm := g.PutOnBattlefield(card, controller)
	if attackerID == uuid.Nil || !g.combat.IsAttacking(attackerID) {
		return perm
	}
	g.combat.AddBlocker(perm.ID(), attackerID)
	g.blockedThisTurn[perm.ID()] = append(g.blockedThisTurn[perm.ID()], attackerID)
	g.FireEvent(GameEvent{
		Type:     EvtEntersBlocking,
		SourceID: perm.ID(),
		TargetID: attackerID,
		PlayerID: controller,
	})
	return perm
}

func (g *Game) isValidDefender(id uuid.UUID) bool {
	for _, p := range g.players {
		if p.PlayerID() == id {
			return true
		}
	}
	return g.FindPermanent(id) != nil
}

// setEffectSource sets the source ID on a continuous effect.
func (g *Game) setEffectSource(e ContinuousEffect, id uuid.UUID) {
	e.SetSourceID(id)
}

// turnFaceUp flips a face-down permanent face up, restoring its original characteristics.
func (g *Game) turnFaceUp(perm *Permanent) {
	if !perm.FaceDown {
		return
	}
	perm.FaceDown = false
	perm.BasePTOverride = nil
	// Restore original abilities from the card
	perm.RuntimeAbilities = nil
	for _, a := range perm.Card.Abilities() {
		cp := a
		perm.RuntimeAbilities = append(perm.RuntimeAbilities, cp)
	}
	// Set ability sources
	for _, a := range perm.RuntimeAbilities {
		a.SetSource(perm.ID())
		a.SetController(perm.Controller)
	}
	// Register continuous effects from static abilities
	for _, a := range perm.RuntimeAbilities {
		if sa, ok := a.(*StaticAbilityHolder); ok {
			for _, e := range sa.Effects {
				g.setEffectSource(e, perm.ID())
				g.effects.Add(e)
			}
		}
	}
	g.effects.Apply(g)
}

// RemoveFromBattlefield removes a permanent and handles cleanup.
func (g *Game) RemoveFromBattlefield(perm *Permanent) {
	// Snapshot LKI before any state mutation so death/leave-triggers can
	// read the dying permanent's controller, types, and P/T after the move.
	g.captureLKI(perm)

	// Capture abilities before removal (for "leaves battlefield" triggers on self)
	selfAbilities := make([]Ability, len(perm.RuntimeAbilities))
	copy(selfAbilities, perm.RuntimeAbilities)
	permID := perm.ID()
	controller := perm.Controller

	// Remove continuous effects sourced from this permanent
	g.effects.Remove(perm.ID())

	// If this was attached to something, remove it from that thing's attachments
	if perm.IsAttached() {
		host := g.FindPermanent(perm.AttachedTo)
		if host != nil {
			filtered := host.Attachments[:0]
			for _, id := range host.Attachments {
				if id != perm.ID() {
					filtered = append(filtered, id)
				}
			}
			host.Attachments = filtered
		}
	}

	// Snapshot attachments before removal so we can clean them up after
	attachments := make([]uuid.UUID, len(perm.Attachments))
	copy(attachments, perm.Attachments)

	// Remove from battlefield
	for i, p := range g.battlefield {
		if p.ID() == perm.ID() {
			g.battlefield = append(g.battlefield[:i], g.battlefield[i+1:]...)
			break
		}
	}

	g.effects.Apply(g)

	evt := GameEvent{
		Type:     EvtLeavesBattlefield,
		SourceID: permID,
		PlayerID: controller,
	}
	g.FireEvent(evt)
	// Also check the removed permanent's own triggers (since it's no longer on battlefield)
	g.checkAbilitiesForEvent(selfAbilities, &evt, permID, controller)

	// Detach equipment immediately. Auras are left for SBAs to put into the
	// graveyard so that "when enchanted creature dies" triggers can still see
	// the attachment relationship when the host's death events fire.
	for _, attID := range attachments {
		att := g.FindPermanent(attID)
		if att == nil {
			continue
		}
		if !att.HasSubType("Aura") {
			att.AttachedTo = uuid.Nil
		}
	}
}

// DestroyPermanent destroys a permanent (sends to graveyard).
func (g *Game) DestroyPermanent(perm *Permanent) {
	if perm.HasKeyword(Indestructible) {
		return
	}
	// Run through the replacement pipeline (regeneration is now a replacement)
	action := NewDestroyPermanentAction(uuid.Nil, perm.ID())
	result := g.effects.ApplyReplacements(action, g)
	if result == nil {
		return // regenerated or otherwise replaced
	}
	controller := perm.Controller
	owner := perm.Card.Owner()
	if owner == uuid.Nil {
		panic("DestroyPermanent: Pemanent has no owner")
	}

	isCreature := perm.HasType(TypeCreature)
	permID := perm.ID()
	card := perm.Card

	g.RemoveFromBattlefield(perm)
	selfAbilities := g.LKIAbilities(permID)

	p := g.GetPlayer(owner)
	if p != nil {
		p.AddToGraveyard(card)
	}

	graveyardEvt := GameEvent{
		Type:     EvtPutIntoGraveyardFromBattlefield,
		SourceID: permID,
		PlayerID: controller,
		FromZone: ZoneBattlefield,
		ToZone:   ZoneGraveyard,
	}
	g.FireEvent(graveyardEvt)
	g.checkAbilitiesForEvent(selfAbilities, &graveyardEvt, permID, controller)

	zoneEvt := GameEvent{
		Type:     EvtZoneChange,
		SourceID: permID,
		PlayerID: controller,
		FromZone: ZoneBattlefield,
		ToZone:   ZoneGraveyard,
	}
	g.FireEvent(zoneEvt)
	g.checkAbilitiesForEvent(selfAbilities, &zoneEvt, permID, controller)

	if isCreature {
		g.creatureDeathsThisTurn++
		diedEvt := GameEvent{
			Type:     EvtCreatureDied,
			SourceID: permID,
			PlayerID: controller,
			FromZone: ZoneBattlefield,
			ToZone:   ZoneGraveyard,
		}
		g.FireEvent(diedEvt)
		g.checkAbilitiesForEvent(selfAbilities, &diedEvt, permID, controller)
	}
}

// TapPermanent taps a permanent and fires the EvtTapped event.
func (g *Game) TapPermanent(perm *Permanent) {
	perm.Tapped = true
	g.FireEvent(GameEvent{
		Type:     EvtTapped,
		SourceID: perm.ID(),
		PlayerID: perm.Controller,
	})
}

// checkAbilitiesForEvent checks a set of abilities (from a removed permanent) for triggers.
func (g *Game) checkAbilitiesForEvent(abilities []Ability, evt *GameEvent, sourceID, controller uuid.UUID) {
	for _, a := range abilities {
		ta, ok := a.(TriggeredAbility)
		if !ok {
			continue
		}
		if !ta.CheckEventType(evt.Type) {
			continue
		}
		if ta.CheckTrigger(evt, g) {
			g.pendingTriggers = append(g.pendingTriggers, &pendingTrigger{
				ability:    ta,
				event:      evt,
				sourceID:   sourceID,
				controller: controller,
			})
		}
	}
}

// PutPermanentIntoGraveyard puts a permanent into its owner's graveyard without
// destroying it. This bypasses indestructible and regeneration. Used by SBAs
// (e.g., 0-toughness creatures) and other rules that move permanents to the
// graveyard without destruction.
func (g *Game) PutPermanentIntoGraveyard(perm *Permanent) {
	controller := perm.Controller
	owner := perm.Card.Owner()
	if owner == uuid.Nil {
		owner = controller
	}

	isCreature := perm.HasType(TypeCreature)
	permID := perm.ID()
	card := perm.Card

	g.RemoveFromBattlefield(perm)
	selfAbilities := g.LKIAbilities(permID)

	p := g.GetPlayer(owner)
	if p != nil {
		p.AddToGraveyard(card)
	}

	graveyardEvt := GameEvent{
		Type:     EvtPutIntoGraveyardFromBattlefield,
		SourceID: permID,
		PlayerID: controller,
		FromZone: ZoneBattlefield,
		ToZone:   ZoneGraveyard,
	}
	g.FireEvent(graveyardEvt)
	g.checkAbilitiesForEvent(selfAbilities, &graveyardEvt, permID, controller)

	zoneEvt := GameEvent{
		Type:     EvtZoneChange,
		SourceID: permID,
		PlayerID: controller,
		FromZone: ZoneBattlefield,
		ToZone:   ZoneGraveyard,
	}
	g.FireEvent(zoneEvt)
	g.checkAbilitiesForEvent(selfAbilities, &zoneEvt, permID, controller)

	if isCreature {
		g.creatureDeathsThisTurn++
		diedEvt := GameEvent{
			Type:     EvtCreatureDied,
			SourceID: permID,
			PlayerID: controller,
			FromZone: ZoneBattlefield,
			ToZone:   ZoneGraveyard,
		}
		g.FireEvent(diedEvt)
		g.checkAbilitiesForEvent(selfAbilities, &diedEvt, permID, controller)
	}
}

// Sacrifice sacrifices a permanent (like destroy but doesn't check
// indestructible). Self-referential triggers fire from the LKI snapshot's
// captured abilities (CR 700.4 / 603.6c: any battlefield → graveyard
// transition is "put into a graveyard," including sacrifice).
func (g *Game) Sacrifice(perm *Permanent) {
	controller := perm.Controller
	owner := perm.Card.Owner()
	if owner == uuid.Nil {
		owner = controller
	}

	isCreature := perm.HasType(TypeCreature)
	permID := perm.ID()
	card := perm.Card

	g.RemoveFromBattlefield(perm)
	selfTriggers := g.LKIAbilities(permID)

	p := g.GetPlayer(owner)
	if p != nil {
		p.AddToGraveyard(card)
	}

	graveyardEvt := GameEvent{
		Type:     EvtPutIntoGraveyardFromBattlefield,
		SourceID: permID,
		PlayerID: controller,
		Flag:     true, // Flag=true means this was a sacrifice (not destroy)
		FromZone: ZoneBattlefield,
		ToZone:   ZoneGraveyard,
	}
	g.FireEvent(graveyardEvt)
	g.checkAbilitiesForEvent(selfTriggers, &graveyardEvt, permID, controller)

	zoneEvt := GameEvent{
		Type:     EvtZoneChange,
		SourceID: permID,
		PlayerID: controller,
		Flag:     true, // sacrifice path
		FromZone: ZoneBattlefield,
		ToZone:   ZoneGraveyard,
	}
	g.FireEvent(zoneEvt)
	g.checkAbilitiesForEvent(selfTriggers, &zoneEvt, permID, controller)

	sacEvt := GameEvent{
		Type:     EvtSacrifice,
		SourceID: permID,
		PlayerID: controller,
		Flag:     isCreature, // Flag=true means the sacrificed permanent was a creature
		FromZone: ZoneBattlefield,
		ToZone:   ZoneGraveyard,
	}
	g.FireEvent(sacEvt)
	g.checkAbilitiesForEvent(selfTriggers, &sacEvt, permID, controller)

	if isCreature {
		g.creatureDeathsThisTurn++
		diedEvt := GameEvent{
			Type:     EvtCreatureDied,
			SourceID: permID,
			PlayerID: controller,
			FromZone: ZoneBattlefield,
			ToZone:   ZoneGraveyard,
		}
		g.FireEvent(diedEvt)
		g.checkAbilitiesForEvent(selfTriggers, &diedEvt, permID, controller)
	}
}

// PlayerDiscard removes a card from the player's hand into their graveyard and
// fires EvtDiscard. SourceID = card ID, PlayerID = discarding player.
// Returns the card and true on success.
func (g *Game) PlayerDiscard(p Player, cardID uuid.UUID) (Card, bool) {
	c, ok := p.DiscardCard(cardID)
	if !ok {
		return nil, false
	}
	g.FireEvent(GameEvent{
		Type:     EvtDiscard,
		SourceID: cardID,
		PlayerID: p.PlayerID(),
	})
	return c, true
}

// PlayerLoseLife reduces a player's life total and fires EvtLifeLost. Used by
// effects that cause direct life loss (CR 119.3). Damage that causes life loss
// fires EvtLifeLost separately from executeDamageToPlayer.
func (g *Game) PlayerLoseLife(p Player, amount int) {
	if amount <= 0 {
		return
	}
	p.LoseLife(amount)
	g.FireEvent(GameEvent{
		Type:     EvtLifeLost,
		PlayerID: p.PlayerID(),
		Amount:   amount,
	})
}

// PlayerGainLife handles life gain with replacement effects (Lich) and
// life-gain amount modifiers (Rhox Faithmender — "If you would gain
// life, you gain twice that much life instead.").
func (g *Game) PlayerGainLife(p Player, amount int) {
	if amount <= 0 {
		return
	}
	amount = g.ApplyLifeGainModifiers(p.PlayerID(), amount)
	if amount <= 0 {
		return
	}
	action := NewLifeGainAction(uuid.Nil, p.PlayerID(), amount)
	result := g.effects.ApplyReplacements(action, g)
	if result == nil {
		return // Lich or other replacement consumed it
	}
	// If the result is still a LifeGainAction, gain the life
	if lga, ok := result.(*LifeGainAction); ok {
		p.GainLife(lga.Amount())
	}
}

// sacrificePermanents sacrifices N nontoken permanents a player controls (for Lich).
func (g *Game) sacrificePermanents(playerID uuid.UUID, count int) {
	sacrificed := 0
	for sacrificed < count {
		var target *Permanent
		for _, p := range g.battlefield {
			if p.Controller == playerID {
				target = p
				break
			}
		}
		if target == nil {
			break
		}
		g.Sacrifice(target)
		sacrificed++
	}
}

// BouncePermanentToHand removes a permanent from the battlefield and adds the
// underlying card to its owner's hand, firing an EvtZoneChange{From: BF,
// To: Hand}. Self-referencing leave triggers fire via the LKI snapshot's
// captured abilities per CR 603.6c.
func (g *Game) BouncePermanentToHand(perm *Permanent) {
	if perm == nil {
		return
	}
	controller := perm.Controller
	permID := perm.ID()
	card := perm.Card
	owner := card.Owner()
	if owner == uuid.Nil {
		owner = controller
	}
	g.RemoveFromBattlefield(perm)
	selfAbilities := g.LKIAbilities(permID)
	if p := g.GetPlayer(owner); p != nil {
		p.AddToHand(card)
	}
	zoneEvt := GameEvent{
		Type:     EvtZoneChange,
		SourceID: permID,
		PlayerID: controller,
		FromZone: ZoneBattlefield,
		ToZone:   ZoneHand,
	}
	g.FireEvent(zoneEvt)
	g.checkAbilitiesForEvent(selfAbilities, &zoneEvt, permID, controller)
}

// ExilePermanent removes a permanent from the battlefield to exile and fires
// an EvtZoneChange{From: Battlefield, To: Exile}. Self-referencing leave
// triggers fire via the LKI snapshot's captured abilities per CR 603.6c.
func (g *Game) ExilePermanent(perm *Permanent) {
	controller := perm.Controller
	permID := perm.ID()
	card := perm.Card
	g.RemoveFromBattlefield(perm)
	selfAbilities := g.LKIAbilities(permID)
	g.exile = append(g.exile, ExiledCard{Card: card})
	zoneEvt := GameEvent{
		Type:     EvtZoneChange,
		SourceID: permID,
		PlayerID: controller,
		FromZone: ZoneBattlefield,
		ToZone:   ZoneExile,
	}
	g.FireEvent(zoneEvt)
	g.checkAbilitiesForEvent(selfAbilities, &zoneEvt, permID, controller)
}

// ExileCard moves a card (from any zone) to the exile zone.
func (g *Game) ExileCard(card Card, exiledBy uuid.UUID) {
	g.exile = append(g.exile, ExiledCard{Card: card, ExiledBy: exiledBy})
}

// FindExiledCard finds an exiled card by its ID.
func (g *Game) FindExiledCard(cardID uuid.UUID) *ExiledCard {
	for i := range g.exile {
		if g.exile[i].Card.ID() == cardID {
			return &g.exile[i]
		}
	}
	return nil
}

// RemoveFromExile removes a card from exile by ID and returns it.
func (g *Game) RemoveFromExile(cardID uuid.UUID) (Card, bool) {
	for i, ec := range g.exile {
		if ec.Card.ID() == cardID {
			g.exile = append(g.exile[:i], g.exile[i+1:]...)
			return ec.Card, true
		}
	}
	return nil, false
}

// RemoveExiledCardBySource removes all exiled cards with the given ExiledBy ID
// and returns them. Used by Tawnos's Coffin and similar cards.
func (g *Game) RemoveExiledCardBySource(exiledBy uuid.UUID) []ExiledCard {
	var found []ExiledCard
	remaining := g.exile[:0]
	for _, ec := range g.exile {
		if ec.ExiledBy == exiledBy {
			found = append(found, ec)
		} else {
			remaining = append(remaining, ec)
		}
	}
	g.exile = remaining
	return found
}

// flushCombatDamageAggregator fires EvtCombatDamageDealt once per (controller,
// recipient-player) pair that took combat damage this step (CR 510.2 wrap-up).
// Used by "whenever one or more creatures you control deal combat damage to a
// player" aggregator triggers. Per-creature/per-target damage is also fired by
// EvtDamageDealt; this aggregator provides once-per-step semantics.
func (g *Game) flushCombatDamageAggregator() {
	if len(g.combatDamageThisStep) == 0 {
		return
	}
	for ctrlID, byRecipient := range g.combatDamageThisStep {
		for recipID, amount := range byRecipient {
			g.FireEvent(GameEvent{
				Type:     EvtCombatDamageDealt,
				PlayerID: ctrlID,
				TargetID: recipID,
				Amount:   amount,
			})
		}
	}
	g.combatDamageThisStep = make(map[uuid.UUID]map[uuid.UUID]int)
	g.combatDamageSourcesThisStep = make(map[uuid.UUID]map[uuid.UUID]map[uuid.UUID]int)
}

// CombatDamageSourcesThisStep returns the per-source combat damage breakdown
// for the given (controllerID, recipientPlayerID) pair during the current
// EvtCombatDamageDealt fire. Returns nil if no combat damage was dealt for
// this pair this step. The map is sourcePermanentID -> amount. Trigger
// condition closures listening to EvtCombatDamageDealt may use this to
// filter on source attributes (e.g., "non-Human creatures you control").
func (g *Game) CombatDamageSourcesThisStep(controllerID, recipientID uuid.UUID) map[uuid.UUID]int {
	byCtrl, ok := g.combatDamageSourcesThisStep[controllerID]
	if !ok {
		return nil
	}
	return byCtrl[recipientID]
}

// CounterSpellOnStack removes a spell from the stack by its source ID.
// The countered spell's card goes to its owner's graveyard.
//
// If the targeted spell is currently uncounterable — either because the
// card was registered with [WithUncounterable] or a static filter
// installed via [RegisterUncounterableStatic] matches — the counter has
// no effect: the spell stays on the stack.
func (g *Game) CounterSpellOnStack(spellID uuid.UUID) {
	if obj := g.stack.FindBySourceID(spellID); obj != nil && g.IsSpellUncounterable(obj) {
		return
	}
	obj := g.stack.RemoveBySourceID(spellID)
	if obj != nil && obj.Card != nil {
		owner := g.GetPlayer(obj.Card.Owner())
		if owner != nil {
			owner.AddToGraveyard(obj.Card)
		}
	}
}

// PlayerDrawCard draws a card for the player and fires EvtCardDrawn.
func (g *Game) PlayerDrawCard(p Player) (Card, bool) {
	c, ok := p.DrawCard()
	if ok {
		g.FireEvent(GameEvent{
			Type:     EvtCardDrawn,
			PlayerID: p.PlayerID(),
		})
	}
	return c, ok
}

// PerformScry implements scry N (CR 701.18): the player looks at the top N
// cards of their library, then puts any number of them on the bottom of their
// library and the rest on top in any order. If the library has fewer than N
// cards, the player scries however many are present. Returns the number of
// cards actually scried.
//
// The placement decision is delegated to Player.ChooseScryPlacement; the engine
// validates the returned IDs and falls back to "all on top, original order" on
// any mismatch so a buggy player implementation cannot lose cards.
func (g *Game) PerformScry(p Player, n int) int {
	if p == nil || n <= 0 {
		return 0
	}
	lib := p.Library()
	if len(lib) == 0 {
		return 0
	}
	count := n
	if count > len(lib) {
		count = len(lib)
	}
	top := make([]Card, count)
	copy(top, lib[:count])

	bottom, topOrder := p.ChooseScryPlacement(top, "scry", g)
	bottom, topOrder = validateScryPlacement(top, bottom, topOrder)

	idToCard := make(map[uuid.UUID]Card, count)
	for _, c := range top {
		idToCard[c.ID()] = c
	}

	rest := lib[count:]
	newLib := make([]Card, 0, len(lib))
	for _, id := range topOrder {
		newLib = append(newLib, idToCard[id])
	}
	newLib = append(newLib, rest...)
	for _, id := range bottom {
		newLib = append(newLib, idToCard[id])
	}
	p.SetLibrary(newLib)

	g.FireEvent(GameEvent{
		Type:     EvtScry,
		PlayerID: p.PlayerID(),
		Amount:   count,
	})
	return count
}

// validateScryPlacement ensures the player's choice is a valid partition of
// `top`. On any inconsistency it returns the safe default (all on top in the
// original order) so cards are never dropped.
func validateScryPlacement(top []Card, bottom, topOrder []uuid.UUID) ([]uuid.UUID, []uuid.UUID) {
	want := make(map[uuid.UUID]bool, len(top))
	for _, c := range top {
		want[c.ID()] = true
	}
	if len(bottom)+len(topOrder) != len(top) {
		return nil, defaultScryTopOrder(top)
	}
	seen := make(map[uuid.UUID]bool, len(top))
	for _, id := range topOrder {
		if !want[id] || seen[id] {
			return nil, defaultScryTopOrder(top)
		}
		seen[id] = true
	}
	for _, id := range bottom {
		if !want[id] || seen[id] {
			return nil, defaultScryTopOrder(top)
		}
		seen[id] = true
	}
	return bottom, topOrder
}

func defaultScryTopOrder(top []Card) []uuid.UUID {
	out := make([]uuid.UUID, len(top))
	for i, c := range top {
		out[i] = c.ID()
	}
	return out
}

// DealDamageToPlayer deals damage to a player, running it through the replacement pipeline.
func (g *Game) DealDamageToPlayer(p Player, amount int, sourceID uuid.UUID) {
	if amount <= 0 {
		return
	}
	action := NewDamageToPlayerAction(sourceID, p.PlayerID(), amount, g.resolvingCombatDamage)
	result := g.effects.ApplyReplacements(action, g)
	if result == nil {
		return
	}
	g.executeAction(result)
}

// executeAction dispatches a post-replacement action to the appropriate executor.
func (g *Game) executeAction(action Action) {
	switch a := action.(type) {
	case *DamageToPlayerAction:
		g.executeDamageToPlayer(a)
	case *DamageToCreatureAction:
		g.executeDamageToCreature(a)
	}
}

// executeDamageToPlayer applies damage to a player after all replacements have been applied.
func (g *Game) executeDamageToPlayer(a *DamageToPlayerAction) {
	p := g.GetPlayer(a.PlayerID())
	if p == nil {
		return
	}
	amount := a.Amount()
	sourceID := a.ActionSource()

	// Minimum life (Ali from Cairo): cap damage so life doesn't go below 1.
	// This is checked here as a fallback for continuous effects that set the
	// GameRules flag directly rather than registering a cycle replacement.
	if g.effects.Rules.IsMinimumLifeActive(p.PlayerID()) {
		maxDamage := p.Life() - 1
		if maxDamage < 0 {
			maxDamage = 0
		}
		if amount > maxDamage {
			amount = maxDamage
		}
		if amount <= 0 {
			return
		}
	}

	// Lich replacement: instead of losing life, sacrifice permanents
	if g.effects.Rules.IsLichActive(g, p.PlayerID()) {
		g.sacrificePermanents(p.PlayerID(), amount)
	} else {
		// CR 119.9: damage dealt to a player causes that player to lose that
		// much life. Fire EvtLifeLost so "whenever a player loses life" triggers
		// see damage-induced life loss.
		p.LoseLife(amount)
		g.FireEvent(GameEvent{
			Type:     EvtLifeLost,
			PlayerID: p.PlayerID(),
			Amount:   amount,
		})
	}
	g.damageTakenThisTurn[p.PlayerID()] += amount
	// Track artifact damage separately (for Reverse Polarity)
	sourceCard := g.findCardForDamageSource(sourceID)
	if sourceCard != nil && sourceCard.HasType(TypeArtifact) {
		g.artifactDamageTakenThisTurn[p.PlayerID()] += amount
	}
	// Combat damage aggregation: track total damage this step per (controller,
	// recipient-player) pair so EvtCombatDamageDealt can fire once per pair
	// after the damage step completes (CR 510.2). Source must be a permanent
	// on the battlefield with a controller.
	if a.IsCombatDamage() {
		if srcPerm := g.FindPermanent(sourceID); srcPerm != nil {
			byCtrl, ok := g.combatDamageThisStep[srcPerm.Controller]
			if !ok {
				byCtrl = make(map[uuid.UUID]int)
				g.combatDamageThisStep[srcPerm.Controller] = byCtrl
			}
			byCtrl[p.PlayerID()] += amount
			byCtrlSrcs, ok := g.combatDamageSourcesThisStep[srcPerm.Controller]
			if !ok {
				byCtrlSrcs = make(map[uuid.UUID]map[uuid.UUID]int)
				g.combatDamageSourcesThisStep[srcPerm.Controller] = byCtrlSrcs
			}
			bySrc, ok := byCtrlSrcs[p.PlayerID()]
			if !ok {
				bySrc = make(map[uuid.UUID]int)
				byCtrlSrcs[p.PlayerID()] = bySrc
			}
			bySrc[sourceID] += amount
		}
	}
	g.FireEvent(GameEvent{
		Type:     EvtDamageDealt,
		SourceID: sourceID,
		TargetID: p.PlayerID(),
		Amount:   amount,
		Flag:     a.IsCombatDamage(),
	})
	if g.onDamageDealt != nil {
		sourceName := "unknown"
		if sc := g.findCardForDamageSource(sourceID); sc != nil {
			sourceName = sc.Name()
		}
		g.onDamageDealt(sourceName, p.Name(), amount, a.IsCombatDamage())
	}
	// Lifelink
	src := g.FindPermanent(sourceID)
	if src != nil && src.HasKeyword(Lifelink) {
		srcPlayer := g.GetPlayer(src.Controller)
		if srcPlayer != nil {
			srcPlayer.GainLife(amount)
		}
	}
	// Face-down: flip the source if it dealt damage to a player
	if src != nil && src.FaceDown {
		g.turnFaceUp(src)
	}
	// Eye for an Eye: reflect damage to source's controller (post-damage, stays inline)
	if reflectEntry, ok := g.effects.Damage.GetDamageReflection(p.PlayerID()); ok {
		if reflectEntry.chosenSource == uuid.Nil || reflectEntry.chosenSource == sourceID {
			g.effects.Damage.ClearDamageReflection(p.PlayerID())
			reflectSourceCard := g.findCardForDamageSource(sourceID)
			if reflectSourceCard != nil {
				sourceOwner := reflectSourceCard.Owner()
				if sourceOwner != uuid.Nil {
					ownerPlayer := g.GetPlayer(sourceOwner)
					if ownerPlayer != nil {
						g.DealDamageToPlayer(ownerPlayer, amount, reflectEntry.eyeSourceID)
					}
				}
			}
		}
	}
}

// DealDamageToPermanent deals damage to a permanent, running it through the replacement pipeline.
func (g *Game) DealDamageToPermanent(perm *Permanent, amount int, sourceID uuid.UUID) {
	if amount <= 0 {
		return
	}

	// Protection from source prevents all damage (static ability, pre-pipeline)
	sourceCard := g.FindCardAnywhere(sourceID)
	if sourceCard != nil && perm.HasProtectionFrom(sourceCard) {
		return
	}

	action := NewDamageToCreatureAction(sourceID, perm.ID(), amount, g.resolvingCombatDamage)
	result := g.effects.ApplyReplacements(action, g)
	if result == nil {
		return
	}
	g.executeAction(result)
}

// executeDamageToCreature applies damage to a creature after all replacements have been applied.
func (g *Game) executeDamageToCreature(a *DamageToCreatureAction) {
	perm := g.FindPermanent(a.PermanentID())
	if perm == nil {
		return
	}
	amount := a.Amount()
	sourceID := a.ActionSource()

	perm.Damage += amount
	// Track which sources dealt damage to this permanent
	if g.damageDealtBy[perm.ID()] == nil {
		g.damageDealtBy[perm.ID()] = make(map[uuid.UUID]bool)
	}
	g.damageDealtBy[perm.ID()][sourceID] = true
	g.FireEvent(GameEvent{
		Type:     EvtDamageDealt,
		SourceID: sourceID,
		TargetID: perm.ID(),
		Amount:   amount,
	})
	if g.onDamageDealt != nil {
		sourceName := "unknown"
		if sc := g.findCardForDamageSource(sourceID); sc != nil {
			sourceName = sc.Name()
		}
		g.onDamageDealt(sourceName, perm.Name(), amount, g.resolvingCombatDamage)
	}
	// Deathtouch / BasiliskTouch
	src := g.FindPermanent(sourceID)
	if src != nil && amount > 0 {
		if src.HasKeyword(Deathtouch) {
			perm.Damage = perm.CurrentToughness(g)
		} else if src.HasKeyword(BasiliskTouch) && !perm.HasSubType("Wall") {
			perm.Damage = perm.CurrentToughness(g)
		}
	}
	// Lifelink
	if src != nil && src.HasKeyword(Lifelink) {
		srcPlayer := g.GetPlayer(src.Controller)
		if srcPlayer != nil {
			g.PlayerGainLife(srcPlayer, amount)
		}
	}
	// Face-down: flip the target if it was dealt damage
	if perm.FaceDown {
		g.turnFaceUp(perm)
	}
	// Face-down: flip the source if it dealt damage
	if src != nil && src.FaceDown {
		g.turnFaceUp(src)
	}
}

// auraHostIsLegal reports whether host satisfies the aura's enchant ability
// (CR 303.4c). It checks the aura's cast-target filter against the host,
// ignoring targeting restrictions (shroud/hexproof don't make an already-attached
// aura fall off — see CR 702.11b).
func auraHostIsLegal(auraCard Card, host *Permanent, g *Game) bool {
	targets := auraCard.CastTargets()
	if len(targets) == 0 {
		return true
	}
	for _, t := range targets {
		switch tt := t.(type) {
		case *CreatureTarget:
			if !host.HasType(TypeCreature) {
				continue
			}
			if filtersMatch(tt.Filters, host, g) {
				return true
			}
		case *PermanentTarget:
			if filtersMatch(tt.Filters, host, g) {
				return true
			}
		default:
			return true
		}
	}
	return false
}

func filtersMatch(filters []PermanentFilter, p *Permanent, g *Game) bool {
	for _, f := range filters {
		if !f.Match(p, g) {
			return false
		}
	}
	return true
}

// Attach attaches source to target (for auras and equipment).
func (g *Game) Attach(sourceID, targetID uuid.UUID) {
	src := g.FindPermanent(sourceID)
	target := g.FindPermanent(targetID)
	if src == nil || target == nil {
		return
	}

	// "Can't be enchanted" — prevent enchantment attachment entirely.
	if src.HasType(TypeEnchantment) && target.HasAttr(AttrCantBeEnchanted) {
		return
	}

	// Detach from current host if any
	if src.IsAttached() {
		oldHost := g.FindPermanent(src.AttachedTo)
		if oldHost != nil {
			filtered := oldHost.Attachments[:0]
			for _, id := range oldHost.Attachments {
				if id != sourceID {
					filtered = append(filtered, id)
				}
			}
			oldHost.Attachments = filtered
		}
	}

	src.AttachedTo = targetID
	target.Attachments = append(target.Attachments, sourceID)

	g.effects.Apply(g)

	g.FireEvent(GameEvent{
		Type:     EvtAttach,
		SourceID: sourceID,
		TargetID: targetID,
	})
}

// fireBecomesTargetEvents fires EvtBecomesTarget for each declared target of a
// stack object that has just been put on the stack with its targets chosen
// (CR 603.6c, 119.5). The event is fired once per distinct target — players
// and permanents alike. evt.SourceID is the spell/ability source (the casting
// card's ID for spells, the source permanent's ID for activated abilities);
// evt.TargetID is the targeted object's ID; evt.PlayerID is the controller of
// the spell/ability; evt.Flag is true for activated abilities, false for spells.
//
// CR 603.6c: "becomes the target" triggered abilities trigger only once per
// event, even if multiple targets are chosen, BUT only the targets that the
// trigger applies to count — i.e. each affected permanent/player sees the
// event independently. This implementation fires one event per distinct target
// so triggers attached to different permanents each see "their" event.
func (g *Game) fireBecomesTargetEvents(obj *StackObject, isAbility bool) {
	if obj == nil {
		return
	}
	if g.timesTargetedThisTurn == nil {
		g.timesTargetedThisTurn = make(map[uuid.UUID]int)
	}
	seen := make(map[uuid.UUID]bool)
	for _, tid := range obj.Targets {
		if tid == uuid.Nil || seen[tid] {
			continue
		}
		seen[tid] = true
		g.timesTargetedThisTurn[tid]++
		g.FireEvent(GameEvent{
			Type:     EvtBecomesTarget,
			SourceID: obj.SourceID,
			TargetID: tid,
			PlayerID: obj.Controller,
			Flag:     isAbility,
		})
	}
}

// TimesTargetedThisTurn returns how many times the given object (permanent or
// player) has become the target of a spell or activated ability this turn.
// Counter resets at the cleanup step.
func (g *Game) TimesTargetedThisTurn(id uuid.UUID) int {
	return g.timesTargetedThisTurn[id]
}

// RegisterDelayedTrigger registers a one-shot delayed trigger that will fire
// when the specified event type occurs.
func (g *Game) RegisterDelayedTrigger(dt *DelayedTrigger) {
	g.delayedTriggers = append(g.delayedTriggers, dt)
}

// FireEvent dispatches an event and checks triggered abilities.
func (g *Game) FireEvent(evt GameEvent) {
	g.recordPerTurnEvent(&evt)
	for _, perm := range g.battlefield {
		for _, a := range perm.RuntimeAbilities {
			ta, ok := UnwrapAbility(a).(TriggeredAbility)
			if !ok {
				continue
			}
			if !ta.CheckEventType(evt.Type) {
				continue
			}
			if ta.CheckTrigger(&evt, g) {
				g.pendingTriggers = append(g.pendingTriggers, &pendingTrigger{
					ability:    ta,
					event:      &evt,
					sourceID:   perm.ID(),
					controller: perm.Controller,
				})
			}
		}
	}

	// Check delayed triggers (one-shot unless Persistent, removed after matching)
	remaining := g.delayedTriggers[:0]
	for _, dt := range g.delayedTriggers {
		if dt.EventType == evt.Type {
			if dt.MatchEventID != uuid.Nil && evt.SourceID != dt.MatchEventID {
				remaining = append(remaining, dt)
				continue
			}
			if dt.MatchPlayerID != uuid.Nil && evt.PlayerID != dt.MatchPlayerID {
				remaining = append(remaining, dt)
				continue
			}
			if dt.MatchTargetID != uuid.Nil && evt.TargetID != dt.MatchTargetID {
				remaining = append(remaining, dt)
				continue
			}
			obj := &StackObject{
				ID:            uuid.New(),
				Controller:    dt.Controller,
				SourceID:      dt.SourceID,
				IsAbility:     true,
				Effects:       dt.Effects,
				Targets:       []uuid.UUID{dt.TargetID},
				EventAmount:   evt.Amount,
				EventSourceID: evt.SourceID,
			}
			g.stack.Push(obj)
			if dt.Persistent {
				remaining = append(remaining, dt)
			}
		} else {
			remaining = append(remaining, dt)
		}
	}
	g.delayedTriggers = remaining
}

// PutTriggersOnStack puts all pending triggers onto the stack.
//
// CR 603.3b: If multiple abilities have triggered since the last time a player
// received priority, the active player's triggered abilities are put on the
// stack in any order the active player chooses, then each non-active player,
// in turn order, puts their triggered abilities on the stack in any order
// they choose. The last-put-on-stack ability ends up on top and resolves
// first.
//
// We partition pendingTriggers by controller into active and non-active
// groups, preserving source order within each group (stable), then push the
// active group first, then the non-active group. This means the non-active
// player's triggers end up on top of the stack and resolve first.
func (g *Game) PutTriggersOnStack() {
	if len(g.pendingTriggers) > 1 {
		activeID := g.ActivePlayerObj().PlayerID()
		active := make([]*pendingTrigger, 0, len(g.pendingTriggers))
		nonActive := make([]*pendingTrigger, 0, len(g.pendingTriggers))
		for _, pt := range g.pendingTriggers {
			if pt.controller == activeID {
				active = append(active, pt)
			} else {
				nonActive = append(nonActive, pt)
			}
		}
		g.pendingTriggers = append(active, nonActive...)
	}
	for _, pt := range g.pendingTriggers {
		obj := &StackObject{
			ID:         uuid.New(),
			Controller: pt.controller,
			SourceID:   pt.sourceID,
			IsAbility:  true,
		}
		obj.Effects = append(obj.Effects, pt.ability.Effects()...)
		// CR 603.3d: when a triggered ability with targets is put on the stack,
		// its controller chooses the targets. Declared AddTarget(...) entries
		// take precedence over the legacy event-derived auto-binding below;
		// triggers without declared targets fall through to the auto-bind so
		// existing card behavior is preserved.
		if declared := pt.ability.Targets(); len(declared) > 0 {
			obj.Targets = g.chooseTriggerTargets(pt, declared)
			if pt.event != nil && pt.event.Amount != 0 {
				if gt, ok := pt.ability.(*GenericTriggered); ok {
					switch gt.eventType {
					case EvtEntersBattlefield:
						obj.XValue = pt.event.Amount
					case EvtZoneChange:
						if pt.event.ToZone == ZoneBattlefield {
							obj.XValue = pt.event.Amount
						}
					case EvtDamageDealt:
						obj.EventAmount = pt.event.Amount
						obj.EventSourceID = pt.event.SourceID
					}
				}
			}
			g.stack.Push(obj)
			continue
		}
		// For triggers that need to pass the event's player as a target
		// (e.g., "deal damage to that land's controller", "that player draws"),
		// store the event PlayerID as a target on the stack object.
		if pt.event != nil {
			if gt, ok := pt.ability.(*GenericTriggered); ok {
				switch gt.eventType {
				case EvtEntersBattlefield:
					// Pass the entering permanent's ID so effects can tap/modify it
					if pt.event.SourceID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.SourceID}
					}
					// Preserve X value from the resolving spell (for X-cost ETB triggers)
					obj.XValue = pt.event.Amount
				case EvtZoneChange:
					// EvtZoneChange replaces the legacy specialized events.
					// Route based on (FromZone, ToZone) so migrated triggers
					// receive the same StackObject context as before.
					switch {
					case pt.event.ToZone == ZoneBattlefield:
						// Mirror EvtEntersBattlefield semantics.
						if pt.event.SourceID != uuid.Nil {
							obj.Targets = []uuid.UUID{pt.event.SourceID}
						}
						obj.XValue = pt.event.Amount
					case pt.event.FromZone == ZoneBattlefield:
						// Battlefield -> elsewhere (graveyard / exile / hand /
						// library). Pass the leaving permanent's ID first
						// (matches EvtCreatureDied's "dead creature ID"
						// convention used by Creature Bond / Sengir Vampire-
						// style triggers) and the controller's ID second
						// (matches EvtPutIntoGraveyardFromBattlefield's "deal
						// damage to its controller" convention).
						if pt.event.SourceID != uuid.Nil {
							obj.Targets = []uuid.UUID{pt.event.SourceID}
							if pt.event.PlayerID != uuid.Nil {
								obj.Targets = append(obj.Targets, pt.event.PlayerID)
							}
						}
					}
				case EvtDrawStep, EvtCardDrawn:
					if pt.event.PlayerID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.PlayerID}
					}
				case EvtSpellCast:
					// Pass the spell's ID and caster's player ID
					if pt.event.SourceID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.SourceID}
						if pt.event.PlayerID != uuid.Nil {
							obj.Targets = append(obj.Targets, pt.event.PlayerID)
						}
					}
				case EvtDamageDealt:
					if pt.event.TargetID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.TargetID}
					}
					obj.EventAmount = pt.event.Amount
					obj.EventSourceID = pt.event.SourceID
				case EvtTapped, EvtAbilityActivated:
					// Pass the permanent's ID so effects can identify it
					if pt.event.SourceID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.SourceID}
					}
				case EvtBecomesTarget:
					// Pass the targeted object's ID and the spell/ability
					// source so effects can either identify "this" (the target,
					// e.g. Departed Deckhand sacrificing itself) or the
					// spell/ability that did the targeting (e.g. Kira
					// countering it). Targets[0] is the targeted object;
					// Targets[1] is the offending spell/ability source.
					if pt.event.TargetID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.TargetID}
						if pt.event.SourceID != uuid.Nil {
							obj.Targets = append(obj.Targets, pt.event.SourceID)
						}
					}
				case EvtCreatureDied:
					// Pass the dead creature's ID so effects can find it in graveyard
					if pt.event.SourceID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.SourceID}
					}
				case EvtPutIntoGraveyardFromBattlefield:
					// Pass the controller's player ID so effects can deal damage/etc.
					if pt.event.PlayerID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.PlayerID}
					}
				case EvtDeclaredBlocker:
					// Pass the blocker's ID and attacker's ID
					if pt.event.SourceID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.SourceID}
						if pt.event.TargetID != uuid.Nil {
							obj.Targets = append(obj.Targets, pt.event.TargetID)
						}
					}
				case EvtCreatureBlocks:
					// Pass the blocker's ID (fired once per combat per blocker)
					if pt.event.SourceID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.SourceID}
					}
				case EvtLifeGained, EvtLifeLost:
					// Preserve the life delta for "gain/lose that much" triggers.
					// We deliberately do NOT auto-bind PlayerID as a target; the
					// affected player is rarely the same as the trigger's
					// "you" (e.g. Exquisite Blood's "you gain that much life"
					// targets the controller, not the opponent who lost life).
					obj.EventAmount = pt.event.Amount
				case EvtDiscard:
					// Pass the discarding player's ID so effects like
					// "that player loses 2 life" target the discarder.
					if pt.event.PlayerID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.PlayerID}
					}
				case EvtSacrifice:
					// Pass the sacrificing player's ID for "you sacrifice"
					// effects that need to identify the controller.
					if pt.event.PlayerID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.PlayerID}
					}
				}
			}
		}
		g.stack.Push(obj)
	}
	g.pendingTriggers = nil
}

// chooseTriggerTargets prompts the trigger's controller to choose targets for
// each declared Target on the triggered ability. Returns the flat list of
// chosen UUIDs that becomes the StackObject's Targets.
//
// If a Target has no legal candidates, it is skipped (a placeholder uuid.Nil
// is appended for required targets so positional indexing in effects survives,
// matching the convention used elsewhere). The trigger may still resolve and
// later fizzle via the normal isTargetStillLegal check at resolution time.
func (g *Game) chooseTriggerTargets(pt *pendingTrigger, declared []Target) []uuid.UUID {
	controller := g.GetPlayer(pt.controller)
	sourceCard := g.FindCardAnywhere(pt.sourceID)
	var out []uuid.UUID
	for _, t := range declared {
		t.Reset()
		possible := t.Possible(pt.controller, sourceCard, g)
		if len(possible) == 0 {
			if t.Min() > 0 {
				out = append(out, uuid.Nil)
			}
			continue
		}
		var chosen []uuid.UUID
		// Some targets specify that the *opponent* (not the trigger's
		// controller) chooses from the legal target set, e.g. Mausoleum
		// Turnkey ("of an opponent's choice"). Such targets implement the
		// OpponentChoosesTarget marker interface; we route the prompt to
		// the opposing player.
		chooser := controller
		if oct, ok := t.(interface{ OpponentChoosesTarget() bool }); ok && oct.OpponentChoosesTarget() {
			if opp := g.GetOpponent(pt.controller); opp != nil {
				chooser = opp
			}
		}
		if chooser != nil {
			chosen = chooser.ChooseTargets(possible, t.Min(), t.Max(), g)
		}
		if len(chosen) == 0 && t.Min() > 0 {
			chosen = possible[:1]
		}
		_ = t.Choose(pt.controller, sourceCard, g, chosen)
		out = append(out, chosen...)
	}
	return out
}

// ResolveStack resolves all objects on the stack (simplified: no priority passing).
func (g *Game) ResolveStack() {
	// Move any pending triggers to the stack first (e.g. from EvtCardDrawn during draw step)
	g.PutTriggersOnStack()
	for !g.stack.IsEmpty() {
		obj := g.stack.Pop()
		g.ResolveStackObject(obj)
		// Check for new triggers after each resolution
		g.PutTriggersOnStack()
	}
}

// isTargetStillLegal checks whether a target is still legal at resolution time.
func (g *Game) isTargetStillLegal(targetID uuid.UUID, sourceCard Card, controller uuid.UUID) bool {
	// Players are always legal targets (targeting doesn't check life/loss status)
	if g.GetPlayer(targetID) != nil {
		return true
	}
	// Permanents must still be on the battlefield and targetable
	if perm := g.FindPermanent(targetID); perm != nil {
		return perm.CanBeTargetedBy(sourceCard, controller, g)
	}
	// Cards in hand are legal if still in hand
	for _, p := range g.players {
		for _, c := range p.Hand() {
			if c.ID() == targetID {
				return true
			}
		}
	}
	// Graveyard cards are legal if still in graveyard
	for _, p := range g.players {
		for _, c := range p.Graveyard() {
			if c.ID() == targetID {
				return true
			}
		}
	}
	// Stack spells are legal if still on the stack
	if g.stack.FindBySourceID(targetID) != nil {
		return true
	}
	// Target no longer exists in any known zone
	return false
}

// ResolveStackObject resolves a single stack object.
func (g *Game) ResolveStackObject(obj *StackObject) {
	// Check for fizzle: if the spell/ability has targets but all are now illegal,
	// it fails to resolve (MTG rule 608.2b)
	if len(obj.Targets) > 0 {
		hasRealTargets := false
		anyLegal := false
		for _, t := range obj.Targets {
			if t == uuid.Nil {
				continue // skip nil targets (used as data slots, not real targets)
			}
			hasRealTargets = true
			if g.isTargetStillLegal(t, obj.Card, obj.Controller) {
				anyLegal = true
				break
			}
		}
		if hasRealTargets && !anyLegal {
			// Spell fizzles — put card in graveyard without resolving effects.
			// Copies of spells (CR 707.10) cease to exist instead of going to
			// any zone.
			if obj.IsCopy {
				g.CheckStateBasedActions()
				return
			}
			if obj.Card != nil && !obj.IsAbility {
				owner := obj.Card.Owner()
				if owner == uuid.Nil {
					owner = obj.Controller
				}
				if g.IsCardMarkedExileInsteadOfGraveyard(obj.Card.ID()) {
					g.ExileCard(obj.Card, obj.Card.ID())
				} else {
					p := g.GetPlayer(owner)
					if p != nil {
						p.AddToGraveyard(obj.Card)
					}
				}
			}
			g.CheckStateBasedActions()
			return
		}
	}

	g.currentX = obj.XValue
	g.currentMode = obj.ModeChoice
	g.currentEventAmount = obj.EventAmount
	g.currentEventSourceID = obj.EventSourceID
	g.resolvingCard = obj.Card
	g.resolvingTargets = obj.Targets
	g.resolvingDamageDistribution = obj.DamageDistribution
	for _, eff := range obj.Effects {
		_ = eff.Apply(g, obj.SourceID, obj.Controller, obj.Targets)
	}
	g.resolvingDamageDistribution = nil

	// Copies of spells cease to exist as they resolve (CR 707.10) — no
	// graveyard, no battlefield, no exile. The effects already ran above.
	if obj.IsCopy {
		g.currentX = 0
		g.currentMode = 0
		g.resolvingCard = nil
		g.resolvingTargets = nil
		g.ClearSacrificed()
		g.CheckStateBasedActions()
		return
	}

	// If this was a spell (not an ability), put the card in the graveyard
	if obj.Card != nil && !obj.IsAbility {
		owner := obj.Card.Owner()
		if owner == uuid.Nil {
			owner = obj.Controller
		}

		// Permanents go to the battlefield instead
		if obj.Card.HasType(TypeCreature) || obj.Card.HasType(TypeArtifact) || obj.Card.HasType(TypeEnchantment) {
			perm := g.PutOnBattlefield(obj.Card, obj.Controller)

			// Handle aura attachment (only for Aura subtype, not all enchantments)
			if obj.Card.HasType(TypeEnchantment) && len(obj.Targets) > 0 {
				for _, st := range obj.Card.SubTypes() {
					if st == "Aura" {
						g.Attach(perm.ID(), obj.Targets[0])
						break
					}
				}
			}

			g.currentX = 0
			g.currentMode = 0
			g.resolvingTargets = nil
			g.ClearSacrificed()
			g.CheckStateBasedActions()
			return
		}

		// Instants and sorceries go to graveyard, unless an active
		// "if would be put into a graveyard, exile it instead" rider
		// applies to this card (e.g. Scholar of the Lost Trove).
		if g.IsCardMarkedExileInsteadOfGraveyard(obj.Card.ID()) {
			g.ExileCard(obj.Card, obj.Card.ID())
		} else {
			p := g.GetPlayer(owner)
			if p != nil {
				p.AddToGraveyard(obj.Card)
			}
		}
	}

	g.currentX = 0
	g.currentMode = 0
	g.resolvingCard = nil
	g.resolvingTargets = nil
	g.ClearSacrificed()

	g.CheckStateBasedActions()
}

// sanitizeDamageDistribution validates a player-chosen damage division for a
// divided-damage spell or ability (CR 601.2d). The returned map is restricted
// to keys present in `targets`, has only non-negative values, and sums to
// exactly `total`. If the player's input is malformed (sum mismatch, unknown
// target, negative entry, missing assignment when total > 0) we fall back to
// "all damage to the first target," which is always a legal distribution
// because total damage must be assigned across the chosen targets.
func sanitizeDamageDistribution(in map[uuid.UUID]int, targets []uuid.UUID, total int) map[uuid.UUID]int {
	if total <= 0 || len(targets) == 0 {
		return nil
	}
	allowed := make(map[uuid.UUID]struct{}, len(targets))
	for _, t := range targets {
		if t == uuid.Nil {
			continue
		}
		allowed[t] = struct{}{}
	}
	out := make(map[uuid.UUID]int, len(in))
	sum := 0
	valid := true
	for k, v := range in {
		if _, ok := allowed[k]; !ok {
			valid = false
			break
		}
		if v < 0 {
			valid = false
			break
		}
		if v > 0 {
			out[k] = v
			sum += v
		}
	}
	if !valid || sum != total || len(out) == 0 {
		out = map[uuid.UUID]int{}
		for _, t := range targets {
			if t != uuid.Nil {
				out[t] = total
				return out
			}
		}
		return nil
	}
	return out
}

// CastSpellByName finds a card in player's hand, puts it on the stack.
func (g *Game) CastSpellByName(playerID uuid.UUID, name string, targets []uuid.UUID, xValues ...int) error {
	p := g.GetPlayer(playerID)
	if p == nil {
		return ErrPlayerNotFound
	}

	// Find card in hand
	var card Card
	for _, c := range p.Hand() {
		if c.Name() == name {
			card = c
			break
		}
	}
	if card == nil {
		return fmt.Errorf("card %s not found in hand", name)
	}

	// CR 307.1, 302.1, 303.1, 301.1 — sorcery-speed timing. Any spell that
	// is not an instant may be cast only during its controller's main phase,
	// when the stack is empty, and when that player is the active player
	// (i.e. could cast a sorcery).
	// CR 702.8 — Flash: "You may cast this spell any time you could cast an
	// instant." A card with Flash bypasses the sorcery-speed gate entirely.
	if !card.HasType(TypeInstant) && !cardHasKeyword(card, Flash) && !g.effects.Rules.HasFlashGrant(playerID, card) {
		if !g.step.IsMainPhase() {
			return ErrSorcerySpeed
		}
		if g.ActivePlayerObj().PlayerID() != playerID {
			return ErrSorcerySpeed
		}
		if !g.stack.IsEmpty() {
			return ErrSorcerySpeed
		}
	}

	// Check expansion block (City in a Bottle)
	if g.effects.Rules.IsCardExpansionBlocked(card.Name()) {
		return fmt.Errorf("can't cast %s: card is from a blocked expansion", card.Name())
	}
	// Check artifact mana restriction (Mishra's Workshop)
	if g.artifactManaOnly[playerID] && !card.HasType(TypeArtifact) {
		return fmt.Errorf("mana restriction: can only cast artifact spells")
	}
	// Check creature mana restriction (Metamorphosis)
	if g.creatureManaOnly[playerID] && !card.HasType(TypeCreature) {
		return fmt.Errorf("mana restriction: can only cast creature spells")
	}

	// Determine X value
	xValue := 0
	if len(xValues) > 0 {
		xValue = xValues[0]
	}

	mc := card.ManaCost()

	// Apply spell cost increases (e.g. Gloom)
	for _, col := range mc.Colors() {
		increase := g.effects.Rules.SpellCostIncrease(col)
		if increase > 0 {
			mc.Generic += increase
			break // only apply once per spell
		}
	}

	// Apply spell cost reductions (by color)
	for _, col := range mc.Colors() {
		reduction := g.effects.Rules.SpellCostReduction(col)
		if reduction > 0 {
			mc.Generic -= reduction
			if mc.Generic < 0 {
				mc.Generic = 0
			}
			break // only apply once per spell
		}
	}

	// Apply spell cost reductions (by type, e.g. Mana Matrix, Planar Gate)
	for _, ct := range card.Types() {
		reduction := g.effects.Rules.SpellTypeCostReduction(ct)
		if reduction > 0 {
			mc.Generic -= reduction
			if mc.Generic < 0 {
				mc.Generic = 0
			}
		}
	}

	// Conditional cost reductions (CR 601.2f). Reduce generic only; never
	// below zero. Covers static-source reducers (Warden of Evos Isle,
	// Dragonlord's Servant, Herald's Horn) and intrinsic self-reducers
	// (Bone Picker, Cryptic Serpent, Ghalta).
	if r := computeConditionalCostReduction(g, playerID, card, mc.Generic); r > 0 {
		mc.Generic -= r
	}

	// Channel: pay life for generic/X costs instead of mana
	if g.effects.Rules.IsChannelActive(playerID) && (mc.Generic > 0 || (mc.HasX && xValue > 0)) {
		// Pay colored portion from pool
		colorMC := mc
		colorMC.Generic = 0
		colorMC.HasX = false
		if !colorMC.IsZero() {
			if !p.ManaPool().CanPay(colorMC) {
				return fmt.Errorf("cannot pay mana cost %s for %s", colorMC, name)
			}
			if err := p.ManaPool().Pay(colorMC); err != nil {
				return err
			}
		}
		// Pay generic + X from life
		lifeCost := mc.Generic
		if mc.HasX {
			lifeCost += xValue * mc.XCount
		}
		g.PlayerLoseLife(p, lifeCost)
	} else {
		// Pay mana cost (auto-pay from pool)
		payMC := mc
		if mc.HasX {
			payMC.Generic += xValue * mc.XCount
		}
		if !payMC.IsZero() {
			if !p.ManaPool().CanPay(payMC) {
				return fmt.Errorf("cannot pay mana cost %s for %s", payMC, name)
			}
			if err := p.ManaPool().Pay(payMC); err != nil {
				return err
			}
		}
	}

	// Pay additional costs (sacrifice, discard, etc.)
	if bc, ok := card.(*BaseCard); ok {
		for _, cost := range bc.AdditionalCosts() {
			if !cost.CanPay(card.ID(), playerID, g) {
				return fmt.Errorf("cannot pay additional cost for %s: %s", name, cost.Text())
			}
			if err := cost.Pay(card.ID(), playerID, g); err != nil {
				return err
			}
		}
	}

	// If an additional cost set g.currentX (e.g. sacrifice-capture-CMC), use it
	if g.currentX != 0 && xValue == 0 {
		xValue = g.currentX
		g.currentX = 0
	}

	// Remove from hand
	p.RemoveFromHand(card.ID())

	// Build effects from spell abilities. Modal spells built with
	// NewModalSpell route through gatherModalSpellTargets; the chosen mode
	// supplies its own effects and targets.
	var effects []Effect
	var modalTargets [][]uuid.UUID
	modeChoice := 0
	if ms, ok := getModalSpellAbility(card); ok {
		mIdx, mTargets := g.gatherModalSpellTargets(p, card, ms)
		modeChoice = mIdx
		effects = append(effects, ms.modes[mIdx].Effects...)
		targets = mTargets
		modalTargets = make([][]uuid.UUID, len(ms.modes))
		modalTargets[mIdx] = mTargets
	} else {
		for _, a := range card.Abilities() {
			if sa, ok := a.(*SpellAbility); ok {
				effects = append(effects, sa.Effects()...)
			}
		}
	}

	obj := &StackObject{
		ID:           uuid.New(),
		Card:         card,
		Controller:   playerID,
		SourceID:     card.ID(),
		Effects:      effects,
		Targets:      targets,
		XValue:       xValue,
		ModeChoice:   modeChoice,
		ModalTargets: modalTargets,
	}

	if modes := card.Modes(); len(modes) > 0 {
		obj.ModeChoice = p.ChooseMode(modes, card.Name())
	}

	for _, eff := range effects {
		if !IsDividedDamageEffect(eff) {
			continue
		}
		total := DividedDamageTotal(eff).Resolve(g, card.ID(), playerID, targets)
		if total > 0 && len(targets) > 0 {
			dist := p.ChooseDamageDistribution(targets, total, card.Name(), g)
			obj.DamageDistribution = sanitizeDamageDistribution(dist, targets, total)
		}
		break
	}

	g.stack.Push(obj)

	// Track instant spells cast per player this turn
	if card.HasType(TypeInstant) {
		g.instantsCastThisTurn[playerID]++
	}

	g.FireEvent(GameEvent{
		Type:     EvtSpellCast,
		SourceID: card.ID(),
		PlayerID: playerID,
	})

	g.fireBecomesTargetEvents(obj, false)

	return nil
}

// addManaFromAbility resolves a mana ability's productions, adding mana to the player's pool.
// AnyColor productions prompt the player to choose a color.
func (g *Game) addManaFromAbility(ma *ManaAbility, p Player, perm *Permanent) {
	for _, prod := range ma.Productions {
		amt := prod.Amount
		if amt <= 0 {
			amt = 1
		}
		// "X mana in any combination of colors": ask once per mana point so
		// the controller can split colors arbitrarily.
		if prod.Color == AnyColor && prod.AnyCombination && amt > 1 {
			for i := 0; i < amt; i++ {
				color := p.ChooseManaColor("add mana")
				p.ManaPool().Add(color, 1)
				g.applyManaBonuses(perm, color, p)
			}
			continue
		}
		color := prod.Color
		if color == AnyColor {
			color = p.ChooseManaColor("add mana")
		}
		p.ManaPool().Add(color, amt)
		g.applyManaBonuses(perm, color, p)
	}
}

// applyManaBonuses checks for mana bonus effects when a permanent is tapped for mana.
func (g *Game) applyManaBonuses(tappedPerm *Permanent, producedColor Color, p Player) {
	for _, perm := range g.battlefield {
		for _, a := range perm.RuntimeAbilities {
			inner := UnwrapAbility(a)
			if mb, ok := inner.(*ManaBonusAbility); ok {
				if mb.AttachedOnly {
					if perm.AttachedTo == tappedPerm.ID() {
						p.ManaPool().Add(mb.BonusMana, 1)
					}
				} else if mb.Filter.Match(tappedPerm, g) {
					if mb.MatchProduced {
						p.ManaPool().Add(producedColor, 1)
					} else {
						p.ManaPool().Add(mb.BonusMana, 1)
					}
				}
			}
		}
	}
}

// CheckStateBasedActions checks and processes state-based actions.
func (g *Game) CheckStateBasedActions() {
	for {
		actions := false

		// Check for creatures with lethal damage (CR 704.5h). Indestructible
		// creatures (CR 702.12b) are skipped — the destroy would be a no-op
		// and setting actions=true would loop the SBA forever.
		var toDestroy []*Permanent
		for _, p := range g.battlefield {
			if p.HasType(TypeCreature) && p.LethalDamage(g) && !p.HasKeyword(Indestructible) {
				toDestroy = append(toDestroy, p)
				actions = true
			}
		}
		for _, p := range toDestroy {
			g.DestroyPermanent(p)
		}

		// Check for creatures with 0 or less toughness (not destruction — bypasses indestructible)
		var zeroToughness []*Permanent
		for _, p := range g.battlefield {
			if p.HasType(TypeCreature) && p.CurrentToughness(g) <= 0 {
				zeroToughness = append(zeroToughness, p)
				actions = true
			}
		}
		for _, p := range zeroToughness {
			g.PutPermanentIntoGraveyard(p)
		}

		// MTG rule 704.5q: +1/+1 and -1/-1 counter annihilation
		for _, p := range g.battlefield {
			plus := p.Counters[P1P1]
			minus := p.Counters[M1M1]
			if plus > 0 && minus > 0 {
				remove := plus
				if minus < remove {
					remove = minus
				}
				p.Counters[P1P1] -= remove
				p.Counters[M1M1] -= remove
				actions = true
			}
		}

		// Check for auras attached to nothing or illegal targets (CR 704.5m / 303.4c).
		var aurasToDrop []*Permanent
		for _, p := range g.battlefield {
			if p.HasSubType("Aura") && p.IsAttached() {
				host := g.FindPermanent(p.AttachedTo)
				if host == nil {
					aurasToDrop = append(aurasToDrop, p)
					actions = true
				} else if host.HasProtectionFrom(p.Card) {
					aurasToDrop = append(aurasToDrop, p)
					actions = true
				} else if !auraHostIsLegal(p.Card, host, g) {
					aurasToDrop = append(aurasToDrop, p)
					actions = true
				}
			}
		}
		for _, a := range aurasToDrop {
			g.DestroyPermanent(a)
		}

		// Equipment attached to a non-creature or missing host becomes unattached
		for _, p := range g.battlefield {
			if p.HasSubType("Equipment") && p.IsAttached() {
				host := g.FindPermanent(p.AttachedTo)
				if host == nil || !host.HasType(TypeCreature) {
					p.AttachedTo = uuid.Nil
					actions = true
				}
			}
		}

		// Sacrifice creatures that require a land type the controller doesn't have
		var toSacrifice []*Permanent
		for _, p := range g.battlefield {
			var landSubtype string
			for _, a := range p.RuntimeAbilities {
				if sa, ok := a.(*SacrificeUnlessLandAbility); ok {
					landSubtype = sa.LandSubtype
					break
				}
			}
			if landSubtype == "" {
				continue
			}
			hasLand := false
			for _, other := range g.battlefield {
				if other.Controller == p.Controller && other.HasSubType(landSubtype) {
					hasLand = true
					break
				}
			}
			if !hasLand {
				toSacrifice = append(toSacrifice, p)
				actions = true
			}
		}
		for _, p := range toSacrifice {
			g.Sacrifice(p)
		}

		// MTG rule 704.5j: Legend rule — if a player controls two or more legendary
		// permanents with the same name, they choose one and sacrifice the rest.
		legendCounts := make(map[uuid.UUID]map[string][]*Permanent) // controller -> name -> perms
		for _, p := range g.battlefield {
			if p.Card.HasSuperType(SuperLegendary) {
				if legendCounts[p.Controller] == nil {
					legendCounts[p.Controller] = make(map[string][]*Permanent)
				}
				legendCounts[p.Controller][p.Name()] = append(legendCounts[p.Controller][p.Name()], p)
			}
		}
		for ctrlID, byName := range legendCounts {
			for _, perms := range byName {
				if len(perms) > 1 {
					player := g.GetPlayer(ctrlID)
					keep := player.ChoosePermanent(perms, "legend rule: keep one", g)
					for _, p := range perms {
						if p != keep {
							g.Sacrifice(p)
						}
					}
					actions = true
				}
			}
		}

		// MTG rule 704.5k: World rule — if two or more permanents have the World
		// supertype, all except the most recent one are put into their owners' graveyards.
		var worldPerms []*Permanent
		for _, p := range g.battlefield {
			if p.Card.HasSuperType(SuperWorld) {
				worldPerms = append(worldPerms, p)
			}
		}
		if len(worldPerms) > 1 {
			// Keep the most recently entered one (last in Battlefield slice)
			keep := worldPerms[len(worldPerms)-1]
			for _, p := range worldPerms {
				if p != keep {
					g.PutPermanentIntoGraveyard(p)
				}
			}
			actions = true
		}

		// MTG rule 704.5c: player with 10 or more poison counters loses
		for _, p := range g.players {
			if p.PoisonCounters() >= 10 {
				p.SetLost()
			}
		}

		// MTG rule 704.5b: player who attempted to draw from empty library loses
		for _, p := range g.players {
			if p.DrewFromEmpty() {
				p.ClearDrewFromEmpty()
				p.SetLost()
			}
		}

		// MTG rule 704.5d / CR 111.7: a token in any zone other than the battlefield
		// ceases to exist. This runs after death/graveyard movement so "dies" triggers
		// fire normally — the trigger is tied to the event, not the card's continued
		// existence in the graveyard.
		for _, p := range g.players {
			var gyTokens []uuid.UUID
			for _, c := range p.Graveyard() {
				if c.IsToken() {
					gyTokens = append(gyTokens, c.ID())
				}
			}
			for _, id := range gyTokens {
				p.RemoveFromGraveyard(id)
				actions = true
			}
			var handTokens []uuid.UUID
			for _, c := range p.Hand() {
				if c.IsToken() {
					handTokens = append(handTokens, c.ID())
				}
			}
			for _, id := range handTokens {
				p.RemoveFromHand(id)
				actions = true
			}
			lib := p.Library()
			kept := lib[:0]
			removed := false
			for _, c := range lib {
				if c.IsToken() {
					removed = true
					continue
				}
				kept = append(kept, c)
			}
			if removed {
				p.SetLibrary(kept)
				actions = true
			}
		}
		exileKept := g.exile[:0]
		exileRemoved := false
		for _, ec := range g.exile {
			if ec.Card != nil && ec.Card.IsToken() {
				exileRemoved = true
				continue
			}
			exileKept = append(exileKept, ec)
		}
		if exileRemoved {
			g.exile = exileKept
			actions = true
		}

		if !actions {
			break
		}
	}

	// Put any pending triggers on the stack
	g.PutTriggersOnStack()
}

func (g *Game) doUntap() {
	active := g.ActivePlayerObj()
	// Island Sanctuary: clear protection at the start of the player's turn
	g.effects.Rules.ClearSanctuary(active.PlayerID())

	landUntapLimit := g.effects.Rules.LandUntapMax
	landsUntapped := 0
	artifactUntapLimit := g.effects.Rules.ArtifactUntapMax
	artifactsUntapped := 0
	creatureUntapLimit := g.effects.Rules.CreatureUntapMax
	creaturesUntapped := 0

	for _, p := range g.battlefield {
		if p.Controller == active.PlayerID() {
			if p.HasAttr(AttrDoesNotUntap) {
				// Does not untap — skip
			} else if p.Tapped && p.HasAttr(AttrMayNotUntap) {
				// Player may choose not to untap
				if !active.ChooseMayAbility("untap " + p.Name()) {
					continue
				}
				p.Tapped = false
				g.FireEvent(GameEvent{Type: EvtBecameUntapped, SourceID: p.ID()})
			} else if p.HasType(TypeLand) && landUntapLimit >= 0 {
				// Land with untap limit in effect
				if p.Tapped && landsUntapped < landUntapLimit {
					p.Tapped = false
					g.FireEvent(GameEvent{Type: EvtBecameUntapped, SourceID: p.ID()})
					landsUntapped++
				}
			} else if p.HasType(TypeArtifact) && !p.HasType(TypeLand) && artifactUntapLimit >= 0 {
				// Artifact (non-land) with untap limit in effect (Damping Field)
				if p.Tapped && artifactsUntapped < artifactUntapLimit {
					p.Tapped = false
					g.FireEvent(GameEvent{Type: EvtBecameUntapped, SourceID: p.ID()})
					artifactsUntapped++
				}
			} else if p.HasType(TypeCreature) && creatureUntapLimit >= 0 {
				// Creature with untap limit in effect (Smoke)
				if p.Tapped && creaturesUntapped < creatureUntapLimit {
					p.Tapped = false
					g.FireEvent(GameEvent{Type: EvtBecameUntapped, SourceID: p.ID()})
					creaturesUntapped++
				}
			} else if p.Tapped {
				p.Tapped = false
				g.FireEvent(GameEvent{Type: EvtBecameUntapped, SourceID: p.ID()})
			} else {
				p.Tapped = false
			}
			p.RevokeBaseAttr(AttrSummonSick)
		}
	}
	g.landsPlayedThisTurn = 0
	g.extraLandPlaysThisTurn = nil
}

func (g *Game) doUpkeepActions() {
	active := g.ActivePlayerObj()

	// Expire "until your next upkeep" effects for the active player.
	g.effects.RemoveUntilYourNextTurn(g, active.PlayerID())

	// Check for graveyard returns (e.g. Nether Shadow)
	g.checkGraveyardReturns(active)

	g.FireEvent(GameEvent{
		Type:     EvtUpkeep,
		PlayerID: active.PlayerID(),
	})
	g.PutTriggersOnStack()
}

// checkGraveyardReturns checks for cards in the graveyard that can return to the
// battlefield at the beginning of their controller's upkeep (e.g. Nether Shadow).
func (g *Game) checkGraveyardReturns(p Player) {
	graveyard := p.Graveyard()
	var toReturn []uuid.UUID

	for i, card := range graveyard {
		var minCreatures int
		for _, a := range card.Abilities() {
			if gra, ok := a.(*GraveyardReturnAbility); ok {
				minCreatures = gra.MinCreaturesAbove
				break
			}
		}
		if minCreatures <= 0 {
			continue
		}
		// Count creature cards above this one (higher indices = more recently added)
		creaturesAbove := 0
		for j := i + 1; j < len(graveyard); j++ {
			if graveyard[j].HasType(TypeCreature) {
				creaturesAbove++
			}
		}
		if creaturesAbove >= minCreatures {
			toReturn = append(toReturn, card.ID())
		}
	}

	for _, id := range toReturn {
		card, ok := p.RemoveFromGraveyard(id)
		if ok {
			g.PutOnBattlefield(card, p.PlayerID())
		}
	}
}

func (g *Game) doDrawActions() {
	active := g.ActivePlayerObj()

	// Fire draw step event after the normal draw so triggers can queue
	g.FireEvent(GameEvent{
		Type:     EvtDrawStep,
		PlayerID: active.PlayerID(),
	})
	g.PutTriggersOnStack()
}

func (g *Game) doDrawNormalDraw() {
	active := g.ActivePlayerObj()

	// First player doesn't draw on turn 1
	if g.turn == 1 && g.activePlayer == 0 {
		return
	}

	// Run through replacement pipeline (skip draw, Aladdin's Lamp, etc.)
	action := NewDrawCardAction(uuid.Nil, active.PlayerID(), true)
	result := g.effects.ApplyReplacements(action, g)
	if result == nil {
		return // draw was replaced (skip draw, Aladdin's Lamp, etc.)
	}

	g.PlayerDrawCard(active)
}

// applyDrawReplacement handles Aladdin's Lamp draw replacement.
// Look at top X cards, choose one, put rest on bottom randomly, draw the chosen card.
func (g *Game) applyDrawReplacement(p Player, count int) {
	lib := p.Library()
	if count > len(lib) {
		count = len(lib)
	}
	if count == 0 {
		return
	}
	candidates := make([]Card, count)
	copy(candidates, lib[:count])
	chosen := p.ChooseCardFromLibrary(candidates, "choose card from Aladdin's Lamp", g)
	if chosen == nil {
		chosen = candidates[0]
	}
	var rest []Card
	for _, c := range candidates {
		if c.ID() != chosen.ID() {
			rest = append(rest, c)
		}
	}
	rand.Shuffle(len(rest), func(i, j int) { rest[i], rest[j] = rest[j], rest[i] })
	newLib := []Card{chosen}
	newLib = append(newLib, lib[count:]...)
	newLib = append(newLib, rest...)
	p.SetLibrary(newLib)
	g.PlayerDrawCard(p)
}

// 1. Rule 508.1: First, the active player declares attackers.
// 2. Rule 508.2: Second, the active player gets priority.
func (g *Game) doDeclareAttackers() {
	active := g.ActivePlayerObj()
	attackerIDs := active.DeclareAttackers(g)
	defender := g.NonActivePlayerObj()

	// Auto-add creatures with MustAttack keyword (from Nettling Imp, etc.)
	declared := make(map[uuid.UUID]bool)
	for _, id := range attackerIDs {
		declared[id] = true
	}
	for _, p := range g.battlefield {
		if p.Controller == active.PlayerID() && p.HasAttr(AttrMustAttack) && !declared[p.ID()] {
			if p.CanDeclareAsAttacker(g) {
				attackerIDs = append(attackerIDs, p.ID())
			}
		}
	}

	for _, id := range attackerIDs {
		atk := g.FindPermanent(id)
		if atk == nil {
			continue
		}
		if !atk.CanDeclareAsAttacker(g) {
			continue
		}
		// Island Sanctuary: only flying or islandwalk creatures can attack
		if g.effects.Rules.IsSanctuaryActive(defender.PlayerID()) {
			if !atk.HasKeyword(Flying) && !atk.HasKeyword(Islandwalk) {
				continue
			}
		}

		// Tap attacker (unless vigilance)
		if !atk.HasKeyword(Vigilance) {
			g.TapPermanent(atk)
		}

		g.combat.AddAttacker(id, defender.PlayerID())
		g.attackedThisTurn[id] = true
		g.FireEvent(GameEvent{
			Type:     EvtDeclaredAttacker,
			SourceID: id,
			PlayerID: active.PlayerID(),
		})
	}

	// Form attacking bands if the player has scripted them.
	if bf, ok := active.(BandFormer); ok {
		for _, band := range bf.GetBandFormations(g.turn, g) {
			if g.isValidBand(band) {
				g.combat.AddBand(band)
			}
		}
	}

	// CR 506.5 — snapshot "attacks alone" once all attackers have been
	// declared this step.
	g.combat.SnapshotAttackedAlone()

	// CR 506.4 / 603.6e: once-per-combat "whenever one or more creatures
	// attack" trigger. Fired only if at least one creature was declared as
	// an attacker. evt.Amount carries the attacker count.
	if len(g.combat.Groups) > 0 {
		g.FireEvent(GameEvent{
			Type:     EvtAttackersDeclared,
			PlayerID: active.PlayerID(),
			Amount:   len(g.combat.Groups),
		})
	}

	g.ResolveStack()
}

// isValidBand checks that a slice of attacker IDs meets banding requirements:
// all must be currently attacking, at least one must have banding, and at most
// one may lack banding.
func (g *Game) isValidBand(memberIDs []uuid.UUID) bool {
	if len(memberIDs) < 2 {
		return false
	}
	bandingCount := 0
	nonBandingCount := 0
	for _, id := range memberIDs {
		perm := g.FindPermanent(id)
		if perm == nil || !g.combat.IsAttacking(id) {
			return false
		}
		if perm.HasKeyword(Banding) {
			bandingCount++
		} else {
			nonBandingCount++
		}
	}
	return bandingCount >= 1 && nonBandingCount <= 1
}

func (g *Game) doDeclareBlockers() {
	nonActive := g.NonActivePlayerObj()
	assignments := nonActive.DeclareBlockers(g)
	if assignments == nil {
		// Even with no scripted blockers, we may need to enforce
		// CR 509.1c "must be blocked if able" before firing the event.
		g.enforceMustBeBlockedIfAble(nonActive.PlayerID())
		g.combat.SnapshotBlockedAlone()
		g.FireEvent(GameEvent{
			Type:     EvtBlockersDecl,
			PlayerID: nonActive.PlayerID(),
		})
		return
	}

	// Check for Lure: if any attacker has MustBeBlocked, redirect all blocks to it
	var luredAttackerID uuid.UUID
	for _, group := range g.combat.Groups {
		atk := g.FindPermanent(group.AttackerID)
		if atk != nil && atk.HasKeyword(MustBeBlocked) {
			luredAttackerID = group.AttackerID
			break
		}
	}

	blockerCount := make(map[uuid.UUID]int) // how many attackers each blocker is assigned to
	var blockerOrder []uuid.UUID            // insertion order for EvtCreatureBlocks (CR 509.3a)
	for _, ba := range assignments {
		blocker := g.FindPermanent(ba.BlockerID)
		attackerID := ba.AttackerID
		// If there's a Lure creature, redirect all blocks to it
		if luredAttackerID != uuid.Nil {
			attackerID = luredAttackerID
		}
		attacker := g.FindPermanent(attackerID)
		if blocker == nil || attacker == nil {
			continue
		}
		if !blocker.CanDeclareAsBlocker(g) {
			continue
		}
		if !CanBlock(blocker, attacker, g) {
			continue
		}
		// Landwalk: if attacker has landwalk and defender controls matching land, can't be blocked
		if HasLandwalkEvasion(attacker, nonActive.PlayerID(), g) {
			continue
		}
		// Check multi-block limit: normally a creature can only block one attacker
		maxBlocks := 1
		if blocker.HasKeyword(CanBlockAny) {
			maxBlocks = 999
		} else if blocker.HasKeyword(CanBlockAdditional) {
			maxBlocks = 2
		}
		if blockerCount[ba.BlockerID] >= maxBlocks {
			continue
		}
		if blockerCount[ba.BlockerID] == 0 {
			blockerOrder = append(blockerOrder, ba.BlockerID)
		}
		blockerCount[ba.BlockerID]++
		g.combat.AddBlocker(ba.BlockerID, attackerID)
		g.blockedThisTurn[ba.BlockerID] = append(g.blockedThisTurn[ba.BlockerID], attackerID)
		g.FireEvent(GameEvent{
			Type:     EvtDeclaredBlocker,
			SourceID: ba.BlockerID,
			TargetID: attackerID,
			PlayerID: nonActive.PlayerID(),
		})
	}

	// CR 509.3a — "Whenever [creature] blocks" fires exactly once per combat
	// per blocking creature, regardless of how many attackers it blocks.
	for _, blockerID := range blockerOrder {
		g.FireEvent(GameEvent{
			Type:     EvtCreatureBlocks,
			SourceID: blockerID,
			PlayerID: nonActive.PlayerID(),
		})
	}

	// Enforce minimum-blocker restrictions (CR 509.1b — Goblin Goon style):
	// if an attacker requires N+ blockers and fewer than N are declared,
	// none of them are legal blockers. Remove them all.
	g.enforceMinimumBlockers()

	// Enforce CR 509.1c "must be blocked if able": for every attacker with
	// AttrMustBeBlockedIfAble, if no blocker has been declared and at least
	// one creature controlled by the defender could legally block it, force
	// one such creature into the block.
	g.enforceMustBeBlockedIfAble(nonActive.PlayerID())

	// CR 506.5 — snapshot "blocks alone" once all blockers have been
	// declared this step.
	g.combat.SnapshotBlockedAlone()

	// Fire EvtBlockersDecl once after all blockers are assigned
	g.FireEvent(GameEvent{
		Type:     EvtBlockersDecl,
		PlayerID: nonActive.PlayerID(),
	})
}

// enforceMinimumBlockers removes blockers from groups whose attacker has a
// minimum-blocker requirement (e.g. Goblin Goon "can't be blocked except by
// three or more creatures") that isn't met.
func (g *Game) enforceMinimumBlockers() {
	for _, group := range g.combat.Groups {
		minN := g.effects.MinBlockers(group.AttackerID)
		if minN <= 0 {
			continue
		}
		if len(group.BlockerIDs) >= minN {
			continue
		}
		// Insufficient blockers — none of them are legal. Drop them all.
		dropped := group.BlockerIDs
		group.BlockerIDs = nil
		for _, bid := range dropped {
			// Also clean up the per-turn blocked tracking for this attacker.
			rest := g.blockedThisTurn[bid][:0]
			for _, aid := range g.blockedThisTurn[bid] {
				if aid != group.AttackerID {
					rest = append(rest, aid)
				}
			}
			g.blockedThisTurn[bid] = rest
		}
	}
}

// enforceMustBeBlockedIfAble implements CR 509.1c: for each attacker with
// AttrMustBeBlockedIfAble, if no blocker is currently declared, force one
// legal blocker controlled by defenderID into the block. Picks the first
// untapped creature that passes CanBlock; ignores creatures already maxed-out
// on blocks.
func (g *Game) enforceMustBeBlockedIfAble(defenderID uuid.UUID) {
	for _, group := range g.combat.Groups {
		atk := g.FindPermanent(group.AttackerID)
		if atk == nil || !atk.HasAttr(AttrMustBeBlockedIfAble) {
			continue
		}
		if len(group.BlockerIDs) > 0 {
			continue
		}
		minN := g.effects.MinBlockers(group.AttackerID)
		if minN < 1 {
			minN = 1
		}
		// Find legal blockers controlled by the defender.
		var candidates []*Permanent
		for _, p := range g.battlefield {
			if p.Controller != defenderID {
				continue
			}
			if !p.CanDeclareAsBlocker(g) {
				continue
			}
			if !CanBlock(p, atk, g) {
				continue
			}
			if HasLandwalkEvasion(atk, defenderID, g) {
				continue
			}
			candidates = append(candidates, p)
		}
		if len(candidates) < minN {
			continue
		}
		for i := 0; i < minN && i < len(candidates); i++ {
			b := candidates[i]
			g.combat.AddBlocker(b.ID(), atk.ID())
			g.blockedThisTurn[b.ID()] = append(g.blockedThisTurn[b.ID()], atk.ID())
			g.FireEvent(GameEvent{
				Type:     EvtDeclaredBlocker,
				SourceID: b.ID(),
				TargetID: atk.ID(),
				PlayerID: defenderID,
			})
			g.FireEvent(GameEvent{
				Type:     EvtCreatureBlocks,
				SourceID: b.ID(),
				PlayerID: defenderID,
			})
		}
	}
}

// doCleanupActions performs cleanup housekeeping and places any triggers on the stack.
// Returns true if triggers were placed on the stack (requiring priority + another cleanup).
func (g *Game) doCleanupActions() bool {
	active := g.ActivePlayerObj()
	g.FireEvent(GameEvent{
		Type:     EvtCleanup,
		PlayerID: active.PlayerID(),
	})
	// Hand size discard: active player discards down to max hand size (CR 514.1)
	p := active
	maxHS := g.effects.Rules.MaxHandSize(p.PlayerID())
	for len(p.Hand()) > maxHS {
		chosen := p.ChooseCardsFromHand(1, "discard to hand size", g)
		if len(chosen) == 0 {
			break
		}
		g.PlayerDiscard(p, chosen[0].ID())
	}
	// Clear damage from all creatures
	for _, p := range g.battlefield {
		p.Damage = 0
	}
	// Clear mana pools
	for _, p := range g.players {
		p.ManaPool().Clear()
	}
	// Remove end-of-turn effects and clear turn-scoped state
	g.effects.RemoveEndOfTurn()
	g.effects.ClearReplacementsEndOfTurn()
	g.exileInsteadCards = map[uuid.UUID]uuid.UUID{}
	g.effects.Damage.ClearEndOfTurn()
	g.effects.Rules.ClearEndOfTurn()
	// Clear persistent delayed triggers (they only last "this turn")
	kept := g.delayedTriggers[:0]
	for _, dt := range g.delayedTriggers {
		if !dt.Persistent {
			kept = append(kept, dt)
		}
	}
	g.delayedTriggers = kept
	// Clear damage tracking
	g.damageDealtBy = make(map[uuid.UUID]map[uuid.UUID]bool)
	g.damageTakenThisTurn = make(map[uuid.UUID]int)
	g.artifactDamageTakenThisTurn = make(map[uuid.UUID]int)
	g.attackedThisTurn = make(map[uuid.UUID]bool)
	g.blockedThisTurn = make(map[uuid.UUID][]uuid.UUID)
	g.instantsCastThisTurn = make(map[uuid.UUID]int)
	g.timesTargetedThisTurn = make(map[uuid.UUID]int)
	g.creatureDeathsThisTurn = 0
	g.clearLKI()
	g.resetPerTurnTrackers()
	// Clear mana restrictions
	g.artifactManaOnly = make(map[uuid.UUID]bool)
	g.creatureManaOnly = make(map[uuid.UUID]bool)
	// Clear last-drawn-card tracking for all players
	for _, p := range g.players {
		p.ClearLastDrawnCard()
	}
	for _, p := range g.battlefield {
		// Clear activation tracking (Charge counters used for per-turn counts)
		p.Counters[Charge] = 0
		// Reset once-per-turn activated abilities
		for _, a := range p.RuntimeAbilities {
			if aa, ok := UnwrapAbility(a).(*SimpleActivatedAbility); ok {
				aa.ResetActivation()
			}
		}
	}
	// MTG 514.3a: if triggers fire during cleanup, put them on stack
	g.PutTriggersOnStack()
	return !g.stack.IsEmpty()
}

// Run executes the game until the stop condition.
func (g *Game) Run(stopTurn int, stopStep PhaseStep, maxTurns int) {
	for g.turn <= maxTurns {
		// Turn-skip (CR 500.11): if the active player's next turn is
		// marked as skipped, consume the skip and advance past this turn
		// without running any steps.
		activeID := g.ActivePlayerObj().PlayerID()
		if g.schedule != nil && g.schedule.consumeTurnSkip(activeID) {
			// Fall through to the extra-turn / next-player logic below.
		} else {
			if g.RunTurn(stopTurn, stopStep) {
				return
			}
		}
		// Check for extra turns
		if len(g.extraTurns) > 0 {
			extraPlayerID := g.extraTurns[0]
			g.extraTurns = g.extraTurns[1:]
			// Find the player index
			for i, p := range g.players {
				if p.PlayerID() == extraPlayerID {
					g.activePlayer = i
					break
				}
			}
		} else {
			// Next turn: swap active player
			g.activePlayer = (g.activePlayer + 1) % len(g.players)
		}
		g.turn++
	}
}

// MaxLandPlays returns the maximum number of lands that can be played this turn.
func (g *Game) MaxLandPlays() int {
	limit := 1
	if g.effects.Rules.UnlimitedLandPlays {
		limit = 999
	}
	if g.extraLandPlaysThisTurn != nil {
		limit += g.extraLandPlaysThisTurn[g.ActivePlayerObj().PlayerID()]
	}
	return limit
}

// GrantExtraLandPlay increases the active player's land-play allowance by n
// for the current turn. Cumulative across multiple grants. Reset alongside
// landsPlayedThisTurn during the cleanup step. Implements the rules-modifier
// half of Explore / Walking Atlas / Azusa-style effects.
func (g *Game) GrantExtraLandPlay(playerID uuid.UUID, n int) {
	if n <= 0 {
		return
	}
	if g.extraLandPlaysThisTurn == nil {
		g.extraLandPlaysThisTurn = make(map[uuid.UUID]int)
	}
	g.extraLandPlaysThisTurn[playerID] += n
}

// ExtraLandPlaysGrantedThisTurn returns the cumulative additional land-play
// allowance granted to the player this turn (not including the base 1).
func (g *Game) ExtraLandPlaysGrantedThisTurn(playerID uuid.UUID) int {
	if g.extraLandPlaysThisTurn == nil {
		return 0
	}
	return g.extraLandPlaysThisTurn[playerID]
}

// playLandCore moves a land from a player's hand to the battlefield and fires
// landfall triggers, but does NOT resolve the stack. Callers are responsible
// for draining the stack (via ResolveStack or RunPriorityRound).
func (g *Game) playLandCore(playerID, cardID uuid.UUID) error {
	if !g.step.IsMainPhase() {
		return fmt.Errorf("can only play lands during a main phase")
	}
	if g.ActivePlayerObj().PlayerID() != playerID {
		return fmt.Errorf("only the active player can play a land")
	}
	if g.landsPlayedThisTurn >= g.MaxLandPlays() {
		return fmt.Errorf("already played a land this turn")
	}

	p := g.GetPlayer(playerID)
	if p == nil {
		return ErrPlayerNotFound
	}

	card, ok := p.RemoveFromHand(cardID)
	if !ok {
		return ErrCardNotInHand
	}
	if !card.HasType(TypeLand) {
		p.AddToHand(card)
		return fmt.Errorf("card is not a land")
	}
	// Check expansion block (City in a Bottle)
	if g.effects.Rules.IsCardExpansionBlocked(card.Name()) {
		p.AddToHand(card)
		return fmt.Errorf("can't play %s: card is from a blocked expansion", card.Name())
	}

	g.PutOnBattlefield(card, playerID)
	g.landsPlayedThisTurn++

	g.FireEvent(GameEvent{
		Type:     EvtLandPlayed,
		SourceID: card.ID(),
		PlayerID: playerID,
		Amount:   g.landsPlayedThisTurn, // which land number this was
	})
	g.PutTriggersOnStack()

	return nil
}

// PlayLand moves a land from a player's hand to the battlefield.
func (g *Game) PlayLand(playerID, cardID uuid.UUID) error {
	if err := g.playLandCore(playerID, cardID); err != nil {
		return err
	}
	g.ResolveStack()
	return nil
}

// TapForMana taps a permanent for mana using its mana ability.
func (g *Game) TapForMana(playerID, permanentID uuid.UUID) error {
	perm := g.FindPermanent(permanentID)
	if perm == nil {
		return ErrPermanentNotFound
	}
	if perm.Controller != playerID {
		return fmt.Errorf("you don't control that permanent")
	}
	if perm.Tapped {
		return fmt.Errorf("permanent is already tapped")
	}

	// Find a mana ability
	for _, a := range perm.RuntimeAbilities {
		if ma, ok := a.(*ManaAbility); ok {
			if !perm.CanTapForEffect(g) {
				return fmt.Errorf("creature has summoning sickness")
			}
			g.TapPermanent(perm)
			p := g.GetPlayer(playerID)
			if p != nil {
				g.addManaFromAbility(ma, p, perm)
			}
			return nil
		}
	}
	return fmt.Errorf("permanent has no mana ability")
}

// manaSourceInfo describes a mana source available for tapping.
type manaSourceInfo struct {
	PermanentID uuid.UUID
	Name        string
	Color       Color
	Amount      int // total mana produced (including all productions)
}

// countManaBonuses returns how many bonus mana a permanent would produce when tapped.
func (g *Game) countManaBonuses(permanentID uuid.UUID) int {
	tappedPerm := g.FindPermanent(permanentID)
	if tappedPerm == nil {
		return 0
	}
	bonus := 0
	for _, perm := range g.battlefield {
		for _, a := range perm.RuntimeAbilities {
			inner := UnwrapAbility(a)
			if mb, ok := inner.(*ManaBonusAbility); ok {
				if mb.AttachedOnly {
					if perm.AttachedTo == tappedPerm.ID() {
						bonus++
					}
				} else if mb.Filter.Match(tappedPerm, g) {
					bonus++
				}
			}
		}
	}
	return bonus
}

// getUntappedManaSources returns all untapped permanents with mana abilities for a player.
func (g *Game) getUntappedManaSources(playerID uuid.UUID) []manaSourceInfo {
	var sources []manaSourceInfo
	for _, perm := range g.battlefield {
		if perm.Controller != playerID || perm.Tapped {
			continue
		}
		// Skip summoning-sick creatures without haste
		if !perm.CanTapForEffect(g) {
			continue
		}
		for _, a := range perm.RuntimeAbilities {
			if ma, ok := a.(*ManaAbility); ok {
				sources = append(sources, manaSourceInfo{
					PermanentID: perm.ID(),
					Name:        perm.Name(),
					Color:       ma.PrimaryColor(),
					Amount:      ma.ProducedAmount(),
				})
				break // one entry per permanent even if it has multiple mana abilities
			}
		}
	}
	return sources
}

// AutoTapForCost taps untapped lands/mana sources to pay a mana cost,
// accounting for mana already in the player's pool.
func (g *Game) AutoTapForCost(playerID uuid.UUID, mc ManaCost) error {
	p := g.GetPlayer(playerID)
	if p == nil {
		return fmt.Errorf("player not found")
	}
	pool := p.ManaPool()
	sources := g.getUntappedManaSources(playerID)

	// Subtract mana already in pool from what we need to tap
	colorNeeds := []struct {
		color Color
		need  int
	}{
		{White, mc.White},
		{Blue, mc.Blue},
		{Black, mc.Black},
		{Red, mc.Red},
		{Green, mc.Green},
	}
	needed := map[Color]int{}
	poolUsedForColor := 0
	poolByColor := map[Color]int{}
	for _, cn := range colorNeeds {
		poolByColor[cn.color] = pool.Count(cn.color)
	}
	for _, cn := range colorNeeds {
		have := poolByColor[cn.color]
		used := min(have, cn.need)
		poolByColor[cn.color] -= used
		poolUsedForColor += used
		needed[cn.color] = cn.need - used
	}
	availableForHybridFromSources := map[Color]int{}
	for _, src := range sources {
		availableForHybridFromSources[src.Color] += src.Amount
	}
	for _, h := range mc.Hybrid {
		pa, pb := poolByColor[h.A], poolByColor[h.B]
		if pa >= pb && pa > 0 {
			poolByColor[h.A] = pa - 1
			poolUsedForColor++
			continue
		}
		if pb > 0 {
			poolByColor[h.B] = pb - 1
			poolUsedForColor++
			continue
		}
		sa, sb := availableForHybridFromSources[h.A], availableForHybridFromSources[h.B]
		if sa >= sb && sa > 0 {
			availableForHybridFromSources[h.A] = sa - 1
			needed[h.A]++
		} else if sb > 0 {
			availableForHybridFromSources[h.B] = sb - 1
			needed[h.B]++
		} else {
			return fmt.Errorf("insufficient mana for hybrid %s", h)
		}
	}
	poolRemaining := pool.TotalMana() - poolUsedForColor
	genericNeeded := mc.Generic - min(mc.Generic, poolRemaining)

	var toTap []uuid.UUID
	surplusMana := 0

	// First pass: tap sources for exact color requirements
	for color, count := range needed {
		for i := 0; i < count; i++ {
			found := false
			for j, src := range sources {
				if src.Color == color && src.PermanentID != uuid.Nil {
					toTap = append(toTap, src.PermanentID)
					surplusMana += g.countManaBonuses(src.PermanentID)
					sources[j].PermanentID = uuid.Nil // mark as used
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("insufficient %s mana", color)
			}
		}
	}

	// Surplus mana from bonuses on color-tapped sources reduces generic need
	genericNeeded -= min(genericNeeded, surplusMana)

	// Second pass: tap remaining sources for generic mana
	for genericNeeded > 0 {
		found := false
		for j, src := range sources {
			if src.PermanentID != uuid.Nil {
				toTap = append(toTap, src.PermanentID)
				produced := src.Amount + g.countManaBonuses(src.PermanentID)
				genericNeeded -= produced
				sources[j].PermanentID = uuid.Nil
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("insufficient mana for generic cost")
		}
	}

	// Actually tap all selected sources
	for _, id := range toTap {
		if err := g.TapForMana(playerID, id); err != nil {
			return err
		}
	}

	return nil
}

// CanAfford returns true if a player has enough mana (pool + untapped sources) to pay a cost.
func (g *Game) CanAfford(playerID uuid.UUID, mc ManaCost) bool {
	p := g.GetPlayer(playerID)
	if p == nil {
		return false
	}

	// Build a hypothetical pool: current pool + what untapped sources would produce
	hypothetical := NewManaPool()
	pool := p.ManaPool()
	for _, color := range []Color{White, Blue, Black, Red, Green, Colorless} {
		hypothetical.Add(color, pool.Count(color))
	}
	for _, src := range g.getUntappedManaSources(playerID) {
		hypothetical.Add(src.Color, src.Amount)
		hypothetical.Add(src.Color, g.countManaBonuses(src.PermanentID))
	}
	hypothetical.ManaConversions = pool.ManaConversions

	return hypothetical.CanPay(mc)
}

// ActivatableInfo describes an activated ability on a permanent that can currently be used.
type ActivatableInfo struct {
	PermanentID   uuid.UUID
	PermanentName string
	AbilityIndex  int
	Description   string
}

// GetCastableSpells returns cards in a player's hand they can currently cast.
func (g *Game) GetCastableSpells(playerID uuid.UUID) []Card {
	p := g.GetPlayer(playerID)
	if p == nil {
		return nil
	}
	isMainPhase := g.step.IsMainPhase()
	isActive := g.ActivePlayerObj().PlayerID() == playerID

	var castable []Card
	for _, card := range p.Hand() {
		if card.HasType(TypeLand) {
			continue
		}
		// CR 702.8 — Flash lets a spell be cast any time you could cast an instant,
		// bypassing the sorcery-speed gate below.
		hasFlash := cardHasKeyword(card, Flash) || g.effects.Rules.HasFlashGrant(p.PlayerID(), card)
		// Sorceries can only be cast at sorcery speed (main phase, active player, empty stack)
		if card.HasType(TypeSorcery) && !hasFlash {
			if !isMainPhase || !isActive || !g.stack.IsEmpty() {
				continue
			}
		}
		// Creatures/artifacts/enchantments are sorcery speed
		if (card.HasType(TypeCreature) || card.HasType(TypeArtifact) || card.HasType(TypeEnchantment)) && !hasFlash {
			if !isMainPhase || !isActive || !g.stack.IsEmpty() {
				continue
			}
		}
		// Check mana (ignore X costs for now - those are always "castable" if base cost met)
		mc := card.ManaCost()
		checkMC := mc
		if mc.HasX {
			// For X spells, check if we can pay the non-X portion
			checkMC = ManaCost{
				Generic: mc.Generic,
				White:   mc.White,
				Blue:    mc.Blue,
				Black:   mc.Black,
				Red:     mc.Red,
				Green:   mc.Green,
			}
		}
		if !g.CanAfford(playerID, checkMC) {
			continue
		}
		// Spells with targets (e.g. auras) can't be cast if no legal targets exist
		if ct := card.CastTargets(); len(ct) > 0 {
			hasLegalTarget := false
			for _, t := range ct {
				if len(t.Possible(playerID, card, g)) > 0 {
					hasLegalTarget = true
					break
				}
			}
			if !hasLegalTarget {
				continue
			}
		}
		castable = append(castable, card)
	}
	return castable
}

// GetPlayableLands returns the lands a player can legally play right now.
// Checks: main phase, active player, land-play limit (respects Fastbond etc.),
// and expansion blocks.
func (g *Game) GetPlayableLands(playerID uuid.UUID) []Card {
	if !g.step.IsMainPhase() {
		return nil
	}
	if g.ActivePlayerObj().PlayerID() != playerID {
		return nil
	}
	if g.landsPlayedThisTurn >= g.MaxLandPlays() {
		return nil
	}
	p := g.GetPlayer(playerID)
	if p == nil {
		return nil
	}
	var lands []Card
	for _, c := range p.Hand() {
		if c.HasType(TypeLand) {
			lands = append(lands, c)
		}
	}
	return lands
}

// GetActivatableAbilities returns activated abilities the player can currently use.
func (g *Game) GetActivatableAbilities(playerID uuid.UUID) []ActivatableInfo {
	var result []ActivatableInfo
	for _, perm := range g.battlefield {
		isOwner := perm.Controller == playerID
		for i, a := range perm.RuntimeAbilities {
			aa, ok := UnwrapAbility(a).(ActivatedAbility)
			if !ok {
				continue
			}
			// Check if this ability can be used by non-controllers
			if !isOwner {
				saa, isSAA := UnwrapAbility(a).(*SimpleActivatedAbility)
				if !isSAA || !saa.AnyPlayerMayUse {
					continue
				}
			}
			// If opponent-only, the controller cannot activate it
			if isOwner {
				saa, isSAA := UnwrapAbility(a).(*SimpleActivatedAbility)
				if isSAA && saa.OpponentOnlyMayUse {
					continue
				}
			}
			// Skip mana abilities - those are handled separately
			if _, isMana := UnwrapAbility(a).(*ManaAbility); isMana {
				continue
			}
			if !aa.CanActivate(playerID, g) {
				continue
			}
			if aa.SorcerySpeed() && !g.step.IsMainPhase() {
				continue
			}
			desc := ""
			for _, e := range aa.Effects() {
				if desc != "" {
					desc += ", "
				}
				desc += e.Text()
			}
			result = append(result, ActivatableInfo{
				PermanentID:   perm.ID(),
				PermanentName: perm.Name(),
				AbilityIndex:  i,
				Description:   desc,
			})
		}
	}
	return result
}

// CastSpellByID casts a spell from a player's hand by card ID. It resolves
// the ID to the card's name and delegates to CastSpellByName, which owns the
// full cost-modification and additional-cost pipeline. Using a thin adapter
// here means the priority loop and scripted tests go through the same code
// path as the harness / interactive layer.
func (g *Game) CastSpellByID(playerID, cardID uuid.UUID, targets []uuid.UUID, xValue int) error {
	p := g.GetPlayer(playerID)
	if p == nil {
		return ErrPlayerNotFound
	}
	var card Card
	for _, c := range p.Hand() {
		if c.ID() == cardID {
			card = c
			break
		}
	}
	if card == nil {
		return ErrCardNotInHand
	}

	// Check expansion block (City in a Bottle)
	if g.effects.Rules.IsCardExpansionBlocked(card.Name()) {
		return fmt.Errorf("can't cast %s: card is from a blocked expansion", card.Name())
	}

	// Compute payment mana cost
	mc := card.ManaCost()
	payMC := mc
	if mc.HasX {
		payMC.Generic += xValue * mc.XCount
	}
	if !payMC.IsZero() && !p.ManaPool().CanPay(payMC) {
		if err := g.AutoTapForCost(playerID, payMC); err != nil {
			return fmt.Errorf("cannot pay for %s: %v", card.Name(), err)
		}
	}
	return g.CastSpellByName(playerID, card.Name(), targets, xValue)
}

// ActivateAbilityByIndex activates an ability on a permanent by index.
func (g *Game) ActivateAbilityByIndex(playerID, permanentID uuid.UUID, abilityIndex int, targets []uuid.UUID) error {
	perm := g.FindPermanent(permanentID)
	if perm == nil {
		return ErrPermanentNotFound
	}
	if abilityIndex < 0 || abilityIndex >= len(perm.RuntimeAbilities) {
		return fmt.Errorf("invalid ability index")
	}

	inner := UnwrapAbility(perm.RuntimeAbilities[abilityIndex])

	// Handle mana abilities (don't use the stack)
	if ma, ok := inner.(*ManaAbility); ok {
		if perm.Tapped || !perm.CanTapForEffect(g) {
			return fmt.Errorf("cannot tap %s for mana", perm.Name())
		}
		g.TapPermanent(perm)
		p := g.GetPlayer(playerID)
		if p != nil {
			g.addManaFromAbility(ma, p, perm)
		}
		g.FireEvent(GameEvent{
			Type:     EvtAbilityActivated,
			SourceID: perm.ID(),
			PlayerID: playerID,
		})
		return nil
	}

	aa, ok := inner.(ActivatedAbility)
	if !ok {
		return fmt.Errorf("not an activated ability")
	}
	if !aa.CanActivate(playerID, g) {
		return fmt.Errorf("cannot activate ability")
	}
	if saa, isSAA := inner.(*SimpleActivatedAbility); isSAA && saa.OpponentOnlyMayUse {
		if perm.Controller == playerID {
			return fmt.Errorf("only opponents may activate this ability")
		}
	}
	// CR 307.5 / 602.5d — "activate only as a sorcery" means the ability can
	// only be activated when its controller could cast a sorcery: main phase,
	// active player, empty stack.
	if aa.SorcerySpeed() {
		if !g.step.IsMainPhase() {
			return ErrSorcerySpeed
		}
		if g.ActivePlayerObj().PlayerID() != playerID {
			return ErrSorcerySpeed
		}
		if !g.stack.IsEmpty() {
			return ErrSorcerySpeed
		}
	}

	// Validate targets
	if len(aa.Targets()) > 0 && len(targets) > 0 {
		for i, t := range aa.Targets() {
			if i >= len(targets) {
				break
			}
			if tf, ok := t.(interface{ Filter() PermanentFilter }); ok {
				targetPerm := g.FindPermanent(targets[i])
				if targetPerm != nil {
					if !tf.Filter().Match(targetPerm, g) {
						return fmt.Errorf("invalid target for ability")
					}
					if !targetPerm.CanBeTargetedBy(perm.Card, playerID, g) {
						return fmt.Errorf("target cannot be targeted")
					}
				}
			} else {
				possible := t.Possible(playerID, perm.Card, g)
				found := false
				for _, pid := range possible {
					if pid == targets[i] {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("invalid target for ability")
				}
			}
		}
	}

	// Auto-tap lands to pay mana costs, then pay all costs
	for _, c := range aa.Costs() {
		if mc, ok := c.(*ManaCostPayment); ok {
			reduced := mc.reducedCost(perm.ID(), g)
			if !reduced.IsZero() {
				if err := g.AutoTapForCost(playerID, reduced); err != nil {
					return err
				}
			}
		}
	}
	for _, c := range aa.Costs() {
		if err := c.Pay(perm.ID(), playerID, g); err != nil {
			return err
		}
	}

	// Mark once-per-turn abilities as used
	if saa, ok := aa.(*SimpleActivatedAbility); ok {
		saa.MarkActivated()
	}

	obj := &StackObject{
		ID:         uuid.New(),
		Controller: playerID,
		SourceID:   perm.ID(),
		IsAbility:  true,
		Targets:    targets,
		XValue:     g.currentX,
	}
	obj.Effects = append(obj.Effects, aa.Effects()...)

	// Modal abilities: choose mode at activation time
	if modes := perm.Card.Modes(); len(modes) > 0 {
		p := g.GetPlayer(playerID)
		if p != nil {
			obj.ModeChoice = p.ChooseMode(modes, perm.Card.Name())
		}
	}

	for _, eff := range aa.Effects() {
		if !IsDividedDamageEffect(eff) {
			continue
		}
		total := DividedDamageTotal(eff).Resolve(g, perm.ID(), playerID, targets)
		if total > 0 && len(targets) > 0 {
			pl := g.GetPlayer(playerID)
			if pl != nil {
				dist := pl.ChooseDamageDistribution(targets, total, perm.Card.Name(), g)
				obj.DamageDistribution = sanitizeDamageDistribution(dist, targets, total)
			}
		}
		break
	}

	g.stack.Push(obj)

	// Fire EvtAbilityActivated
	hasTapCost := false
	for _, c := range aa.Costs() {
		if _, ok := c.(*tapSourceCost); ok {
			hasTapCost = true
			break
		}
	}
	g.FireEvent(GameEvent{
		Type:     EvtAbilityActivated,
		SourceID: perm.ID(),
		PlayerID: playerID,
		Flag:     hasTapCost,
	})

	g.fireBecomesTargetEvents(obj, true)

	return nil
}

// ResolveTopOfStack resolves just the top item on the stack.
func (g *Game) ResolveTopOfStack() {
	if g.stack.IsEmpty() {
		return
	}
	obj := g.stack.Pop()
	g.ResolveStackObject(obj)
	g.PutTriggersOnStack()
}

// IsGameOver returns true if any player has 0 or less life.
func (g *Game) IsGameOver() bool {
	for _, p := range g.players {
		if !p.IsAlive() {
			return true
		}
	}
	return false
}

// Winner returns the name of the winning player, or "" if no winner yet.
func (g *Game) Winner() string {
	for _, p := range g.players {
		if !p.IsAlive() {
			return g.GetOpponent(p.PlayerID()).Name()
		}
	}
	return ""
}

func (g *Game) Stack() *Stack {
	return g.stack
}
