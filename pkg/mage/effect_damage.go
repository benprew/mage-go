package mage

import (
	"fmt"

	"github.com/google/uuid"
)

// RevealTopAndDealDamage reveals the top card of the controller's
// library and deals damage equal to that card's mana value to the
// resolving target (player or permanent). Used by Riddle of Lightning:
// "Choose any target. Scry 3, then reveal the top card of your library.
// Riddle of Lightning deals damage equal to that card's mana value to
// that permanent or player." The reveal does not mutate the library.
func RevealTopAndDealDamage() Effect {
	return FuncEffect(
		"reveal top card; deal damage equal to its mana value to target",
		EffectProperties{Outcome: OutcomeDetriment},
		func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
			if len(targets) == 0 {
				return nil
			}
			ctrl := g.GetPlayer(controller)
			if ctrl == nil {
				return nil
			}
			top := g.RevealTopN(ctrl, 1)
			if len(top) == 0 {
				return nil
			}
			amount := top[0].ManaCost().CMC()
			if amount <= 0 {
				return nil
			}
			tid := targets[0]
			if perm := g.FindPermanent(tid); perm != nil {
				g.DealDamageToPermanent(perm, amount, sourceID)
				return nil
			}
			if p := g.GetPlayer(tid); p != nil {
				g.DealDamageToPlayer(p, amount, sourceID)
			}
			return nil
		},
	)
}

// gainLifeEffect gains life for the controller.
type gainLifeEffect struct {
	amount int
}

// GainLife creates an effect that gains life for the controller.
func GainLife(amount int) Effect {
	return DataEffect(&gainLifeEffect{amount: amount})
}

// GainLifeStep returns the EffectData for use in pipelines/ForEach/Modal.
func GainLifeStep(amount int) EffectData { return &gainLifeEffect{amount: amount} }

func (e *gainLifeEffect) EffectText() string {
	return fmt.Sprintf("gain %d life", e.amount)
}
func (e *gainLifeEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit, LifeGain: e.amount}
}

// gainLifeDynamicEffect gains life for the controller from a dynamic ValueSource.
type gainLifeDynamicEffect struct {
	amount ValueSource
}

// GainLifeAmount creates an effect that gains life for the controller equal to a dynamic value.
func GainLifeAmount(amount ValueSource) Effect {
	return DataEffect(&gainLifeDynamicEffect{amount: amount})
}

func (e *gainLifeDynamicEffect) EffectText() string {
	return "gain " + e.amount.Text() + " life"
}
func (e *gainLifeDynamicEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// gainLifeTargetEffect gains life for a target player (or controller as fallback).
type gainLifeTargetEffect struct {
	amount ValueSource
}

// GainLifeTarget creates an effect that gains life for a target player (or controller as fallback).
func GainLifeTarget(amount ValueSource) Effect {
	return DataEffect(&gainLifeTargetEffect{amount: amount})
}

func (e *gainLifeTargetEffect) EffectText() string {
	switch e.amount.(type) {
	case xValue:
		return "target player gains X life"
	case eventAmountValue:
		return "target player gains that much life"
	}
	return fmt.Sprintf("target player gains %d life", e.amount.Resolve(nil, uuid.Nil, uuid.Nil, nil))
}
func (e *gainLifeTargetEffect) EffectProps() EffectProperties {
	lg := 0
	switch e.amount.(type) {
	case xValue, eventAmountValue:
		// dynamic; report 0 for static heuristics
	default:
		lg = e.amount.Resolve(nil, uuid.Nil, uuid.Nil, nil)
	}
	return EffectProperties{Outcome: OutcomeBenefit, LifeGain: lg}
}

// poisonTargetPlayerEffect gives the target player a poison counter.
type poisonTargetPlayerEffect struct {
	amount int
}

// PoisonTargetPlayer creates an effect that gives the target player N poison counters.
func PoisonTargetPlayer(amount int) Effect {
	return DataEffect(&poisonTargetPlayerEffect{amount: amount})
}

func (e *poisonTargetPlayerEffect) EffectText() string {
	return fmt.Sprintf("target player gets %d poison counter(s)", e.amount)
}
func (e *poisonTargetPlayerEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// loseLifeDynamicEffect causes the controller to lose life from a dynamic ValueSource.
type loseLifeDynamicEffect struct {
	amount ValueSource
}

// LoseLifeAmount creates an effect that causes the controller to lose life equal to a dynamic value.
func LoseLifeAmount(amount ValueSource) Effect {
	return DataEffect(&loseLifeDynamicEffect{amount: amount})
}

func (e *loseLifeDynamicEffect) EffectText() string {
	return "lose " + e.amount.Text() + " life"
}
func (e *loseLifeDynamicEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// loseLifeTargetEffect causes a target player (or the controller if no target)
// to lose life equal to a dynamic value. Used for "target opponent loses N life".
type loseLifeTargetEffect struct {
	amount ValueSource
}

// TargetPlayerLoseLife creates an effect that causes the target player (first
// player target) to lose the given amount of life. Falls back to the controller
// if no target is supplied.
func TargetPlayerLoseLife(amount ValueSource) Effect {
	return DataEffect(&loseLifeTargetEffect{amount: amount})
}

func (e *loseLifeTargetEffect) EffectText() string {
	return "target player loses " + e.amount.Text() + " life"
}
func (e *loseLifeTargetEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// loseLifeEffect causes the controller to lose life.
type loseLifeEffect struct {
	amount int
}

// LoseLife creates an effect that causes the controller to lose life.
func LoseLife(amount int) Effect {
	return DataEffect(&loseLifeEffect{amount: amount})
}

func (e *loseLifeEffect) EffectText() string {
	return fmt.Sprintf("you lose %d life", e.amount)
}
func (e *loseLifeEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// dealDamageEffect deals damage to a target (creature, player, or planeswalker).
type dealDamageEffect struct {
	amount ValueSource
}

// DealDamage creates an effect that deals damage to a target.
func DealDamage(amount ValueSource) Effect {
	return DataEffect(&dealDamageEffect{amount: amount})
}

// DealDamageStep returns the EffectData for use in pipelines/ForEach.
func DealDamageStep(amount ValueSource) EffectData { return &dealDamageEffect{amount: amount} }

func (e *dealDamageEffect) EffectText() string {
	if _, ok := e.amount.(xValue); ok {
		return "deal X damage to target"
	}
	return fmt.Sprintf("deal %d damage to target", e.amount.Resolve(nil, uuid.Nil, uuid.Nil, nil))
}
func (e *dealDamageEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, DamageValue: e.amount}
}

// dealDividedDamageEffect deals N damage divided as the controller chose at
// cast/activation time among any number of targets (CR 601.2d). The per-target
// distribution is recorded on the StackObject (DamageDistribution) when the
// spell is cast and is read back at resolution. Used by Flames of the
// Firebrand and similar spells.
type dealDividedDamageEffect struct {
	total ValueSource
}

// DealDividedDamage creates an effect that deals `total` damage divided among
// the spell's targets according to the StackObject.DamageDistribution chosen
// at cast time. Pair with TargetUpToNCreaturesOrPlayers and pass the resulting
// Target to NewMultiTargetSpell.
func DealDividedDamage(total ValueSource) Effect {
	return DataEffect(&dealDividedDamageEffect{total: total})
}

func (e *dealDividedDamageEffect) EffectText() string {
	return fmt.Sprintf("deal %s damage divided as you choose among any number of targets", e.total.Text())
}
func (e *dealDividedDamageEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, DamageValue: e.total}
}

// IsDividedDamageEffect returns true if e is a divided-damage effect (used by
// the spell-cast flow to prompt the controller for a distribution).
func IsDividedDamageEffect(e Effect) bool {
	if a, ok := e.(*dataEffectAdapter); ok {
		_, isDiv := a.data.(*dealDividedDamageEffect)
		return isDiv
	}
	return false
}

// DividedDamageTotal returns the ValueSource carrying the total damage of a
// divided-damage effect, or nil if e is not one.
func DividedDamageTotal(e Effect) ValueSource {
	if a, ok := e.(*dataEffectAdapter); ok {
		if d, isDiv := a.data.(*dealDividedDamageEffect); isDiv {
			return d.total
		}
	}
	return nil
}

// dealDamageToAllCreaturesEffect deals damage to all creatures matching an optional filter.
type dealDamageToAllCreaturesEffect struct {
	amount ValueSource
	filter PermanentFilter
}

// DealDamageToAllCreatures creates an effect that deals damage to all matching creatures.
func DealDamageToAllCreatures(amount ValueSource, filter PermanentFilter) Effect {
	return DataEffect(&dealDamageToAllCreaturesEffect{amount: amount, filter: filter})
}

func (e *dealDamageToAllCreaturesEffect) EffectText() string {
	if _, ok := e.amount.(xValue); ok {
		return "deal X damage to each creature"
	}
	return fmt.Sprintf("deal %d damage to each creature", e.amount.Resolve(nil, uuid.Nil, uuid.Nil, nil))
}
func (e *dealDamageToAllCreaturesEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, DamageValue: e.amount, Mass: true}
}

// dealDamageToPlayersEffect deals damage to players selected by a PlayerSelector.
type dealDamageToPlayersEffect struct {
	amount   ValueSource
	selector PlayerSelector
}

// DealDamageToPlayers creates an effect that deals damage to players selected by the selector.
func DealDamageToPlayers(amount ValueSource, selector PlayerSelector) Effect {
	return DataEffect(&dealDamageToPlayersEffect{amount: amount, selector: selector})
}

// DealDamageToPlayersStep returns the EffectData for use in pipelines/ForEach/Modal.
func DealDamageToPlayersStep(amount ValueSource, selector PlayerSelector) EffectData {
	return &dealDamageToPlayersEffect{amount: amount, selector: selector}
}

func (e *dealDamageToPlayersEffect) EffectText() string {
	return fmt.Sprintf("deal %s damage to %s", e.amount.Text(), e.selector.Text())
}
func (e *dealDamageToPlayersEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, DamageValue: e.amount}
}

// handSizeDamageEffect deals damage to the active player based on hand size vs a threshold.
type handSizeDamageEffect struct {
	threshold int
	above     bool // true = damage for cards above threshold (Black Vise); false = below (The Rack)
}

// HandSizeDamageEffect creates an effect that deals damage to the active player based on
// their hand size relative to a threshold. If above is true, damage = hand - threshold
// (Black Vise: threshold 4). If above is false, damage = threshold - hand (The Rack: threshold 3).
func HandSizeDamageEffect(threshold int, above bool) Effect {
	return DataEffect(&handSizeDamageEffect{threshold: threshold, above: above})
}

// BlackViseEffect creates the Black Vise damage effect (hand size minus 4).
func BlackViseEffect() Effect {
	return HandSizeDamageEffect(4, true)
}

// TheRackEffect creates The Rack damage effect (3 minus hand size).
func TheRackEffect() Effect {
	return HandSizeDamageEffect(3, false)
}

func (e *handSizeDamageEffect) EffectText() string {
	if e.above {
		return fmt.Sprintf("deal damage to active player equal to cards in hand minus %d", e.threshold)
	}
	return fmt.Sprintf("deal damage to active player equal to %d minus cards in hand", e.threshold)
}
func (e *handSizeDamageEffect) EffectProps() EffectProperties { return EffectProperties{} }

// preventAllCombatDamageEffect prevents all combat damage this turn (Fog).
type preventAllCombatDamageEffect struct{}

// PreventAllCombatDamage creates an effect that prevents all combat damage this turn (e.g. Fog).
func PreventAllCombatDamage() Effect {
	return DataEffect(&preventAllCombatDamageEffect{})
}

func (e *preventAllCombatDamageEffect) EffectText() string {
	return "prevent all combat damage that would be dealt this turn"
}
func (e *preventAllCombatDamageEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// preventDamageToTargetEffect sets a damage prevention shield on a target.
type preventDamageToTargetEffect struct {
	amount ValueSource
}

// PreventDamageToTarget creates an effect that prevents damage to a target.
func PreventDamageToTarget(amount ValueSource) Effect {
	return DataEffect(&preventDamageToTargetEffect{amount: amount})
}

func (e *preventDamageToTargetEffect) EffectText() string {
	if _, ok := e.amount.(xValue); ok {
		return "Prevent the next X damage to target"
	}
	return fmt.Sprintf("Prevent the next %d damage to target", e.amount.Resolve(nil, uuid.Nil, uuid.Nil, nil))
}
func (e *preventDamageToTargetEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// sacrificeOrDamageEffect sacrifices a creature you control, or deals damage
// to the source's controller if no creature is available.
type sacrificeOrDamageEffect struct {
	damage int
}

// SacrificeCreatureOrDamage creates an effect where the controller sacrifices a creature,
// or takes damage if no creature is available (e.g. Lord of the Pit upkeep).
func SacrificeCreatureOrDamage(damage int) Effect {
	return DataEffect(&sacrificeOrDamageEffect{damage: damage})
}

func (e *sacrificeOrDamageEffect) EffectText() string {
	return fmt.Sprintf("Sacrifice a creature or take %d damage", e.damage)
}
func (e *sacrificeOrDamageEffect) EffectProps() EffectProperties { return EffectProperties{} }
