package mage

import (
	"fmt"

	"github.com/google/uuid"
)

// gainLifeEffect gains life for the controller.
type gainLifeEffect struct {
	amount int
}

// GainLife creates an effect that gains life for the controller.
func GainLife(amount int) Effect {
	return DataEffect(&gainLifeEffect{amount: amount})
}

func (e *gainLifeEffect) EffectText() string {
	return fmt.Sprintf("gain %d life", e.amount)
}
func (e *gainLifeEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit, LifeGain: e.amount}
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
	if _, ok := e.amount.(xValue); ok {
		return "target player gains X life"
	}
	return fmt.Sprintf("target player gains %d life", e.amount.Resolve(nil, uuid.Nil, uuid.Nil, nil))
}
func (e *gainLifeTargetEffect) EffectProps() EffectProperties {
	lg := 0
	if _, ok := e.amount.(xValue); !ok {
		lg = e.amount.Resolve(nil, uuid.Nil, uuid.Nil, nil)
	}
	return EffectProperties{Outcome: OutcomeBenefit, LifeGain: lg}
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

func (e *dealDamageEffect) EffectText() string {
	if _, ok := e.amount.(xValue); ok {
		return "deal X damage to target"
	}
	return fmt.Sprintf("deal %d damage to target", e.amount.Resolve(nil, uuid.Nil, uuid.Nil, nil))
}
func (e *dealDamageEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, DamageValue: e.amount}
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
