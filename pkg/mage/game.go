package mage

import (
	"errors"
	"fmt"
	"maps"
	"math/rand"
	"slices"

	. "github.com/benprew/mage-go/pkg/mage/core"

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
	ErrAnteDisabled      = errors.New("ante is not enabled for this game")
	ErrAnteSettled       = errors.New("ante result has already been settled")
)

// ExiledCard records a card in exile and the reason for the exile.
//
// FaceDown: When true, the card is in exile face down (CR 707, 406.3).
// The engine hides the card characteristics from unauthorized players.
// Cards such as Gonti, Lord of Luxury use this field to hide cards from opponents.
//
// RevealedTo: Contains the IDs of players permitted to inspect the card (CR 408).
// This list usually includes the exiling player and the card owner.
// Other players see only that a face-down card is in exile.
// Unauthorized players cannot inspect card characteristics.
type ExiledCard struct {
	Card       Card
	ExiledBy   uuid.UUID // ID of the permanent/spell that caused the exile
	FaceDown   bool
	RevealedTo []uuid.UUID
}

// VisibleTo reports whether the given player may inspect this exiled card's
// identity. Face-up exiled cards are visible to everyone; face-down exiled
// cards are visible only to players in RevealedTo.
func (ec *ExiledCard) VisibleTo(playerID uuid.UUID) bool {
	if !ec.FaceDown {
		return true
	}
	return slices.Contains(ec.RevealedTo, playerID)
}

// Game is the central game state and engine.
type Game struct {
	players        []Player
	anteEnabled    bool
	originalOwners map[uuid.UUID]uuid.UUID
	anteResult     []OwnershipChange
	anteSettled    bool
	battlefield    []*Permanent
	// Search clones share battlefield permanent pointers until a branch writes
	// to a permanent. ownedPermanents contains IDs whose pointers are private to
	// this Game after sharing began.
	battlefieldShared      bool
	battlefieldSliceShared bool
	ownedPermanents        map[uuid.UUID]struct{}
	exile                  []ExiledCard // exile zone with metadata
	stack                  *Stack
	combat                 *Combat
	effects                *EffectManager
	layer2Controllers      map[uuid.UUID]uuid.UUID
	manaScratch            []manaSourceInfo

	turn         int
	step         PhaseStep
	activePlayer int // index into players

	// Event handling
	pendingTriggers []*pendingTrigger

	// armedStateTriggers tracks state-triggered abilities (CR 603.8) that have
	// fired but not yet rearmed. Key is the (sourceID, abilityID) pair. A
	// state trigger only re-fires once its condition has gone false and then
	// true again — armedStateTriggers[key] == true means "fired since last
	// observed false; do not fire again until it's seen false."
	armedStateTriggers map[stateTriggerKey]bool

	// Extra turns
	extraTurns []uuid.UUID // player IDs who get extra turns

	// Per-turn step schedule and pending skips (CR 500.7–500.11).
	schedule *TurnSchedule

	// ResolutionState owns transient resolution and cost scratch state.
	resolution ResolutionState

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

	// Game-long damage tracking by permanent (e.g. The Fallen)
	damageDealtToPlayersByPermanent    map[uuid.UUID]map[uuid.UUID]bool
	damageDealtToPermanentsByPermanent map[uuid.UUID]map[uuid.UUID]bool

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

	// Creatures that attacked this turn (survives combat reset for end-of-turn checks)
	attackedThisTurn map[uuid.UUID]bool

	// Creatures that attacked during a player's last turn: playerID -> permID -> bool (for Tangle Kelp)
	attackedLastTurn map[uuid.UUID]map[uuid.UUID]bool

	// Blockers this turn: key = blocker ID, value = attacker IDs it blocked
	// Survives combat reset for post-combat checks (e.g., Glyph of Reincarnation)
	blockedThisTurn map[uuid.UUID][]uuid.UUID

	// Instant spells cast this turn per player (for Ichneumon Druid, etc.)
	instantsCastThisTurn  map[uuid.UUID]int
	sorceriesCastThisTurn map[uuid.UUID]int

	// Creature deaths this turn (total count across all players)
	creatureDeathsThisTurn int

	// Number of untapped lands the active player controlled at the start of
	// this turn (snapshot taken before the untap step). Read by Power Surge.
	untappedLandsAtTurnStart map[uuid.UUID]int

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

	// Permanent currently being put onto the battlefield during PutOnBattlefield,
	// before it is appended to g.battlefield. Looked up by FindPermanent so
	// that counter-placement replacements (ETB additional, doubling) can match
	// the entering permanent during ETB resolution.
	enteringPermanent *Permanent

	// skipNextUntap tracks object-specific one-shot untap replacements. Entries
	// survive control changes and are consumed only by an actual untap attempt
	// during the permanent's then-controller's untap step.
	skipNextUntap map[uuid.UUID]int

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

	// Random integer results (for test determinism; popped in order)
	randomResults []int

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

	// Per-turn trackers (see per_turn_trackers.go). Reset by
	// resetPerTurnTrackers in the turn-end cleanup pipeline.
	discardCountThisTurn       map[uuid.UUID]int  // playerID -> discards this turn
	lifeGainedThisTurn         map[uuid.UUID]int  // playerID -> life gained this turn
	permDamageReceivedThisTurn map[uuid.UUID]int  // permID/playerID -> damage taken this turn
	attackedOrBlockedThisTurn  map[uuid.UUID]bool // permID -> attacked or blocked this turn
	playerCastSpellThisTurn    map[uuid.UUID]bool // playerID -> cast any spell this turn
	playerAttackedThisTurn     map[uuid.UUID]bool // playerID -> declared at least one attacker this turn
	cardsDrawnThisTurn         map[uuid.UUID]int  // playerID -> count of cards drawn this turn (per Zurzoth, Chaos Rider et al.)
	cardsLeftGraveyardThisTurn map[uuid.UUID]int  // playerID -> cards that left that player's graveyard this turn
	cardsPutIntoExileThisTurn  int                // total cards put into exile this turn
	exileZoneChangesPending    map[uuid.UUID]int  // cardID -> ZoneExile events already counted, awaiting ExileCard append

	// Per-duel objective counters (see per_duel_trackers.go). Unlike the
	// per-turn trackers these are NOT reset between turns; they accumulate for
	// the whole game and are read at game-over (used by the s30 quest system).
	duelSpellsCastByColor map[uuid.UUID]map[Color]int    // playerID -> color -> spells cast
	duelSpellsCastByType  map[uuid.UUID]map[CardType]int // playerID -> type -> spells cast
	duelLandsPlayed       map[uuid.UUID]int              // playerID -> lands played
	duelAttackersDeclared map[uuid.UUID]int              // playerID -> attackers declared
	duelCreatureDeaths    map[uuid.UUID]int              // controllerID -> own creatures that died
	duelNonCombatDamage   map[uuid.UUID]int              // dealerID -> non-combat damage dealt to the opposing player

	// customState is a per-game string-keyed bag for set-specific keyword
	// support to stash auxiliary state (e.g. Paradigm "have I resolved a
	// spell with this name yet?" tracking). Populate via paradigmStateOf
	// and similar accessors in keyword_sos.go. Survives the lifetime of
	// the game; cleared per-game via NewGame.
	customState map[string]any
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
	MatchFromZone Zone      // for EvtZoneChange: ZoneAny to skip the from check
	MatchToZone   Zone      // for EvtZoneChange: ZoneAny to skip the to check
	MatchFlag     bool      // if true, only fire when evt.Flag is true (e.g. combat damage)
	Persistent    bool      // if true, trigger is not consumed after firing
}

type pendingTrigger struct {
	ability    TriggeredAbility
	event      *GameEvent
	sourceID   uuid.UUID
	controller uuid.UUID
}

type stateTriggerKey struct {
	sourceID  uuid.UUID
	abilityID uuid.UUID
}

// NewGame creates a new 2-player game.
func NewGame(playerA, playerB Player) *Game {
	return newGame(playerA, playerB, false)
}

func newGame(playerA, playerB Player, anteEnabled bool) *Game {
	return &Game{
		players:                            []Player{playerA, playerB},
		anteEnabled:                        anteEnabled,
		originalOwners:                     make(map[uuid.UUID]uuid.UUID),
		stack:                              NewStack(),
		combat:                             NewCombat(),
		effects:                            NewEffectManager(),
		resolution:                         NewResolutionState(),
		turn:                               1,
		damageDealtBy:                      make(map[uuid.UUID]map[uuid.UUID]bool),
		damageDealtToPlayersByPermanent:    make(map[uuid.UUID]map[uuid.UUID]bool),
		damageDealtToPermanentsByPermanent: make(map[uuid.UUID]map[uuid.UUID]bool),
		damageTakenThisTurn:                make(map[uuid.UUID]int),
		artifactDamageTakenThisTurn:        make(map[uuid.UUID]int),
		combatDamageThisStep:               make(map[uuid.UUID]map[uuid.UUID]int),
		combatDamageSourcesThisStep:        make(map[uuid.UUID]map[uuid.UUID]map[uuid.UUID]int),
		attackedThisTurn:                   make(map[uuid.UUID]bool),
		blockedThisTurn:                    make(map[uuid.UUID][]uuid.UUID),
		instantsCastThisTurn:               make(map[uuid.UUID]int),
		sorceriesCastThisTurn:              make(map[uuid.UUID]int),
		timesTargetedThisTurn:              make(map[uuid.UUID]int),
		armedStateTriggers:                 make(map[stateTriggerKey]bool),
		schedule:                           newTurnSchedule(),
		exileInsteadCards:                  make(map[uuid.UUID]uuid.UUID),
		customState:                        make(map[string]any),
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

// PhaseOut phases the permanent with the given ID out of the battlefield, along
// with any Auras and Equipment attached to it (CR 702.26f, "phasing out
// indirectly"). Counters, attachments, and other on-permanent state are
// retained while phased out (CR 702.26d). It returns the IDs that were phased
// out — the permanent followed by its attachments — so the caller can later
// phase exactly that set back in with PhaseIn.
func (g *Game) PhaseOut(id uuid.UUID) []uuid.UUID {
	perm := g.MutablePermanent(id)
	if perm == nil {
		return nil
	}
	perm.PhasedOut = true
	phased := []uuid.UUID{id}
	for _, p := range g.battlefield {
		if p.PhasedOut || p.AttachedTo != id {
			continue
		}
		if !p.HasSubType("Aura") && !p.HasSubType("Equipment") {
			continue
		}
		att := g.MutablePermanent(p.ID())
		if att == nil {
			continue
		}
		att.PhasedOut = true
		phased = append(phased, att.ID())
	}
	return phased
}

// PhaseIn phases the given permanents back onto the battlefield. IDs that are
// not currently phased out are skipped.
func (g *Game) PhaseIn(ids []uuid.UUID) {
	for _, id := range ids {
		perm := g.mutablePermanentIncludingPhased(id)
		if perm != nil && perm.PhasedOut {
			perm.PhasedOut = false
		}
	}
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

// MutablePermanent returns an owned battlefield permanent pointer suitable for
// mutation. Search clones initially share permanent pointers; the first write
// to a shared permanent clones that permanent and replaces the battlefield
// entry in this Game only. Phased-out permanents are invisible.
func (g *Game) MutablePermanent(id uuid.UUID) *Permanent {
	for i, p := range g.battlefield {
		if p.PhasedOut {
			continue
		}
		if p.ID() == id {
			return g.mutablePermanentAt(i)
		}
	}
	if g.enteringPermanent != nil && g.enteringPermanent.ID() == id {
		return g.enteringPermanent
	}
	return nil
}

// mutablePermanentIncludingPhased is the mutable counterpart to
// FindPermanentIncludingPhased.
func (g *Game) mutablePermanentIncludingPhased(id uuid.UUID) *Permanent {
	for i, p := range g.battlefield {
		if p.ID() == id {
			return g.mutablePermanentAt(i)
		}
	}
	return nil
}

// MutablePermanentIncludingPhased is the exported mutable counterpart to
// FindPermanentIncludingPhased. Returns an owned battlefield permanent
// pointer suitable for mutation (triggers copy-on-write under search).
func (g *Game) MutablePermanentIncludingPhased(id uuid.UUID) *Permanent {
	return g.mutablePermanentIncludingPhased(id)
}

func (g *Game) mutablePermanentAt(i int) *Permanent {
	p := g.battlefield[i]
	if !g.battlefieldShared {
		return p
	}
	if g.ownedPermanents != nil {
		if _, ok := g.ownedPermanents[p.ID()]; ok {
			return p
		}
	}
	g.ensureBattlefieldSliceOwned()
	cp := new(Permanent)
	clonePermanentInto(cp, p)
	g.battlefield[i] = cp
	g.addOwnedPermanent(cp)
	return cp
}

func (g *Game) ensureBattlefieldSliceOwned() {
	if !g.battlefieldSliceShared {
		return
	}
	if len(g.battlefield) == 0 {
		g.battlefield = nil
		g.battlefieldSliceShared = false
		return
	}
	cp := make([]*Permanent, len(g.battlefield))
	copy(cp, g.battlefield)
	g.battlefield = cp
	g.battlefieldSliceShared = false
}

func (g *Game) addOwnedPermanent(p *Permanent) {
	if p == nil || !g.battlefieldShared {
		return
	}
	if g.ownedPermanents == nil {
		g.ownedPermanents = make(map[uuid.UUID]struct{})
	}
	g.ownedPermanents[p.ID()] = struct{}{}
}

// FindPermanentByName finds a permanent by name on the battlefield (first match).
// Phased-out permanents are invisible.
func (g *Game) FindPermanentByName(name string, controller uuid.UUID) *Permanent {
	for _, p := range g.battlefield {
		if p.PhasedOut {
			continue
		}
		if p.Name() == name && p.ControllerID() == controller {
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
	if rc := g.resolution.ResolvingCard(); rc != nil && rc.ID() == id {
		return rc
	}
	for _, pl := range g.players {
		for _, c := range pl.Library() {
			if c.ID() == id {
				return c
			}
		}
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
		for _, c := range pl.Ante() {
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
	if rc := g.resolution.ResolvingCard(); rc != nil && rc.ID() == sourceID {
		return rc
	}
	return g.FindCardAnywhere(sourceID)
}

// FlipCoin simulates a coin flip. Returns true for "win" (heads).
// If CoinFlipResults is non-empty, pops from the front (for test determinism).
func (g *Game) FlipCoin(playerID uuid.UUID) bool {
	if len(g.coinFlipResults) > 0 {
		result := g.coinFlipResults[0]
		g.coinFlipResults = g.coinFlipResults[1:]
		return result
	}
	return g.RandIntn(2) == 0
}

// TryPayMana attempts to pay a mana cost using floating mana and any untapped
// mana sources the player may activate. It is suitable for resolving effects
// that ask a player to pay mana outside the spell-casting pipeline.
func (g *Game) TryPayMana(playerID uuid.UUID, manaCostStr string) bool {
	cost := ParseManaCost(manaCostStr)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{
		Controller: playerID,
		Costs:      []Cost{&ManaCostPayment{MC: cost}},
	})
	if err != nil {
		return false
	}
	return tx.Commit() == nil
}

// PutOnBattlefield puts a card onto the battlefield under the given controller.
func (g *Game) PutOnBattlefield(card Card, controller uuid.UUID) *Permanent {
	return g.putOnBattlefield(card, controller, nil, uuid.Nil)
}

func (g *Game) putOnBattlefield(card Card, controller uuid.UUID, colors *[]Color, colorSourceID uuid.UUID) *Permanent {
	if card.HasType(TypeLand) && g.effects.Rules.LandsCantEnter() {
		return nil
	}
	perm := NewPermanent(card, controller)
	if colors != nil {
		override := append([]Color(nil), (*colors)...)
		perm.ColorOverride = &override
	}
	g.addOwnedPermanent(perm)
	perm.turnControlGained = g.turn

	g.syncAbilityContext(perm)

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
		if xc, ok := a.(*EntersWithXCountersAbility); ok && g.resolution.X() > 0 {
			g.AddCountersWithReplacement(perm, xc.CounterType, g.resolution.X(), perm.ID(), true)
			break
		}
	}

	// Add fixed N counters if configured (replacement effect, not a trigger).
	baseCtrTypes := map[CounterType]bool{}
	for _, a := range perm.RuntimeAbilities {
		if nc, ok := a.(*EntersWithNCountersAbility); ok {
			g.AddCountersWithReplacement(perm, nc.CounterType, nc.Count, perm.ID(), true)
			baseCtrTypes[nc.CounterType] = true
		}
	}
	for _, a := range perm.RuntimeAbilities {
		if xc, ok := a.(*EntersWithXCountersAbility); ok {
			baseCtrTypes[xc.CounterType] = true
		}
	}

	// Add computed counters if configured (CR 614.1c self-replacement whose
	// count depends on board state, e.g. Towering Titan: X = total toughness
	// of other creatures you control).
	for _, a := range perm.RuntimeAbilities {
		if cc, ok := a.(*EntersWithComputedCountersAbility); ok && cc.Compute != nil {
			n := cc.Compute(g, perm)
			if n > 0 {
				g.AddCountersWithReplacement(perm, cc.CounterType, n, perm.ID(), true)
			}
			baseCtrTypes[cc.CounterType] = true
		}
	}

	// CR 614.1c: "enters with N counters" effects from other sources (Oona's
	// Blackguard, Winding Constrictor) are self-replacements applied during the
	// ETB process even when the entering permanent has no native "enters with"
	// clause for that counter type. Synthesize a 0-amount AddCountersAction so
	// the registered etbAdditionalCountersReplacement effects can intercept it
	// (CR 614.5: multiple such effects combine into a single application).
	for _, ct := range g.etbAdditionalCounterTypesFor(perm) {
		if baseCtrTypes[ct] {
			continue
		}
		action := NewAddCountersAction(perm.ID(), perm.ID(), ct, 0, true)
		result := g.effects.ApplyReplacements(action, g)
		if aca, ok := result.(*AddCountersAction); ok && aca.Amount() > 0 {
			perm.AddCounter(aca.CounterType(), aca.Amount())
		}
	}

	// Copy creature on ETB (Vesuvan Doppelganger): copy target creature's P/T and keywords
	for _, a := range perm.RuntimeAbilities {
		if _, ok := a.(*CopyCreatureOnETBAbility); ok && len(g.resolution.ResolvingTargets()) > 0 {
			target := g.FindPermanent(g.resolution.ResolvingTargets()[0])
			if target != nil {
				g.effects.AddCopyEffect(perm.ID(), target)
			}
			break
		}
	}

	for _, a := range perm.RuntimeAbilities {
		if asEnter, ok := UnwrapAbility(a).(AsEntersBattlefieldAbility); ok {
			if !asEnter.OnEnter(g, perm) {
				owner := perm.Card.Owner()
				if owner == uuid.Nil {
					owner = perm.ControllerID()
				}
				p := g.GetPlayer(owner)
				if p != nil {
					p.AddToGraveyard(perm.Card)
				}
				return nil
			}
		}
	}

	g.ensureBattlefieldSliceOwned()
	g.battlefield = append(g.battlefield, perm)
	if colors != nil {
		g.effects.Add(permanentColorEffect(perm, *colors, colorSourceID))
	}

	// Register continuous effects from static abilities
	for _, a := range perm.RuntimeAbilities {
		if sa, ok := a.(*StaticAbilityHolder); ok {
			for _, e := range sa.Effects {
				// Set the source ID on the continuous effect
				g.setEffectSource(e, perm.ID())
				g.effects.Add(&printedAbilityContinuousEffect{ContinuousEffect: e})
			}
		}
	}

	// Run unconditional ETB effects (e.g. Primal Clay mode choice)
	for _, a := range perm.RuntimeAbilities {
		if etb, ok := a.(*ETBEffectAbility); ok {
			_ = ApplyEffect(g, etb.Effect, perm.ID(), controller, nil)
		}
	}

	// Run ETB-with-targets effects (e.g. Oubliette exile on entry)
	for _, a := range perm.RuntimeAbilities {
		if etb, ok := a.(*ETBWithTargetsAbility); ok && len(g.resolution.ResolvingTargets()) > 0 {
			_ = ApplyEffect(g, etb.Effect, perm.ID(), controller, g.resolution.ResolvingTargets())
			break
		}
	}

	g.effects.Apply(g)

	g.FireEvent(GameEvent{
		Type:     EvtZoneChange,
		SourceID: perm.ID(),
		PlayerID: controller,
		Amount:   g.resolution.X(), // preserve X from resolving spell for ETB triggers
		FromZone: ZoneAny,          // engine doesn't model the precise origin of an ETB
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
	perm = g.MutablePermanent(perm.ID())
	if perm == nil {
		return
	}
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
	g.syncAbilityContext(perm)
	// Register continuous effects from static abilities
	for _, a := range perm.RuntimeAbilities {
		if sa, ok := a.(*StaticAbilityHolder); ok {
			for _, e := range sa.Effects {
				g.setEffectSource(e, perm.ID())
				g.effects.Add(&printedAbilityContinuousEffect{ContinuousEffect: e})
			}
		}
	}
	g.effects.Apply(g)
}

// RemoveFromBattlefield removes a permanent and handles cleanup.
func (g *Game) RemoveFromBattlefield(perm *Permanent) {
	if perm == nil {
		return
	}
	delete(g.skipNextUntap, perm.ID())
	perm = g.mutablePermanentIncludingPhased(perm.ID())
	if perm == nil {
		return
	}
	// Snapshot LKI before any state mutation so death/leave-triggers can
	// read the dying permanent's controller, types, and P/T after the move.
	g.captureLKI(perm)

	// Remove continuous effects sourced from this permanent
	g.effects.Remove(perm.ID())

	// If this was attached to something, remove it from that thing's attachments
	if perm.IsAttached() {
		host := g.MutablePermanent(perm.AttachedTo)
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
	g.ensureBattlefieldSliceOwned()
	for i, p := range g.battlefield {
		if p.ID() == perm.ID() {
			g.battlefield = append(g.battlefield[:i], g.battlefield[i+1:]...)
			if g.ownedPermanents != nil {
				delete(g.ownedPermanents, perm.ID())
			}
			break
		}
	}

	g.effects.Apply(g)
	// Leave-zone trigger dispatch (CR 603.6c) happens in the destination-
	// specific path (Destroy / Sacrifice / Exile / Bounce /
	// PutPermanentIntoGraveyard) via EvtZoneChange + LKIAbilities. The
	// permanent's runtime abilities are captured into LKI by captureLKI
	// above.

	// Detach equipment immediately. Auras are left for SBAs to put into the
	// graveyard so that "when enchanted creature dies" triggers can still see
	// the attachment relationship when the host's death events fire.
	for _, attID := range attachments {
		att := g.MutablePermanent(attID)
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
	controller := perm.ControllerID()
	owner := perm.Card.Owner()
	if owner == uuid.Nil {
		panic("DestroyPermanent: Pemanent has no owner")
	}

	isCreature := perm.HasType(TypeCreature)
	permID := perm.ID()
	card := perm.Card

	isToken := perm.IsToken

	g.RemoveFromBattlefield(perm)
	selfAbilities := g.LKIAbilities(permID)

	if !isToken {
		p := g.GetPlayer(owner)
		if p != nil {
			p.AddToGraveyard(card)
		}
	}

	graveyardEvt := GameEvent{
		Type:     EvtZoneChange,
		SourceID: permID,
		PlayerID: controller,
		FromZone: ZoneBattlefield,
		ToZone:   ZoneGraveyard,
	}
	g.FireEvent(graveyardEvt)
	g.checkAbilitiesForEvent(selfAbilities, &graveyardEvt, permID, controller)

	if isCreature {
		g.creatureDeathsThisTurn++
		g.recordCreatureDeath(controller)
	}
}

// TapPermanent taps a permanent and fires the EvtTapped event.
func (g *Game) TapPermanent(perm *Permanent) {
	if perm == nil {
		return
	}
	perm = g.MutablePermanent(perm.ID())
	if perm == nil {
		return
	}
	perm.Tapped = true
	g.FireEvent(GameEvent{
		Type:     EvtTapped,
		SourceID: perm.ID(),
		PlayerID: perm.ControllerID(),
	})
}

// checkAbilitiesForEvent checks a set of abilities (from a removed permanent) for triggers.
func (g *Game) checkAbilitiesForEvent(abilities []Ability, evt *GameEvent, sourceID, controller uuid.UUID) {
	for _, a := range abilities {
		ta, ok := UnwrapAbility(a).(TriggeredAbility)
		if !ok {
			continue
		}
		ta.SetSource(sourceID)
		ta.SetController(controller)
		if ta.IsStateTrigger() {
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

func (g *Game) syncAbilityContext(perm *Permanent) {
	if perm == nil {
		return
	}
	for _, ability := range perm.RuntimeAbilities {
		unwrapped := UnwrapAbility(ability)
		unwrapped.SetSource(perm.ID())
		unwrapped.SetController(perm.ControllerID())
	}
}

// PutPermanentIntoGraveyard puts a permanent into its owner's graveyard without
// destroying it. This bypasses indestructible and regeneration. Used by SBAs
// (e.g., 0-toughness creatures) and other rules that move permanents to the
// graveyard without destruction.
func (g *Game) PutPermanentIntoGraveyard(perm *Permanent) {
	controller := perm.ControllerID()
	owner := perm.Card.Owner()
	if owner == uuid.Nil {
		owner = controller
	}

	isCreature := perm.HasType(TypeCreature)
	isToken := perm.IsToken
	permID := perm.ID()
	card := perm.Card

	g.RemoveFromBattlefield(perm)
	selfAbilities := g.LKIAbilities(permID)

	if !isToken {
		p := g.GetPlayer(owner)
		if p != nil {
			p.AddToGraveyard(card)
		}
	}

	graveyardEvt := GameEvent{
		Type:     EvtZoneChange,
		SourceID: permID,
		PlayerID: controller,
		FromZone: ZoneBattlefield,
		ToZone:   ZoneGraveyard,
	}
	g.FireEvent(graveyardEvt)
	g.checkAbilitiesForEvent(selfAbilities, &graveyardEvt, permID, controller)

	if isCreature {
		g.creatureDeathsThisTurn++
		g.recordCreatureDeath(controller)
	}
}

// MoveFromGraveyard removes a card from the graveyard of a player.
// The function sends an EvtZoneChange event for the card.
// The function also sends an EvtCardsLeftGraveyard event with Amount set to 1.
// The function does not put the card into the destination zone.
// The caller must put the card into the destination zone.
//
// The function returns the removed card, or nil and false if the card was not present.
//
// When multiple cards leave a graveyard together (CR 603.10), use MoveCardsFromGraveyard.
func (g *Game) MoveFromGraveyard(playerID, cardID uuid.UUID, to Zone) (Card, bool) {
	p := g.GetPlayer(playerID)
	if p == nil {
		return nil, false
	}
	card, ok := p.RemoveFromGraveyard(cardID)
	if !ok {
		return nil, false
	}
	g.FireEvent(GameEvent{
		Type:     EvtZoneChange,
		SourceID: cardID,
		PlayerID: playerID,
		FromZone: ZoneGraveyard,
		ToZone:   to,
	})
	g.FireEvent(GameEvent{
		Type:     EvtCardsLeftGraveyard,
		PlayerID: playerID,
		Amount:   1,
	})
	return card, true
}

// MoveFromAnyGraveyard removes a card from whichever player's graveyard
// contains it and returns that player's ID for zone-change attribution.
func (g *Game) MoveFromAnyGraveyard(cardID uuid.UUID, to Zone) (Card, uuid.UUID, bool) {
	for _, player := range g.players {
		if card, ok := g.MoveFromGraveyard(player.PlayerID(), cardID, to); ok {
			return card, player.PlayerID(), true
		}
	}
	return nil, uuid.Nil, false
}

// MoveCardsFromGraveyard removes the listed cards from playerID's graveyard,
// firing per-card EvtZoneChange events and a SINGLE EvtCardsLeftGraveyard
// event with Amount = number of cards actually removed (CR 603.10 — multiple
// cards moving via the same effect form one zone-change event for the
// purposes of "one or more cards leave your graveyard" triggers). Returns
// the removed cards in input order, skipping any that weren't found.
func (g *Game) MoveCardsFromGraveyard(playerID uuid.UUID, cardIDs []uuid.UUID, to Zone) []Card {
	p := g.GetPlayer(playerID)
	if p == nil {
		return nil
	}
	removed := make([]Card, 0, len(cardIDs))
	for _, id := range cardIDs {
		c, ok := p.RemoveFromGraveyard(id)
		if !ok {
			continue
		}
		removed = append(removed, c)
		g.FireEvent(GameEvent{
			Type:     EvtZoneChange,
			SourceID: id,
			PlayerID: playerID,
			FromZone: ZoneGraveyard,
			ToZone:   to,
		})
	}
	if len(removed) > 0 {
		g.FireEvent(GameEvent{
			Type:     EvtCardsLeftGraveyard,
			PlayerID: playerID,
			Amount:   len(removed),
		})
	}
	return removed
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

// PlayerDiscardByEffect discards a card through the replacement pipeline.
// Costs and turn-based discards must continue to use PlayerDiscard.
func (g *Game) PlayerDiscardByEffect(p Player, cardID, sourceID uuid.UUID) (Card, bool) {
	if p == nil {
		return nil, false
	}
	action := NewDiscardAction(sourceID, p.PlayerID(), cardID)
	result := g.effects.ApplyReplacements(action, g)
	if result == nil {
		return nil, false
	}
	discard, ok := result.(*DiscardAction)
	if !ok {
		return nil, false
	}
	return g.executeDiscard(discard)
}

func (g *Game) executeDiscard(action *DiscardAction) (Card, bool) {
	p := g.GetPlayer(action.PlayerID())
	if p == nil {
		return nil, false
	}
	c, ok := p.RemoveFromHand(action.CardID())
	if !ok {
		return nil, false
	}
	if action.Destination() == ZoneLibrary {
		p.SetLibrary(append([]Card{c}, p.Library()...))
	} else {
		p.AddToGraveyard(c)
	}
	g.FireEvent(GameEvent{Type: EvtDiscard, SourceID: c.ID(), PlayerID: p.PlayerID()})
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

// BouncePermanentToHand removes a permanent from the battlefield and adds the
// underlying card to its owner's hand, firing an EvtZoneChange{From: BF,
// To: Hand}. Self-referencing leave triggers fire via the LKI snapshot's
// captured abilities per CR 603.6c.
func (g *Game) BouncePermanentToHand(perm *Permanent) {
	if perm == nil {
		return
	}
	controller := perm.ControllerID()
	permID := perm.ID()
	card := perm.Card
	isToken := perm.IsToken
	owner := card.Owner()
	if owner == uuid.Nil {
		owner = controller
	}
	g.RemoveFromBattlefield(perm)
	selfAbilities := g.LKIAbilities(permID)
	if !isToken {
		// CR 111.7 / 111.10g: tokens cease to exist when they leave the
		// battlefield. The zone-change event still fires, but the card is
		// not added to a hand.
		if p := g.GetPlayer(owner); p != nil {
			p.AddToHand(card)
		}
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
	controller := perm.ControllerID()
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
	g.consumePendingExileZoneChange(permID)
}

// ExileCard moves a card (from any zone) to the exile zone.
func (g *Game) ExileCard(card Card, exiledBy uuid.UUID) {
	g.exile = append(g.exile, ExiledCard{Card: card, ExiledBy: exiledBy})
	g.recordCardPutIntoExile(card)
}

// ExileCardFaceDown moves a card to the exile zone face down. Only players in
// revealedTo may inspect the card's identity (via FindExiledCard / GetExile +
// ExiledCard.VisibleTo). For Gonti-style "exile face down" effects, only the
// exiling player (not the card's owner) sees the identity — pass that player's
// ID in revealedTo. Owners of face-down exiled cards do not automatically see
// the identity, matching Gonti's printed ruling.
func (g *Game) ExileCardFaceDown(card Card, exiledBy uuid.UUID, revealedTo ...uuid.UUID) {
	rev := append([]uuid.UUID(nil), revealedTo...)
	g.exile = append(g.exile, ExiledCard{
		Card:       card,
		ExiledBy:   exiledBy,
		FaceDown:   true,
		RevealedTo: rev,
	})
	g.recordCardPutIntoExile(card)
}

// RevealExiledCardTo grants the given player permission to see the identity of
// the face-down exiled card with the given ID. No-op if the card is face up
// (already public) or not in exile.
func (g *Game) RevealExiledCardTo(cardID, playerID uuid.UUID) {
	for i := range g.exile {
		if g.exile[i].Card.ID() != cardID {
			continue
		}
		ec := &g.exile[i]
		if !ec.FaceDown {
			return
		}
		if slices.Contains(ec.RevealedTo, playerID) {
			return
		}
		ec.RevealedTo = append(ec.RevealedTo, playerID)
		return
	}
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

// PlayerDrawCard draws a card for the player, running the replacement
// pipeline first so that "if you would draw a card" replacement effects
// (Aladdin's Lamp, Ormos's empty-library counters, etc.) intercept the
// draw. When the action is fully replaced, no card is drawn and (false,
// nil) is returned; otherwise the next library card moves to hand and
// EvtCardDrawn fires. The draw is marked isNormalDraw=false because this
// entry point is used by effect-driven draws; the turn-based draw step
// builds its own action with isNormalDraw=true.
func (g *Game) PlayerDrawCard(p Player) (Card, bool) {
	if p == nil {
		return nil, false
	}
	action := NewDrawCardAction(uuid.Nil, p.PlayerID(), false)
	result := g.effects.ApplyReplacements(action, g)
	if result == nil {
		return nil, false
	}
	return g.drawCardRaw(p)
}

// drawCardRaw performs the underlying library-to-hand transfer and fires
// EvtCardDrawn without running the replacement pipeline. Used by the
// normal draw step (which applies replacements itself) and by
// replacement implementations that need to draw after rearranging the
// library (Aladdin's Lamp).
func (g *Game) drawCardRaw(p Player) (Card, bool) {
	c, ok := p.DrawCard()
	if ok {
		g.FireEvent(GameEvent{
			Type:     EvtCardDrawn,
			PlayerID: p.PlayerID(),
		})
	}
	return c, ok
}

// PerformScry executes the scry action for N cards (CR 701.18).
// The player examines the top N cards of the library.
// The player can put any number of these cards on the bottom of the library.
// The player puts the remaining cards on top of the library in any order.
// If the library contains fewer than N cards, the player examines all available cards.
// The function returns the quantity of cards examined.
//
// The method delegates card placement to Player.ChooseScryPlacement.
// The engine validates the returned IDs.
// If an ID mismatch occurs, the engine puts all cards on top in the original order.
func (g *Game) PerformScry(p Player, n int) int {
	if p == nil || n <= 0 {
		return 0
	}
	lib := p.Library()
	if len(lib) == 0 {
		return 0
	}
	count := min(n, len(lib))
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

// PerformSurveil implements surveil N (CR 701.42): the player looks at the top
// N cards of their library, then puts any number of them into their graveyard
// and the rest on top of their library in any order. If the library has fewer
// than N cards, the player surveils however many are present. Returns the
// number of cards actually surveiled.
//
// The placement decision is delegated to Player.ChooseSurveilPlacement; the
// engine validates the returned IDs and falls back to "all on top, original
// order" on any mismatch so a buggy player implementation cannot lose cards.
func (g *Game) PerformSurveil(p Player, n int) int {
	if p == nil || n <= 0 {
		return 0
	}
	lib := p.Library()
	if len(lib) == 0 {
		return 0
	}
	count := min(n, len(lib))
	top := make([]Card, count)
	copy(top, lib[:count])

	graveyard, topOrder := p.ChooseSurveilPlacement(top, "surveil", g)
	graveyard, topOrder = validateScryPlacement(top, graveyard, topOrder)

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
	p.SetLibrary(newLib)
	for _, id := range graveyard {
		if c, ok := idToCard[id]; ok {
			p.AddToGraveyard(c)
		}
	}
	return count
}

// validateScryPlacement ensures the player's choice is a valid partition of
// `top`. On any inconsistency it returns the safe default (all on top in the
// original order) so cards are never dropped.
func validateScryPlacement(top []Card, bottom, topOrder []uuid.UUID) (validBottom, validTopOrder []uuid.UUID) {
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
		maxDamage := max(p.Life()-1, 0)
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
	if sourceID != uuid.Nil {
		if g.damageDealtToPlayersByPermanent == nil {
			g.damageDealtToPlayersByPermanent = make(map[uuid.UUID]map[uuid.UUID]bool)
		}
		if g.damageDealtToPlayersByPermanent[sourceID] == nil {
			g.damageDealtToPlayersByPermanent[sourceID] = make(map[uuid.UUID]bool)
		}
		g.damageDealtToPlayersByPermanent[sourceID][p.PlayerID()] = true
	}
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
			byCtrl, ok := g.combatDamageThisStep[srcPerm.ControllerID()]
			if !ok {
				byCtrl = make(map[uuid.UUID]int)
				g.combatDamageThisStep[srcPerm.ControllerID()] = byCtrl
			}
			byCtrl[p.PlayerID()] += amount
			byCtrlSrcs, ok := g.combatDamageSourcesThisStep[srcPerm.ControllerID()]
			if !ok {
				byCtrlSrcs = make(map[uuid.UUID]map[uuid.UUID]int)
				g.combatDamageSourcesThisStep[srcPerm.ControllerID()] = byCtrlSrcs
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
		srcPlayer := g.GetPlayer(src.ControllerID())
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
	if sourceCard != nil && perm.HasProtectionFromInGame(sourceCard, g) {
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
	perm := g.MutablePermanent(a.PermanentID())
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
	if sourceID != uuid.Nil {
		if g.damageDealtToPermanentsByPermanent == nil {
			g.damageDealtToPermanentsByPermanent = make(map[uuid.UUID]map[uuid.UUID]bool)
		}
		if g.damageDealtToPermanentsByPermanent[sourceID] == nil {
			g.damageDealtToPermanentsByPermanent[sourceID] = make(map[uuid.UUID]bool)
		}
		g.damageDealtToPermanentsByPermanent[sourceID][perm.ID()] = true
	}
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
		srcPlayer := g.GetPlayer(src.ControllerID())
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

// HasDealtDamageToPlayer reports whether the given source permanent has dealt damage to playerID this game.
func (g *Game) HasDealtDamageToPlayer(sourceID, playerID uuid.UUID) bool {
	if g.damageDealtToPlayersByPermanent == nil {
		return false
	}
	if m, ok := g.damageDealtToPlayersByPermanent[sourceID]; ok {
		return m[playerID]
	}
	return false
}

// HasDealtDamageToPermanent reports whether the given source permanent has dealt damage to permID this game.
func (g *Game) HasDealtDamageToPermanent(sourceID, permID uuid.UUID) bool {
	if g.damageDealtToPermanentsByPermanent == nil {
		return false
	}
	if m, ok := g.damageDealtToPermanentsByPermanent[sourceID]; ok {
		return m[permID]
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
	src := g.MutablePermanent(sourceID)
	target := g.MutablePermanent(targetID)
	if src == nil || target == nil {
		return
	}

	// "Can't be enchanted" — prevent enchantment attachment entirely.
	if src.HasType(TypeEnchantment) && target.HasAttr(AttrCantBeEnchanted) {
		return
	}

	// Detach from current host if any
	if src.IsAttached() {
		oldHost := g.MutablePermanent(src.AttachedTo)
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

// fireBecomesTargetEvents sends an EvtBecomesTarget event for each declared target.
// The function runs after a player puts an object on the stack and chooses targets (CR 603.6c, 119.5).
// The engine sends one event for each distinct target permanent or player.
// Field evt.SourceID contains the ID of the spell or ability source.
// Field evt.TargetID contains the ID of the targeted object.
// Field evt.PlayerID contains the ID of the spell or ability controller.
// Field evt.Flag is true for activated abilities and false for spells.
//
// CR 603.6c states that abilities trigger only once for each event when a spell has multiple targets.
// However, each targeted permanent or player receives the event independently.
// The engine sends one event for each distinct target.
// This ensures that triggers on different permanents receive separate events.
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
	g.recordPerDuelEvent(&evt)
	for _, perm := range g.battlefield {
		g.syncAbilityContext(perm)
		for _, a := range perm.RuntimeAbilities {
			ta, ok := UnwrapAbility(a).(TriggeredAbility)
			if !ok {
				continue
			}
			if ta.IsStateTrigger() {
				continue
			}
			if ta.TriggerSourceZone() != ZoneBattlefield {
				continue
			}
			if !ta.CheckEventType(evt.Type) {
				continue
			}
			// Default-zone triggers are battlefield-only; explicit
			// non-battlefield zones are scanned in the cross-zone loop below.
			if gt, ok := ta.(*GenericTriggered); ok && !gt.FunctionsInZone(ZoneBattlefield) {
				continue
			}
			if ta.CheckTrigger(&evt, g) {
				g.pendingTriggers = append(g.pendingTriggers, &pendingTrigger{
					ability:    ta,
					event:      &evt,
					sourceID:   perm.ID(),
					controller: perm.ControllerID(),
				})
			}
		}
	}

	// Scan card-level abilities that function while the source is in a
	// non-battlefield zone (CR 113.6) — currently graveyard, for cards like
	// Pia Nalaar, Consul of Revival and Nether Shadow. Card.Abilities() returns
	// the immutable card-level ability list (not Permanent.RuntimeAbilities);
	// we set source/controller transiently so condition predicates see the
	// right IDs while the trigger is queued.
	for _, pl := range g.players {
		ownerID := pl.PlayerID()
		for _, c := range pl.Graveyard() {
			for _, a := range c.Abilities() {
				gt, ok := UnwrapAbility(a).(*GenericTriggered)
				if !ok {
					continue
				}
				if !gt.FunctionsInZone(ZoneGraveyard) {
					continue
				}
				if !gt.CheckEventType(evt.Type) {
					continue
				}
				gt.SetSource(c.ID())
				gt.SetController(ownerID)
				if !gt.CheckTrigger(&evt, g) {
					continue
				}
				g.pendingTriggers = append(g.pendingTriggers, &pendingTrigger{
					ability:    gt,
					event:      &evt,
					sourceID:   c.ID(),
					controller: ownerID,
				})
			}
		}
	}

	// Scan spell abilities that function while their source spell is on the
	// stack (CR 113.6i), such as "When you cast this spell, copy it...".
	for _, obj := range g.stack.Objects() {
		if obj == nil || obj.IsAbility || obj.Card == nil {
			continue
		}
		for _, a := range obj.Card.Abilities() {
			gt, ok := UnwrapAbility(a).(*GenericTriggered)
			if !ok {
				continue
			}
			if !gt.FunctionsInZone(ZoneStack) {
				continue
			}
			if !gt.CheckEventType(evt.Type) {
				continue
			}
			gt.SetSource(obj.SourceID)
			gt.SetController(obj.Controller)
			if !gt.CheckTrigger(&evt, g) {
				continue
			}
			g.pendingTriggers = append(g.pendingTriggers, &pendingTrigger{
				ability:    gt,
				event:      &evt,
				sourceID:   obj.SourceID,
				controller: obj.Controller,
			})
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
			if dt.MatchFlag && !evt.Flag {
				remaining = append(remaining, dt)
				continue
			}
			if evt.Type == EvtZoneChange {
				// Default unset zone matchers to ZoneAny so callers that
				// don't care about the from/to don't have to set them.
				wantFrom := dt.MatchFromZone
				if wantFrom == 0 {
					wantFrom = ZoneAny
				}
				wantTo := dt.MatchToZone
				if wantTo == 0 {
					wantTo = ZoneAny
				}
				if wantFrom != ZoneAny && evt.FromZone != wantFrom {
					remaining = append(remaining, dt)
					continue
				}
				if wantTo != ZoneAny && evt.ToZone != wantTo {
					remaining = append(remaining, dt)
					continue
				}
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
			g.pushStack(obj)
			if dt.Persistent {
				remaining = append(remaining, dt)
			}
		} else {
			remaining = append(remaining, dt)
		}
	}
	g.delayedTriggers = remaining
	g.queueParadigmRecurringTriggers(&evt)
}

// CheckStateTriggers evaluates state-triggered abilities (CR 603.8) on all battlefield permanents.
// A state trigger starts once when its condition changes from false to true.
// While the condition remains true, the ability does not trigger again.
// The ability triggers again only after the condition becomes false and then true again.
// The engine appends new triggers to pendingTriggers.
//
// Call this function after state-based actions are checked and before a player receives priority.
func (g *Game) CheckStateTriggers() {
	seen := make(map[stateTriggerKey]bool)
	for _, perm := range g.battlefield {
		g.syncAbilityContext(perm)
		for _, a := range perm.RuntimeAbilities {
			ta, ok := UnwrapAbility(a).(TriggeredAbility)
			if !ok || !ta.IsStateTrigger() {
				continue
			}
			key := stateTriggerKey{sourceID: perm.ID(), abilityID: ta.AbilityID()}
			seen[key] = true
			cond := ta.CheckTrigger(nil, g)
			if !cond {
				delete(g.armedStateTriggers, key)
				continue
			}
			if g.armedStateTriggers[key] {
				continue
			}
			g.armedStateTriggers[key] = true
			g.pendingTriggers = append(g.pendingTriggers, &pendingTrigger{
				ability:    ta,
				sourceID:   perm.ID(),
				controller: perm.ControllerID(),
			})
		}
	}
	// Drop entries for sources no longer on the battlefield so a re-entered
	// instance starts fresh.
	for key := range g.armedStateTriggers {
		if !seen[key] {
			delete(g.armedStateTriggers, key)
		}
	}
}

// PutTriggersOnStack puts all pending triggers onto the stack.
//
// CR 603.3b specifies the trigger order when players receive priority.
// The active player puts triggered abilities on the stack first in any chosen order.
// Each non-active player in turn order then puts triggered abilities on the stack.
// The last ability put on the stack resolves first.
//
// The engine divides pending triggers into an active group and a non-active group.
// The engine pushes the active group first.
// The engine then pushes the non-active group.
// Thus, the triggers of the non-active player resolve first.
//
// Within each group, the engine reverses the event source order.
// Therefore, triggers from older permanents resolve first.
// CR 603.3b permits any order.
// The engine uses the XMage order to maintain deterministic test results.
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
		reverseTriggers(active)
		reverseTriggers(nonActive)
		active = append(active, nonActive...)
		g.pendingTriggers = active
	}
	for _, pt := range g.pendingTriggers {
		targetSource := g.FindCardAnywhere(pt.sourceID)
		obj := &StackObject{
			ID:         uuid.New(),
			Controller: pt.controller,
			SourceID:   pt.sourceID,
			IsAbility:  true,
		}
		// CR 603.1f / 603.3d: a modal triggered ability picks its mode as it
		// goes on the stack, then gathers targets only for that mode. Push the
		// chosen mode's effects/targets onto the stack object and skip the
		// legacy declared-targets and event-derived auto-binding paths below.
		if gt, ok := pt.ability.(*GenericTriggered); ok && gt.IsModal() {
			ctrl := g.GetPlayer(pt.controller)
			modes := gt.Modes()
			labels := make([]string, len(modes))
			for i, m := range modes {
				labels[i] = m.Label
			}
			idx := 0
			if ctrl != nil {
				reason := "modal trigger"
				if c := g.FindCardAnywhere(pt.sourceID); c != nil {
					reason = c.Name()
				}
				idx = ctrl.ChooseMode(labels, reason)
				if idx < 0 || idx >= len(modes) {
					idx = 0
				}
			}
			chosen := modes[idx]
			obj.Effects = append(obj.Effects, chosen.Effects...)
			obj.ModeChoice = idx
			if len(chosen.Targets) > 0 {
				obj.Targets = g.chooseTriggerTargets(pt, chosen.Targets)
				obj.TargetSpecs = expandTargetSpecs(chosen.Targets, obj.Targets, obj.XValue)
				obj.TargetSource = targetSource
			}
			g.pushStack(obj)
			continue
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
			obj.TargetSpecs = expandTargetSpecs(declared, obj.Targets, obj.XValue)
			obj.TargetSource = targetSource
			g.pushStack(obj)
			continue
		}
		// Bind event context needed by resolving triggered effects. Some events
		// expose their player through targets[0]; zone changes instead expose the
		// moved object so effects can inspect it or its controller.
		if pt.event != nil {
			if gt, ok := pt.ability.(*GenericTriggered); ok {
				switch gt.eventType {
				case EvtZoneChange:
					// Route by (FromZone, ToZone) for stack-object context.
					switch {
					case pt.event.ToZone == ZoneBattlefield:
						// ETB: pass the entering permanent's ID so effects can
						// tap/modify it; preserve X for X-cost ETB triggers.
						if pt.event.SourceID != uuid.Nil {
							obj.Targets = []uuid.UUID{pt.event.SourceID}
						}
						obj.XValue = pt.event.Amount
					case pt.event.FromZone == ZoneBattlefield:
						// Battlefield -> elsewhere (graveyard / exile / hand /
						// library). Pass the leaving permanent's ID first
						// (the dies/leaves convention - Creature Bond reading
						// the dead creature's toughness, Sengir Vampire
						// finding it in the graveyard) and the controller's
						// ID second (Dingus Egg "deal damage to its
						// controller", post-LKI lookups).
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
				case EvtTappedForMana:
					if pt.event.PlayerID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.PlayerID}
					}
					obj.EventAmount = pt.event.Amount
					obj.EventSourceID = pt.event.SourceID
				case EvtTapped, EvtAbilityActivated:
					// Pass the permanent's ID so effects can identify it
					if pt.event.SourceID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.SourceID}
					}
				case EvtDeclaredAttacker:
					// Pass the declared attacker's ID so effects can identify which
					// creature attacked (Hellrider's "deal 1 damage to the player
					// or planeswalker it's attacking").
					obj.EventSourceID = pt.event.SourceID
					if pt.event.SourceID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.SourceID}
					}
				case EvtCombatDamageDealt:
					// Pass the damaging controller and recipient via the dedicated
					// event-context fields (EventSourceID = recipient, EventAmount
					// = total damage). We deliberately do NOT auto-bind Targets[0]
					// here — DrawCards-style effects fall back to controller when
					// Targets is empty, which is the correct behavior for "draw a
					// card" triggers (Keeper of Fables). Per-step triggers that
					// need the recipient (Oona's Blackguard) read it via
					// g.EventSourceID().
					obj.EventSourceID = pt.event.TargetID
					obj.EventAmount = pt.event.Amount
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
				case EvtDeclaredBlocker:
					// Pass the blocker's ID and attacker's ID
					if pt.event.SourceID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.SourceID}
						if pt.event.TargetID != uuid.Nil {
							obj.Targets = append(obj.Targets, pt.event.TargetID)
						}
					}
				case EvtLifeGained, EvtLifeLost:
					// Preserve the life delta for "gain/lose that much" triggers.
					// We deliberately do NOT auto-bind PlayerID as a target; the
					// affected player is rarely the same as the trigger's
					// "you" (e.g. Exquisite Blood's "you gain that much life"
					// targets the controller, not the opponent who lost life).
					obj.EventAmount = pt.event.Amount
				case EvtAttackersDeclared:
					// Preserve the attacker count for "gain that much life" /
					// "draw that many cards" attack-aggregate triggers (Path of
					// Bravery's gain-life clause).
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
		g.pushStack(obj)
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
		if pair, ok := t.(*randomActivePlayerExchangePairTarget); ok {
			out = append(out, pair.chooseForTrigger(pt.controller, sourceCard, g)...)
			continue
		}
		if random, ok := t.(*randomTarget); ok {
			chosen := g.chooseRandomTargets(pt.controller, sourceCard, random, 0)
			if len(chosen) == 0 && t.Min() > 0 {
				out = append(out, uuid.Nil)
			} else {
				out = append(out, chosen...)
			}
			continue
		}
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

func reverseTriggers(s []*pendingTrigger) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
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

// zoneOfTarget reports the zone the given target id currently occupies, or
// ZoneAny if the id is a player or can't be located in any zone.
func (g *Game) zoneOfTarget(id uuid.UUID) Zone {
	if id == uuid.Nil || g.GetPlayer(id) != nil {
		return ZoneAny
	}
	if g.FindPermanent(id) != nil {
		return ZoneBattlefield
	}
	if g.stack.FindBySourceID(id) != nil {
		return ZoneStack
	}
	for _, ec := range g.exile {
		if ec.Card != nil && ec.Card.ID() == id {
			return ZoneExile
		}
	}
	for _, p := range g.players {
		for _, c := range p.Graveyard() {
			if c.ID() == id {
				return ZoneGraveyard
			}
		}
		for _, c := range p.Library() {
			if c.ID() == id {
				return ZoneLibrary
			}
		}
		for _, c := range p.Hand() {
			if c.ID() == id {
				return ZoneHand
			}
		}
		for _, c := range p.Ante() {
			if c.ID() == id {
				return ZoneAnte
			}
		}
	}
	return ZoneAny
}

// Used to track game-specific state before pushing object onto stack
func (g *Game) pushStack(obj *StackObject) {
	g.chooseRandomCounterDistribution(obj)
	// Track target zone when targeted, needed for CR 608.2b
	obj.TargetZones = make(map[uuid.UUID]Zone, len(obj.Targets))
	for _, t := range obj.Targets {
		obj.TargetZones[t] = g.zoneOfTarget(t)
	}

	g.stack.Push(obj)
}

// Check if target is still legal at resolution time. Used in stack responses and
// resolutions.
func (g *Game) isTargetStillLegal(targetID uuid.UUID, sourceCard Card, controller uuid.UUID, expectedZone Zone, spec Target) bool {
	if g.GetPlayer(targetID) != nil {
		return true
	}
	// CR 608.2b: "A target that's no longer in the zone it was in when it was
	// targeted is illegal."
	if g.zoneOfTarget(targetID) != expectedZone {
		return false
	}
	if perm := g.FindPermanent(targetID); perm != nil {
		if !perm.CanBeTargetedBy(sourceCard, controller, g) {
			return false
		}
	}
	if spec != nil && !slices.Contains(spec.Possible(controller, sourceCard, g), targetID) {
		return false
	}
	return true
}

// ResolveStackObject resolves a single stack object.
func (g *Game) ResolveStackObject(obj *StackObject) {
	// Check for fizzle: if the spell/ability has targets but all are now illegal,
	// it fails to resolve (MTG rule 608.2b)

	defer g.CheckStateBasedActions()
	defer g.ClearSacrificed()
	defer g.resolution.SetColorOverride(obj.SourceID, obj.ColorOverride)()

	// CR 608.2b: a spell/ability with target(s) fails to resolve only if ALL of
	// them are illegal at resolution. uuid.Nil entries are positional
	// placeholders for target slots that had no legal candidate — not real
	// targets — so they never count as "has a target", and every slot is kept
	// in place (illegal ones nulled) so effects that read targets by index
	// (e.g. Hungry Flames) stay aligned.
	resolvedTargets := make([]uuid.UUID, len(obj.Targets))
	hasRealTarget := false
	anyLegal := false
	for i, t := range obj.Targets {
		if t != uuid.Nil {
			hasRealTarget = true
			var spec Target
			if i < len(obj.TargetSpecs) {
				spec = obj.TargetSpecs[i]
			}
			sourceCard := obj.Card
			if obj.TargetSource != nil {
				sourceCard = obj.TargetSource
			}
			if g.isTargetStillLegal(t, sourceCard, obj.Controller, obj.TargetZones[t], spec) {
				resolvedTargets[i] = t
				anyLegal = true
				continue
			}
		}
		resolvedTargets[i] = uuid.Nil
	}

	if hasRealTarget && !anyLegal {
		g.cleanupSpell(obj)
		return
	}

	defer g.resolution.Begin(obj)()
	for _, eff := range obj.Effects {
		_ = ApplyEffect(g, eff, obj.SourceID, obj.Controller, resolvedTargets)
	}
	g.resolution.ClearDistributions()

	// Copies of spells cease to exist as they resolve (CR 707.10) — no
	// graveyard, no battlefield, no exile. The effects already ran above.
	if obj.IsCopy {
		return
	}

	// If this was a spell (not an ability), put the card in the graveyard
	if obj.Card != nil && !obj.IsAbility {
		// Permanents go to the battlefield instead
		if obj.Card.HasType(TypeCreature) || obj.Card.HasType(TypeArtifact) || obj.Card.HasType(TypeEnchantment) || obj.Card.HasType(TypePlaneswalker) {
			perm := g.putOnBattlefield(obj.Card, obj.Controller, obj.ColorOverride, obj.SourceID)

			// Handle aura attachment (only for Aura subtype, not all enchantments)
			if perm != nil && obj.Card.HasType(TypeEnchantment) && len(obj.Targets) > 0 {
				if slices.Contains(obj.Card.SubTypes(), "Aura") {
					g.Attach(perm.ID(), obj.Targets[0])
				}
			}
			return
		}

		g.cleanupSpell(obj)
	}
}

// Move spell to graveyard/exile as appropriate. Used for resolving cast
// instants/sorceries and fizzled spells
func (g *Game) cleanupSpell(obj *StackObject) {
	// do nothing for non-spells
	if obj.Card == nil || obj.IsAbility {
		return
	}

	// Copies of spells cease to exist (CR 707.10) — no graveyard, no exile.
	if obj.IsCopy {
		return
	}

	owner := obj.Card.Owner()
	if owner == uuid.Nil {
		owner = obj.Controller
	}
	if obj.ExileOnLeaveStack || g.IsCardMarkedExileInsteadOfGraveyard(obj.Card.ID()) {
		g.ExileCard(obj.Card, obj.Card.ID())
	} else {
		p := g.GetPlayer(owner)
		if p != nil {
			p.AddToGraveyard(obj.Card)
		}
	}
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

func (g *Game) validateActionTargets(controller uuid.UUID, sourceCard Card, specs []Target, chosen []uuid.UUID, x int, label string) error {
	if len(specs) == 0 {
		return nil
	}
	offset := 0
	for i, spec := range specs {
		minimum, maximum := TargetBounds(spec, x)
		remainingRequired := 0
		for _, later := range specs[i+1:] {
			laterMinimum, _ := TargetBounds(later, x)
			remainingRequired += laterMinimum
		}
		available := max(0, len(chosen)-offset-remainingRequired)
		count := min(maximum, available)
		if count < minimum {
			return fmt.Errorf("not enough targets for %s", label)
		}
		possible := spec.Possible(controller, sourceCard, g)
		seen := make(map[uuid.UUID]bool, count)
		for _, id := range chosen[offset : offset+count] {
			if seen[id] || !slices.Contains(possible, id) {
				return fmt.Errorf("invalid target for %s", label)
			}
			seen[id] = true
		}
		offset += count
	}
	if offset != len(chosen) {
		return fmt.Errorf("too many targets for %s", label)
	}
	return nil
}

func (g *Game) validateVariableSpellTargets(controller uuid.UUID, sourceCard Card, chosen []uuid.UUID, x int) error {
	specs := sourceCard.CastTargets()
	if len(specs) != 1 {
		return nil
	}
	spec := specs[0]
	minTargets, maxTargets := TargetBounds(spec, x)
	if _, variable := spec.(VariableTarget); !variable {
		return nil
	}
	if len(chosen) < minTargets || len(chosen) > maxTargets {
		return fmt.Errorf("%s requires exactly %d targets", sourceCard.Name(), minTargets)
	}
	possible := spec.Possible(controller, sourceCard, g)
	seen := make(map[uuid.UUID]bool, len(chosen))
	for _, id := range chosen {
		if seen[id] || !slices.Contains(possible, id) {
			return fmt.Errorf("invalid target for %s", sourceCard.Name())
		}
		seen[id] = true
	}
	return nil
}

func addActionManaCost(total *ManaCost, add ManaCost) {
	total.Generic += add.Generic
	total.White += add.White
	total.Blue += add.Blue
	total.Black += add.Black
	total.Red += add.Red
	total.Green += add.Green
	total.Hybrid = append(total.Hybrid, add.Hybrid...)
	if add.HasX {
		total.HasX = true
		total.XCount += add.XCount
	}
}

func (g *Game) prepareActionCosts(costs []Cost, targets []uuid.UUID, x int) []Cost {
	ctx := actionCostContext{Targets: append([]uuid.UUID(nil), targets...), XValue: x}
	resolved := make([]Cost, 0, len(costs))
	var mana ManaCost
	for _, cost := range costs {
		if contextual, ok := cost.(contextualActionCost); ok {
			cost = contextual.resolveActionCost(ctx)
		}
		if payment, ok := cost.(*ManaCostPayment); ok {
			addActionManaCost(&mana, payment.MC)
			continue
		}
		resolved = append(resolved, cost)
	}
	if !mana.IsZero() {
		resolved = append([]Cost{&ManaCostPayment{MC: mana}}, resolved...)
	}
	return resolved
}

func newStackObject(controller, sourceID uuid.UUID, card Card, effects []Effect, targets []uuid.UUID, xValue int, isAbility bool) *StackObject {
	obj := &StackObject{
		ID:         uuid.New(),
		Card:       card,
		Controller: controller,
		SourceID:   sourceID,
		IsAbility:  isAbility,
		Targets:    targets,
		XValue:     xValue,
	}
	obj.Effects = append(obj.Effects, effects...)
	return obj
}

func chooseModeForStackObject(obj *StackObject, modes []string, chooser Player, sourceName string) {
	if len(modes) == 0 || chooser == nil {
		return
	}
	obj.ModeChoice = chooser.ChooseMode(modes, sourceName)
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
	// Check player-level cast prohibition (Angelic Arbiter, etc.)
	if g.effects.Rules.PlayerCantCastSpells(playerID) {
		return fmt.Errorf("can't cast %s: a continuous effect prevents this player from casting spells", card.Name())
	}
	// Determine X value
	xValue := 0
	if len(xValues) > 0 {
		xValue = xValues[0]
	}
	if xValue < 0 {
		return fmt.Errorf("x cannot be negative")
	}
	targets = g.acquireRandomTargets(playerID, card, card.CastTargets(), targets, xValue)
	targets = g.acquireOpponentChosenTargets(playerID, card, card.CastTargets(), targets)
	if err := g.validateActionTargets(playerID, card, card.CastTargets(), targets, xValue, "spell"); err != nil {
		return err
	}
	if err := g.validateVariableSpellTargets(playerID, card, targets, xValue); err != nil {
		return err
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
	if r := computeConditionalCostReduction(g, playerID, card, mc.Generic, targets); r > 0 {
		mc.Generic -= r
	}

	payMC := mc
	lifeLoss := 0
	// Channel: pay life for the printed generic/X portion instead of mana.
	if g.effects.Rules.IsChannelActive(playerID) && (mc.Generic > 0 || (mc.HasX && xValue > 0)) {
		payMC.Generic = 0
		payMC.HasX = false
		payMC.XCount = 0
		lifeLoss = mc.Generic
		if mc.HasX {
			lifeLoss += xValue * mc.XCount
		}
	}

	var totalCosts []Cost
	if bc, ok := card.(*BaseCard); ok {
		totalCosts = append(totalCosts, bc.AdditionalCosts()...)
	}

	var actionCosts []Cost
	for _, a := range card.Abilities() {
		if sa, ok := a.(*SpellAbility); ok && sa.Kind() == ActionSpell {
			actionCosts = append(actionCosts, sa.Costs()...)
		}
	}
	actionCosts = g.prepareActionCosts(actionCosts, targets, xValue)
	totalCosts = append(totalCosts, actionCosts...)

	payment, err := g.prepareSpellPaymentTransaction(spellPaymentSpec{
		Card:       card,
		Controller: playerID,
		Zone:       ZoneHand,
		Mana:       payMC,
		LifeLoss:   lifeLoss,
		Costs:      totalCosts,
		Targets:    targets,
		XValue:     xValue,
	})
	if err != nil {
		return fmt.Errorf("cannot pay total cost for %s: %w", name, err)
	}
	if err := payment.Commit(); err != nil {
		return err
	}

	// If an additional cost set g.resolution.X (e.g. sacrifice-capture-CMC), use it
	if g.resolution.X() != 0 && xValue == 0 {
		xValue = g.resolution.X()
		g.resolution.SetX(0)
	}

	_, err = g.pushCastSpellObject(castStackObjectOptions{
		Card:         card,
		Controller:   playerID,
		Targets:      targets,
		XValue:       xValue,
		CastZone:     ZoneHand,
		SnapshotCast: true,
	})
	return err
}

// addManaFromAbility resolves a mana ability's current productions, then runs
// its immediate post-production effects.
func (g *Game) addManaFromAbility(ma *ManaAbility, p Player, perm *Permanent) (int, error) {
	productions := ma.currentProductions(g, perm.ID())
	produced := g.addManaProductions(productions, p, perm)
	return produced, g.runManaPostProduction(ma, p, perm)
}

func (g *Game) runManaPostProduction(ma *ManaAbility, p Player, perm *Permanent) error {
	if ma == nil || len(ma.postProduction) == 0 {
		return nil
	}
	// Claim a copy-on-write permanent before card-defined follow-up effects run.
	// This keeps callbacks that locate the source through FindPermanent from
	// mutating a permanent shared with another search branch.
	perm = g.MutablePermanent(perm.ID())
	if perm == nil {
		return nil
	}
	ctx := &EffectContext{
		Game:       g,
		SourceID:   perm.ID(),
		Controller: p.PlayerID(),
		Vars:       make(map[string]any),
	}
	for _, effect := range ma.postProduction {
		if err := effect.Apply(ctx); err != nil {
			return err
		}
	}
	return nil
}

// addManaProductions adds mana to the player's pool from a list of productions.
// AnyColor productions prompt the player to choose a color. Used by both the
// proper *ManaAbility path and the *SimpleActivatedAbility tap-for-mana path.
func (g *Game) addManaProductions(productions []ManaProduction, p Player, perm *Permanent) int {
	return g.addManaProductionsForColor(productions, p, perm, Colorless)
}

// addManaProductionsForColor is the autotap-friendly variant: when
// preferredColor is non-Colorless, AnyColor productions resolve to
// preferredColor without prompting the player. Used by TapForManaWithColor
// so a requested color is honored end-to-end. AnyCombination productions
// prompt when this helper is used directly; automatic payment executes its
// solver-selected concrete combination through applyManaSolution.
func (g *Game) addManaProductionsForColor(productions []ManaProduction, p Player, perm *Permanent, preferredColor Color) int {
	var producedColors []Color
	producedAmount := 0
	for _, prod := range productions {
		amt := prod.Amount
		if amt <= 0 {
			amt = 1
		}
		producedAmount += amt
		// "X mana in any combination of colors": ask once per mana point so
		// the controller can split colors arbitrarily.
		if prod.Color == AnyColor && prod.AnyCombination && amt > 1 {
			for i := 0; i < amt; i++ {
				color := p.ChooseManaColor("add mana")
				addManaProductionToPool(p.ManaPool(), color, 1, prod.Restriction)
				producedColors = appendProducedColor(producedColors, color)
			}
			continue
		}
		color := prod.Color
		if color == AnyColor {
			if preferredColor != Colorless {
				color = preferredColor
			} else {
				color = p.ChooseManaColor("add mana")
			}
		}
		addManaProductionToPool(p.ManaPool(), color, amt, prod.Restriction)
		producedColors = appendProducedColor(producedColors, color)
	}
	g.applyManaBonuses(perm, producedColors, p)
	return producedAmount
}

func addManaProductionToPool(pool *ManaPool, color Color, amount int, restriction ManaRestriction) {
	if restriction != nil {
		pool.AddRestricted(color, amount, restriction)
		return
	}
	pool.Add(color, amount)
}

func (g *Game) fireTappedForMana(sourceID, playerID uuid.UUID, amount int) {
	if amount <= 0 {
		return
	}
	g.FireEvent(GameEvent{
		Type:     EvtTappedForMana,
		SourceID: sourceID,
		PlayerID: playerID,
		Amount:   amount,
	})
}

// applyManaBonuses checks for mana bonus effects when a permanent is tapped for mana.
func (g *Game) applyManaBonuses(tappedPerm *Permanent, producedColors []Color, p Player) {
	for _, bonus := range g.manaBonuses(tappedPerm.ID()) {
		bonusColor := Color(bonus)
		if bonus == MatchProduced {
			if len(producedColors) == 0 {
				continue
			}
			bonusColor = producedColors[0]
			if len(producedColors) > 1 {
				chosen := p.ChooseManaColor("add bonus mana")
				if slices.Contains(producedColors, chosen) {
					bonusColor = chosen
				}
			}
		}
		p.ManaPool().Add(bonusColor, 1)
	}
}

func appendProducedColor(colors []Color, color Color) []Color {
	if slices.Contains(colors, color) {
		return colors
	}
	return append(colors, color)
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

		// CR 704.5i: a planeswalker with loyalty 0 is put into its owner's
		// graveyard. Loyalty-activated abilities, attacking planeswalkers, and
		// the legacy damage-redirection rules are not implemented.
		var zeroLoyalty []*Permanent
		for _, p := range g.battlefield {
			if p.HasType(TypePlaneswalker) && int(p.Counters[Loyalty]) <= 0 {
				zeroLoyalty = append(zeroLoyalty, p)
				actions = true
			}
		}
		for _, p := range zeroLoyalty {
			g.PutPermanentIntoGraveyard(p)
		}

		// MTG rule 704.5q: +1/+1 and -1/-1 counter annihilation
		for _, p := range g.battlefield {
			plus := p.Counters[P1P1]
			minus := p.Counters[M1M1]
			if plus > 0 && minus > 0 {
				remove := min(minus, plus)
				p = g.MutablePermanent(p.ID())
				if p == nil {
					continue
				}
				p.Counters[P1P1] -= remove
				p.Counters[M1M1] -= remove
				actions = true
			}
		}

		// Check for auras attached to nothing or illegal targets (CR 704.5m / 303.4c).
		var aurasToDrop []*Permanent
		for _, p := range g.battlefield {
			if p.PhasedOut {
				continue
			}
			if p.HasSubType("Aura") && p.IsAttached() {
				host := g.FindPermanent(p.AttachedTo)
				if host == nil {
					aurasToDrop = append(aurasToDrop, p)
					actions = true
				} else if host.HasProtectionFromInGame(p.Card, g) {
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
			if p.PhasedOut {
				continue
			}
			if p.HasSubType("Equipment") && p.IsAttached() {
				host := g.FindPermanent(p.AttachedTo)
				if host == nil || !host.HasType(TypeCreature) {
					p = g.MutablePermanent(p.ID())
					if p == nil {
						continue
					}
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
				if other.ControllerID() == p.ControllerID() && other.HasSubType(landSubtype) {
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
			g.DoSacrifice(p)
		}

		// Sacrifice creatures if controller controls a specific subtype (e.g. Goblins of the Flarg)
		var toSacrificeControls []*Permanent
		for _, p := range g.battlefield {
			var forbidSubtype string
			for _, a := range p.RuntimeAbilities {
				if sa, ok := a.(*SacrificeIfControlsAbility); ok {
					forbidSubtype = sa.Subtype
					break
				}
			}
			if forbidSubtype == "" {
				continue
			}
			controlsForbid := false
			for _, other := range g.battlefield {
				if other.ID() != p.ID() && other.ControllerID() == p.ControllerID() && other.HasSubType(forbidSubtype) {
					controlsForbid = true
					break
				}
			}
			if controlsForbid {
				toSacrificeControls = append(toSacrificeControls, p)
				actions = true
			}
		}
		for _, p := range toSacrificeControls {
			g.DoSacrifice(p)
		}

		// MTG rule 704.5j: Legend rule — if a player controls two or more legendary
		// permanents with the same name, they choose one and sacrifice the rest.
		var legendCounts map[uuid.UUID]map[string][]*Permanent // controller -> name -> perms
		for _, p := range g.battlefield {
			if p.Card.HasSuperType(SuperLegendary) {
				if legendCounts == nil {
					legendCounts = make(map[uuid.UUID]map[string][]*Permanent)
				}
				if legendCounts[p.ControllerID()] == nil {
					legendCounts[p.ControllerID()] = make(map[string][]*Permanent)
				}
				legendCounts[p.ControllerID()][p.Name()] = append(legendCounts[p.ControllerID()][p.Name()], p)
			}
		}
		for ctrlID, byName := range legendCounts {
			for _, perms := range byName {
				if len(perms) > 1 {
					player := g.GetPlayer(ctrlID)
					keep := player.ChoosePermanent(perms, "legend rule: keep one", g)
					for _, p := range perms {
						if p != keep {
							g.PutPermanentIntoGraveyard(p)
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
		// ceases to exist. Token status lives on Permanent, so when a token permanent
		// leaves the battlefield its underlying Card is just a regular card — it is
		// skipped by DestroyPermanent/bounce rather than cleaned up here.

		if !actions {
			break
		}
	}

	// Put any pending triggers on the stack
	g.PutTriggersOnStack()
}

// UntapPermanent untaps the given permanent unless it has a stun counter
// (CR 122.1g — "If a permanent with a stun counter would become untapped,
// remove a stun counter from it instead. It doesn't untap.") If the
// permanent has at least one stun counter, exactly one is removed and the
// permanent stays tapped; no EvtBecameUntapped fires. Otherwise, if the
// permanent is currently tapped it untaps and EvtBecameUntapped fires.
// Returns true if the permanent actually untapped.
func (g *Game) UntapPermanent(p *Permanent) bool {
	if p == nil {
		return false
	}
	p = g.MutablePermanent(p.ID())
	if p == nil {
		return false
	}
	if p.Counters[Stun] > 0 {
		p.RemoveCounter(Stun, 1)
		return false
	}
	if !p.Tapped {
		return false
	}
	p.Tapped = false
	g.FireEvent(GameEvent{Type: EvtBecameUntapped, SourceID: p.ID()})
	return true
}

// SkipNextUntap causes the specific battlefield object to skip its next untap
// attempt during its then-controller's untap step.
func (g *Game) SkipNextUntap(permanentID uuid.UUID) {
	if g.FindPermanent(permanentID) == nil {
		return
	}
	if g.skipNextUntap == nil {
		g.skipNextUntap = make(map[uuid.UUID]int)
	}
	g.skipNextUntap[permanentID]++
}

func (g *Game) untapPermanentDuringUntapStep(p *Permanent) bool {
	if p == nil || !p.Tapped {
		return false
	}
	if remaining := g.skipNextUntap[p.ID()]; remaining > 0 {
		if remaining == 1 {
			delete(g.skipNextUntap, p.ID())
		} else {
			g.skipNextUntap[p.ID()] = remaining - 1
		}
		return false
	}
	return g.UntapPermanent(p)
}

func (g *Game) doUntap() {
	active := g.ActivePlayerObj()
	// Island Sanctuary: clear protection at the start of the player's turn
	g.effects.Rules.ClearSanctuary(active.PlayerID())

	// Snapshot the active player's untapped land count before the untap loop
	// runs (CR 502.1 happens at the very start of the turn). Power Surge and
	// similar effects read this at upkeep.
	if g.untappedLandsAtTurnStart == nil {
		g.untappedLandsAtTurnStart = make(map[uuid.UUID]int)
	}
	count := 0
	for _, p := range g.battlefield {
		if p.ControllerID() == active.PlayerID() && p.HasType(TypeLand) && !p.Tapped {
			count++
		}
	}
	g.untappedLandsAtTurnStart[active.PlayerID()] = count

	landUntapLimit := g.effects.Rules.LandUntapMax
	landsUntapped := 0
	artifactUntapLimit := g.effects.Rules.ArtifactUntapMax
	artifactsUntapped := 0
	creatureUntapLimit := g.effects.Rules.CreatureUntapMax
	creaturesUntapped := 0

	for _, p := range g.battlefield {
		if p.ControllerID() == active.PlayerID() {
			if p.HasAttr(AttrDoesNotUntap) {
				// Does not untap — skip
			} else if p.Tapped && p.HasAttr(AttrMayNotUntap) {
				// Player may choose not to untap, but only when staying tapped
				// still maintains a continuous effect on a target that remains
				// on the battlefield. Once that target is gone (e.g. Tawnos's
				// Weaponry's boosted creature or Phyrexian Gremlins' tapped
				// artifact left play), there is nothing left to maintain, so
				// untap automatically instead of asking.
				hasTapEffect, targetLives := g.effects.SourceTapMaintainedStatus(g, p.ID())
				autoUntap := hasTapEffect && !targetLives
				if !autoUntap && !active.ChooseMayAbility("untap "+p.Name()) {
					continue
				}
				g.untapPermanentDuringUntapStep(p)
			} else if p.HasType(TypeLand) && landUntapLimit >= 0 {
				// Land with untap limit in effect
				if p.Tapped && landsUntapped < landUntapLimit {
					if g.untapPermanentDuringUntapStep(p) {
						landsUntapped++
					}
				}
			} else if p.HasType(TypeArtifact) && !p.HasType(TypeLand) && artifactUntapLimit >= 0 {
				// Artifact (non-land) with untap limit in effect (Damping Field)
				if p.Tapped && artifactsUntapped < artifactUntapLimit {
					if g.untapPermanentDuringUntapStep(p) {
						artifactsUntapped++
					}
				}
			} else if p.HasType(TypeCreature) && creatureUntapLimit >= 0 {
				// Creature with untap limit in effect (Smoke)
				if p.Tapped && creaturesUntapped < creatureUntapLimit {
					if g.untapPermanentDuringUntapStep(p) {
						creaturesUntapped++
					}
				}
			} else if p.Tapped {
				g.untapPermanentDuringUntapStep(p)
			}
			if mp := g.MutablePermanent(p.ID()); mp != nil {
				mp.RevokeBaseAttr(AttrSummonSick)
			}
		}
	}
	g.landsPlayedThisTurn = 0
	g.extraLandPlaysThisTurn = nil
}

// TODO this should "tell" turn to do upkeep actions and give turn a list of actions to do
// this should iterate over permanents & cards in the game and ask if they have upkeep actions to do
// What the fuck is this method doing here? Why isn't it part of Turn or similar?
func (g *Game) doUpkeepActions() {
	active := g.ActivePlayerObj()

	// Expire "until your next upkeep" effects for the active player.
	g.effects.RemoveUntilYourNextTurn(g, active.PlayerID())

	g.FireEvent(GameEvent{
		Type:     EvtUpkeep,
		PlayerID: active.PlayerID(),
	})
	g.PutTriggersOnStack()
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

	// Run through replacement pipeline (skip draw, Aladdin's Lamp, etc.)
	action := NewDrawCardAction(uuid.Nil, active.PlayerID(), true)
	result := g.effects.ApplyReplacements(action, g)
	if result == nil {
		return // draw was replaced (skip draw, Aladdin's Lamp, etc.)
	}

	g.drawCardRaw(active)
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
	newLib := make([]Card, 1, 1+len(lib)-count+len(rest))
	newLib[0] = chosen
	newLib = append(newLib, lib[count:]...)
	newLib = append(newLib, rest...)
	p.SetLibrary(newLib)
	g.drawCardRaw(p)
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
		if p.ControllerID() == active.PlayerID() && p.HasAttr(AttrMustAttack) && !declared[p.ID()] {
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

		// Attack costs (CR 508.1e): declined or unpayable means not declared.
		if !g.PayAttackCosts(atk, active.PlayerID(), true) {
			continue
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

// AttackedDuringLastTurn reports whether the given creature attacked during playerID's most recent turn.
func (g *Game) AttackedDuringLastTurn(playerID, permID uuid.UUID) bool {
	if m, ok := g.attackedLastTurn[playerID]; ok {
		return m[permID]
	}
	return false
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
		firstForBlocker := blockerCount[ba.BlockerID] == 0
		blockerCount[ba.BlockerID]++
		g.combat.AddBlocker(ba.BlockerID, attackerID)
		g.blockedThisTurn[ba.BlockerID] = append(g.blockedThisTurn[ba.BlockerID], attackerID)
		// CR 509.3a — Flag=true marks the once-per-combat "Whenever ~ blocks"
		// firing for this blocker; subsequent attackers fire with Flag=false
		// for "blocks a creature" per-pair triggers only.
		g.FireEvent(GameEvent{
			Type:     EvtDeclaredBlocker,
			SourceID: ba.BlockerID,
			TargetID: attackerID,
			PlayerID: nonActive.PlayerID(),
			Flag:     firstForBlocker,
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
		// Insufficient blockers — none of them are legal. Drop them all,
		// and reset the Blocked flag so the attacker is treated as unblocked
		// for damage assignment (CR 509.1b — those creatures aren't legal
		// blockers, so the attacker was never legally blocked).
		dropped := group.BlockerIDs
		group.BlockerIDs = nil
		group.Blocked = false
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
		minN := max(g.effects.MinBlockers(group.AttackerID), 1)
		// Find legal blockers controlled by the defender.
		var candidates []*Permanent
		for _, p := range g.battlefield {
			if p.ControllerID() != defenderID {
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
				Flag:     true,
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
	for maxHS >= 0 && len(p.Hand()) > maxHS {
		chosen := p.ChooseCardsFromHand(1, "discard to hand size", g)
		if len(chosen) == 0 {
			break
		}
		g.PlayerDiscard(p, chosen[0].ID())
	}
	// Clear damage from all creatures
	for _, p := range g.battlefield {
		if p.Damage == 0 {
			continue
		}
		p = g.MutablePermanent(p.ID())
		if p == nil {
			continue
		}
		p.Damage = 0
	}
	// Clear mana pools
	for _, p := range g.players {
		p.ManaPool().Clear()
	}
	// Remove end-of-turn effects and clear turn-scoped state
	g.effects.RemoveEndOfTurn()
	g.effects.Apply(g)
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

	// Save active player's attackers to attackedLastTurn before clearing
	if g.attackedLastTurn == nil {
		g.attackedLastTurn = make(map[uuid.UUID]map[uuid.UUID]bool)
	}
	activeID := active.PlayerID()
	lastMap := make(map[uuid.UUID]bool, len(g.attackedThisTurn))
	maps.Copy(lastMap, g.attackedThisTurn)
	g.attackedLastTurn[activeID] = lastMap

	g.attackedThisTurn = make(map[uuid.UUID]bool)
	g.blockedThisTurn = make(map[uuid.UUID][]uuid.UUID)
	g.instantsCastThisTurn = make(map[uuid.UUID]int)
	g.sorceriesCastThisTurn = make(map[uuid.UUID]int)
	g.timesTargetedThisTurn = make(map[uuid.UUID]int)
	g.creatureDeathsThisTurn = 0
	g.clearLKI()
	g.resetPerTurnTrackers()
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
		} else if g.RunTurn(stopTurn, stopStep) {
			return
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
	activeID := g.ActivePlayerObj().PlayerID()
	if g.extraLandPlaysThisTurn != nil {
		limit += g.extraLandPlaysThisTurn[activeID]
	}
	limit += g.effects.Rules.AdditionalLandPlays(activeID)
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

// AddRevealedTopCardEffect marks playerID as playing with the top card of
// their library revealed for this Apply() cycle (Future Sight, Oracle of
// Mul Daya, Magus of the Future). Card implementations should normally
// register the RevealTopCardOfLibrary continuous effect via
// WithStaticAbility; this method is the direct entry point for callers
// (UI, AI) and tests that need to set the flag explicitly.
func (g *Game) AddRevealedTopCardEffect(playerID uuid.UUID) {
	g.effects.Rules.AddRevealedTopCard(playerID)
}

// IsTopCardRevealed reports whether the given player is currently playing
// with the top card of their library revealed.
func (g *Game) IsTopCardRevealed(playerID uuid.UUID) bool {
	return g.effects.Rules.IsTopCardRevealed(playerID)
}

// AddPlayLandsFromZone permits playerID to play lands from the given zone
// (in addition to their hand) for this Apply() cycle. Used by Oracle of
// Mul Daya and similar cards via PlayLandsFromTopOfLibrary.
func (g *Game) AddPlayLandsFromZone(playerID uuid.UUID, zone Zone) {
	g.effects.Rules.AddPlayLandsFromZone(playerID, zone)
}

// CanPlayLandsFromZone reports whether playerID may currently play lands
// from the given zone.
func (g *Game) CanPlayLandsFromZone(playerID uuid.UUID, zone Zone) bool {
	return g.effects.Rules.CanPlayLandsFromZone(playerID, zone)
}

// AddAdditionalLandPlay registers a static-ability per-cycle additional
// land-play allowance for the player. Re-registered each Apply() cycle by
// the source's continuous effect, so it auto-clears when the source
// leaves the battlefield (Azusa, Oracle of Mul Daya, Exploration).
func (g *Game) AddAdditionalLandPlay(playerID uuid.UUID, n int) {
	g.effects.Rules.AddAdditionalLandPlay(playerID, n)
}

// playLandCore moves a land from a player's hand (or, with an active
// AddPlayLandsFromZone(ZoneLibrary) grant, from the top of their library)
// to the battlefield and fires landfall triggers, but does NOT resolve the
// stack. Callers are responsible for draining the stack (via ResolveStack
// or RunPriorityRound).
func (g *Game) playLandCore(playerID, cardID uuid.UUID) error {
	if g.effects.Rules.CantPlayLands() {
		return fmt.Errorf("players can't play lands")
	}
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
	fromZone := ZoneHand
	if !ok {
		// Try the top of library if the player has a grant for that zone.
		if g.effects.Rules.CanPlayLandsFromZone(playerID, ZoneLibrary) {
			lib := p.Library()
			if len(lib) > 0 && lib[0].ID() == cardID {
				card = lib[0]
				p.SetLibrary(lib[1:])
				ok = true
				fromZone = ZoneLibrary
			}
		}
	}
	if !ok {
		return ErrCardNotInHand
	}
	if !card.HasType(TypeLand) {
		if fromZone == ZoneLibrary {
			// Restore to top of library.
			p.SetLibrary(append([]Card{card}, p.Library()...))
		} else {
			p.AddToHand(card)
		}
		return fmt.Errorf("card is not a land")
	}
	// Check expansion block (City in a Bottle)
	if g.effects.Rules.IsCardExpansionBlocked(card.Name()) {
		if fromZone == ZoneLibrary {
			p.SetLibrary(append([]Card{card}, p.Library()...))
		} else {
			p.AddToHand(card)
		}
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

// TapForMana taps a permanent for mana using its first mana ability.
// Equivalent to TapForManaWithColor with no color preference. Recognizes both
// *ManaAbility (built via WithManaAbility/WithMultiManaAbility) and
// *SimpleActivatedAbility whose only cost is tapping and whose effects are
// mana-producing (built via WithActivatedAbility(AddMana(...), Tap())
// — e.g. Mana Vault).
func (g *Game) TapForMana(playerID, permanentID uuid.UUID) error {
	return g.TapForManaWithColor(playerID, permanentID, Colorless)
}

// TapForManaWithColor taps a permanent for mana, picking the mana ability
// whose productions match preferredColor when the permanent has more than
// one mana ability (e.g. Underground Sea, which is registered as separate
// "{T}: Add {U}" and "{T}: Add {B}" abilities). Pass Colorless to mean
// "no preference" — the first mana ability is used, matching the prior
// TapForMana behavior. If no ability matches preferredColor, the first
// mana ability is used as a fallback.
func (g *Game) TapForManaWithColor(playerID, permanentID uuid.UUID, preferredColor Color) error {
	perm := g.FindPermanent(permanentID)
	if perm == nil {
		return ErrPermanentNotFound
	}
	if perm.ControllerID() != playerID {
		return fmt.Errorf("you don't control that permanent")
	}
	if perm.Tapped {
		return fmt.Errorf("permanent is already tapped")
	}
	if perm.HasAttr(AttrCantActivate) {
		return fmt.Errorf("cannot activate mana ability of %s", perm.Name())
	}

	var chosen []ManaProduction
	var chosenAbility *ManaAbility
	var firstProd []ManaProduction
	var firstAbility *ManaAbility
	for _, a := range perm.RuntimeAbilities {
		productions := abilityManaProductions(a, g, perm.ID())
		if productions == nil {
			continue
		}
		if firstProd == nil {
			firstProd = productions
			firstAbility, _ = UnwrapAbility(a).(*ManaAbility)
		}
		if preferredColor != Colorless && productionsMatchColor(productions, preferredColor) {
			chosen = productions
			chosenAbility, _ = UnwrapAbility(a).(*ManaAbility)
			break
		}
	}
	if chosen == nil {
		chosen = firstProd
		chosenAbility = firstAbility
	}
	if chosen == nil {
		return fmt.Errorf("permanent has no mana ability")
	}
	if !perm.CanTapForEffect(g) {
		return fmt.Errorf("creature has summoning sickness")
	}
	g.TapPermanent(perm)
	if p := g.GetPlayer(playerID); p != nil {
		produced := g.addManaProductionsForColor(chosen, p, perm, preferredColor)
		if err := g.runManaPostProduction(chosenAbility, p, perm); err != nil {
			return err
		}
		g.fireTappedForMana(perm.ID(), playerID, produced)
	}
	return nil
}

// productionsMatchColor reports whether the production list can satisfy a
// request for preferredColor. A specific-color production matches its color;
// an AnyColor production matches every color request.
func productionsMatchColor(productions []ManaProduction, preferredColor Color) bool {
	for _, p := range productions {
		if p.Color == preferredColor || p.Color == AnyColor {
			return true
		}
	}
	return false
}

// abilityManaProductions returns the mana productions for an ability that acts
// as a tap-for-mana mana source, or nil if the ability is not such a source.
// Recognizes both *ManaAbility and *SimpleActivatedAbility whose only cost is
// tapping and whose effects all produce mana (AddMana / AddAnyMana).
func abilityManaProductions(a Ability, g GameReader, sourceID uuid.UUID) []ManaProduction {
	a = UnwrapAbility(a)
	switch ab := a.(type) {
	case *ManaAbility:
		return ab.currentProductions(g, sourceID)
	case *SimpleActivatedAbility:
		return activatedManaProductions(ab)
	}
	return nil
}

// activatedManaProductions extracts mana productions from a SimpleActivatedAbility
// that behaves as a tap-for-mana ability — exactly one cost (tapping the source)
// and all effects are AddMana / AddAnyMana. Returns nil if the shape doesn't match.
func activatedManaProductions(a *SimpleActivatedAbility) []ManaProduction {
	if len(a.costs) != 1 {
		return nil
	}
	if _, ok := a.costs[0].(*tap); !ok {
		return nil
	}
	if len(a.effects) == 0 {
		return nil
	}
	var productions []ManaProduction
	for _, e := range a.effects {
		switch d := e.(type) {
		case *addManaEffect:
			productions = append(productions, ManaProduction{Color: d.color, Amount: d.amount})
		case *addAnyManaEffect:
			productions = append(productions, ManaProduction{Color: AnyColor, Amount: d.amount})
		default:
			return nil
		}
	}
	return productions
}

// manaSourceInfo describes one untapped permanent and each exact mana ability
// it can currently activate. Colors is the union used only by preservation
// scoring; payment planning operates on Abilities without merging their output.
type manaSourceInfo struct {
	PermanentID uuid.UUID
	Colors      []Color
	Abilities   []manaSourceAbility
	Bonuses     []ManaBonusColor
}

// AutoTapHint informs the smart auto-tap algorithm about the action being paid
// for. Callers pass a hint so the algorithm can deprioritize tapping the
// permanent that is activating an ability and weight color preferences by
// other spells the player still wants to cast.
type AutoTapHint struct {
	// ReservedSources lists permanents that the surrounding action must keep
	// untapped while mana is produced. Attack declarations use this for chosen
	// attackers; multi-source total costs may reserve more than one permanent.
	ReservedSources []uuid.UUID
	// ActivationSource is the permanent whose activated ability is being paid
	// for (zero UUID if not applicable, e.g. when casting a spell). Sources
	// matching this id are penalized so we only tap them as a last resort.
	ActivationSource uuid.UUID
	// ActivationTapsSource is true when the ability being activated includes
	// a {T} cost on its source. In that case the source will be tapped during
	// cost payment anyway, so it must be hard-excluded from auto-tap to avoid
	// a "source already tapped" failure later.
	ActivationTapsSource bool
	// CastingCard is the card currently being cast (zero UUID if not casting).
	// Its colored pips are excluded from the hand-demand calculation so we
	// don't count the cost we're paying for against itself.
	CastingCard uuid.UUID
}

var manaPoolColors = [...]Color{White, Blue, Black, Red, Green, Colorless}

func (g *Game) manaBonuses(permanentID uuid.UUID) []ManaBonusColor {
	tappedPerm := g.FindPermanent(permanentID)
	if tappedPerm == nil {
		return nil
	}
	var bonuses []ManaBonusColor
	for _, perm := range g.battlefield {
		for _, a := range perm.RuntimeAbilities {
			inner := UnwrapAbility(a)
			if mb, ok := inner.(*ManaBonusAbility); ok {
				if mb.AttachedOnly {
					if perm.AttachedTo == tappedPerm.ID() {
						bonuses = append(bonuses, ManaBonusColor(mb.BonusMana))
					}
				} else if mb.Filter.Match(tappedPerm, g) {
					if mb.MatchProduced {
						bonuses = append(bonuses, MatchProduced)
					} else {
						bonuses = append(bonuses, ManaBonusColor(mb.BonusMana))
					}
				}
			}
		}
	}
	return bonuses
}

// getUntappedManaSources returns all untapped permanents with mana abilities for a player.
func (g *Game) getUntappedManaSources(playerID uuid.UUID) []manaSourceInfo {
	var sources []manaSourceInfo
	return g.appendUntappedManaSources(playerID, sources)
}

func (g *Game) appendUntappedManaSources(playerID uuid.UUID, sources []manaSourceInfo) []manaSourceInfo {
	for _, perm := range g.battlefield {
		if perm.ControllerID() != playerID || perm.Tapped {
			continue
		}
		if perm.HasAttr(AttrCantActivate) {
			continue
		}
		// Skip summoning-sick creatures without haste
		if !perm.CanTapForEffect(g) {
			continue
		}
		var colors []Color
		seen := [AnyColor + 1]bool{}
		var abilities []manaSourceAbility
		for abilityIndex, ability := range perm.RuntimeAbilities {
			sourceAbility, ok := manaSourceAbilityForPlanning(ability, perm.ID(), abilityIndex, g)
			if !ok {
				continue
			}
			abilities = append(abilities, sourceAbility)
			for _, p := range sourceAbility.Productions {
				for _, c := range expandProductionColor(p.Color) {
					if !seen[c] {
						seen[c] = true
						colors = append(colors, c)
					}
				}
			}
		}
		if len(abilities) == 0 {
			continue
		}
		if len(colors) == 0 {
			colors = []Color{Colorless}
		}
		sources = append(sources, manaSourceInfo{
			PermanentID: perm.ID(),
			Colors:      colors,
			Abilities:   abilities,
			Bonuses:     g.manaBonuses(perm.ID()),
		})
	}
	return sources
}

// expandProductionColor returns the concrete colors a ManaProduction can
// produce. AnyColor expands to the five colored colors; everything else
// maps to itself.
func expandProductionColor(c Color) []Color {
	if c == AnyColor {
		return []Color{White, Blue, Black, Red, Green}
	}
	return []Color{c}
}

func productionsTotalAmount(productions []ManaProduction) int {
	total := 0
	for _, p := range productions {
		if p.Amount <= 0 {
			total++
		} else {
			total += p.Amount
		}
	}
	return total
}

func maxManaSourceOutput(source manaSourceInfo) int {
	maximum := 0
	for _, ability := range source.Abilities {
		maximum = max(maximum, productionsTotalAmount(ability.Productions)+len(source.Bonuses))
	}
	return maximum
}

// AutoTapForCost taps untapped lands/mana sources to pay a mana cost,
// accounting for mana already in the player's pool. Equivalent to
// AutoTapForCostWithHint with a zero hint; smart-tap heuristics only kick in
// when callers pass an AutoTapHint via AutoTapForCostWithHint.
func (g *Game) AutoTapForCost(playerID uuid.UUID, mc ManaCost) error {
	return g.AutoTapForCostWithHint(playerID, mc, AutoTapHint{})
}

// AutoTapForCostWithHint taps untapped lands/mana sources to pay a mana cost,
// using the hint to deprioritize the activated-ability source and to weight
// color preferences by other castable spells in hand. See AutoTapHint for
// hint semantics. Gathers solver inputs from Game state, delegates the
// decision to SolveMana, and executes the exact planned abilities and choices.
func (g *Game) AutoTapForCostWithHint(playerID uuid.UUID, mc ManaCost, hint AutoTapHint) error {
	return g.autoTapForCost(playerID, mc, hint, nil)
}

func (g *Game) autoTapForCost(playerID uuid.UUID, mc ManaCost, hint AutoTapHint, spellCtx *SpellPaymentContext) error {
	solution, err := g.planManaForCost(playerID, mc, hint, spellCtx)
	if err != nil {
		return err
	}
	return g.applyManaSolution(playerID, solution)
}

func (g *Game) planManaForCost(playerID uuid.UUID, mc ManaCost, hint AutoTapHint, spellCtx *SpellPaymentContext) (*ManaSolution, error) {
	p := g.GetPlayer(playerID)
	if p == nil {
		return nil, ErrPlayerNotFound
	}
	if mc.IsZero() {
		return &ManaSolution{}, nil
	}
	sources := g.getUntappedManaSources(playerID)

	// Remove sources reserved for another part of the total cost or action.
	if len(hint.ReservedSources) > 0 || hint.ActivationTapsSource && hint.ActivationSource != uuid.Nil {
		filtered := sources[:0]
		for _, src := range sources {
			reserved := hint.ActivationTapsSource && src.PermanentID == hint.ActivationSource
			if !reserved {
				if slices.Contains(hint.ReservedSources, src.PermanentID) {
					reserved = true
				}
			}
			if !reserved {
				filtered = append(filtered, src)
			}
		}
		sources = filtered
	}

	handDemand := g.computeHandDemand(playerID, hint.CastingCard)
	scores := make([]int, len(sources))
	for i, src := range sources {
		scores[i] = g.preservationScore(src, hint, handDemand)
	}

	solverInputs := ManaSolverInputs{
		Pool:         p.ManaPool(),
		Cost:         mc,
		Sources:      sources,
		Scores:       scores,
		Conversions:  p.ManaPool().ManaConversions,
		SpellContext: spellCtx,
	}
	return SolveMana(solverInputs)
}

// preservationScore returns the score for a mana source under the smart-tap
// heuristic. Higher = prefer to keep untapped (tap last). See AutoTapHint.
func (g *Game) preservationScore(src manaSourceInfo, hint AutoTapHint, handDemand [AnyColor + 1]int) int {
	score := 0
	// Heavy penalty for the activated-ability's own source, so it only taps
	// as a last resort. (When ActivationTapsSource is true the source has
	// already been filtered out.)
	if hint.ActivationSource != uuid.Nil && src.PermanentID == hint.ActivationSource {
		score += 1000
	}
	// Each non-mana activated ability on the permanent adds utility — prefer
	// to keep utility lands like Strip Mine and Mishra's Factory untapped.
	score += 10 * g.utilityAbilityCount(src.PermanentID)
	// Preserve mana sources whose active tap triggers are detrimental, such as
	// City of Brass dealing damage to its controller.
	score += g.manaSourceDrawbackScore(src.PermanentID)
	// Color flexibility: colorless-only sources (Sol Ring, Wastes) are least
	// flexible and tap first. Each colored option adds +1 — duals are
	// preserved more than basics, AnyColor sources more than duals.
	for _, c := range src.Colors {
		if c == Colorless {
			continue
		}
		score++
		// Demand from other castable hand spells — preserve sources whose
		// color is still needed for unplayed cards.
		if c >= 0 && int(c) <= int(AnyColor) {
			score += handDemand[c]
		}
	}
	return score
}

func (g *Game) manaSourceDrawbackScore(permID uuid.UUID) int {
	perm := g.FindPermanent(permID)
	if perm == nil {
		return 0
	}
	score := 0
	for _, ability := range perm.RuntimeAbilities {
		triggered, ok := UnwrapAbility(ability).(TriggeredAbility)
		if !ok || !triggered.CheckEventType(EvtTapped) {
			continue
		}
		for _, effect := range triggered.Effects() {
			props := effect.Properties()
			if props.Outcome != OutcomeDetriment {
				continue
			}
			severity := 1
			if props.DamageValue != nil {
				severity = max(props.DamageValue.Resolve(g, perm.ID(), perm.ControllerID(), nil), 1)
			}
			score += 25 * severity
		}
	}
	return score
}

// computeHandDemand sums colored pip demand across spells in the player's
// hand, excluding the card identified by `exclude` (the spell we're currently
// paying for). Hybrid pips count toward both their colors. Used to bias
// auto-tap toward preserving lands whose color matches still-castable spells.
func (g *Game) computeHandDemand(playerID, exclude uuid.UUID) [AnyColor + 1]int {
	var demand [AnyColor + 1]int
	p := g.GetPlayer(playerID)
	if p == nil {
		return demand
	}
	for _, c := range p.Hand() {
		if c.ID() == exclude {
			continue
		}
		if c.HasType(TypeLand) {
			continue
		}
		mc := c.ManaCost()
		demand[White] += mc.White
		demand[Blue] += mc.Blue
		demand[Black] += mc.Black
		demand[Red] += mc.Red
		demand[Green] += mc.Green
		for _, h := range mc.Hybrid {
			demand[h.A]++
			demand[h.B]++
		}
	}
	return demand
}

// utilityAbilityCount returns the number of activated abilities on a permanent
// that are NOT tap-for-mana abilities. Mana-producing abilities (both
// *ManaAbility and the SimpleActivatedAbility {T}: Add X form) are skipped so
// Mana Vault and similar cost-laden mana sources don't get a utility penalty.
func (g *Game) utilityAbilityCount(permID uuid.UUID) int {
	perm := g.FindPermanent(permID)
	if perm == nil {
		return 0
	}
	count := 0
	for abilityIndex, a := range perm.RuntimeAbilities {
		inner := UnwrapAbility(a)
		if _, ok := inner.(ActivatedAbility); !ok {
			continue
		}
		if _, ok := manaSourceAbilityForPlanning(a, perm.ID(), abilityIndex, g); ok {
			continue
		}
		count++
	}
	return count
}

// HypotheticalMana returns the total mana a player could produce right now:
// floating pool plus the per-tap output (and Mana Flare-style bonuses) of
// every untapped mana source. Used by spell-evaluation heuristics that just
// want a count, not an affordability decision — for color-aware checks use
// CanAfford/MaxXValue, which route through SolveMana.
func (g *Game) HypotheticalMana(playerID uuid.UUID) int {
	p := g.GetPlayer(playerID)
	if p == nil {
		return 0
	}
	total := p.ManaPool().TotalMana()
	sources := g.appendUntappedManaSources(playerID, g.manaScratch[:0])
	g.manaScratch = sources
	for _, src := range sources {
		total += maxManaSourceOutput(src)
	}
	return total
}

// CanAfford returns true if a player has enough mana (pool + untapped sources)
// to pay a cost. spellCtx is non-nil when checking affordability for casting
// a specific spell (so restricted mana can count); nil for ability costs.
func (g *Game) CanAfford(playerID uuid.UUID, mc ManaCost, spellCtx *SpellPaymentContext) bool {
	p := g.GetPlayer(playerID)
	if p == nil {
		return false
	}
	sources := g.appendUntappedManaSources(playerID, g.manaScratch[:0])
	g.manaScratch = sources
	return CanSolveMana(ManaSolverInputs{
		Pool:         p.ManaPool(),
		Cost:         mc,
		Sources:      sources,
		Conversions:  p.ManaPool().ManaConversions,
		SpellContext: spellCtx,
	})
}

// MaxXValue returns the maximum X value a player can pay for a spell with cost mc,
// considering mana in pool plus untapped sources. Binary-searches via SolveMana
// so dual lands and conversions are honored.
func (g *Game) MaxXValue(playerID uuid.UUID, mc ManaCost, spellCtx *SpellPaymentContext) int {
	if !mc.HasX || mc.XCount == 0 {
		return 0
	}
	p := g.GetPlayer(playerID)
	if p == nil {
		return 0
	}
	sources := g.appendUntappedManaSources(playerID, g.manaScratch[:0])
	g.manaScratch = sources
	scores := make([]int, len(sources))
	conv := p.ManaPool().ManaConversions
	pool := p.ManaPool()

	// Upper bound: every source taps for its full Amount + bonus, plus pool.
	upperMana := pool.TotalMana()
	for _, src := range sources {
		upperMana += maxManaSourceOutput(src)
	}
	tryX := func(x int) bool {
		cost := ManaCost{
			Generic: mc.Generic + x*mc.XCount,
			White:   mc.White,
			Blue:    mc.Blue,
			Black:   mc.Black,
			Red:     mc.Red,
			Green:   mc.Green,
			Hybrid:  mc.Hybrid,
		}
		_, err := SolveMana(ManaSolverInputs{
			Pool:         pool,
			Cost:         cost,
			Sources:      sources,
			Scores:       scores,
			Conversions:  conv,
			SpellContext: spellCtx,
		})
		return err == nil
	}
	if !tryX(0) {
		return 0
	}
	hi := upperMana / mc.XCount
	if hi == 0 {
		return 0
	}
	lo := 0
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if tryX(mid) {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo
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
		if !g.CanAfford(playerID, checkMC, SpellContextForCard(card)) {
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
		isOwner := perm.ControllerID() == playerID
		for i, a := range perm.RuntimeAbilities {
			inner := UnwrapAbility(a)
			aa, ok := inner.(ActivatedAbility)
			if !ok {
				continue
			}
			// An ActionDefinition implements ActivatedAbility regardless of kind.
			// Spell-kind actions are the resolution effect of a cast spell, not
			// something a player activates from the battlefield — surfacing them
			// would let the search re-fire an aura's ETB effect indefinitely.
			if def, ok := inner.(*ActionDefinition); ok && def.Kind() != ActionActivated {
				continue
			}
			// Check if this ability can be used by non-controllers
			if !isOwner {
				saa, isSAA := inner.(*SimpleActivatedAbility)
				if !isSAA || !saa.IsAnyPlayerAbility() {
					continue
				}
			}
			// If opponent-only, the controller cannot activate it
			if isOwner {
				saa, isSAA := inner.(*SimpleActivatedAbility)
				if isSAA && saa.IsOpponentOnlyAbility() {
					continue
				}
			}
			// Skip mana abilities - those are handled separately
			if _, isMana := inner.(*ManaAbility); isMana {
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
		if perm.ControllerID() != playerID {
			return fmt.Errorf("only the controller may activate mana abilities")
		}
		if perm.HasAttr(AttrCantActivate) {
			return fmt.Errorf("cannot activate mana ability of %s", perm.Name())
		}
		if perm.Tapped || !perm.CanTapForEffect(g) {
			return fmt.Errorf("cannot tap %s for mana", perm.Name())
		}
		g.TapPermanent(perm)
		p := g.GetPlayer(playerID)
		produced := 0
		if p != nil {
			var err error
			produced, err = g.addManaFromAbility(ma, p, perm)
			if err != nil {
				return err
			}
		}
		g.FireEvent(GameEvent{
			Type:     EvtAbilityActivated,
			SourceID: perm.ID(),
			PlayerID: playerID,
		})
		g.fireTappedForMana(perm.ID(), playerID, produced)
		return nil
	}

	aa, ok := inner.(ActivatedAbility)
	if !ok {
		return fmt.Errorf("not an activated ability")
	}
	if def, ok := inner.(*ActionDefinition); ok && def.Kind() != ActionActivated {
		return fmt.Errorf("not an activated ability")
	}
	saa, isSAA := inner.(*SimpleActivatedAbility)
	if perm.ControllerID() != playerID {
		if !isSAA || !saa.IsAnyPlayerAbility() {
			return fmt.Errorf("only the controller may activate this ability")
		}
	}
	if perm.ControllerID() == playerID && isSAA && saa.IsOpponentOnlyAbility() {
		return fmt.Errorf("only opponents may activate this ability")
	}
	if perm.HasAttr(AttrCantActivateNonManaAbilities) {
		return fmt.Errorf("non-mana activated abilities of %s are prevented", perm.Name())
	}
	if !aa.CanActivate(playerID, g) {
		return fmt.Errorf("cannot activate ability")
	}

	actionTargets := aa.Targets()
	actionEffects := aa.Effects()
	modeChoice := -1
	var modalTargets [][]uuid.UUID
	if modal, ok := aa.(interface{ Modes() []Mode }); ok && len(modal.Modes()) > 0 {
		modes := modal.Modes()
		chooser := g.GetPlayer(playerID)
		if effectChooser, ok := chooser.(EffectModeChooser); ok {
			modeChoice = effectChooser.ChooseModeWithEffects(modes, perm.Card.Name(), g)
		} else if chooser != nil {
			labels := make([]string, len(modes))
			for i, mode := range modes {
				labels[i] = mode.Label
			}
			modeChoice = chooser.ChooseMode(labels, perm.Card.Name())
		}
		if modeChoice < 0 || modeChoice >= len(modes) {
			modeChoice = 0
		}
		mode := modes[modeChoice]
		actionTargets = mode.Targets
		actionEffects = mode.Effects
		targets = g.promptTargetsForList(playerID, perm.Card, actionTargets)
		modalTargets = make([][]uuid.UUID, len(modes))
		modalTargets[modeChoice] = append([]uuid.UUID(nil), targets...)
	}

	// When the caller supplied no targets but the ability's targeting is
	// forced (e.g. "you", or a single legal target), fill them in here so the
	// ability doesn't fizzle. The interactive layer relies on this to skip
	// prompting for targets that offer no real choice.
	if len(targets) == 0 {
		if forced := ForcedActivationTargets(playerID, perm.Card, actionTargets, g); forced != nil {
			targets = forced
		}
	}
	targets = g.acquireRandomTargets(playerID, perm.Card, actionTargets, targets, g.resolution.X())
	targets = g.acquireOpponentChosenTargets(playerID, perm.Card, actionTargets, targets)

	if err := g.validateActionTargets(playerID, perm.Card, actionTargets, targets, g.resolution.X(), "ability"); err != nil {
		return err
	}

	// Lock and validate the total activation cost before changing live state.
	// If the ability includes {T}, reserve its source from mana planning.
	hasTapCost := false
	for _, c := range aa.Costs() {
		if _, ok := c.(*tap); ok {
			hasTapCost = true
			break
		}
	}
	hint := AutoTapHint{ActivationSource: perm.ID(), ActivationTapsSource: hasTapCost}
	preparedCosts := g.prepareActionCosts(aa.Costs(), targets, g.resolution.X())
	payment, err := g.prepareActionPaymentTransaction(actionPaymentSpec{
		Controller:               playerID,
		SourceID:                 perm.ID(),
		Costs:                    preparedCosts,
		Targets:                  targets,
		XValue:                   g.resolution.X(),
		Hint:                     hint,
		ApplyActivationReduction: true,
	})
	if err != nil {
		return fmt.Errorf("cannot pay activation costs: %w", err)
	}
	if err := payment.Commit(); err != nil {
		return err
	}

	// Mark once-per-turn abilities as used
	if saa, ok := aa.(*SimpleActivatedAbility); ok {
		saa.MarkActivated()
	}

	obj := newStackObject(playerID, perm.ID(), nil, actionEffects, targets, g.resolution.X(), true)
	obj.TargetSpecs = expandTargetSpecs(actionTargets, targets, g.resolution.X())
	obj.TargetSource = perm.Card
	obj.ModeChoice = modeChoice
	obj.ModalTargets = modalTargets

	if produced := tappedForManaAmount(aa, actionEffects, actionTargets); produced > 0 {
		g.FireEvent(GameEvent{
			Type:     EvtAbilityActivated,
			SourceID: perm.ID(),
			PlayerID: playerID,
			Flag:     true,
		})
		g.ResolveStackObject(obj)
		g.fireTappedForMana(perm.ID(), playerID, produced)
		return nil
	}

	if modeChoice < 0 {
		chooseModeForStackObject(obj, perm.Card.Modes(), g.GetPlayer(playerID), perm.Card.Name())
	}

	for _, eff := range actionEffects {
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

	g.pushStack(obj)

	g.FireEvent(GameEvent{
		Type:     EvtAbilityActivated,
		SourceID: perm.ID(),
		PlayerID: playerID,
		Flag:     hasTapCost,
	})

	g.fireBecomesTargetEvents(obj, true)

	return nil
}

func tappedForManaAmount(ability ActivatedAbility, effects []Effect, targets []Target) int {
	if len(targets) != 0 {
		return 0
	}
	hasTapCost := false
	for _, cost := range ability.Costs() {
		if _, ok := cost.(*tap); ok {
			hasTapCost = true
			break
		}
	}
	if !hasTapCost {
		return 0
	}
	amount := 0
	for _, effect := range effects {
		switch add := effect.(type) {
		case *addManaEffect:
			if add.amount > 0 {
				amount += add.amount
			}
		case *addAnyManaEffect:
			if add.amount > 0 {
				amount += add.amount
			}
		}
	}
	return amount
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
