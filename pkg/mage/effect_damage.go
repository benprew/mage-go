package mage

import (
	"fmt"

	"github.com/google/uuid"
)

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
	switch e.(type) {
	case *dealDividedDamageEffect, *allocatedDamageEffect:
		return true
	case *sequenceEffect:
		return singleAllocatedDamage(e) != nil
	}
	return false
}

// DividedDamageTotal returns the ValueSource carrying the total damage of a
// divided-damage effect, or nil if e is not one.
func DividedDamageTotal(e Effect) ValueSource {
	switch d := e.(type) {
	case *dealDividedDamageEffect:
		return d.total
	case *allocatedDamageEffect:
		return d.allocationValue()
	case *sequenceEffect:
		if allocation := singleAllocatedDamage(d); allocation != nil {
			return allocation.allocationValue()
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

func (e *preventAllCombatDamageEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "install-combat-prevention", Text: e.Text()}
}

type preventDamageToYouByCreaturesWithFlyingEffect struct{}

// PreventDamageToYouByCreaturesWithFlying prevents all damage that would be dealt to the controller this turn by creatures with flying.
func PreventDamageToYouByCreaturesWithFlying() Effect {
	return &preventDamageToYouByCreaturesWithFlyingEffect{}
}

func (e *preventDamageToYouByCreaturesWithFlyingEffect) Text() string {
	return "prevent all damage that would be dealt to you this turn by creatures with flying"
}

func (e *preventDamageToYouByCreaturesWithFlyingEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

func (e *preventDamageToYouByCreaturesWithFlyingEffect) Apply(ctx *EffectContext) error {
	ctx.Game.PreventDamageToPlayerByCreaturesWithFlying(ctx.Controller)
	return nil
}

// PreventDamageEffect sets a damage prevention shield on selected permanents
// or the targeted player.
type PreventDamageEffect struct {
	amount   ValueSource
	selector TargetSelector
}

// PreventDamageToTarget creates an effect that prevents damage to a target.
// Chain Targeting to select a permanent without adding a rules target.
func PreventDamageToTarget(amount ValueSource) *PreventDamageEffect {
	return &PreventDamageEffect{
		amount:   amount,
		selector: ToTarget(),
	}
}

func (e *PreventDamageEffect) Targeting(selector TargetSelector) *PreventDamageEffect {
	e.selector = selector
	return e
}

func (e *PreventDamageEffect) Text() string {
	object := "target"
	if e.selector.Kind == KindSource {
		object = "source"
	}
	if _, ok := e.amount.(xValue); ok {
		return "Prevent the next X damage to " + object
	}
	return fmt.Sprintf("Prevent the next %d damage to %s", e.amount.Resolve(nil, uuid.Nil, uuid.Nil, nil), object)
}
func (e *PreventDamageEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

func (e *PreventDamageEffect) Apply(ctx *EffectContext) error {
	amount := e.amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	for _, perm := range resolvePermanents(ctx, e.selector) {
		ctx.Game.AddPreventionShield(perm.ID(), amount)
	}
	if e.selector.Kind == KindTarget && len(ctx.Targets) > 0 {
		player := ctx.Game.GetPlayer(ctx.Targets[0])
		if player != nil {
			ctx.Game.AddPreventionShield(player.PlayerID(), amount)
		}
	}
	return nil
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
	ctx.Game.gainLifeWithEvent(targetPlayer, amount)
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
	if sourceCard != nil && perm.HasProtectionFromInGame(sourceCard, ctx.Game) {
		return nil
	}

	ctx.Game.DealDamageToPermanent(perm, amount, ctx.SourceID)
	return nil
}

// Illegal targets keep no share. The remaining shares form one damage instruction.
func (*dealDividedDamageEffect) Apply(ctx *EffectContext) error {
	dist := ctx.DamageDistribution
	if dist == nil {
		dist = ctx.Game.ResolvingDamageDistribution()
	}
	if len(dist) == 0 {
		return nil
	}
	source := ctx.Game.CaptureDamageSource(ctx.SourceID)
	if !source.Valid() {
		source.id = ctx.SourceID
	}
	assignments := make([]DamageAssignment, 0, len(ctx.Targets))
	for _, targetID := range ctx.Targets {
		amount := dist[targetID]
		if amount <= 0 {
			continue
		}
		if player := ctx.Game.GetPlayer(targetID); player != nil {
			assignments = append(assignments, PlayerDamageAssignment(player, amount, source))
		} else if permanent := ctx.Game.FindPermanent(targetID); permanent != nil {
			assignments = append(assignments, PermanentDamageAssignment(permanent, amount, source))
		}
	}
	ctx.Game.DealSimultaneousDamage(assignments...)
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

func (e *sacrificeOrDamageEffect) Apply(ctx *EffectContext) error {
	candidates := ctx.Game.FilterBattlefield(And(ControlledBy(ctx.Controller), IsCreature, NotID(ctx.SourceID)))
	if len(candidates) > 0 {
		player := ctx.Game.GetPlayer(ctx.Controller)
		chosen := player.ChoosePermanent(candidates, "sacrifice", ctx.Game)
		if chosen != nil {
			ctx.Game.DoSacrifice(chosen)
			return nil
		}
	}
	player := ctx.Game.GetPlayer(ctx.Controller)
	if player != nil {
		ctx.Game.DealDamageToPlayer(player, e.damage, ctx.SourceID)
	}
	return nil
}
