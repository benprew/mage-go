package mage

// Custom-keyword support for the Secrets of Strixhaven set.
//
// The set introduces seven set-specific keywords that don't appear in the
// engine's built-in keyword catalog. This file collects engine-level helpers
// each card-side card can wire up via small CardOptions:
//
//	Prepared    — ETB grants a sorcery-speed "cast a copy of [spell] for free"
//	              ability that unprepares the creature on use.
//	Repartee    — Triggered ability shorthand: "Whenever you cast an instant
//	              or sorcery spell that targets a creature, …".
//	Paradigm    — Self-exile-on-resolve plus a recurring main-phase free copy
//	              from exile, gated on "after you first resolve a spell with
//	              this name".
//	Opus        — Cast-trigger shorthand whose effect closure receives the
//	              total amount of mana the controller spent to cast that
//	              spell (CR 118.9 — colored + generic together).
//	Increment   — Cast-trigger shorthand: "if mana spent > self.power or
//	              self.toughness, put a +1/+1 counter on this".
//	Infusion    — Convenience flag for "if you gained life this turn" — the
//	              effect closure receives the per-turn life-gain count.
//	Grandeur    — Activated ability whose cost includes "discard another
//	              card with the same name as this".
//
// All helpers are safe to combine with normal CardOptions / triggers; they
// produce regular Ability / Cost / Effect values.

import (
	"fmt"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

// =============================================================================
// Prepared
// =============================================================================

// IsPrepared reports whether the permanent currently has the AttrPrepared
// flag set. A "prepared" creature is one whose Prepared spell has not yet
// been cast as a copy. See WithPreparedSpell.
func (g *Game) IsPrepared(permID uuid.UUID) bool {
	perm := g.FindPermanent(permID)
	if perm == nil {
		return false
	}
	return perm.HasAttr(AttrPrepared)
}

// SetPrepared toggles the AttrPrepared flag on a permanent. Used to wire the
// "enters prepared" ETB and the "doing so unprepares it" cleanup.
func (g *Game) SetPrepared(permID uuid.UUID, prepared bool) {
	perm := g.FindPermanent(permID)
	if perm == nil {
		return
	}
	if prepared {
		if !perm.HasAttr(AttrPrepared) {
			perm.GrantBaseAttr(AttrPrepared)
		}
	} else {
		// Clear all base seeds for AttrPrepared so the attr goes to zero.
		for perm.HasAttr(AttrPrepared) {
			perm.RevokeBaseAttr(AttrPrepared)
		}
	}
}

// CastPreparedSpellCopy casts a copy of the Prepared creature's spell side.
// It builds a fresh Card from the supplied factory, pushes it onto the stack
// as a copy (CR 706 / 707.10 — copies cease to exist when they resolve or
// fizzle), prompts the controller for any required targets, fires
// EvtSpellCast, and unprepares the source permanent. Returns an error if the
// permanent can't be found, isn't prepared, or the spell card cannot be
// constructed.
func (g *Game) CastPreparedSpellCopy(playerID, permID uuid.UUID, spellFactory func() Card) error {
	perm := g.FindPermanent(permID)
	if perm == nil {
		return fmt.Errorf("permanent not found")
	}
	if !perm.HasAttr(AttrPrepared) {
		return fmt.Errorf("permanent is not prepared")
	}
	if perm.Controller != playerID {
		return fmt.Errorf("only the controller may cast the prepared spell")
	}
	if spellFactory == nil {
		return fmt.Errorf("nil prepared spell factory")
	}
	pl := g.GetPlayer(playerID)
	if pl == nil {
		return ErrPlayerNotFound
	}

	card := spellFactory()
	if card == nil {
		return fmt.Errorf("prepared spell factory returned nil")
	}
	card.SetID(uuid.New())
	card.SetOwner(playerID)

	// Gather effects from the card's SpellAbility.
	var effects []Effect
	for _, a := range card.Abilities() {
		if sa, ok := a.(*SpellAbility); ok {
			effects = append(effects, sa.Effects()...)
		}
	}

	obj := &StackObject{
		ID:         uuid.New(),
		Card:       card,
		Controller: playerID,
		SourceID:   card.ID(),
		Effects:    effects,
		IsCopy:     true,
		CastZone:   ZoneAny,
		// Forward the resolving spell's X (CR 706.10c — copies of an X-cost
		// spell copy the chosen X). When CastPreparedSpellCopy is called from
		// inside a resolving effect (the typical Prepared activation), the
		// engine has stashed the originating spell's X in g.currentX.
		XValue: g.currentX,
	}

	// Prompt for targets on each declared SpellAbility target.
	for _, a := range card.Abilities() {
		sa, ok := a.(*SpellAbility)
		if !ok {
			continue
		}
		if len(sa.Targets()) == 0 {
			continue
		}
		obj.Targets = g.promptTargetsForList(playerID, card, sa.Targets())
		break
	}

	g.stack.Push(obj)
	if card.HasType(TypeInstant) {
		g.instantsCastThisTurn[playerID]++
	}
	if card.HasType(TypeSorcery) {
		g.sorceriesCastThisTurn[playerID]++
	}
	g.FireEvent(GameEvent{
		Type:     EvtSpellCast,
		SourceID: card.ID(),
		PlayerID: playerID,
	})
	g.fireBecomesTargetEvents(obj, false)

	// Unprepare the source permanent.
	g.SetPrepared(permID, false)
	return nil
}

// WithPreparedSpell installs the Prepared keyword on a creature. The
// `spellFactory` returns a fresh Card representing the spell-side that will
// be copied when the controller activates the ability. The spell card's
// SpellAbility (with its targets and effects) is used; the card itself never
// enters any zone (the cast is always a copy on the stack).
//
// The option adds:
//   - An ETB trigger that calls Game.SetPrepared(self, true).
//   - A sorcery-speed activated ability with no mana cost gated on
//     IsPrepared(self) that calls Game.CastPreparedSpellCopy.
//
// Cards generally still want to declare the front-face creature stats with
// NewCreature; WithPreparedSpell layers the Prepared mechanics on top.
func WithPreparedSpell(spellFactory func() Card) CardOption {
	return func(c *BaseCard) {
		// Mark the card with the keyword seed so AttrPrepared shows in keyword
		// listings / cardHasKeyword lookups (e.g. for a hypothetical
		// "creatures with prepare spells" filter — Biblioplex Tomekeeper).
		if c.attrSeeds == nil {
			c.attrSeeds = make(map[Attr]int)
		}
		// Seed at zero so the creature only becomes prepared via ETB; the
		// AttrSeeds map is also probed by cardHasKeyword which only cares
		// about >0 — so we use a distinct seed key (see HasPreparedSpell
		// below). We mark this with a separate sentinel via SubTypes? No —
		// keep it simple: stash on a side map tracked by HasPreparedSpell.
		preparedSpellFactories[c.Name()] = spellFactory
		// ETB: become prepared.
		c.AddAbility(EntersBattlefieldTrigger(FuncEffect(
			"this creature enters prepared",
			EffectProperties{},
			func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
				g.SetPrepared(sourceID, true)
				return nil
			}), false))
		// Activated ability: cast a copy of the spell.
		castCopy := FuncEffect(
			"cast a copy of this creature's spell",
			EffectProperties{},
			func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
				return g.CastPreparedSpellCopy(controller, sourceID, spellFactory)
			})
		ab := NewActivatedAbility(castCopy, ManaCostOf("{0}"),
			WithSorcerySpeed(),
			WithActivationCondition(func(g *Game, src *Permanent, controller uuid.UUID) bool {
				return src != nil && src.HasAttr(AttrPrepared)
			}),
		)
		c.AddAbility(ab)
	}
}

// preparedSpellFactories stores the spell factory associated with a
// Prepared creature card, keyed by card name. Card-level lookup
// (HasPreparedSpell) is per-card-name rather than per-instance because each
// invocation of a card factory creates a fresh *BaseCard pointer, so a
// pointer-keyed map would miss every lookup against a card on the battlefield.
var preparedSpellFactories = map[string]func() Card{}

// HasPreparedSpell reports whether the given card was registered with a
// Prepared spell side via WithPreparedSpell.
func HasPreparedSpell(c Card) bool {
	if c == nil {
		return false
	}
	_, has := preparedSpellFactories[c.Name()]
	return has
}

// =============================================================================
// Repartee
// =============================================================================

// RepartreeTrigger / WheneverYouCastInstantOrSorceryTargetingCreature
// fires whenever the controller casts an instant or sorcery spell that
// declares a target which is a creature (or a creature card in any zone).
// The effect runs in the standard trigger context, so target selection /
// SetCondition can be layered.
func WheneverYouCastInstantOrSorceryTargetingCreatureTrigger(effect Effect, optional bool) *GenericTriggered {
	return NewTriggered(EvtSpellCast, optional, effect).
		SetConditionData(spellCastByControllerTargetsCreature{})
}

// spellCastByControllerTargetsCreature checks: controller cast the spell;
// the spell is an instant or sorcery; at least one of its declared targets
// is a creature on the battlefield (CR 116.6 / 603.6c — at the time the
// spell goes on the stack the prospective target's identity is read from
// live state).
type spellCastByControllerTargetsCreature struct{}

func (spellCastByControllerTargetsCreature) CheckTriggerCond(evt *GameEvent, g GameReader, _, controllerID uuid.UUID) bool {
	if evt.PlayerID != controllerID {
		return false
	}
	card := g.FindCardAnywhere(evt.SourceID)
	if card == nil {
		return false
	}
	if !card.HasType(TypeInstant) && !card.HasType(TypeSorcery) {
		return false
	}
	obj := g.FindStackObject(evt.SourceID)
	if obj == nil {
		return false
	}
	for _, tid := range collectAllTargets(obj) {
		if tid == uuid.Nil {
			continue
		}
		if perm := g.FindPermanent(tid); perm != nil && perm.HasAttr(AttrIsCreature) {
			return true
		}
	}
	return false
}

// collectAllTargets returns every UUID referenced by the StackObject's
// targets, including modal targets, in declaration order. uuid.Nil entries
// (representing "no target chosen for an optional 0+ slot") are skipped.
func collectAllTargets(obj *StackObject) []uuid.UUID {
	var out []uuid.UUID
	out = append(out, obj.Targets...)
	for _, mt := range obj.ModalTargets {
		out = append(out, mt...)
	}
	return out
}

// =============================================================================
// Opus / Increment — "amount of mana spent to cast" predicate
// =============================================================================

// ManaSpentToCast returns the total amount of mana that was spent to cast
// the spell whose StackObject is given. It sums the values in
// CastContext.ColorsSpent (which records both colored and generic drains).
// Returns 0 if the stack object or its CastContext is nil.
func ManaSpentToCast(obj *StackObject) int {
	if obj == nil || obj.CastContext == nil {
		return 0
	}
	n := 0
	for _, v := range obj.CastContext.ColorsSpent {
		n += v
	}
	return n
}

// ManaSpentForSpellEvent looks up the StackObject for evt.SourceID (assumed
// to be an EvtSpellCast event) on the live stack and returns the mana spent
// to cast it. Returns 0 if the spell is not on the stack.
func ManaSpentForSpellEvent(evt *GameEvent, g GameReader) int {
	if evt == nil {
		return 0
	}
	return ManaSpentToCast(g.FindStackObject(evt.SourceID))
}

// WheneverYouCastSpellWithManaSpent is a trigger shorthand: fires whenever
// the controller casts a spell of any type matching `filters`. The effect
// closure runs in a normal stack-effect context, but it can read the mana
// spent through the cast context — typical use is to wrap the closure in a
// FuncEffect that calls Game.ResolvingCastContext() or to pass the mana
// amount via OpusEffect / IncrementEffect.
func WheneverYouCastSpellWithManaSpent(effect Effect, optional bool, filters ...CardFilter) *GenericTriggered {
	return WheneverYouCastSpellTrigger(effect, optional, filters...)
}

// OpusEffect builds an Effect that runs `fn` with the total mana that was
// spent to cast the triggering spell, looked up via ResolvingCastContext.
// (Opus triggers are non-targeted; the closure receives the mana value, the
// game, the source permanent's ID, and its controller.)
//
// Used by Opus-keyword cards like Deluge Virtuoso, Exhibition Tidecaller,
// Muse Seeker.
func OpusEffect(text string, fn func(g *Game, sourceID, controller uuid.UUID, manaSpent int) error) Effect {
	return FuncEffect(text, EffectProperties{}, func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
		spent := opusManaSpentForResolvingTrigger(g)
		return fn(g, sourceID, controller, spent)
	})
}

// opusManaSpentForResolvingTrigger reads the StackObject of the spell that
// triggered the currently-resolving Opus / Increment ability. The ability's
// trigger fires on EvtSpellCast and is pushed on the stack while the spell
// is still beneath it; we identify the spell by walking down through the
// stack, skipping abilities and the resolving Opus trigger itself, and pick
// the first SpellCast object below.
func opusManaSpentForResolvingTrigger(g *Game) int {
	objs := g.stack.Objects()
	for i := len(objs) - 1; i >= 0; i-- {
		obj := objs[i]
		if obj.IsAbility {
			continue
		}
		// The triggering spell is the one most recently cast that lies
		// under the resolving trigger. The resolving stack object has
		// already been popped before Apply runs, so the deepest spell
		// from the top is the right one.
		return ManaSpentToCast(obj)
	}
	return 0
}

// =============================================================================
// Increment
// =============================================================================

// IncrementTrigger creates the standard Increment cast-trigger: "Whenever you
// cast a spell, if the amount of mana you spent is greater than this
// creature's power or toughness, put a +1/+1 counter on this creature." The
// caller passes the source permanent's name only for trigger text; the
// engine resolves source live at trigger time.
func IncrementTrigger() *GenericTriggered {
	effect := OpusEffect(
		"if mana spent > this creature's power or toughness, put a +1/+1 counter on this",
		func(g *Game, sourceID, controller uuid.UUID, manaSpent int) error {
			perm := g.FindPermanent(sourceID)
			if perm == nil {
				return nil
			}
			pow := perm.CurrentPower(g)
			tou := perm.CurrentToughness(g)
			if manaSpent > pow || manaSpent > tou {
				perm.AddCounter(P1P1, 1)
			}
			return nil
		})
	return WheneverYouCastSpellTrigger(effect, false)
}

// =============================================================================
// Infusion — "if you gained life this turn" predicate
// =============================================================================

// IfControllerGainedLifeThisTurn returns true if the supplied controller has
// gained at least one point of life this turn. Wraps the existing
// PlayerLifeGainedThisTurn tracker.
func IfControllerGainedLifeThisTurn(g GameReader, controller uuid.UUID) bool {
	if pg, ok := g.(*Game); ok {
		return pg.PlayerLifeGainedThisTurn(controller) > 0
	}
	return false
}

// LifeGainedThisTurnFor returns the amount of life the controller has
// gained this turn (used by Moseo's "X is the amount of life you gained
// this turn"-style effects).
func LifeGainedThisTurnFor(g GameReader, controller uuid.UUID) int {
	if pg, ok := g.(*Game); ok {
		return pg.PlayerLifeGainedThisTurn(controller)
	}
	return 0
}

// InfusionEffect wraps an Effect so it only applies if the controller has
// gained life this turn. Otherwise it is a no-op.
func InfusionEffect(text string, inner Effect) Effect {
	return FuncEffect(text, EffectProperties{},
		func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
			if !IfControllerGainedLifeThisTurn(g, controller) {
				return nil
			}
			return inner.Apply(g, sourceID, controller, targets)
		})
}

// =============================================================================
// Grandeur — "Discard another card with the same name as this" cost
// =============================================================================

// DiscardAnotherCardNamedSelfCost returns a Cost that requires the
// controller to discard a card from hand whose name equals the source
// card's name and which is not the source card itself (CR 702.62 / Future
// Sight Grandeur). The cost is unpayable when no such card exists in hand.
func DiscardAnotherCardNamedSelfCost() Cost {
	return &discardAnotherNamedSelfCost{}
}

type discardAnotherNamedSelfCost struct{}

func (c *discardAnotherNamedSelfCost) sourceName(sourceID uuid.UUID, g *Game) string {
	if card := g.FindCardAnywhere(sourceID); card != nil {
		return card.Name()
	}
	return ""
}

func (c *discardAnotherNamedSelfCost) findCandidate(sourceID, controller uuid.UUID, g *Game) Card {
	p := g.GetPlayer(controller)
	if p == nil {
		return nil
	}
	name := c.sourceName(sourceID, g)
	if name == "" {
		return nil
	}
	for _, card := range p.Hand() {
		if card == nil {
			continue
		}
		if card.ID() == sourceID {
			continue
		}
		if card.Name() == name {
			return card
		}
	}
	return nil
}

func (c *discardAnotherNamedSelfCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	return c.findCandidate(sourceID, controller, g) != nil
}

func (c *discardAnotherNamedSelfCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	cand := c.findCandidate(sourceID, controller, g)
	if cand == nil {
		return fmt.Errorf("no card with same name in hand")
	}
	if _, ok := g.PlayerDiscard(p, cand.ID()); !ok {
		return fmt.Errorf("failed to discard %s", cand.Name())
	}
	return nil
}

func (c *discardAnotherNamedSelfCost) Text() string { return "Discard another card with the same name" }

// =============================================================================
// Paradigm
// =============================================================================

// ParadigmExileTracker records, per (player, card-name) pair, whether the
// player has resolved their "first" Paradigm spell with that name. Once a
// spell of the named card has resolved at least once for a controller, the
// per-main-phase recurring free-copy trigger (RegisterParadigmRecurringCast)
// is allowed to fire.
//
// Keyed by (controller-UUID, lowercased card name). Uses Game's per-game
// custom-state map (g.SetCustomState/GetCustomState) — kept on Game itself
// to survive cloning.

const paradigmStateKey = "sos.paradigm"

type paradigmState struct {
	// Resolved[playerID][name] — has this player resolved a spell with this name?
	Resolved map[uuid.UUID]map[string]bool
	// ExiledIDs[playerID][name] — the exiled card IDs of paradigm spells
	// belonging to this player, keyed by name; the first matching id is
	// recast at each main phase.
	ExiledIDs map[uuid.UUID]map[string]uuid.UUID
}

func paradigmStateOf(g *Game) *paradigmState {
	if g.customState == nil {
		g.customState = make(map[string]any)
	}
	v, ok := g.customState[paradigmStateKey]
	if !ok {
		st := &paradigmState{
			Resolved:  map[uuid.UUID]map[string]bool{},
			ExiledIDs: map[uuid.UUID]map[string]uuid.UUID{},
		}
		g.customState[paradigmStateKey] = st
		return st
	}
	return v.(*paradigmState)
}

// RecordParadigmResolution marks that the named player has resolved a spell
// with the given name at least once this game. Subsequent main-phase
// trigger checks will allow the recurring free-cast. Idempotent.
func (g *Game) RecordParadigmResolution(playerID uuid.UUID, name string) {
	st := paradigmStateOf(g)
	if st.Resolved[playerID] == nil {
		st.Resolved[playerID] = map[string]bool{}
	}
	st.Resolved[playerID][name] = true
}

// HasResolvedParadigmSpell reports whether the named player has resolved a
// spell with the given name at least once. Used by the recurring main-phase
// trigger ("After you first resolve a spell with this name, you may cast a
// copy of it from exile…").
func (g *Game) HasResolvedParadigmSpell(playerID uuid.UUID, name string) bool {
	st := paradigmStateOf(g)
	if st.Resolved[playerID] == nil {
		return false
	}
	return st.Resolved[playerID][name]
}

// RegisterParadigmExiledCopy records an exiled card as the "paradigm copy"
// of `name` for `playerID`. The recurring main-phase trigger will use this
// recorded card ID to identify which exiled card to cast for free.
func (g *Game) RegisterParadigmExiledCopy(playerID uuid.UUID, name string, cardID uuid.UUID) {
	st := paradigmStateOf(g)
	if st.ExiledIDs[playerID] == nil {
		st.ExiledIDs[playerID] = map[string]uuid.UUID{}
	}
	st.ExiledIDs[playerID][name] = cardID
}

// ParadigmExiledCopy returns the exiled card ID registered for (playerID,
// name) if any, and a bool indicating presence.
func (g *Game) ParadigmExiledCopy(playerID uuid.UUID, name string) (uuid.UUID, bool) {
	st := paradigmStateOf(g)
	if st.ExiledIDs[playerID] == nil {
		return uuid.Nil, false
	}
	id, ok := st.ExiledIDs[playerID][name]
	return id, ok
}

// =============================================================================
// helpers
// =============================================================================

// ManaCostOf is a thin wrapper around the engine's mana cost parser so the
// SOS keyword helpers don't need to import the parser explicitly. It panics
// on an invalid cost string (matching the behaviour of the registered
// engine helper).
func init() {
	// Compile-time check that AttrPrepared is in the keyword range so it
	// shows up alongside Flying / Trample / etc. in keyword-listing logic.
	if !IsKeywordAttr(AttrPrepared) {
		panic("AttrPrepared must be in the keyword attr range")
	}
}
