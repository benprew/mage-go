package mage

import (
	"fmt"

	"github.com/google/uuid"
	. "github.com/mage/mage/pkg/mage/core"
)

// gainLifeEffect gains life for the controller.
type gainLifeEffect struct {
	amount int
}

// GainLife creates an effect that gains life for the controller.
func GainLife(amount int) Effect {
	return &gainLifeEffect{amount: amount}
}

func (e *gainLifeEffect) Apply(g GameMutator, _, controller uuid.UUID, _ []uuid.UUID) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	g.PlayerGainLife(p, e.amount)
	if !g.IsLichActive(controller) {
		g.FireEvent(GameEvent{Type: EvtLifeGained, PlayerID: controller, Amount: e.amount})
	}
	return nil
}

func (e *gainLifeEffect) Text() string {
	return fmt.Sprintf("gain %d life", e.amount)
}
func (e *gainLifeEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit, LifeGain: e.amount}
}

// gainLifeTargetEffect gains life for a target player (or controller as fallback).
type gainLifeTargetEffect struct {
	amount ValueSource
}

// GainLifeTarget creates an effect that gains life for a target player (or controller as fallback).
func GainLifeTarget(amount ValueSource) Effect {
	return &gainLifeTargetEffect{amount: amount}
}

func (e *gainLifeTargetEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	var targetPlayer Player
	if len(targets) > 0 {
		targetPlayer = g.GetPlayer(targets[0])
	}
	if targetPlayer == nil {
		targetPlayer = g.GetPlayer(controller)
	}
	if targetPlayer == nil {
		return ErrPlayerNotFound
	}
	amount := e.amount.Resolve(g, sourceID, controller)
	g.PlayerGainLife(targetPlayer, amount)
	if !g.IsLichActive(targetPlayer.PlayerID()) {
		g.FireEvent(GameEvent{Type: EvtLifeGained, PlayerID: targetPlayer.PlayerID(), Amount: amount})
	}
	return nil
}

func (e *gainLifeTargetEffect) Text() string {
	if _, ok := e.amount.(xValue); ok {
		return "target player gains X life"
	}
	return fmt.Sprintf("target player gains %d life", e.amount.Resolve(nil, uuid.Nil, uuid.Nil))
}
func (e *gainLifeTargetEffect) Properties() EffectProperties {
	lg := 0
	if _, ok := e.amount.(xValue); !ok {
		lg = e.amount.Resolve(nil, uuid.Nil, uuid.Nil)
	}
	return EffectProperties{Outcome: OutcomeBenefit, LifeGain: lg}
}

// loseLifeEffect causes the controller to lose life.
type loseLifeEffect struct {
	amount int
}

// LoseLife creates an effect that causes the controller to lose life.
func LoseLife(amount int) Effect {
	return &loseLifeEffect{amount: amount}
}

func (e *loseLifeEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	p.LoseLife(e.amount)
	g.FireEvent(GameEvent{Type: EvtLifeLost, PlayerID: controller, Amount: e.amount})
	return nil
}

func (e *loseLifeEffect) Text() string {
	return fmt.Sprintf("you lose %d life", e.amount)
}
func (e *loseLifeEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// dealDamageEffect deals damage to a target (creature, player, or planeswalker).
type dealDamageEffect struct {
	props  EffectProperties
	amount ValueSource
}

// DealDamage creates an effect that deals damage to a target.
func DealDamage(amount ValueSource) Effect {
	return &dealDamageEffect{
		props:  EffectProperties{Outcome: OutcomeDetriment, DamageValue: amount},
		amount: amount,
	}
}

func (e *dealDamageEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for damage")
	}
	amount := e.amount.Resolve(g, sourceID, controller)
	if amount <= 0 {
		return nil
	}
	targetID := targets[0]

	// Check if target is a player
	for _, pl := range g.AllPlayers() {
		if pl.PlayerID() == targetID {
			g.DealDamageToPlayer(pl, amount, sourceID)
			return nil
		}
	}

	// Otherwise target is a permanent
	perm := g.FindPermanent(targetID)
	if perm == nil {
		return nil // target gone, fizzle
	}

	// Check protection
	sourceCard := g.FindCardAnywhere(sourceID)
	if sourceCard != nil && perm.HasProtectionFrom(sourceCard) {
		return nil // damage prevented by protection
	}

	g.DealDamageToPermanent(perm, amount, sourceID)
	return nil
}

func (e *dealDamageEffect) Text() string {
	if _, ok := e.amount.(xValue); ok {
		return "deal X damage to target"
	}
	return fmt.Sprintf("deal %d damage to target", e.amount.Resolve(nil, uuid.Nil, uuid.Nil))
}
func (e *dealDamageEffect) Properties() EffectProperties { return e.props }

// dealDamageToAllCreaturesEffect deals damage to all creatures matching an optional filter.
type dealDamageToAllCreaturesEffect struct {
	props  EffectProperties
	amount ValueSource
	filter PermanentFilter
}

// DealDamageToAllCreatures creates an effect that deals damage to all matching creatures.
func DealDamageToAllCreatures(amount ValueSource, filter PermanentFilter) Effect {
	return &dealDamageToAllCreaturesEffect{
		props:  EffectProperties{Outcome: OutcomeDetriment, DamageValue: amount, Mass: true},
		amount: amount, filter: filter,
	}
}

func (e *dealDamageToAllCreaturesEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	amount := e.amount.Resolve(g, sourceID, controller)
	if amount <= 0 {
		return nil
	}
	f := IsCreature
	if !e.filter.IsZero() {
		f = And(IsCreature, e.filter)
	}
	for _, p := range g.FilterBattlefield(f) {
		g.DealDamageToPermanent(p, amount, sourceID)
	}
	return nil
}

func (e *dealDamageToAllCreaturesEffect) Text() string {
	if _, ok := e.amount.(xValue); ok {
		return "deal X damage to each creature"
	}
	return fmt.Sprintf("deal %d damage to each creature", e.amount.Resolve(nil, uuid.Nil, uuid.Nil))
}
func (e *dealDamageToAllCreaturesEffect) Properties() EffectProperties { return e.props }

// dealDamageToPlayersEffect deals damage to players selected by a PlayerSelector.
type dealDamageToPlayersEffect struct {
	amount   ValueSource
	selector PlayerSelector
}

// DealDamageToPlayers creates an effect that deals damage to players selected by the selector.
func DealDamageToPlayers(amount ValueSource, selector PlayerSelector) Effect {
	return &dealDamageToPlayersEffect{amount: amount, selector: selector}
}

func (e *dealDamageToPlayersEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	amount := e.amount.Resolve(g, sourceID, controller)
	if amount <= 0 {
		return nil
	}
	playerIDs := e.selector.Select(g, sourceID, controller, targets)
	for _, pid := range playerIDs {
		p := g.GetPlayer(pid)
		if p != nil {
			g.DealDamageToPlayer(p, amount, sourceID)
		}
	}
	return nil
}

func (e *dealDamageToPlayersEffect) Text() string {
	return fmt.Sprintf("deal %s damage to %s", e.amount.Text(), e.selector.Text())
}
func (e *dealDamageToPlayersEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, DamageValue: e.amount}
}

// blackViseEffect deals damage to the active player based on hand size > 4.
type blackViseEffect struct{}

// BlackViseEffect creates an effect that deals damage to the active player equal to
// cards in hand minus 4 (Black Vise).
func BlackViseEffect() Effect {
	return &blackViseEffect{}
}

func (e *blackViseEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	active := g.ActivePlayerObj()
	handSize := len(active.Hand())
	if handSize > 4 {
		damage := handSize - 4
		active.LoseLife(damage)
	}
	return nil
}

func (e *blackViseEffect) Text() string {
	return "Deal damage to active player equal to cards in hand minus 4"
}
func (e *blackViseEffect) Properties() EffectProperties { return EffectProperties{} }

// preventAllCombatDamageEffect prevents all combat damage this turn (Fog).
type preventAllCombatDamageEffect struct{}

// PreventAllCombatDamage creates an effect that prevents all combat damage this turn (e.g. Fog).
func PreventAllCombatDamage() Effect {
	return &preventAllCombatDamageEffect{}
}

func (e *preventAllCombatDamageEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	g.SetPreventCombatDamage()
	return nil
}

func (e *preventAllCombatDamageEffect) Text() string {
	return "prevent all combat damage that would be dealt this turn"
}
func (e *preventAllCombatDamageEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// preventDamageToTargetEffect sets a damage prevention shield on a target.
type preventDamageToTargetEffect struct {
	amount ValueSource
}

// PreventDamageToTarget creates an effect that prevents damage to a target.
func PreventDamageToTarget(amount ValueSource) Effect {
	return &preventDamageToTargetEffect{amount: amount}
}

func (e *preventDamageToTargetEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	amount := e.amount.Resolve(g, sourceID, controller)
	perm := g.FindPermanent(targets[0])
	if perm != nil {
		g.AddPreventionShield(perm.ID(), amount)
		return nil
	}
	// Prevent damage to player
	player := g.GetPlayer(targets[0])
	if player != nil {
		g.AddPreventionShield(player.PlayerID(), amount)
	}
	return nil
}

func (e *preventDamageToTargetEffect) Text() string {
	if _, ok := e.amount.(xValue); ok {
		return "Prevent the next X damage to target"
	}
	return fmt.Sprintf("Prevent the next %d damage to target", e.amount.Resolve(nil, uuid.Nil, uuid.Nil))
}
func (e *preventDamageToTargetEffect) Properties() EffectProperties {
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
	return &sacrificeOrDamageEffect{damage: damage}
}

func (e *sacrificeOrDamageEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	// Collect creatures that can be sacrificed (not the source itself)
	candidates := g.FilterBattlefield(And(ControlledBy(controller), IsCreature, NotID(sourceID)))
	if len(candidates) > 0 {
		player := g.GetPlayer(controller)
		chosen := player.ChoosePermanent(candidates, "sacrifice", g)
		if chosen != nil {
			g.Sacrifice(chosen)
			return nil
		}
	}
	// No creature available - deal damage to controller
	player := g.GetPlayer(controller)
	if player != nil {
		g.DealDamageToPlayer(player, e.damage, sourceID)
	}
	return nil
}

func (e *sacrificeOrDamageEffect) Text() string {
	return fmt.Sprintf("Sacrifice a creature or take %d damage", e.damage)
}
func (e *sacrificeOrDamageEffect) Properties() EffectProperties { return EffectProperties{} }
