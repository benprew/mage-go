package mage

import (
	"fmt"

	. "github.com/benprew/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// gainLifeEffect gains life for the controller.
type gainLifeEffect struct {
	amount int
}

// GainLife creates an effect that gains life for the controller.
func GainLife(amount int) Effect {
	return &gainLifeEffect{amount: amount}
}

// GainLifeStep returns the Effect for use in pipelines/ForEach/Modal.
func GainLifeStep(amount int) Effect { return &gainLifeEffect{amount: amount} }

func (e *gainLifeEffect) Text() string {
	return fmt.Sprintf("gain %d life", e.amount)
}
func (e *gainLifeEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit, LifeGain: e.amount}
}

// gainLifeDynamicEffect gains life for the controller from a dynamic ValueSource.
type gainLifeDynamicEffect struct {
	amount ValueSource
}

// GainLifeAmount creates an effect that gains life for the controller equal to a dynamic value.
func GainLifeAmount(amount ValueSource) Effect {
	return &gainLifeDynamicEffect{amount: amount}
}

func (e *gainLifeDynamicEffect) Text() string {
	return "gain " + e.amount.Text() + " life"
}
func (e *gainLifeDynamicEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// gainLifeTargetEffect gains life for a target player (or controller as fallback).
type gainLifeTargetEffect struct {
	amount ValueSource
}

// GainLifeTarget creates an effect that gains life for a target player (or controller as fallback).
func GainLifeTarget(amount ValueSource) Effect {
	return &gainLifeTargetEffect{amount: amount}
}

func (e *gainLifeTargetEffect) Text() string {
	switch e.amount.(type) {
	case xValue:
		return "target player gains X life"
	case eventAmountValue:
		return "target player gains that much life"
	}
	return fmt.Sprintf("target player gains %d life", e.amount.Resolve(nil, uuid.Nil, uuid.Nil, nil))
}
func (e *gainLifeTargetEffect) Properties() EffectProperties {
	lg := 0
	switch e.amount.(type) {
	case xValue, eventAmountValue:
		// dynamic; report 0 for static heuristics
	default:
		lg = e.amount.Resolve(nil, uuid.Nil, uuid.Nil, nil)
	}
	return EffectProperties{Outcome: OutcomeBenefit, LifeGain: lg}
}

// poisonTargetPlayerEffect gives a player poison counters. By default the
// player is read from ctx.Targets[0]. Use Targeting to override with a
// PlayerSelector.
type poisonTargetPlayerEffect struct {
	amount int
	sel    PlayerSelector
}

// PoisonTargetPlayer creates an effect that gives the target player N poison
// counters. Chain .Targeting(sel) to pick the player(s) via a PlayerSelector.
func PoisonTargetPlayer(amount int) *poisonTargetPlayerEffect {
	return &poisonTargetPlayerEffect{amount: amount}
}

// Targeting overrides the default target (ctx.Targets[0]) with a PlayerSelector.
func (e *poisonTargetPlayerEffect) Targeting(sel PlayerSelector) *poisonTargetPlayerEffect {
	e.sel = sel
	return e
}

func (e *poisonTargetPlayerEffect) Text() string {
	return fmt.Sprintf("target player gets %d poison counter(s)", e.amount)
}
func (e *poisonTargetPlayerEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// loseLifeDynamicEffect causes the controller to lose life from a dynamic ValueSource.
type loseLifeDynamicEffect struct {
	amount ValueSource
}

// LoseLifeAmount creates an effect that causes the controller to lose life equal to a dynamic value.
func LoseLifeAmount(amount ValueSource) Effect {
	return &loseLifeDynamicEffect{amount: amount}
}

func (e *loseLifeDynamicEffect) Text() string {
	return "lose " + e.amount.Text() + " life"
}
func (e *loseLifeDynamicEffect) Properties() EffectProperties {
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
	return &loseLifeTargetEffect{amount: amount}
}

func (e *loseLifeTargetEffect) Text() string {
	return "target player loses " + e.amount.Text() + " life"
}
func (e *loseLifeTargetEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// loseLifeEffect causes the controller to lose life.
type loseLifeEffect struct {
	amount int
}

// LoseLife creates an effect that causes the controller to lose life.
func LoseLife(amount int) Effect {
	return &loseLifeEffect{amount: amount}
}

func (e *loseLifeEffect) Text() string {
	return fmt.Sprintf("you lose %d life", e.amount)
}
func (e *loseLifeEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// dealDamageEffect deals damage to a target (creature, player, or planeswalker).
type dealDamageEffect struct {
	amount ValueSource
}

// DealDamage creates an effect that deals damage to a target.
func DealDamage(amount ValueSource) Effect {
	return &dealDamageEffect{amount: amount}
}

// DealDamageStep returns the Effect for use in pipelines/ForEach.
func DealDamageStep(amount ValueSource) Effect { return &dealDamageEffect{amount: amount} }

func (e *dealDamageEffect) Text() string {
	if _, ok := e.amount.(xValue); ok {
		return "deal X damage to target"
	}
	return fmt.Sprintf("deal %d damage to target", e.amount.Resolve(nil, uuid.Nil, uuid.Nil, nil))
}
func (e *dealDamageEffect) Properties() EffectProperties {
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
	return &dealDividedDamageEffect{total: total}
}

func (e *dealDividedDamageEffect) Text() string {
	return fmt.Sprintf("deal %s damage divided as you choose among any number of targets", e.total.Text())
}
func (e *dealDividedDamageEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, DamageValue: e.total}
}

// IsDividedDamageEffect returns true if e is a divided-damage effect (used by
// the spell-cast flow to prompt the controller for a distribution).
func IsDividedDamageEffect(e Effect) bool {
	_, isDiv := e.(*dealDividedDamageEffect)
	return isDiv
}

// DividedDamageTotal returns the ValueSource carrying the total damage of a
// divided-damage effect, or nil if e is not one.
func DividedDamageTotal(e Effect) ValueSource {
	if d, isDiv := e.(*dealDividedDamageEffect); isDiv {
		return d.total
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
	return &dealDamageToAllCreaturesEffect{amount: amount, filter: filter}
}

func (e *dealDamageToAllCreaturesEffect) Text() string {
	if _, ok := e.amount.(xValue); ok {
		return "deal X damage to each creature"
	}
	return fmt.Sprintf("deal %d damage to each creature", e.amount.Resolve(nil, uuid.Nil, uuid.Nil, nil))
}
func (e *dealDamageToAllCreaturesEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, DamageValue: e.amount, Mass: true}
}

// dealDamageToPlayersEffect deals damage to players selected by a PlayerSelector.
type dealDamageToPlayersEffect struct {
	amount   ValueSource
	selector PlayerSelector
}

// DealDamageToPlayers creates an effect that deals damage to players selected by the selector.
func DealDamageToPlayers(amount ValueSource, selector PlayerSelector) Effect {
	return &dealDamageToPlayersEffect{amount: amount, selector: selector}
}

// DealDamageToPlayersStep returns the Effect for use in pipelines/ForEach/Modal.
func DealDamageToPlayersStep(amount ValueSource, selector PlayerSelector) Effect {
	return &dealDamageToPlayersEffect{amount: amount, selector: selector}
}

func (e *dealDamageToPlayersEffect) Text() string {
	return fmt.Sprintf("deal %s damage to %s", e.amount.Text(), e.selector.Text())
}
func (e *dealDamageToPlayersEffect) Properties() EffectProperties {
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
	return &handSizeDamageEffect{threshold: threshold, above: above}
}

// BlackViseEffect creates the Black Vise damage effect (hand size minus 4).
func BlackViseEffect() Effect {
	return HandSizeDamageEffect(4, true)
}

// TheRackEffect creates The Rack damage effect (3 minus hand size).
func TheRackEffect() Effect {
	return HandSizeDamageEffect(3, false)
}

func (e *handSizeDamageEffect) Text() string {
	if e.above {
		return fmt.Sprintf("deal damage to active player equal to cards in hand minus %d", e.threshold)
	}
	return fmt.Sprintf("deal damage to active player equal to %d minus cards in hand", e.threshold)
}
func (e *handSizeDamageEffect) Properties() EffectProperties { return EffectProperties{} }

// preventAllCombatDamageEffect prevents all combat damage this turn (Fog).
type preventAllCombatDamageEffect struct{}

// PreventAllCombatDamage creates an effect that prevents all combat damage this turn (e.g. Fog).
func PreventAllCombatDamage() Effect {
	return &preventAllCombatDamageEffect{}
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

func (e *preventDamageToTargetEffect) Text() string {
	if _, ok := e.amount.(xValue); ok {
		return "Prevent the next X damage to target"
	}
	return fmt.Sprintf("Prevent the next %d damage to target", e.amount.Resolve(nil, uuid.Nil, uuid.Nil, nil))
}
func (e *preventDamageToTargetEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

type preventDamageToSourceEffect struct {
	amount ValueSource
}

func (e *preventDamageToTargetEffect) Apply(ctx *EffectContext) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	amount := e.amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	perm := ctx.Game.FindPermanent(ctx.Targets[0])
	if perm != nil {
		ctx.Game.AddPreventionShield(perm.ID(), amount)
		return nil
	}
	player := ctx.Game.GetPlayer(ctx.Targets[0])
	if player != nil {
		ctx.Game.AddPreventionShield(player.PlayerID(), amount)
	}
	return nil
}

// PreventDamageToSource creates an effect that prevents the next damage to
// the permanent that is the source of the resolving ability.
func PreventDamageToSource(amount ValueSource) Effect {
	return &preventDamageToSourceEffect{amount: amount}
}

func (e *preventDamageToSourceEffect) Text() string {
	if _, ok := e.amount.(xValue); ok {
		return "Prevent the next X damage to source"
	}
	return fmt.Sprintf("Prevent the next %d damage to source", e.amount.Resolve(nil, uuid.Nil, uuid.Nil, nil))
}

func (e *preventDamageToSourceEffect) Properties() EffectProperties {
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

func (e *sacrificeOrDamageEffect) Text() string {
	return fmt.Sprintf("Sacrifice a creature or take %d damage", e.damage)
}
func (e *sacrificeOrDamageEffect) Properties() EffectProperties { return EffectProperties{} }

// --- Apply methods (moved from executor.go) ---

func (e *gainLifeEffect) Apply(ctx *EffectContext) error {
	p := ctx.Game.GetPlayer(ctx.Controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	ctx.Game.PlayerGainLife(p, e.amount)
	if !ctx.Game.IsLichActive(ctx.Controller) {
		ctx.Game.FireEvent(GameEvent{Type: EvtLifeGained, PlayerID: ctx.Controller, Amount: e.amount})
	}
	return nil
}

func (e *poisonTargetPlayerEffect) Apply(ctx *EffectContext) error {
	var playerIDs []uuid.UUID
	switch {
	case e.sel != nil:
		playerIDs = e.sel.Select(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	case len(ctx.Targets) > 0:
		playerIDs = []uuid.UUID{ctx.Targets[0]}
	}

	for _, pid := range playerIDs {
		p := ctx.Game.GetPlayer(pid)
		if p == nil {
			continue
		}
		p.AddPoisonCounters(e.amount)
	}
	return nil
}

func (e *gainLifeDynamicEffect) Apply(ctx *EffectContext) error {
	amount := e.amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	if amount <= 0 {
		return nil
	}
	p := ctx.Game.GetPlayer(ctx.Controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	ctx.Game.PlayerGainLife(p, amount)
	if !ctx.Game.IsLichActive(ctx.Controller) {
		ctx.Game.FireEvent(GameEvent{Type: EvtLifeGained, PlayerID: ctx.Controller, Amount: amount})
	}
	return nil
}

func (e *gainLifeTargetEffect) Apply(ctx *EffectContext) error {
	var targetPlayer Player
	if len(ctx.Targets) > 0 {
		targetPlayer = ctx.Game.GetPlayer(ctx.Targets[0])
	}
	if targetPlayer == nil {
		targetPlayer = ctx.Game.GetPlayer(ctx.Controller)
	}
	if targetPlayer == nil {
		return ErrPlayerNotFound
	}
	amount := e.amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	ctx.Game.PlayerGainLife(targetPlayer, amount)
	if !ctx.Game.IsLichActive(targetPlayer.PlayerID()) {
		ctx.Game.FireEvent(GameEvent{Type: EvtLifeGained, PlayerID: targetPlayer.PlayerID(), Amount: amount})
	}
	return nil
}

func (e *loseLifeDynamicEffect) Apply(ctx *EffectContext) error {
	amount := e.amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	if amount <= 0 {
		return nil
	}
	p := ctx.Game.GetPlayer(ctx.Controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	ctx.Game.PlayerLoseLife(p, amount)
	return nil
}

func (e *loseLifeEffect) Apply(ctx *EffectContext) error {
	p := ctx.Game.GetPlayer(ctx.Controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	ctx.Game.PlayerLoseLife(p, e.amount)
	return nil
}

func (e *loseLifeTargetEffect) Apply(ctx *EffectContext) error {
	var targetPlayer Player
	if len(ctx.Targets) > 0 {
		targetPlayer = ctx.Game.GetPlayer(ctx.Targets[0])
	}
	if targetPlayer == nil {
		targetPlayer = ctx.Game.GetPlayer(ctx.Controller)
	}
	if targetPlayer == nil {
		return ErrPlayerNotFound
	}
	amount := e.amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	if amount <= 0 {
		return nil
	}
	ctx.Game.PlayerLoseLife(targetPlayer, amount)
	return nil
}

func (e *dealDamageEffect) Apply(ctx *EffectContext) error {
	if len(ctx.Targets) == 0 {
		return fmt.Errorf("no target for damage")
	}
	amount := e.amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	if amount <= 0 {
		return nil
	}
	targetID := ctx.Targets[0]

	for _, pl := range ctx.Game.AllPlayers() {
		if pl.PlayerID() == targetID {
			ctx.Game.DealDamageToPlayer(pl, amount, ctx.SourceID)
			return nil
		}
	}

	perm := ctx.Game.FindPermanent(targetID)
	if perm == nil {
		return nil
	}

	sourceCard := ctx.Game.FindCardAnywhere(ctx.SourceID)
	if sourceCard != nil && perm.HasProtectionFrom(sourceCard) {
		return nil
	}

	ctx.Game.DealDamageToPermanent(perm, amount, ctx.SourceID)
	return nil
}

// execDealDividedDamage reads the per-target damage distribution chosen at
// cast/activation time (stored on the StackObject and forwarded into ctx via
// resolvingDamageDistribution) and applies it. CR 601.2d/609.3.5: a divided-
// damage spell's distribution is chosen on announcement and is frozen; if a
// target later becomes illegal, only that target's share is wasted — the
// remaining targets still take their shares. We honor that by skipping
// ctx.Targets entries that no longer point at a valid creature/player.
func (*dealDividedDamageEffect) Apply(ctx *EffectContext) error {
	dist := ctx.DamageDistribution
	if dist == nil {
		dist = ctx.Game.resolvingDamageDistribution
	}
	if len(dist) == 0 {
		return nil
	}
	for _, tid := range ctx.Targets {
		amt, ok := dist[tid]
		if !ok || amt <= 0 {
			continue
		}
		applied := false
		for _, pl := range ctx.Game.AllPlayers() {
			if pl.PlayerID() == tid {
				ctx.Game.DealDamageToPlayer(pl, amt, ctx.SourceID)
				applied = true
				break
			}
		}
		if applied {
			continue
		}
		perm := ctx.Game.FindPermanent(tid)
		if perm == nil {
			continue
		}
		sourceCard := ctx.Game.FindCardAnywhere(ctx.SourceID)
		if sourceCard != nil && perm.HasProtectionFrom(sourceCard) {
			continue
		}
		ctx.Game.DealDamageToPermanent(perm, amt, ctx.SourceID)
	}
	return nil
}

func (e *dealDamageToAllCreaturesEffect) Apply(ctx *EffectContext) error {
	amount := e.amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	if amount <= 0 {
		return nil
	}
	f := IsCreature
	if !e.filter.IsZero() {
		f = And(IsCreature, e.filter)
	}
	for _, p := range ctx.Game.FilterBattlefield(f) {
		ctx.Game.DealDamageToPermanent(p, amount, ctx.SourceID)
	}
	return nil
}

func (e *dealDamageToPlayersEffect) Apply(ctx *EffectContext) error {
	amount := e.amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	if amount <= 0 {
		return nil
	}
	playerIDs := e.selector.Select(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	for _, pid := range playerIDs {
		p := ctx.Game.GetPlayer(pid)
		if p != nil {
			ctx.Game.DealDamageToPlayer(p, amount, ctx.SourceID)
		}
	}
	return nil
}

func (e *handSizeDamageEffect) Apply(ctx *EffectContext) error {
	var p Player
	if len(ctx.Targets) > 0 {
		p = ctx.Game.GetPlayer(ctx.Targets[0])
	}
	if p == nil {
		p = ctx.Game.ActivePlayerObj()
	}
	handSize := len(p.Hand())
	var damage int
	if e.above {
		damage = handSize - e.threshold
	} else {
		damage = e.threshold - handSize
	}
	if damage > 0 {
		ctx.Game.DealDamageToPlayer(p, damage, ctx.SourceID)
	}
	return nil
}

func (*preventAllCombatDamageEffect) Apply(ctx *EffectContext) error {
	ctx.Game.SetPreventCombatDamage()
	return nil
}

func (e *preventDamageToSourceEffect) Apply(ctx *EffectContext) error {
	if ctx.Game.FindPermanent(ctx.SourceID) == nil {
		return nil
	}
	amount := e.amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	ctx.Game.AddPreventionShield(ctx.SourceID, amount)
	return nil
}

func (e *sacrificeOrDamageEffect) Apply(ctx *EffectContext) error {
	candidates := ctx.Game.FilterBattlefield(And(ControlledBy(ctx.Controller), IsCreature, NotID(ctx.SourceID)))
	if len(candidates) > 0 {
		player := ctx.Game.GetPlayer(ctx.Controller)
		chosen := player.ChoosePermanent(candidates, "sacrifice", ctx.Game)
		if chosen != nil {
			ctx.Game.Sacrifice(chosen)
			return nil
		}
	}
	player := ctx.Game.GetPlayer(ctx.Controller)
	if player != nil {
		ctx.Game.DealDamageToPlayer(player, e.damage, ctx.SourceID)
	}
	return nil
}
