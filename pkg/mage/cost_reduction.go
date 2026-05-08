package mage

import (
	"fmt"

	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// ---------------------------------------------------------------------------
// Conditional spell cost reduction (CR 601.2f)
//
// CR 601.2f: "The player determines the total cost of the spell. ... If the
// total cost includes a mana payment, the player then has a chance to activate
// mana abilities. ... Cost increases are applied before cost reductions."
//
// Two flavors are supported here:
//
//   1. Static reductions sourced from a permanent on the battlefield, applying
//      to spells cast by some player that match a spell-filter (e.g. Warden of
//      Evos Isle, Dragonlord's Servant, Herald's Horn).
//
//   2. Self cost reductions intrinsic to the casting card itself, reducing
//      the cost only when THIS card is cast (e.g. Bone Picker, Cryptic Serpent,
//      Ghalta, Primal Hunger).
//
// Both flavors only ever reduce the *generic* portion of a mana cost (a cost
// can be reduced to zero generic but its colored requirements are unchanged).
// A spell's generic cost cannot be reduced below zero on a single application.
// ---------------------------------------------------------------------------

// SpellPredicate is a predicate evaluated at cast time to decide whether a
// cost reducer applies to the spell being cast.
//
// The arguments are:
//   - g: the live game (so a predicate may inspect game state)
//   - controller: the player casting the spell
//   - card: the card being cast
//   - sourceID: the permanent ID that registered the reducer (uuid.Nil for
//     self-reducers — the reduction is intrinsic to the casting card)
type SpellPredicate func(g *Game, controller uuid.UUID, card Card, sourceID uuid.UUID) bool

// SpellAmount returns the amount of generic mana a reducer should remove from
// the spell's cost. Receives the same context as SpellPredicate so the amount
// can be dynamic (graveyard count, total power, etc.).
type SpellAmount func(g *Game, controller uuid.UUID, card Card, sourceID uuid.UUID) int

// SpellCondition is an optional global gate (e.g. "if you control a Wizard",
// "if a creature died this turn"). If nil the reducer is unconditional.
type SpellCondition func(g *Game, controller uuid.UUID, card Card, sourceID uuid.UUID) bool

// FixedAmount returns a SpellAmount that always returns n.
func FixedAmount(n int) SpellAmount {
	return func(_ *Game, _ uuid.UUID, _ Card, _ uuid.UUID) int { return n }
}

// SpellCostReducer is a registered cost-reduction entry on GameRules. It is
// installed by ReduceSpellCostStatic while the source permanent is on the
// battlefield, and consulted at cast time.
type SpellCostReducer struct {
	SourceID   uuid.UUID
	Controller uuid.UUID // the player whose spells are affected (typically the source's controller)
	Filter     SpellPredicate
	Amount     SpellAmount
	Condition  SpellCondition
	Label      string
}

// applies reports whether this reducer should reduce the cost of the given
// cast. The cast's sourceID is the *card being cast*; the reducer's own
// SourceID is the permanent that registered it.
func (r *SpellCostReducer) applies(g *Game, casterID uuid.UUID, card Card) bool {
	if r.Controller != uuid.Nil && r.Controller != casterID {
		return false
	}
	if r.Filter != nil && !r.Filter(g, casterID, card, r.SourceID) {
		return false
	}
	if r.Condition != nil && !r.Condition(g, casterID, card, r.SourceID) {
		return false
	}
	return true
}

// ---------------------------------------------------------------------------
// Spell predicates (filters) — cast-time card-level
// ---------------------------------------------------------------------------

// SpellAny matches every spell.
func SpellAny() SpellPredicate {
	return func(_ *Game, _ uuid.UUID, _ Card, _ uuid.UUID) bool { return true }
}

// SpellHasType matches spells of the given card type (creature, artifact, etc.).
func SpellHasType(t CardType) SpellPredicate {
	return func(_ *Game, _ uuid.UUID, c Card, _ uuid.UUID) bool {
		return c.HasType(t)
	}
}

// SpellHasSubType matches spells whose card has the given (case-sensitive)
// subtype, e.g. "Dragon" or "Wizard".
func SpellHasSubType(st string) SpellPredicate {
	return func(_ *Game, _ uuid.UUID, c Card, _ uuid.UUID) bool {
		return c.HasSubType(st)
	}
}

// SpellHasKeyword matches spells whose card was registered with the given
// keyword attr (e.g. Flying for Warden of Evos Isle).
func SpellHasKeyword(kw Attr) SpellPredicate {
	return func(_ *Game, _ uuid.UUID, c Card, _ uuid.UUID) bool {
		return cardHasKeyword(c, kw)
	}
}

// SpellIsSelf matches only the very card that registered the reducer (used by
// activated/triggered abilities that grant their owner card a cost reduction).
// Compares card IDs — the casting card must be the same Card instance.
func SpellIsSelf() SpellPredicate {
	return func(_ *Game, _ uuid.UUID, c Card, sourceID uuid.UUID) bool {
		return c.ID() == sourceID
	}
}

// SpellSubTypeMatchesChosen matches spells whose card has a subtype equal to
// the source permanent's ChosenSubtype field (Herald's Horn pattern).
func SpellSubTypeMatchesChosen() SpellPredicate {
	return func(g *Game, _ uuid.UUID, c Card, sourceID uuid.UUID) bool {
		perm := g.FindPermanent(sourceID)
		if perm == nil || perm.ChosenSubtype == "" {
			return false
		}
		return c.HasSubType(perm.ChosenSubtype)
	}
}

// SpellsAnd combines spell predicates with logical AND.
func SpellsAnd(ps ...SpellPredicate) SpellPredicate {
	return func(g *Game, controller uuid.UUID, c Card, sourceID uuid.UUID) bool {
		for _, p := range ps {
			if p == nil {
				continue
			}
			if !p(g, controller, c, sourceID) {
				return false
			}
		}
		return true
	}
}

// SpellsOr combines spell predicates with logical OR.
func SpellsOr(ps ...SpellPredicate) SpellPredicate {
	return func(g *Game, controller uuid.UUID, c Card, sourceID uuid.UUID) bool {
		for _, p := range ps {
			if p != nil && p(g, controller, c, sourceID) {
				return true
			}
		}
		return false
	}
}

// ---------------------------------------------------------------------------
// Spell conditions (global state predicates)
// ---------------------------------------------------------------------------

// CondCreatureDiedThisTurn matches when at least one creature has died this
// turn (Bone Picker).
func CondCreatureDiedThisTurn() SpellCondition {
	return func(g *Game, _ uuid.UUID, _ Card, _ uuid.UUID) bool {
		return g.creatureDeathsThisTurn > 0
	}
}

// CondCardLeftYourGraveyardThisTurn matches when at least one card has left
// the casting player's graveyard this turn.
func CondCardLeftYourGraveyardThisTurn() SpellCondition {
	return func(g *Game, controller uuid.UUID, _ Card, _ uuid.UUID) bool {
		return g.PlayerHadCardLeaveGraveyardThisTurn(controller)
	}
}

// CondControlsMatching matches when the casting player controls at least one
// permanent satisfying f (e.g. "if you control a Wizard", "if you control a
// creature with flying").
func CondControlsMatching(f PermanentFilter) SpellCondition {
	return func(g *Game, controller uuid.UUID, _ Card, _ uuid.UUID) bool {
		for _, p := range g.battlefield {
			if p.Controller != controller {
				continue
			}
			if f.Match(p, g) {
				return true
			}
		}
		return false
	}
}

// CondAnd combines conditions with logical AND. A nil entry is ignored.
func CondAnd(cs ...SpellCondition) SpellCondition {
	return func(g *Game, controller uuid.UUID, c Card, sourceID uuid.UUID) bool {
		for _, cd := range cs {
			if cd == nil {
				continue
			}
			if !cd(g, controller, c, sourceID) {
				return false
			}
		}
		return true
	}
}

// ---------------------------------------------------------------------------
// Dynamic amount sources
// ---------------------------------------------------------------------------

// AmountByGraveyardCount returns the count of cards in the casting player's
// graveyard matching the given CardFilter (Cryptic Serpent: instant/sorcery).
func AmountByGraveyardCount(f CardFilter) SpellAmount {
	return func(g *Game, controller uuid.UUID, _ Card, _ uuid.UUID) int {
		p := g.GetPlayer(controller)
		if p == nil {
			return 0
		}
		n := 0
		for _, c := range p.Graveyard() {
			if f.IsZero() || f.Match(c) {
				n++
			}
		}
		return n
	}
}

// AmountByTotalPower returns the total power of permanents the casting player
// controls matching f (Ghalta, Primal Hunger: total power of creatures).
// Power is counted from the permanent's current value (post-effects) and
// negative values are clamped to zero (CR 208.3).
func AmountByTotalPower(f PermanentFilter) SpellAmount {
	return func(g *Game, controller uuid.UUID, _ Card, _ uuid.UUID) int {
		total := 0
		for _, p := range g.battlefield {
			if p.Controller != controller {
				continue
			}
			if !f.Match(p, g) {
				continue
			}
			pw := max(p.Card.Power(), 0)
			total += pw
		}
		return total
	}
}

// AmountByPermanentCount returns the count of permanents the casting player
// controls matching f.
func AmountByPermanentCount(f PermanentFilter) SpellAmount {
	return func(g *Game, controller uuid.UUID, _ Card, _ uuid.UUID) int {
		n := 0
		for _, p := range g.battlefield {
			if p.Controller != controller {
				continue
			}
			if f.Match(p, g) {
				n++
			}
		}
		return n
	}
}

// ---------------------------------------------------------------------------
// Static-ability constructor (battlefield source)
// ---------------------------------------------------------------------------

// ReduceSpellCostStatic returns a continuous effect that, while the source
// permanent is on the battlefield, registers a cost reducer matching every
// spell its controller casts that satisfies filter (and condition, if any).
// The reduction is applied to generic mana only.
//
// Example: Dragonlord's Servant — "Dragon spells you cast cost {1} less."
//
//	WithStaticAbility(ReduceSpellCostStatic(
//	    SpellsAnd(SpellHasType(TypeCreature), SpellHasSubType("Dragon")),
//	    FixedAmount(1), nil,
//	))
func ReduceSpellCostStatic(filter SpellPredicate, amount SpellAmount, condition SpellCondition) ContinuousEffect {
	return ReduceSpellCostStaticLabeled("", filter, amount, condition)
}

// ReduceSpellCostStaticLabeled is like ReduceSpellCostStatic but lets the
// caller attach a debug label that surfaces in trace output.
func ReduceSpellCostStaticLabeled(label string, filter SpellPredicate, amount SpellAmount, condition SpellCondition) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		perm := g.FindPermanent(sourceID)
		if perm == nil {
			return nil
		}
		g.effects.Rules.AddSpellCostReducer(SpellCostReducer{
			SourceID:   sourceID,
			Controller: perm.Controller,
			Filter:     filter,
			Amount:     amount,
			Condition:  condition,
			Label:      label,
		})
		return nil
	})
}

// ---------------------------------------------------------------------------
// Self cost reduction ability (intrinsic to casting card)
// ---------------------------------------------------------------------------

// SelfCostReductionAbility is a static-ish ability borne by the card itself
// that reduces the generic portion of its own mana cost at cast time. Unlike
// ReduceSpellCostStatic it works while the card is in hand — the cost
// computation walks the casting card's abilities directly.
//
// Examples:
//   - Bone Picker: amount=FixedAmount(3),
//     condition=CondCreatureDiedThisTurn()
//   - Cryptic Serpent: amount=AmountByGraveyardCount(IsInstantOrSorceryCard)
//   - Ghalta, Primal Hunger:
//     amount=AmountByTotalPower(And(IsCreature, ControlledByCaster))
type SelfCostReductionAbility struct {
	BaseAbility
	Amount    SpellAmount
	Condition SpellCondition
	// TargetPredicate, if non-nil, gates the reduction based on the chosen
	// targets for this cast. It is consulted in addition to Condition. CR
	// 601.2f says total cost is determined after modes and targets are
	// chosen, so target-conditional reducers like Savage Stomp ("This spell
	// costs {2} less to cast if it targets a Dinosaur you control") evaluate
	// here. If a cost-reduction query has no targets context (e.g. a UI
	// hint before the player picks targets), TargetPredicate is treated as
	// false and the reducer does not apply.
	TargetPredicate func(g *Game, controller uuid.UUID, card Card, targets []uuid.UUID) bool
	Label           string
}

// SelfCostReduction creates a self cost reduction ability. The optional
// condition gates the reduction (returns false ⇒ no reduction).
func SelfCostReduction(amount SpellAmount, condition SpellCondition) *SelfCostReductionAbility {
	return &SelfCostReductionAbility{
		BaseAbility: BaseAbility{id: uuid.New(), abilityType: AbilityStatic},
		Amount:      amount,
		Condition:   condition,
	}
}

// SelfCostReductionLabeled is like SelfCostReduction but with a debug label.
func SelfCostReductionLabeled(label string, amount SpellAmount, condition SpellCondition) *SelfCostReductionAbility {
	a := SelfCostReduction(amount, condition)
	a.Label = label
	return a
}

// WithTargetConditionalCostReduction attaches a self cost-reduction ability
// to a card whose applicability depends on the chosen targets at cast time
// (CR 601.2f). The predicate receives the full target list (multi-target
// spells include all chosen IDs in declaration order).
//
// Example: Savage Stomp — "This spell costs {2} less to cast if it targets
// a Dinosaur you control."
//
//	WithTargetConditionalCostReduction(2, func(g *Game, controller uuid.UUID, c Card, targets []uuid.UUID) bool {
//	    for _, id := range targets {
//	        if perm := g.FindPermanent(id); perm != nil && perm.Controller == controller && perm.Card.HasSubType("Dinosaur") {
//	            return true
//	        }
//	    }
//	    return false
//	})
func WithTargetConditionalCostReduction(amount int, predicate func(g *Game, controller uuid.UUID, card Card, targets []uuid.UUID) bool) CardOption {
	return func(c *BaseCard) {
		ab := SelfCostReduction(FixedAmount(amount), nil)
		ab.TargetPredicate = predicate
		c.AddAbility(ab)
	}
}

// WithSelfCostReduction attaches a SelfCostReductionAbility to a card.
//
// Example: Bone Picker — "This costs {3} less to cast if a creature died this turn."
//
//	NewCreature("Bone Picker", "{3}{B}", 3, 2,
//	    WithSelfCostReduction(FixedAmount(3), CondCreatureDiedThisTurn()))
func WithSelfCostReduction(amount SpellAmount, condition SpellCondition) CardOption {
	return func(c *BaseCard) { c.AddAbility(SelfCostReduction(amount, condition)) }
}

// applySelfCostReductions walks card.Abilities() and returns the total generic
// reduction owed by intrinsic SelfCostReductionAbility entries. The targets
// slice carries the chosen targets for this cast (per CR 601.2f, the total
// cost is determined after modes and targets are chosen). Pass nil for
// pre-target queries — target-conditional reducers will skip themselves.
func applySelfCostReductions(g *Game, controller uuid.UUID, card Card, targets []uuid.UUID) int {
	total := 0
	for _, a := range card.Abilities() {
		scr, ok := UnwrapAbility(a).(*SelfCostReductionAbility)
		if !ok {
			continue
		}
		if scr.Condition != nil && !scr.Condition(g, controller, card, card.ID()) {
			continue
		}
		if scr.TargetPredicate != nil {
			if targets == nil || !scr.TargetPredicate(g, controller, card, targets) {
				continue
			}
		}
		if scr.Amount == nil {
			continue
		}
		n := scr.Amount(g, controller, card, card.ID())
		if n > 0 {
			total += n
		}
	}
	return total
}

// applyExternalCostReductions consults registered SpellCostReducers and
// returns the total generic reduction owed for the cast.
func applyExternalCostReductions(g *Game, controller uuid.UUID, card Card) int {
	total := 0
	for _, r := range g.effects.Rules.SpellCostReducers {
		if !r.applies(g, controller, card) {
			continue
		}
		if r.Amount == nil {
			continue
		}
		n := r.Amount(g, controller, card, r.SourceID)
		if n > 0 {
			total += n
		}
	}
	return total
}

// computeConditionalCostReduction is the entry point used by the cast-cost
// pipeline. It returns the total generic reduction owed (capped to the
// caller's current generic cost). The targets slice carries the chosen
// targets for the cast (CR 601.2f: total cost is calculated after modes
// and targets are chosen). Pass nil for pre-target queries.
func computeConditionalCostReduction(g *Game, controller uuid.UUID, card Card, currentGeneric int, targets []uuid.UUID) int {
	red := min(max(applySelfCostReductions(g, controller, card, targets)+applyExternalCostReductions(g, controller, card), 0), currentGeneric)
	return red
}

// ConditionalSpellCostReduction reports the generic-mana reduction that would
// be applied to the printed cost of card if controller cast it right now,
// accounting for both self cost reductions and registered external reducers.
// The returned value is capped to the card's printed generic cost (a cost
// can't be reduced below zero generic in a single cast). Exported for engine
// tests and tools.
func (g *Game) ConditionalSpellCostReduction(controller uuid.UUID, card Card) int {
	if card == nil {
		return 0
	}
	return computeConditionalCostReduction(g, controller, card, card.ManaCost().Generic, nil)
}

// ConditionalSpellCostReductionWithTargets is like ConditionalSpellCostReduction
// but evaluates target-conditional reducers (CR 601.2f) against the given
// chosen-target list. Used by the cast pipeline and for engine tests.
func (g *Game) ConditionalSpellCostReductionWithTargets(controller uuid.UUID, card Card, targets []uuid.UUID) int {
	if card == nil {
		return 0
	}
	return computeConditionalCostReduction(g, controller, card, card.ManaCost().Generic, targets)
}

// String formats a reducer for debugging.
func (r SpellCostReducer) String() string {
	if r.Label != "" {
		return fmt.Sprintf("CostReducer(%s)", r.Label)
	}
	return fmt.Sprintf("CostReducer(src=%s)", r.SourceID)
}
