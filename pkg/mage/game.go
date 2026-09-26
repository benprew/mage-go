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
	players           []Player
	anteEnabled       bool
	originalOwners    map[uuid.UUID]uuid.UUID
	anteResult        []OwnershipChange
	anteSettled       bool
	zones             ZoneSystem
	stack             *Stack
	combat            *Combat
	effects           *EffectManager
	layer2Controllers map[uuid.UUID]uuid.UUID
	mana              ManaSystem

	// TurnSystem owns turn, step, active player, extra turns, scheduling, and untap skips.
	turns TurnSystem

	// TriggerSystem owns pending, delayed, and state-triggered abilities.
	triggers TriggerSystem

	// ResolutionState owns transient resolution and cost scratch state.
	resolution ResolutionState

	// TrackerSystem owns turn-scoped and duel-scoped observations and counters.
	trackers TrackerSystem

	// DamageSystem owns damage execution, history, aggregation, and reflection.
	damage      DamageSystem
	damageBatch *damageBatch

	// Mill amount modifiers (CR 614 replacement-style) keyed by source permanent ID.
	// Each entry maps milled-player ID -> proposed amount -> new amount; modifiers
	// stack and are dropped when the source leaves the battlefield (cleared in Apply).
	millModifiers []millModifierEntry

	// Life-gain amount modifiers (CR 614 replacement-style) keyed by source
	// permanent ID. Used by Rhox Faithmender ("If you would gain life, you
	// gain twice that much life instead.") and similar effects.
	lifeGainModifiers []lifeGainModifierEntry

	// RandomSource encapsulates random number and coin flip outcomes
	random RandomSource

	// Priority handler — called when a player receives priority.
	// If nil, the engine drains the stack atomically (legacy behavior).
	onPriority PriorityHandler

	// AfterPriorityAction is called after a non-pass priority action is executed.
	// Used by the interactive layer for logging and display.
	afterPriorityAction func(g *Game, playerIdx int, action PriorityAction)

	// BeforeStackResolve is called before the top of the stack is resolved
	// during a priority round. Used by the interactive layer for logging.
	beforeStackResolve func(g *Game)

	// Control flags
	stopped bool

	// customState is a per-game string-keyed bag for set-specific keyword
	// support to stash auxiliary state (e.g. Paradigm "have I resolved a
	// spell with this name yet?" tracking). Populate via paradigmStateOf
	// and similar accessors in keyword_sos.go. Survives the lifetime of
	// the game; cleared per-game via NewGame.
	customState map[string]any
}

func (g *Game) ActivePlayer() int {
	return g.turns.ActivePlayerIndex()
}

// NewGame creates a new 2-player game.
func NewGame(playerA, playerB Player) *Game {
	return newGame(playerA, playerB, false)
}

func newGame(playerA, playerB Player, anteEnabled bool) *Game {
	return &Game{
		players:        []Player{playerA, playerB},
		anteEnabled:    anteEnabled,
		originalOwners: make(map[uuid.UUID]uuid.UUID),
		zones:          NewZoneSystem(),
		stack:          NewStack(),
		combat:         NewCombat(),
		effects:        NewEffectManager(),
		resolution:     NewResolutionState(),
		trackers:       NewTrackerSystem(),
		random:         NewRandomSource(),
		triggers:       NewTriggerSystem(),
		mana:           NewManaSystem(),
		damage:         NewDamageSystem(),
		turns:          NewTurnSystem(),
		customState:    make(map[string]any),
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
	if len(g.players) == 0 {
		return nil
	}
	idx := g.turns.ActivePlayerIndex()
	if idx < 0 || idx >= len(g.players) {
		return nil
	}
	return g.players[idx]
}

// NonActivePlayerObj returns the non-active player.
func (g *Game) NonActivePlayerObj() Player {
	if len(g.players) < 2 {
		return nil
	}
	idx := (g.turns.ActivePlayerIndex() + 1) % len(g.players)
	return g.players[idx]
}

// FindPermanent finds a permanent by ID on the battlefield.
// Phased-out permanents are invisible.
func (g *Game) FindPermanent(id uuid.UUID) *Permanent {
	return g.zones.FindPermanent(id)
}

// PhaseOut phases the permanent with the given ID out of the battlefield, along
// with any Auras and Equipment attached to it (CR 702.26f, "phasing out
// indirectly"). Counters, attachments, and other on-permanent state are
// retained while phased out (CR 702.26d). It returns the IDs that were phased
// out — the permanent followed by its attachments — so the caller can later
// phase exactly that set back in with PhaseIn.
func (g *Game) PhaseOut(id uuid.UUID) []uuid.UUID {
	phased := g.zones.PhaseOut(id)
	for _, permanentID := range phased {
		g.combat.RemoveFromCombat(permanentID)
	}
	return phased
}

// PhaseIn phases the given permanents back onto the battlefield. IDs that are
// not currently phased out are skipped.
func (g *Game) PhaseIn(ids []uuid.UUID) {
	g.zones.PhaseIn(ids)
}

// FindPermanentIncludingPhased finds a permanent by ID even if phased out.
func (g *Game) FindPermanentIncludingPhased(id uuid.UUID) *Permanent {
	return g.zones.FindPermanentIncludingPhased(id)
}

// MutablePermanent returns an owned battlefield permanent pointer suitable for
// mutation. Search clones initially share permanent pointers; the first write
// to a shared permanent clones that permanent and replaces the battlefield
// entry in this Game only. Phased-out permanents are invisible.
func (g *Game) MutablePermanent(id uuid.UUID) *Permanent {
	return g.zones.MutablePermanent(id)
}

// mutablePermanentIncludingPhased is the mutable counterpart to
// FindPermanentIncludingPhased.
func (g *Game) mutablePermanentIncludingPhased(id uuid.UUID) *Permanent {
	return g.zones.MutablePermanentIncludingPhased(id)
}

// MutablePermanentIncludingPhased is the exported mutable counterpart to
// FindPermanentIncludingPhased. Returns an owned battlefield permanent
// pointer suitable for mutation (triggers copy-on-write under search).
func (g *Game) MutablePermanentIncludingPhased(id uuid.UUID) *Permanent {
	return g.zones.MutablePermanentIncludingPhased(id)
}

// FindPermanentByName finds a permanent by name on the battlefield (first match).
// Phased-out permanents are invisible.
func (g *Game) FindPermanentByName(name string, controller uuid.UUID) *Permanent {
	return g.zones.FindPermanentByName(name, controller)
}

// AnyBattlefield returns true if any permanent on the battlefield matches f.
// Phased-out permanents are invisible.
func (g *Game) AnyBattlefield(f PermanentFilter) bool {
	return g.zones.AnyBattlefield(f, g)
}

// FilterBattlefield returns all permanents on the battlefield matching f.
// Phased-out permanents are invisible.
func (g *Game) FilterBattlefield(f PermanentFilter) []*Permanent {
	return g.zones.FilterBattlefield(f, g)
}

// CountBattlefield returns the number of permanents on the battlefield matching f.
// Phased-out permanents are invisible.
func (g *Game) CountBattlefield(f PermanentFilter) int {
	return g.zones.CountBattlefield(f, g)
}

// FindCardAnywhere finds a card by ID anywhere in the game.
// Also checks the currently resolving card (which may be in limbo between
// stack pop and graveyard placement during resolution).
func (g *Game) FindCardAnywhere(id uuid.UUID) Card {
	for _, p := range g.zones.battlefield {
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
	for _, ec := range g.zones.exile {
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
	return g.random.FlipCoin(playerID)
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
	perm := g.prepareBattlefieldEntry(card, controller, colors)
	if perm == nil {
		return nil
	}
	g.registerBattlefieldEntry(perm, colors, colorSourceID)
	g.completeBattlefieldEntry(perm, controller)
	return perm
}

func (g *Game) putTokensOnBattlefieldTogether(cards []Card, controller uuid.UUID) {
	permanents := make([]*Permanent, 0, len(cards))
	for _, card := range cards {
		if perm := g.prepareBattlefieldEntry(card, controller, nil); perm != nil {
			permanents = append(permanents, perm)
		}
	}
	for _, perm := range permanents {
		g.registerBattlefieldEntry(perm, nil, uuid.Nil)
	}
	for _, perm := range permanents {
		g.completeBattlefieldEntry(perm, controller)
	}
}

func (g *Game) prepareBattlefieldEntry(card Card, controller uuid.UUID, colors *[]Color) *Permanent {
	if card.HasType(TypeLand) && g.effects.Rules.LandsCantEnter() {
		return nil
	}
	perm := NewPermanent(card, controller)
	if colors != nil {
		override := append([]Color(nil), (*colors)...)
		perm.ColorOverride = &override
	}
	g.zones.addOwnedPermanent(perm)
	perm.turnControlGained = g.turns.Turn()

	g.syncAbilityContext(perm)

	// EntersTapped keyword check — consumed on entry, attr cleared immediately after.
	if perm.HasKeyword(EntersTapped) || g.effects.Rules.ShouldEnterTapped(perm) {
		perm.Tapped = true
		perm.RevokeBaseAttr(EntersTapped)
	}

	// Expose the entering permanent to FindPermanent for the duration of ETB
	// resolution so counter-placement replacements (ETB additional counters,
	// doublers) can match it before it joins g.battlefield.
	g.zones.SetEnteringPermanent(perm)
	defer g.zones.ClearEnteringPermanent()

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

	return perm
}

func (g *Game) registerBattlefieldEntry(perm *Permanent, colors *[]Color, colorSourceID uuid.UUID) {
	g.zones.AddPermanent(perm)
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
}

func (g *Game) completeBattlefieldEntry(perm *Permanent, controller uuid.UUID) {
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
	if perm == nil || defenderID == uuid.Nil || !g.isValidDefender(defenderID) {
		return perm
	}
	g.combat.AddAttackerIncarnation(perm.ID(), perm.incarnationID, defenderID)
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
	if perm == nil || attackerID == uuid.Nil || !g.combat.IsAttacking(attackerID) {
		return perm
	}
	wasBlocked := g.attackerIsBlocked(attackerID)
	g.combat.AddBlockerIncarnation(perm.ID(), perm.incarnationID, attackerID)
	g.trackers.Turn.RecordBlocked(perm.ID(), attackerID)
	if attacker := g.FindPermanent(attackerID); attacker != nil {
		g.FireEvent(GameEvent{Type: EvtCreatureBlockedBy, SourceID: attackerID, TargetID: perm.ID(), PlayerID: attacker.ControllerID()})
		g.FireEvent(GameEvent{Type: EvtBlockerGroup, SourceID: attackerID, RelatedIDs: []uuid.UUID{perm.ID()}, PlayerID: attacker.ControllerID()})
	}
	if !wasBlocked {
		g.fireCreatureBecomesBlocked(attackerID)
	}
	return perm
}

func (g *Game) attackerIsBlocked(attackerID uuid.UUID) bool {
	for _, group := range g.combat.Groups {
		if group.AttackerID == attackerID {
			return group.Blocked
		}
	}
	return false
}

func (g *Game) fireUnblockedAttackers() {
	for _, group := range g.combat.Groups {
		if group.Blocked || !g.combat.IsAttacking(group.AttackerID) {
			continue
		}
		attacker := g.FindPermanent(group.AttackerID)
		if attacker == nil {
			continue
		}
		g.FireEvent(GameEvent{Type: EvtAttacksUnblocked, SourceID: attacker.ID(), PlayerID: attacker.ControllerID(), TargetID: group.DefenderID})
	}
}

func (g *Game) blockedAttackersBeforeDeclaration() map[uuid.UUID]bool {
	before := make(map[uuid.UUID]bool, len(g.combat.Groups))
	for _, group := range g.combat.Groups {
		before[group.AttackerID] = group.Blocked
	}
	return before
}

type combatBlockPair struct {
	attacker uuid.UUID
	blocker  uuid.UUID
	object   uuid.UUID
}

func (g *Game) blockerPairsBeforeDeclaration() map[combatBlockPair]bool {
	before := make(map[combatBlockPair]bool)
	for _, group := range g.combat.Groups {
		for i, blockerID := range group.BlockerIDs {
			if i < len(group.BlockerIncarnations) {
				before[combatBlockPair{group.AttackerID, blockerID, group.BlockerIncarnations[i]}] = true
			}
		}
	}
	return before
}

func (g *Game) fireNewBlockerPairs(before map[combatBlockPair]bool) {
	for _, group := range g.combat.Groups {
		attacker := g.FindPermanent(group.AttackerID)
		if attacker == nil {
			continue
		}
		for i, blockerID := range group.BlockerIDs {
			if i >= len(group.BlockerIncarnations) {
				continue
			}
			pair := combatBlockPair{group.AttackerID, blockerID, group.BlockerIncarnations[i]}
			blocker := g.FindPermanent(blockerID)
			if before[pair] || blocker == nil || blocker.incarnationID != pair.object {
				continue
			}
			g.FireEvent(GameEvent{Type: EvtCreatureBlockedBy, SourceID: attacker.ID(), TargetID: blockerID, PlayerID: attacker.ControllerID()})
		}
	}
}

func (g *Game) fireNewBlockerGroups(before map[combatBlockPair]bool) {
	type group struct {
		source     uuid.UUID
		blocking   bool
		controller uuid.UUID
		related    []uuid.UUID
	}
	var groups []group
	type groupKey struct {
		source   uuid.UUID
		blocking bool
	}
	indices := make(map[groupKey]int)
	add := func(source, other, controller uuid.UUID, blocking bool) {
		key := groupKey{source, blocking}
		index, ok := indices[key]
		if !ok {
			index = len(groups)
			indices[key] = index
			groups = append(groups, group{source: source, blocking: blocking, controller: controller})
		}
		groups[index].related = append(groups[index].related, other)
	}
	for _, combatGroup := range g.combat.Groups {
		attacker := g.FindPermanent(combatGroup.AttackerID)
		if attacker == nil || attacker.PhasedOut {
			continue
		}
		for i, blockerID := range combatGroup.BlockerIDs {
			if i >= len(combatGroup.BlockerIncarnations) {
				continue
			}
			pair := combatBlockPair{combatGroup.AttackerID, blockerID, combatGroup.BlockerIncarnations[i]}
			blocker := g.FindPermanent(blockerID)
			if before[pair] || blocker == nil || blocker.PhasedOut || blocker.incarnationID != pair.object {
				continue
			}
			add(attacker.ID(), blockerID, attacker.ControllerID(), false)
			add(blockerID, attacker.ID(), blocker.ControllerID(), true)
		}
	}
	for _, grouped := range groups {
		g.FireEvent(GameEvent{Type: EvtBlockerGroup, SourceID: grouped.source, PlayerID: grouped.controller, Flag: grouped.blocking, RelatedIDs: grouped.related})
	}
}

func (g *Game) fireDeclaredBlockerPairs(before map[combatBlockPair]bool) {
	seen := make(map[uuid.UUID]bool)
	for pair := range before {
		seen[pair.blocker] = true
	}
	for _, group := range g.combat.Groups {
		for i, blockerID := range group.BlockerIDs {
			if i >= len(group.BlockerIncarnations) {
				continue
			}
			pair := combatBlockPair{group.AttackerID, blockerID, group.BlockerIncarnations[i]}
			blocker := g.FindPermanent(blockerID)
			if before[pair] || blocker == nil || blocker.incarnationID != pair.object {
				continue
			}
			g.FireEvent(GameEvent{Type: EvtDeclaredBlocker, SourceID: blockerID, TargetID: group.AttackerID, PlayerID: blocker.ControllerID(), Flag: !seen[blockerID]})
			seen[blockerID] = true
		}
	}
}

func (g *Game) fireCreatureBecomesBlocked(attackerID uuid.UUID) {
	attacker := g.FindPermanent(attackerID)
	if attacker != nil {
		g.FireEvent(GameEvent{Type: EvtCreatureBecomesBlocked, SourceID: attackerID, PlayerID: attacker.ControllerID()})
	}
}

func (g *Game) fireNewlyBlockedAttackers(before map[uuid.UUID]bool) {
	for _, group := range g.combat.Groups {
		if !before[group.AttackerID] && group.Blocked && len(group.BlockerIDs) > 0 {
			g.fireCreatureBecomesBlocked(group.AttackerID)
		}
	}
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
	g.turns.RemoveSkipNextUntap(perm.ID())
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
	g.zones.RemovePermanent(perm.ID())

	if g.triggers.zoneBatch == nil {
		g.effects.Apply(g)
	}
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
			att.attachedIncarnation = uuid.Nil
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

	if isCreature {
		g.trackers.Turn.RecordCreatureDeath()
		g.recordCreatureDeath(controller)
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
}

// TapPermanent taps a permanent and fires the EvtTapped event.
func (g *Game) TapPermanent(perm *Permanent) {
	if perm == nil {
		return
	}
	perm = g.MutablePermanent(perm.ID())
	if perm == nil || perm.Tapped {
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
	if g.triggers.zoneBatch != nil && isBattlefieldLeaveEvent(evt) {
		return
	}
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
			g.queuePendingTrigger(&pendingTrigger{
				eventCaptures: captureDetectedTriggerEvent(ta, evt, g, g, sourceID),
				ability:       ta,
				event:         evt,
				sourceID:      sourceID,
				controller:    controller,
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

	if isCreature {
		g.trackers.Turn.RecordCreatureDeath()
		g.recordCreatureDeath(controller)
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
	g.playersLoseLife([]Player{p}, amount)
}

// playersLoseLife changes all life totals before events can observe them.
func (g *Game) playersLoseLife(players []Player, amount int) {
	if amount <= 0 {
		return
	}
	for _, p := range players {
		p.LoseLife(amount)
	}
	for _, p := range players {
		g.FireEvent(GameEvent{
			Type:     EvtLifeLost,
			PlayerID: p.PlayerID(),
			Amount:   amount,
		})
	}
}

// PlayerGainLife applies modifiers and replacements without event dispatch.
func (g *Game) PlayerGainLife(p Player, amount int) {
	g.applyLifeGain(p, amount)
}

func (g *Game) gainLifeWithEvent(p Player, amount int) {
	if gained := g.applyLifeGain(p, amount); gained > 0 {
		g.FireEvent(GameEvent{Type: EvtLifeGained, PlayerID: p.PlayerID(), Amount: gained})
	}
}

// applyLifeGain returns the amount gained after replacement processing.
func (g *Game) applyLifeGain(p Player, amount int) int {
	if amount <= 0 {
		return 0
	}
	amount = g.ApplyLifeGainModifiers(p.PlayerID(), amount)
	if amount <= 0 {
		return 0
	}
	action := NewLifeGainAction(uuid.Nil, p.PlayerID(), amount)
	result := g.effects.ApplyReplacements(action, g)
	if gain, ok := result.(*LifeGainAction); ok && gain.Amount() > 0 {
		p.GainLife(gain.Amount())
		return gain.Amount()
	}
	return 0
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
	g.zones.ExileCard(card, uuid.Nil)
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
	g.zones.ExileCard(card, exiledBy)
	g.recordCardPutIntoExile(card)
}

// ExileCardFaceDown moves a card to the exile zone face down. Only players in
// revealedTo may inspect the card's identity (via FindExiledCard / GetExile +
// ExiledCard.VisibleTo). For Gonti-style "exile face down" effects, only the
// exiling player (not the card's owner) sees the identity — pass that player's
// ID in revealedTo. Owners of face-down exiled cards do not automatically see
// the identity, matching Gonti's printed ruling.
func (g *Game) ExileCardFaceDown(card Card, exiledBy uuid.UUID, revealedTo ...uuid.UUID) {
	g.zones.ExileCardFaceDown(card, exiledBy, revealedTo...)
	g.recordCardPutIntoExile(card)
}

// RevealExiledCardTo grants the given player permission to see the identity of
// the face-down exiled card with the given ID. No-op if the card is face up
// (already public) or not in exile.
func (g *Game) RevealExiledCardTo(cardID, playerID uuid.UUID) {
	g.zones.RevealExiledCardTo(cardID, playerID)
}

// FindExiledCard finds an exiled card by its ID.
func (g *Game) FindExiledCard(cardID uuid.UUID) *ExiledCard {
	return g.zones.FindExiledCard(cardID)
}

// RemoveFromExile removes a card from exile by ID and returns it.
func (g *Game) RemoveFromExile(cardID uuid.UUID) (Card, bool) {
	return g.zones.RemoveFromExile(cardID)
}

// RemoveExiledCardBySource removes all exiled cards with the given ExiledBy ID
// and returns them. Used by Tawnos's Coffin and similar cards.
func (g *Game) RemoveExiledCardBySource(exiledBy uuid.UUID) []ExiledCard {
	return g.zones.RemoveExiledCardBySource(exiledBy)
}

func (g *Game) flushCombatDamageAggregator() {
	g.damage.FlushCombatDamageAggregator(g)
}

// CombatDamageSourcesThisStep returns the per-source combat damage breakdown
// for the given (controllerID, recipientPlayerID) pair during the current
// EvtCombatDamageDealt fire. Returns nil if no combat damage was dealt for
// this pair this step. The map is sourcePermanentID -> amount. Trigger
// condition closures listening to EvtCombatDamageDealt may use this to
// filter on source attributes (e.g., "non-Human creatures you control").
func (g *Game) CombatDamageSourcesThisStep(controllerID, recipientID uuid.UUID) map[uuid.UUID]int {
	return g.damage.CombatDamageSourcesThisStep(controllerID, recipientID)
}

// CounterSpellOnStack removes a spell from the stack by its source ID.
// The countered spell's card goes to its owner's graveyard.
//
// If the targeted spell is currently uncounterable — either because the
// card was registered with [WithUncounterable] or a static filter
// installed via [RegisterUncounterableStatic] matches — the counter has
// no effect: the spell stays on the stack.
func (g *Game) CounterSpellOnStack(spellID uuid.UUID) {
	g.counterStackObject(g.stack.FindBySourceID(spellID))
}

// CounterStackObject counters a spell by its stack object ID.
func (g *Game) CounterStackObject(id uuid.UUID) {
	obj := g.stack.FindByID(id)
	if obj == nil || obj.IsAbility || obj.Card == nil {
		return
	}
	g.counterStackObject(obj)
}

func (g *Game) counterStackObject(obj *StackObject) {
	if obj == nil || g.IsSpellUncounterable(obj) {
		return
	}
	g.stack.RemoveByID(obj.ID)
	if obj.Card != nil && !obj.IsCopy {
		if owner := g.GetPlayer(obj.Card.Owner()); owner != nil {
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

// PerformScry executes the scry action for N cards (CR 701.22).
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
		g.FireEvent(GameEvent{Type: EvtScry, PlayerID: p.PlayerID()})
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

// PerformSurveil implements surveil N (CR 701.25): the player looks at the top
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
		g.FireEvent(GameEvent{Type: EvtSurveil, PlayerID: p.PlayerID()})
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
	g.FireEvent(GameEvent{Type: EvtSurveil, PlayerID: p.PlayerID(), Amount: count})
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
	g.damage.DealDamageToPlayer(g, p, amount, sourceID)
}

// DealDamageToPlayerFromSource keeps the damage source identity after it leaves.
func (g *Game) DealDamageToPlayerFromSource(p Player, amount int, source DamageSource) {
	g.damage.DealDamageToPlayerFromSource(g, p, amount, source)
}

func (g *Game) executeDamageAction(action Action) int {
	switch a := action.(type) {
	case *DamageToPlayerAction:
		return g.damage.ExecuteDamageToPlayer(g, a)
	case *DamageToCreatureAction:
		return g.damage.ExecuteDamageToCreature(g, a)
	}
	return 0
}

// DealDamageToPermanent deals damage to a permanent, running it through the replacement pipeline.
func (g *Game) DealDamageToPermanent(perm *Permanent, amount int, sourceID uuid.UUID) {
	g.damage.DealDamageToPermanent(g, perm, amount, sourceID)
}

// DealDamageToPermanentFromSource keeps the damage source identity after it leaves.
func (g *Game) DealDamageToPermanentFromSource(perm *Permanent, amount int, source DamageSource) {
	g.damage.DealDamageToPermanentFromSource(g, perm, amount, source)
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
	return g.damage.HasDealtDamageToPlayer(sourceID, playerID)
}

// HasDealtDamageToPermanent reports whether the given source permanent has dealt damage to permID this game.
func (g *Game) HasDealtDamageToPermanent(sourceID, permID uuid.UUID) bool {
	return g.damage.HasDealtDamageToPermanent(sourceID, permID)
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
	src.attachedIncarnation = target.incarnationID
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
	seen := make(map[uuid.UUID]bool)
	for _, tid := range obj.Targets {
		if tid == uuid.Nil || seen[tid] {
			continue
		}
		seen[tid] = true
		g.trackers.Turn.RecordTimesTargeted(tid)
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
	return g.trackers.Turn.TimesTargeted(id)
}

// RegisterDelayedTrigger registers a one-shot delayed trigger that will fire
// when the specified event type occurs.
func (g *Game) RegisterDelayedTrigger(dt *DelayedTrigger) {
	if dt != nil {
		captured := *dt
		dt = &captured
		if p := g.FindPermanentIncludingPhased(dt.SourceID); !dt.sourceCaptured && p != nil {
			dt.sourceIncarnation = p.incarnationID
		}
	}
	g.triggers.RegisterDelayed(dt)
}

// DelayedTriggers returns all active delayed triggers.
func (g *Game) DelayedTriggers() []*DelayedTrigger {
	return g.triggers.Delayed()
}

// FireEvent dispatches an event and checks triggered abilities.
func (g *Game) FireEvent(evt GameEvent) {
	if g.damageBatch != nil {
		g.damageBatch.events = append(g.damageBatch.events, evt)
		return
	}
	if batch := g.triggers.zoneBatch; batch != nil && isBattlefieldLeaveEvent(&evt) {
		batch.events = append(batch.events, evt)
		return
	}
	g.fireEvent(evt, nil)
}

func (g *Game) fireEvent(evt GameEvent, before *Game) {
	defer g.CheckStateTriggers()
	g.recordPerTurnEvent(&evt)
	g.recordPerDuelEvent(&evt)
	observer := g
	if before != nil {
		observer = before
	}
	for _, perm := range observer.zones.battlefield {
		if perm.PhasedOut {
			continue
		}
		observer.syncAbilityContext(perm)
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
			if ta.CheckTrigger(&evt, observer) {
				g.queuePendingTrigger(&pendingTrigger{
					eventCaptures: captureDetectedTriggerEvent(ta, &evt, observer, g, perm.ID()),
					ability:       ta,
					event:         &evt,
					sourceID:      perm.ID(),
					controller:    perm.ControllerID(),
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
				g.queuePendingTrigger(&pendingTrigger{
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
			g.queuePendingTrigger(&pendingTrigger{
				ability:    gt,
				event:      &evt,
				sourceID:   obj.SourceID,
				controller: obj.Controller,
			})
		}
	}

	// Check delayed triggers (one-shot unless Persistent, removed after matching)
	var remaining []*DelayedTrigger
	for _, dt := range g.triggers.Delayed() {
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
			if (dt.MatchFlag && !evt.Flag) || (dt.matchFlagFalse && evt.Flag) {
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
			if dt.interveningCondition != nil && !dt.interveningCondition.checkState(g, StateContext{SourceID: dt.SourceID, ControllerID: dt.Controller}) {
				remaining = append(remaining, dt)
				continue
			}
			g.triggers.AddPending(&pendingTrigger{
				delayed:           dt,
				sourceID:          dt.SourceID,
				sourceIncarnation: dt.sourceIncarnation,
				controller:        dt.Controller,
				event:             &evt,
			})
			if dt.Persistent {
				remaining = append(remaining, dt)
			}
		} else {
			remaining = append(remaining, dt)
		}
	}
	g.triggers.SetDelayed(remaining)
	g.queueParadigmRecurringTriggers(&evt)
}

// CheckStateTriggers records state triggers that have no pending or active instance.
// A condition can trigger again after its ability leaves the stack (CR 603.8).
func (g *Game) CheckStateTriggers() {
	if g.triggers.zoneBatch != nil || g.triggers.stateChecksPaused {
		return
	}
	seen := make(map[stateTriggerKey]bool)
	for _, perm := range g.zones.battlefield {
		if perm.PhasedOut {
			continue
		}
		g.syncAbilityContext(perm)
		for _, a := range perm.RuntimeAbilities {
			ta, ok := UnwrapAbility(a).(TriggeredAbility)
			if !ok || !ta.IsStateTrigger() {
				continue
			}
			key := stateTriggerKey{sourceID: perm.ID(), abilityID: ta.AbilityID(), incarnationID: perm.incarnationID}
			seen[key] = true
			if g.stateTriggerActive(key) {
				g.triggers.ArmStateTrigger(key)
				continue
			}
			g.triggers.DisarmStateTrigger(key)
			if !ta.CheckTrigger(nil, g) {
				continue
			}
			g.triggers.ArmStateTrigger(key)
			g.queuePendingTrigger(&pendingTrigger{
				stateKey:   key,
				ability:    ta,
				sourceID:   perm.ID(),
				controller: perm.ControllerID(),
			})
		}
	}
	// Drop entries for sources no longer on the battlefield so a re-entered
	// instance starts fresh.
	g.triggers.ClearUnseenArmed(seen)
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
	pending := g.triggers.Pending()
	if len(pending) > 1 {
		activeID := g.ActivePlayerObj().PlayerID()
		active := make([]*pendingTrigger, 0, len(pending))
		nonActive := make([]*pendingTrigger, 0, len(pending))
		for _, pt := range pending {
			if pt.controller == activeID {
				active = append(active, pt)
			} else {
				nonActive = append(nonActive, pt)
			}
		}
		reverseTriggers(active)
		reverseTriggers(nonActive)
		active = append(active, nonActive...)
		g.triggers.SetPending(active)
	}
	for _, pt := range g.triggers.Pending() {
		targetSource := g.FindCardAnywhere(pt.sourceID)
		obj := &StackObject{
			stateKey:          pt.stateKey,
			sourceIncarnation: pt.sourceIncarnation,
			ID:                uuid.New(),
			Controller:        pt.controller,
			SourceID:          pt.sourceID,
			IsAbility:         true,
		}
		if dt := pt.delayed; dt != nil && len(dt.modes) == 0 {
			obj.Effects = append([]Effect(nil), dt.Effects...)
			obj.Targets = []uuid.UUID{dt.TargetID}
			if len(dt.targets) > 0 {
				var legal bool
				obj.Targets, obj.TargetSpecs, legal = g.chooseTriggerTargets(pt, dt.targets)
				if !legal {
					continue
				}
				obj.TargetSource = targetSource
			}
			obj.EventAmount = pt.event.Amount
			obj.EventSourceID = pt.event.SourceID
			captureTriggerEventEffects(pt, obj)
			g.pushStack(obj)
			continue
		}
		// CR 603.3c permits only modes with available required targets.
		var modes []Mode
		if dt := pt.delayed; dt != nil {
			modes = dt.modes
			obj.EventAmount = pt.event.Amount
			obj.EventSourceID = pt.event.SourceID
		} else if gt, ok := pt.ability.(*GenericTriggered); ok {
			modes = gt.Modes()
		}
		if len(modes) > 0 {
			ctrl := g.GetPlayer(pt.controller)
			var labels []string
			var indices []int
			for i, mode := range modes {
				legal := true
				for _, target := range mode.Targets {
					if len(uniqueTargetCandidates(target.Possible(pt.controller, targetSource, g))) < target.Min() {
						legal = false
						break
					}
				}
				if legal {
					labels = append(labels, mode.Label)
					indices = append(indices, i)
				}
			}
			if len(indices) == 0 || ctrl == nil {
				continue
			}
			reason := "modal trigger"
			if targetSource != nil {
				reason = targetSource.Name()
			}
			choice := ctrl.ChooseMode(labels, reason)
			if choice < 0 || choice >= len(indices) {
				continue
			}
			idx := indices[choice]
			chosen := modes[idx]
			obj.Effects = append(obj.Effects, chosen.Effects...)
			obj.ModeChoice = idx
			if len(chosen.Targets) > 0 {
				var legal bool
				obj.Targets, obj.TargetSpecs, legal = g.chooseTriggerTargets(pt, chosen.Targets)
				if !legal {
					continue
				}
				obj.TargetSource = targetSource
			}
			captureTriggerEventEffects(pt, obj)
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
			var legal bool
			obj.Targets, obj.TargetSpecs, legal = g.chooseTriggerTargets(pt, declared)
			if !legal {
				continue
			}
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
			obj.TargetSource = targetSource
			captureTriggerEventEffects(pt, obj)
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
				case EvtScry, EvtSurveil:
					obj.EventAmount = pt.event.Amount
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
				// Typed event subjects are not targets (CR 115.1d).
				if gt.eventDefinition != nil {
					obj.Targets = nil
				}
			}
		}
		captureTriggerEventEffects(pt, obj)
		g.pushStack(obj)
	}
	g.triggers.ClearPending()
}

// chooseTriggerTargets returns false when a required target group cannot be chosen.
// CR 603.3d removes such a trigger before a player receives priority.
func (g *Game) chooseTriggerTargets(pt *pendingTrigger, declared []Target) ([]uuid.UUID, []Target, bool) {
	controller := g.GetPlayer(pt.controller)
	sourceCard := g.FindCardAnywhere(pt.sourceID)
	var out []uuid.UUID
	var aligned []Target
	for _, t := range declared {
		if pair, ok := t.(*randomActivePlayerExchangePairTarget); ok {
			chosen := pair.chooseForTrigger(pt.controller, sourceCard, g)
			if len(chosen) < t.Min() {
				return nil, nil, false
			}
			out = append(out, chosen...)
			for range chosen {
				aligned = append(aligned, t)
			}
			continue
		}
		if random, ok := t.(*randomTarget); ok {
			chosen := g.chooseRandomTargets(pt.controller, sourceCard, random, 0)
			if len(chosen) < t.Min() {
				return nil, nil, false
			}
			out = append(out, chosen...)
			for range chosen {
				aligned = append(aligned, t)
			}
			continue
		}
		t.Reset()
		possible := t.Possible(pt.controller, sourceCard, g)
		if len(possible) < t.Min() {
			return nil, nil, false
		}
		if len(possible) == 0 {
			continue
		}
		chooser := controller
		if isOpponentChosenTarget(t) {
			if opp := g.GetOpponent(pt.controller); opp != nil {
				chooser = opp
			}
		}
		var chosen []uuid.UUID
		if chooser != nil {
			chosen = chooser.ChooseTargets(possible, t.Min(), t.Max(), g)
		}
		if len(chosen) == 0 && t.Min() > 0 {
			chosen = possible[:t.Min()]
		}
		if _, typed := t.(*targetDeclaration); typed {
			if len(chosen) < t.Min() || len(chosen) > t.Max() {
				return nil, nil, false
			}
			seen := make(map[uuid.UUID]bool, len(chosen))
			for _, id := range chosen {
				if seen[id] || !slices.Contains(possible, id) {
					return nil, nil, false
				}
				seen[id] = true
			}
		}
		if err := t.Choose(pt.controller, sourceCard, g, chosen); err != nil {
			return nil, nil, false
		}
		out = append(out, chosen...)
		for range chosen {
			aligned = append(aligned, t)
		}
	}
	return out, aligned, true
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
	if g.stack.FindByID(id) != nil {
		return ZoneStack
	}
	if g.stack.FindBySourceID(id) != nil {
		return ZoneStack
	}
	for _, ec := range g.zones.exile {
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

func (g *Game) graveyardVisit(id uuid.UUID) uuid.UUID {
	for _, player := range g.players {
		for _, card := range player.Graveyard() {
			if card.ID() == id {
				if visited, ok := player.(interface{ graveyardVisit(uuid.UUID) uuid.UUID }); ok {
					return visited.graveyardVisit(id)
				}
			}
		}
	}
	return uuid.Nil
}

// Used to track game-specific state before pushing object onto stack
func (g *Game) pushStack(obj *StackObject) {
	g.chooseRandomCounterDistribution(obj)
	// Track target zone when targeted, needed for CR 608.2b
	for _, id := range obj.Targets {
		if _, captured := obj.targetIncarnations[id]; captured {
			continue
		}
		var incarnation uuid.UUID
		if p := g.FindPermanent(id); p != nil {
			incarnation = p.incarnationID
		} else {
			incarnation = g.graveyardVisit(id)
		}
		if incarnation != uuid.Nil {
			if obj.targetIncarnations == nil {
				obj.targetIncarnations = make(map[uuid.UUID]uuid.UUID)
			}
			obj.targetIncarnations[id] = incarnation
		}
	}
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
	g.resolveStackObject(obj)
}

func (g *Game) resolveStackObject(obj *StackObject) (produced int, colors []Color) {
	previousMana, previousTracking := g.resolution.manaProduced, g.resolution.manaTracking
	previousColors := g.resolution.manaColors
	g.resolution.manaColors = nil
	g.resolution.manaProduced, g.resolution.manaTracking = 0, true
	defer func() {
		produced = g.resolution.manaProduced
		colors = g.resolution.manaColors
		g.resolution.manaColors = previousColors
		g.resolution.manaProduced, g.resolution.manaTracking = previousMana, previousTracking
	}()
	oldIncarnation, oldCaptured := g.resolution.sourceIncarnation, g.resolution.sourceCaptured
	g.resolution.sourceIncarnation, g.resolution.sourceCaptured = obj.sourceIncarnation, true
	defer func() { g.resolution.sourceIncarnation, g.resolution.sourceCaptured = oldIncarnation, oldCaptured }()
	previousState := g.triggers.resolvingState
	g.triggers.resolvingState = obj.stateKey
	defer g.CheckStateBasedActions()
	defer func() {
		g.triggers.resolvingState = previousState
		g.CheckStateTriggers()
	}()

	// Check for fizzle: if the spell/ability has targets but all are now illegal,
	// it fails to resolve (MTG rule 608.2b)

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
			incarnationLegal := true
			if incarnation, ok := obj.targetIncarnations[t]; ok {
				if obj.TargetZones[t] == ZoneGraveyard {
					incarnationLegal = g.graveyardVisit(t) == incarnation
				} else {
					p := g.FindPermanent(t)
					incarnationLegal = p != nil && p.incarnationID == incarnation
				}
			}
			if incarnationLegal && g.isTargetStillLegal(t, sourceCard, obj.Controller, obj.TargetZones[t], spec) {
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
		_ = applyEffectWithCaptures(g, eff, obj.SourceID, obj.Controller, resolvedTargets, obj.capturedNumbers, obj.capturedPermanents)
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
	return
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
	return g.castSpellByName(playerID, name, targets, nil, xValues...)
}

// CastSpellByNameWithTargetGroups casts a spell with one choice slice per declared target group.
func (g *Game) CastSpellByNameWithTargetGroups(playerID uuid.UUID, name string, groups [][]uuid.UUID, xValues ...int) error {
	return g.castSpellByName(playerID, name, nil, groups, xValues...)
}

func (g *Game) castSpellByName(playerID uuid.UUID, name string, targets []uuid.UUID, groups [][]uuid.UUID, xValues ...int) error {
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

	if err := g.checkSpellStep(card); err != nil {
		return err
	}

	// CR 307.1, 302.1, 303.1, 301.1 — sorcery-speed timing. Any spell that
	// is not an instant may be cast only during its controller's main phase,
	// when the stack is empty, and when that player is the active player
	// (i.e. could cast a sorcery).
	// CR 702.8 — Flash: "You may cast this spell any time you could cast an
	// instant." A card with Flash bypasses the sorcery-speed gate entirely.
	if !card.HasType(TypeInstant) && !cardHasKeyword(card, Flash) && !g.effects.Rules.HasFlashGrant(playerID, card) {
		if !g.turns.Step().IsMainPhase() {
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
	grouped := false
	for _, ability := range card.Abilities() {
		if action, ok := ability.(*ActionDefinition); ok && action.kind == ActionSpell && action.groupedTargets {
			grouped = true
			break
		}
	}
	if grouped != (groups != nil) {
		return fmt.Errorf("spell target group announcement does not match declaration")
	}
	var aligned []Target
	if grouped {
		var err error
		targets, aligned, err = g.prepareSpellTargetGroups(playerID, card, groups, xValue)
		if err != nil {
			return err
		}
	}
	selection, err := g.prepareModalSpellSelection(p, card, targets, xValue, "")
	if err != nil {
		return err
	}
	if selection != nil {
		targets = selection.targets
	} else if !grouped {
		targets = g.acquireRandomTargets(playerID, card, card.CastTargets(), targets, xValue)
		targets = g.acquireOpponentChosenTargets(playerID, card, card.CastTargets(), targets)
		if err := g.validateActionTargets(playerID, card, card.CastTargets(), targets, xValue, "spell"); err != nil {
			return err
		}
		if err := g.validateVariableSpellTargets(playerID, card, targets, xValue); err != nil {
			return err
		}
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
		ModalSelection:  selection,
		Card:            card,
		Controller:      playerID,
		Targets:         targets,
		TargetSpecs:     aligned,
		XValue:          xValue,
		CastZone:        ZoneHand,
		SnapshotCast:    true,
		CapturedNumbers: payment.payment.paidNumbers,
	})
	return err
}

// CheckStateBasedActions checks and processes state-based actions.
func (g *Game) CheckStateBasedActions() {
	g.checkStateBasedActions()
}

func (g *Game) checkStateBasedActions() bool {
	performed := false
	for {
		actions := false

		var toDestroy, toGraveyard []*Permanent
		for _, p := range g.zones.battlefield {
			if p.HasType(TypeCreature) && p.CurrentToughness(g) <= 0 || p.HasType(TypePlaneswalker) && int(p.Counters[Loyalty]) <= 0 {
				toGraveyard = append(toGraveyard, p)
			} else if p.HasType(TypeCreature) && p.LethalDamage(g) && !p.HasKeyword(Indestructible) {
				toDestroy = append(toDestroy, p)
			}
		}
		if len(toDestroy)+len(toGraveyard) > 0 {
			actions = true
			g.withZoneChangeBatch(func() {
				g.DestroyPermanents(toDestroy)
				for _, p := range toGraveyard {
					if current := g.FindPermanent(p.ID()); current != nil && current.incarnationID == p.incarnationID {
						g.PutPermanentIntoGraveyard(current)
					}
				}
			})
		}

		// MTG rule 704.5q: +1/+1 and -1/-1 counter annihilation
		for _, p := range g.zones.battlefield {
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
		for _, p := range g.zones.battlefield {
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
		for _, p := range g.zones.battlefield {
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
					p.attachedIncarnation = uuid.Nil
					actions = true
				}
			}
		}

		// MTG rule 704.5j: Legend rule — if a player controls two or more legendary
		// permanents with the same name, they choose one and sacrifice the rest.
		var legendCounts map[uuid.UUID]map[string][]*Permanent // controller -> name -> perms
		for _, p := range g.zones.battlefield {
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
		for _, p := range g.zones.battlefield {
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
		performed = true
	}

	// Put any pending triggers on the stack
	g.PutTriggersOnStack()
	return performed
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

// SkipNextUntap prevents untapping during the next untap step of the current
// controller. Repeated applications expire during that same step.
func (g *Game) SkipNextUntap(permanentID uuid.UUID) {
	if g.FindPermanent(permanentID) == nil {
		return
	}
	g.turns.SkipNextUntap(permanentID)
}

func (g *Game) untapPermanentDuringUntapStep(p *Permanent) bool {
	if p == nil || !p.Tapped {
		return false
	}
	return g.UntapPermanent(p)
}

func (g *Game) doUntap() {
	active := g.ActivePlayerObj()
	capturedUntap := g.turns.consumeCapturedUntap(active.PlayerID())
	// Island Sanctuary: clear protection at the start of the player's turn
	g.effects.Rules.ClearSanctuary(active.PlayerID())

	// Snapshot the active player's untapped land count before the untap loop
	// runs (CR 502.1 happens at the very start of the turn). Power Surge and
	// similar effects read this at upkeep.
	count := 0
	for _, p := range g.zones.battlefield {
		if p.ControllerID() == active.PlayerID() && p.HasType(TypeLand) && !p.Tapped {
			count++
		}
	}
	g.trackers.Turn.SetUntappedLandsAtTurnStart(active.PlayerID(), count)

	landUntapLimit := g.effects.Rules.LandUntapMax
	landsUntapped := 0
	artifactUntapLimit := g.effects.Rules.ArtifactUntapMax
	artifactsUntapped := 0
	creatureUntapLimit := g.effects.Rules.CreatureUntapMax
	creaturesUntapped := 0

	for _, p := range g.zones.battlefield {
		if p.ControllerID() == active.PlayerID() {
			if g.turns.ConsumeSkipNextUntap(p.ID()) || capturedUntap[p.ID()] || p.HasAttr(AttrDoesNotUntap) {
				// The restriction expires even when no untap is possible.
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
	g.trackers.Turn.ResetLandsPlayed()
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
	declaredCount := 0
	for _, id := range attackerIDs {
		declared[id] = true
	}
	for _, p := range g.zones.battlefield {
		if p.ControllerID() == active.PlayerID() && p.HasAttr(AttrMustAttack) && !declared[p.ID()] {
			if p.CanDeclareAsAttacker(g) {
				attackerIDs = append(attackerIDs, p.ID())
			}
		}
	}

	for _, id := range attackerIDs {
		if g.combat.IsAttacking(id) {
			continue
		}
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

		g.combat.AddAttackerIncarnation(id, atk.incarnationID, defender.PlayerID())
		declaredCount++
		g.trackers.Turn.RecordAttacked(id)
		g.trackers.Turn.RecordAttackedIncarnation(id, atk.incarnationID)
		g.FireEvent(GameEvent{
			Type:     EvtDeclaredAttacker,
			SourceID: id,
			PlayerID: active.PlayerID(),
			TargetID: defender.PlayerID(),
		})
	}

	// Form attacking bands if the player has scripted them.
	if bf, ok := active.(BandFormer); ok {
		for _, band := range bf.GetBandFormations(g.turns.Turn(), g) {
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
	if declaredCount > 0 {
		g.FireEvent(GameEvent{
			Type:     EvtAttackersDeclared,
			PlayerID: active.PlayerID(),
			Amount:   declaredCount,
		})
	}

	g.ResolveStack()
}

// AttackedDuringLastTurn reports whether the given creature attacked during playerID's most recent turn.
func (g *Game) AttackedDuringLastTurn(playerID, permID uuid.UUID) bool {
	if m := g.trackers.Turn.AttackedLastTurn(playerID); m != nil {
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
	blockedBefore := g.blockedAttackersBeforeDeclaration()
	pairsBefore := g.blockerPairsBeforeDeclaration()
	assignments := nonActive.DeclareBlockers(g)
	if assignments == nil {
		// Even with no scripted blockers, we may need to enforce
		// CR 509.1c "must be blocked if able" before firing the event.
		g.enforceMustBeBlockedIfAble(nonActive.PlayerID())
		g.combat.SnapshotBlockedAlone()
		g.fireDeclaredBlockerPairs(pairsBefore)
		g.fireNewBlockerPairs(pairsBefore)
		g.fireNewBlockerGroups(pairsBefore)
		g.fireNewlyBlockedAttackers(blockedBefore)
		g.fireUnblockedAttackers()
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
		blockerCount[ba.BlockerID]++
		g.combat.AddBlockerIncarnation(ba.BlockerID, blocker.incarnationID, attackerID)
		g.trackers.Turn.RecordBlocked(ba.BlockerID, attackerID)
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
	g.fireDeclaredBlockerPairs(pairsBefore)
	g.fireNewBlockerPairs(pairsBefore)
	g.fireNewBlockerGroups(pairsBefore)
	g.fireNewlyBlockedAttackers(blockedBefore)
	g.fireUnblockedAttackers()

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
		group.BlockerIncarnations = nil
		group.Blocked = false
		for _, bid := range dropped {
			// Also clean up the per-turn blocked tracking for this attacker.
			g.trackers.Turn.RemoveBlockedAttacker(bid, group.AttackerID)
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
		for _, p := range g.zones.battlefield {
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
			g.combat.AddBlockerIncarnation(b.ID(), b.incarnationID, atk.ID())
			g.trackers.Turn.RecordBlocked(b.ID(), atk.ID())
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
// It returns true if state-based actions or triggers require another cleanup step.
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
	// CR 514.2 includes phased-out permanents in damage removal.
	for _, p := range g.zones.battlefield {
		if p.Damage == 0 {
			continue
		}
		p = g.MutablePermanentIncludingPhased(p.ID())
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
	g.zones.ClearExileInstead()
	g.damage.expireEndOfTurnEffects()
	g.effects.Rules.ClearEndOfTurn()
	// Clear non-persistent delayed triggers (they only last "this turn")
	g.triggers.ClearEndOfTurn()
	// CR 514.3a keeps turn history available during cleanup priority.
	performed := g.checkStateBasedActions()
	if performed || !g.stack.IsEmpty() {
		return true
	}
	// Pending cleanup abilities can still need departure information.
	g.zones.ClearLKI()
	g.trackers.Turn.ResetForNewTurn(active.PlayerID())
	g.damage.ClearEndOfTurn()
	// Clear last-drawn-card tracking for all players
	for _, p := range g.players {
		p.ClearLastDrawnCard()
	}
	for _, p := range g.zones.battlefield {
		p = g.MutablePermanentIncludingPhased(p.ID())
		// Clear activation tracking (Charge counters used for per-turn counts)
		p.Counters[Charge] = 0
		// Cloned games can share abilities before this reset.
		p.RuntimeAbilities = clonePaymentAbilities(p.RuntimeAbilities)
		for _, a := range p.RuntimeAbilities {
			if aa, ok := UnwrapAbility(a).(*SimpleActivatedAbility); ok {
				aa.ResetActivation()
			}
		}
	}
	return false
}

// Run executes the game until the stop condition.
func (g *Game) Run(stopTurn int, stopStep PhaseStep, maxTurns int) {
	for g.turns.Turn() <= maxTurns {
		// Turn-skip (CR 500.11): if the active player's next turn is
		// marked as skipped, consume the skip and advance past this turn
		// without running any steps.
		activeID := g.ActivePlayerObj().PlayerID()
		if g.turns.ConsumeTurnSkip(activeID) {
			// Fall through to the extra-turn / next-player logic below.
		} else if g.RunTurn(stopTurn, stopStep) {
			return
		}
		// Check for extra turns
		if extraPlayerID, ok := g.turns.PopExtraTurn(); ok {
			// Find the player index
			for i, p := range g.players {
				if p.PlayerID() == extraPlayerID {
					g.turns.SetActivePlayerIndex(i)
					break
				}
			}
		} else {
			// Next turn: swap active player
			g.turns.SetActivePlayerIndex((g.turns.ActivePlayerIndex() + 1) % len(g.players))
		}
		g.turns.IncrementTurn()
	}
}

// MaxLandPlays returns the maximum number of lands that can be played this turn.
func (g *Game) MaxLandPlays() int {
	limit := 1
	if g.effects.Rules.UnlimitedLandPlays {
		limit = 999
	}
	activeID := g.ActivePlayerObj().PlayerID()
	limit += g.trackers.Turn.ExtraLandPlays(activeID)
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
	g.trackers.Turn.AddExtraLandPlays(playerID, n)
}

// ExtraLandPlaysGrantedThisTurn returns the cumulative additional land-play
// allowance granted to the player this turn (not including the base 1).
func (g *Game) ExtraLandPlaysGrantedThisTurn(playerID uuid.UUID) int {
	return g.trackers.Turn.ExtraLandPlays(playerID)
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
	if !g.turns.Step().IsMainPhase() {
		return fmt.Errorf("can only play lands during a main phase")
	}
	if g.ActivePlayerObj().PlayerID() != playerID {
		return fmt.Errorf("only the active player can play a land")
	}
	if g.trackers.Turn.LandsPlayed() >= g.MaxLandPlays() {
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
	g.trackers.Turn.RecordLandPlayed()

	g.FireEvent(GameEvent{
		Type:     EvtLandPlayed,
		SourceID: card.ID(),
		PlayerID: playerID,
		Amount:   g.trackers.Turn.LandsPlayed(), // which land number this was
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
	return g.mana.TapForMana(g, playerID, permanentID, Colorless)
}

func (g *Game) TapForManaWithColor(playerID, permanentID uuid.UUID, preferredColor Color) error {
	return g.mana.TapForMana(g, playerID, permanentID, preferredColor)
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
	if len(a.costs) != 1 || len(a.targets) != 0 {
		return nil
	}
	if _, ok := a.costs[0].(*tap); !ok {
		return nil
	}
	productions, ok := activatedManaEffects(a.effects)
	if !ok {
		return nil
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

// AutoTapForCost taps untapped lands/mana sources to pay a mana cost,
// accounting for mana already in the player's pool.
func (g *Game) AutoTapForCost(playerID uuid.UUID, mc ManaCost) error {
	return g.mana.AutoTapForCost(g, playerID, mc, AutoTapHint{}, nil)
}

// AutoTapForCostWithHint taps untapped lands/mana sources to pay a mana cost with a hint.
func (g *Game) AutoTapForCostWithHint(playerID uuid.UUID, mc ManaCost, hint AutoTapHint) error {
	return g.mana.AutoTapForCost(g, playerID, mc, hint, nil)
}

func (g *Game) HypotheticalMana(playerID uuid.UUID) int {
	return g.mana.HypotheticalMana(g, playerID)
}

func (g *Game) CanAfford(playerID uuid.UUID, mc ManaCost, spellCtx *SpellPaymentContext) bool {
	return g.mana.CanAfford(g, playerID, mc, spellCtx)
}

func (g *Game) MaxXValue(playerID uuid.UUID, mc ManaCost, spellCtx *SpellPaymentContext) int {
	return g.mana.MaxXValue(g, playerID, mc, spellCtx)
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
	isMainPhase := g.turns.Step().IsMainPhase()
	isActive := g.ActivePlayerObj().PlayerID() == playerID

	var castable []Card
	for _, card := range p.Hand() {
		if card.HasType(TypeLand) {
			continue
		}
		if g.checkSpellStep(card) != nil {
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
	if !g.turns.Step().IsMainPhase() {
		return nil
	}
	if g.ActivePlayerObj().PlayerID() != playerID {
		return nil
	}
	if g.trackers.Turn.LandsPlayed() >= g.MaxLandPlays() {
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
	for _, perm := range g.zones.battlefield {
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
			// Automatic payment handles these mana abilities.
			if _, isMana := inner.(*ManaAbility); isMana || abilityManaProductions(inner, g, perm.ID()) != nil {
				continue
			}
			if !aa.CanActivate(playerID, g) {
				continue
			}
			if aa.SorcerySpeed() && !g.turns.Step().IsMainPhase() {
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
	return g.activateAbilityByIndex(playerID, permanentID, abilityIndex, targets, nil)
}

// ActivateAbilityByIndexWithTargetGroups announces one choice list per target declaration.
func (g *Game) ActivateAbilityByIndexWithTargetGroups(playerID, permanentID uuid.UUID, abilityIndex int, groups [][]uuid.UUID) error {
	if groups == nil {
		return fmt.Errorf("activation target groups are required")
	}
	return g.activateAbilityByIndex(playerID, permanentID, abilityIndex, nil, groups)
}

func (g *Game) activateAbilityByIndex(playerID, permanentID uuid.UUID, abilityIndex int, targets []uuid.UUID, groups [][]uuid.UUID) error {
	perm := g.FindPermanent(permanentID)
	if perm == nil {
		return ErrPermanentNotFound
	}
	if abilityIndex < 0 || abilityIndex >= len(perm.RuntimeAbilities) {
		return fmt.Errorf("invalid ability index")
	}

	inner := UnwrapAbility(perm.RuntimeAbilities[abilityIndex])
	if groups != nil {
		if _, ok := inner.(*ManaAbility); ok {
			return fmt.Errorf("mana ability does not declare target groups")
		}
	}

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
			produced, err = g.mana.AddManaFromAbility(g, ma, p, perm)
			if err != nil {
				return err
			}
		}
		g.FireEvent(GameEvent{
			Type:     EvtAbilityActivated,
			SourceID: perm.ID(),
			PlayerID: playerID,
		})
		g.mana.FireTappedForMana(g, perm.ID(), playerID, produced)
		return nil
	}

	aa, ok := inner.(ActivatedAbility)
	if !ok {
		return fmt.Errorf("not an activated ability")
	}
	if def, ok := inner.(*ActionDefinition); ok && def.Kind() != ActionActivated {
		return fmt.Errorf("not an activated ability")
	}
	grouped := false
	if def, ok := inner.(*ActionDefinition); ok {
		grouped = def.groupedTargets
	}
	if grouped != (groups != nil) {
		return fmt.Errorf("activation target group announcement does not match declaration")
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
	if perm.HasAttr(AttrCantActivateNonManaAbilities) && !canProduceMana(aa.Effects(), aa.Targets()) {
		return fmt.Errorf("non-mana activated abilities of %s are prevented", perm.Name())
	}
	if !aa.CanActivate(playerID, g) {
		return fmt.Errorf("cannot activate ability")
	}

	actionTargets := aa.Targets()
	actionEffects := aa.Effects()
	var alignedTargets []Target
	modeChoice := -1
	var modalTargets [][]uuid.UUID
	if modal, ok := aa.(interface{ Modes() []Mode }); ok && len(modal.Modes()) > 0 {
		modes := modal.Modes()
		var available []Mode
		var indices []int
		for i, mode := range modes {
			legal := true
			for _, target := range mode.Targets {
				if len(uniqueTargetCandidates(target.Possible(playerID, perm.Card, g))) < target.Min() {
					legal = false
					break
				}
			}
			if legal {
				available = append(available, mode)
				indices = append(indices, i)
			}
		}
		if len(available) == 0 {
			return fmt.Errorf("no legal activation mode")
		}
		chooser := g.GetPlayer(playerID)
		if effectChooser, ok := chooser.(EffectModeChooser); ok {
			modeChoice = effectChooser.ChooseModeWithEffects(available, perm.Card.Name(), g)
		} else if chooser != nil {
			labels := make([]string, len(available))
			for i, mode := range available {
				labels[i] = mode.Label
			}
			modeChoice = chooser.ChooseMode(labels, perm.Card.Name())
		}
		if modeChoice < 0 || modeChoice >= len(available) {
			return fmt.Errorf("invalid activation mode choice")
		}
		modeChoice = indices[modeChoice]
		mode := modes[modeChoice]
		actionTargets = mode.Targets
		actionEffects = mode.Effects
		targets = g.promptTargetsForList(playerID, perm.Card, actionTargets)
		modalTargets = make([][]uuid.UUID, len(modes))
		modalTargets[modeChoice] = append([]uuid.UUID(nil), targets...)
	}

	if grouped {
		var err error
		targets, alignedTargets, err = g.prepareTargetGroups(playerID, perm.Card, actionTargets, groups, g.resolution.X(), "ability target group")
		if err != nil {
			return err
		}
	}
	// When the caller supplied no targets but the ability's targeting is
	// forced (e.g. "you", or a single legal target), fill them in here so the
	// ability doesn't fizzle. The interactive layer relies on this to skip
	// prompting for targets that offer no real choice.
	if !grouped && len(targets) == 0 {
		if forced := ForcedActivationTargets(playerID, perm.Card, actionTargets, g); forced != nil {
			targets = forced
		}
	}
	if !grouped {
		targets = g.acquireRandomTargets(playerID, perm.Card, actionTargets, targets, g.resolution.X())
		targets = g.acquireOpponentChosenTargets(playerID, perm.Card, actionTargets, targets)
	}

	if !grouped {
		if err := g.validateActionTargets(playerID, perm.Card, actionTargets, targets, g.resolution.X(), "ability"); err != nil {
			return err
		}
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
	var capturedNumbers map[*NumberBinding]int
	var capturedPermanents map[*PermanentBinding]capturedPermanent
	if definition, ok := aa.(*ActionDefinition); ok {
		captureSpecs := actionTargets
		if grouped {
			captureSpecs = alignedTargets
		}
		capturedNumbers, capturedPermanents = definition.captureValues(g, perm, playerID, captureSpecs, targets)
	}
	if err := payment.Commit(); err != nil {
		return err
	}
	if len(payment.paidNumbers) > 0 {
		if capturedNumbers == nil {
			capturedNumbers = make(map[*NumberBinding]int)
		}
		maps.Copy(capturedNumbers, payment.paidNumbers)
	}

	// Mark once-per-turn abilities as used
	if saa, ok := aa.(*SimpleActivatedAbility); ok {
		saa.markActivatedInGame(g)
	}

	obj := newStackObject(playerID, perm.ID(), nil, actionEffects, targets, g.resolution.X(), true)
	obj.capturedNumbers = capturedNumbers
	obj.capturedPermanents = capturedPermanents
	obj.TargetSpecs = expandTargetSpecs(actionTargets, targets, g.resolution.X())
	if grouped {
		obj.TargetSpecs = append([]Target(nil), alignedTargets...)
	}
	obj.sourceIncarnation = perm.incarnationID
	obj.TargetSource = perm.Card
	obj.ModeChoice = modeChoice
	obj.ModalTargets = modalTargets

	if canProduceMana(actionEffects, actionTargets) {
		g.FireEvent(GameEvent{
			Type:     EvtAbilityActivated,
			SourceID: perm.ID(),
			PlayerID: playerID,
			Flag:     hasTapCost,
		})
		produced, colors := g.resolveStackObject(obj)
		if hasTapCost && produced > 0 {
			g.mana.ApplyManaBonuses(g, perm, colors, g.GetPlayer(playerID))
			g.mana.FireTappedForMana(g, perm.ID(), playerID, produced)
		}
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

func (g *Game) stateTriggerActive(key stateTriggerKey) bool {
	if g.triggers.resolvingState == key {
		return true
	}
	for _, pt := range g.triggers.Pending() {
		if pt.stateKey == key {
			return true
		}
	}
	for _, obj := range g.stack.Objects() {
		if obj.stateKey == key {
			return true
		}
	}
	return false
}

func (g *Game) queuePendingTrigger(pt *pendingTrigger) {
	if p := g.FindPermanent(pt.sourceID); p != nil {
		pt.sourceIncarnation = p.incarnationID
	} else if lki := g.LKI(pt.sourceID); lki != nil && lki.Snapshot != nil {
		pt.sourceIncarnation = lki.Snapshot.incarnationID
	}
	g.triggers.AddPending(pt)
}
