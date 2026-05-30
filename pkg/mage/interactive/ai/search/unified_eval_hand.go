package search

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// handValue scores p's hand contextually: each card is valued by what it would
// do if cast given the current game state, not by a flat per-card constant.
// Lands keep their flat low value; everything else is dispatched by class.
func handValue(g *mage.Game, p mage.Player) float64 {
	var v float64
	for _, c := range p.Hand() {
		v += cardHandValue(g, p, c)
	}
	return v
}

func cardHandValue(g *mage.Game, p mage.Player, c mage.Card) float64 {
	if c.HasType(core.TypeLand) {
		return wLandInHand
	}
	// Property-based dispatch: classify by Effect.Properties() on the spell
	// ability rather than by card name. Covers damage spells (Lightning Bolt),
	// boost combat tricks (Giant Growth), life gain (Healing Salve), and
	// single-target removal (Terror, Swords to Plowshares).
	if v, ok := spellEffectValue(g, p, c); ok {
		return v
	}
	// Name-based fallback for cards whose value isn't reflected in spell-effect
	// Properties: auras (boost lives on the resulting permanent's static
	// ability), enchantments (continuous effect, not a spell effect), and
	// X-spells using FuncEffect (Disintegrate doesn't expose DamageValue).
	switch c.Name() {
	case "Disintegrate":
		// X-cost: budget = available colored Red mana (untapped Mountains).
		x := availableColorMana(g, p, core.Red)
		// Disintegrate also costs {X}{R}, so X effectively spends one less.
		if x > 0 {
			x--
		}
		return burnAnyValue(g, p, x)
	case "Crusade":
		return globalPumpValue(g, p, 1, 1, isWhiteCreature)
	case "Holy Strength":
		return localAuraValue(g, p, 1, 2)
	case "Unholy Strength":
		return localAuraValue(g, p, 2, 1)
	case "Firebreathing":
		// Activated +1/+0 per {R}; value as one-shot pump for currently
		// available red mana on the largest creature.
		return activatedPumpValue(g, p, core.Red, 1, 0)
	}
	return wSpellInHand
}

// spellEffectValue dispatches a hand card by introspecting its spell ability's
// Effect.Properties(). Returns (value, true) when the card matches a known
// effect shape; (0, false) when nothing matched and the caller should fall
// back to name-based dispatch.
//
// Order matters: damage > boost > life-gain > single-target removal. Damage
// outranks "detrimental" because damage effects are also OutcomeDetriment but
// have a more specific valuation model (lethal-on-creature vs. finisher).
func spellEffectValue(g *mage.Game, p mage.Player, c mage.Card) (float64, bool) {
	action := spellAction(c)
	if action == nil {
		return 0, false
	}
	effects := action.Effects()

	for _, e := range effects {
		if mage.IsDamageEffect(e) {
			// Resolve with the live game; Fixed values ignore it, X reads
			// g.XValue() which is 0 outside a spell context — fine, since X
			// spells like Disintegrate use FuncEffect (no DamageValue) and
			// fall to the name-based path anyway.
			amount := e.Properties().DamageValue.Resolve(g, uuid.Nil, uuid.Nil, nil)
			return burnAnyValue(g, p, amount), true
		}
	}
	for _, e := range effects {
		props := e.Properties()
		if props.PowerBoost > 0 || props.ToughnessBoost > 0 {
			return combatTrickValue(g, p, props.PowerBoost, props.ToughnessBoost), true
		}
	}
	for _, e := range effects {
		if lg := topLifeGain(e); lg > 0 {
			return lifeGainValue(p, lg), true
		}
	}
	for _, e := range effects {
		props := e.Properties()
		if props.Outcome != mage.OutcomeDetriment {
			continue
		}
		if props.DamageValue != nil || props.Mass || props.IsBounce {
			continue
		}
		if len(action.Targets()) == 0 {
			continue
		}
		return removalValueFromTarget(g, p, c, action.Targets()[0]), true
	}
	return 0, false
}

// spellAction returns the card's spell ability (the one cast from hand), or
// nil if the card has no spell ability (e.g. a land).
func spellAction(c mage.Card) *mage.ActionDefinition {
	for _, ab := range c.Abilities() {
		if a, ok := ab.(*mage.ActionDefinition); ok && a.Kind() == mage.ActionSpell {
			return a
		}
	}
	return nil
}

// topLifeGain returns the LifeGain amount from an Effect's Properties, or
// the first LifeGain found among a ModalEffectData's modes (Healing Salve).
// Modal wrappers don't propagate child Properties, so we peek inside.
func topLifeGain(e mage.Effect) int {
	if lg := e.Properties().LifeGain; lg > 0 {
		return lg
	}
	if modal, ok := e.(*mage.ModalEffectData); ok {
		for _, m := range modal.Modes {
			if lg := m.Properties().LifeGain; lg > 0 {
				return lg
			}
		}
	}
	return 0
}

// removalValueFromTarget scores a single-target removal spell using the
// engine's Target.Possible to enumerate legal targets. This automatically
// honors target-filter restrictions like Terror's "non-black, non-artifact"
// without us re-implementing them.
func removalValueFromTarget(g *mage.Game, owner mage.Player, c mage.Card, t mage.Target) float64 {
	opp := g.GetOpponent(owner.PlayerID())
	if opp == nil {
		return wSpellInHand
	}
	bestKill := 0.0
	for _, id := range t.Possible(owner.PlayerID(), c, g) {
		perm := g.FindPermanent(id)
		if perm == nil {
			continue
		}
		if perm.Controller != opp.PlayerID() || !perm.HasType(core.TypeCreature) {
			continue
		}
		pt := float64(perm.CurrentPower(g) + perm.CurrentToughness(g))
		pt *= evasionFactor(g, perm, owner.PlayerID())
		if pt > bestKill {
			bestKill = pt
		}
	}
	v := handRemovalEffectScale * wBoardDiff * bestKill
	floor := handRemovalEffectScale * wBoardDiff * threatCeiling
	if v < floor {
		v = floor
	}
	return v
}

// burnAnyValue scores a damage spell that can target a creature or a player.
// It picks max over (best lethal-on-a-creature, value-as-finisher-on-opp).
// Floor at threatCeiling so an unused burn spell is still valuable for what's
// coming or for closing the game.
func burnAnyValue(g *mage.Game, owner mage.Player, dmg int) float64 {
	if dmg <= 0 {
		return wSpellInHand
	}
	opp := g.GetOpponent(owner.PlayerID())
	if opp == nil {
		return wSpellInHand
	}

	// Best creature kill: among opponent's creatures with toughness ≤ dmg,
	// take the largest P+T we'd remove.
	bestKill := 0.0
	for _, perm := range g.AllBattlefield() {
		if perm.Controller != opp.PlayerID() || !perm.HasType(core.TypeCreature) {
			continue
		}
		if perm.CurrentToughness(g) > dmg {
			continue
		}
		pt := float64(perm.CurrentPower(g) + perm.CurrentToughness(g))
		// Weight by evasion the same way boardPower does — killing a flier the
		// opponent has no answer to is worth more than the same P+T on a vanilla.
		pt *= evasionFactor(g, perm, owner.PlayerID())
		if pt > bestKill {
			bestKill = pt
		}
	}

	// Finisher value: damage to opponent's life. This rises sharply as opp
	// approaches death, so early face-burn looks bad and lethal looks great.
	finisher := lifePressure(opp.Life()-dmg) - lifePressure(opp.Life())

	v := handRemovalEffectScale*wBoardDiff*bestKill + wPressure*finisher
	floor := handRemovalEffectScale * wBoardDiff * threatCeiling
	if v < floor {
		v = floor
	}
	return v
}

// globalPumpValue scores a Crusade-style enchantment by what it would do if
// cast right now: count my matching creatures and credit (P+T) per creature.
// Floor at wSpellInHand so early-game (no creatures yet) it's not literally 0.
func globalPumpValue(g *mage.Game, owner mage.Player, p, t int, eligible func(*mage.Permanent) bool) float64 {
	count := 0
	for _, perm := range g.AllBattlefield() {
		if perm.Controller != owner.PlayerID() || !perm.HasType(core.TypeCreature) {
			continue
		}
		if eligible(perm) {
			count++
		}
	}
	v := handPermanentEffectScale * wBoardDiff * float64(count) * float64(p+t)
	if v < wSpellInHand {
		v = wSpellInHand
	}
	return v
}

// localAuraValue scores an aura that pumps one creature by +p/+t. Picks my
// best creature to target.
func localAuraValue(g *mage.Game, owner mage.Player, p, t int) float64 {
	if !ownerHasCreature(g, owner) {
		return wSpellInHand
	}
	v := handPermanentEffectScale * wBoardDiff * float64(p+t)
	if v < wSpellInHand {
		v = wSpellInHand
	}
	return v
}

// activatedPumpValue scores an aura (Firebreathing) that pumps once per mana
// of a given color. Approximates the immediate one-shot pump using available
// untapped lands of that color.
func activatedPumpValue(g *mage.Game, owner mage.Player, color core.Color, p, t int) float64 {
	if !ownerHasCreature(g, owner) {
		return wSpellInHand
	}
	mana := availableColorMana(g, owner, color)
	v := handPermanentEffectScale * wBoardDiff * float64(mana) * float64(p+t)
	if v < wSpellInHand {
		v = wSpellInHand
	}
	return v
}

// combatTrickValue is a one-shot EOT pump (Giant Growth). Worth less than a
// permanent pump because it doesn't stick — score conservatively.
func combatTrickValue(g *mage.Game, p mage.Player, pBoost, tBoost int) float64 {
	if !ownerHasCreature(g, p) {
		return wSpellInHand
	}
	// Half of board-equivalent value to reflect EOT-only nature.
	return 0.5 * wBoardDiff * float64(pBoost+tBoost)
}

// lifeGainValue scores life gain by the pressure delta it would relieve.
// Healing 3 life is meaningless at 20 life and crucial at 5 life.
func lifeGainValue(p mage.Player, n int) float64 {
	cur := p.Life()
	gain := lifePressure(cur) - lifePressure(cur+n)
	v := wPressure * gain
	if v < wSpellInHand {
		v = wSpellInHand
	}
	return v
}

// ── helpers ─────────────────────────────────────────────────────────────────

func ownerHasCreature(g *mage.Game, owner mage.Player) bool {
	for _, perm := range g.AllBattlefield() {
		if perm.Controller == owner.PlayerID() && perm.HasType(core.TypeCreature) {
			return true
		}
	}
	return false
}

// availableColorMana counts untapped lands controlled by p that produce the
// given color. Approximation: assumes basic-land mapping (Mountain→Red, etc.)
// via card names. Good enough for the current card pool.
func availableColorMana(g *mage.Game, p mage.Player, color core.Color) int {
	produces := basicLandColor(color)
	if produces == "" {
		return 0
	}
	n := 0
	for _, perm := range g.AllBattlefield() {
		if perm.Controller != p.PlayerID() || !perm.HasType(core.TypeLand) || perm.Tapped {
			continue
		}
		if perm.Name() == produces {
			n++
		}
	}
	return n
}

func basicLandColor(c core.Color) string {
	switch c {
	case core.White:
		return "Plains"
	case core.Blue:
		return "Island"
	case core.Black:
		return "Swamp"
	case core.Red:
		return "Mountain"
	case core.Green:
		return "Forest"
	}
	return ""
}

func isWhiteCreature(perm *mage.Permanent) bool {
	return slices.Contains(perm.Card.ManaCost().Colors(), core.White)
}
