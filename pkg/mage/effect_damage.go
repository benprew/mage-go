package mage

import (
	"fmt"
	"uuid"
)

// GainLife creates an effect that gains life for the controller.
func GainLife(amount int) *selectedLifeEffect {
	return &selectedLifeEffect{player: SelectController(), amount: Fixed(amount)}
}

// GainLifeForPlayers creates an effect that gains life for each selected player.
func GainLifeForPlayers(player PlayerSelector, amount ValueSource) *selectedLifeEffect {
	return &selectedLifeEffect{player: player, amount: amount}
}

// GainLifeStep returns the Effect for use in pipelines/ForEach/Modal.
func GainLifeStep(amount int) Effect { return GainLife(amount) }

// GainLifeAmount creates an effect that gains life for the controller equal to a dynamic value.
func GainLifeAmount(amount ValueSource) Effect {
	return GainLifeForPlayers(SelectController(), amount)
}

// GainLifeTarget creates an effect that gains life for a target player (or controller as fallback).
func GainLifeTarget(amount ValueSource) Effect {
	return GainLifeForPlayers(legacyPlayerReference{}, amount)
}

// poisonTargetPlayerEffect gives a player poison counters. By default the
// player is read from ctx.legacyTargets()[0]. Use Targeting to override with a
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

// Targeting overrides the default target (ctx.legacyTargets()[0]) with a PlayerSelector.
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

// LoseLifeAmount creates an effect that causes the controller to lose life equal to a dynamic value.
func LoseLifeAmount(amount ValueSource) Effect {
	return LoseLifeForPlayers(SelectController(), amount)
}

// TargetPlayerLoseLife creates an effect that causes the target player (first
// player target) to lose the given amount of life. Falls back to the controller
// if no target is supplied.
func TargetPlayerLoseLife(amount ValueSource) Effect {
	return LoseLifeForPlayers(legacyPlayerReference{}, amount)
}

// LoseLife creates an effect that causes the controller to lose life.
func LoseLife(amount int) *selectedLifeEffect {
	return &selectedLifeEffect{player: SelectController(), amount: Fixed(amount), lose: true}
}

// LoseLifeForPlayers creates an effect that causes each selected player to lose life.
func LoseLifeForPlayers(player PlayerSelector, amount ValueSource) *selectedLifeEffect {
	return &selectedLifeEffect{player: player, amount: amount, lose: true}
}

// dealDamageEffect deals damage to a target (creature, player, or planeswalker).
type dealDamageEffect struct {
	mass      bool
	source    ObjectSelector
	recipient DamageRecipientSelector
	amount    ValueSource
}

// DealDamage creates an effect that deals damage from the resolving source to the first target.
func DealDamage(amount ValueSource) *dealDamageEffect {
	return &dealDamageEffect{amount: amount, source: legacyDamageSourceReference{}, recipient: legacyFirstReference{}}
}

// DealDamageFrom creates an effect that deals damage from source to recipient.
func DealDamageFrom(source ObjectSelector, recipient DamageRecipientSelector, amount ValueSource) *dealDamageEffect {
	_, mass := recipient.(battlefieldObjects)
	return &dealDamageEffect{source: source, recipient: recipient, amount: amount, mass: mass}
}

func (e *dealDamageEffect) From(source ObjectSelector) *dealDamageEffect {
	e.source = source
	return e
}

func (e *dealDamageEffect) Targeting(recipient DamageRecipientSelector) *dealDamageEffect {
	e.recipient = recipient
	_, e.mass = recipient.(battlefieldObjects)
	return e
}

// DealDamageStep returns the Effect for use in pipelines/ForEach.
func DealDamageStep(amount ValueSource) Effect { return DealDamage(amount) }

func (e *dealDamageEffect) Text() string {
	return "deal " + e.amount.Text() + " damage to " + e.recipient.Text()
}
func (e *dealDamageEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, DamageValue: e.amount, Mass: e.mass}
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

// dealDamageDividedEvenlyEffect deals N damage divided evenly, rounded down, among
// all remaining legal targets as the spell or ability resolves (CR 107.1c, Fireball).
type dealDamageDividedEvenlyEffect struct {
	total ValueSource
}

// DealDamageDividedEvenly creates an effect that divides total damage evenly, rounded down,
// among all legal targets on resolution.
func DealDamageDividedEvenly(total ValueSource) Effect {
	return &dealDamageDividedEvenlyEffect{total: total}
}

func (e *dealDamageDividedEvenlyEffect) Text() string {
	return fmt.Sprintf("deals %s damage divided evenly, rounded down, among any number of targets", e.total.Text())
}

func (e *dealDamageDividedEvenlyEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, DamageValue: e.total}
}

func (e *dealDamageDividedEvenlyEffect) Apply(ctx *EffectContext) error {
	totalDamage := e.total.Resolve(ctx)
	if totalDamage <= 0 {
		return nil
	}

	targets := ctx.legacyTargets()
	if len(targets) == 0 {
		return nil
	}

	var legalTargets []uuid.UUID
	sourceCard := ctx.Game.FindCardAnywhere(ctx.SourceID)
	for _, tid := range targets {
		if pl := ctx.Game.GetPlayer(tid); pl != nil {
			legalTargets = append(legalTargets, tid)
			continue
		}
		if perm := ctx.Game.FindPermanent(tid); perm != nil {
			if sourceCard != nil && perm.HasProtectionFromInGame(sourceCard, ctx.Game) {
				continue
			}
			legalTargets = append(legalTargets, tid)
		}
	}

	if len(legalTargets) == 0 {
		return nil
	}

	damagePerTarget := totalDamage / len(legalTargets)
	if damagePerTarget <= 0 {
		return nil
	}

	for _, tid := range legalTargets {
		if pl := ctx.Game.GetPlayer(tid); pl != nil {
			ctx.Game.DealDamageToPlayer(pl, damagePerTarget, ctx.SourceID)
		} else if perm := ctx.Game.FindPermanent(tid); perm != nil {
			ctx.Game.DealDamageToPermanent(perm, damagePerTarget, ctx.SourceID)
		}
	}
	return nil
}

// DealDamageToAllCreatures creates an effect that deals damage to all matching creatures.
func DealDamageToAllCreatures(amount ValueSource, filter PermanentFilter) Effect {
	return DealDamageFrom(legacyDamageSourceReference{}, BattlefieldObjects(And(IsCreature, filter)), amount)
}

// DealDamageToPlayers creates an effect that deals damage to players selected by the selector.
func DealDamageToPlayers(amount ValueSource, selector PlayerSelector) Effect {
	return DealDamageFrom(legacyDamageSourceReference{}, selector, amount)
}

// DealDamageToPlayersStep returns the Effect for use in pipelines/ForEach/Modal.
func DealDamageToPlayersStep(amount ValueSource, selector PlayerSelector) Effect {
	return DealDamageFrom(legacyDamageSourceReference{}, selector, amount)
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

// preventCombatDamageToAndByTargetEffect prevents all combat damage dealt to and by target creature this turn (Maze of Ith).
type preventCombatDamageToAndByTargetEffect struct{}

// PreventCombatDamageToAndByTarget creates an effect preventing all combat damage dealt to and by target creature this turn.
func PreventCombatDamageToAndByTarget() Effect {
	return &preventCombatDamageToAndByTargetEffect{}
}

func (e *preventCombatDamageToAndByTargetEffect) Text() string {
	return "prevent all combat damage that would be dealt to and dealt by target creature this turn"
}

func (e *preventCombatDamageToAndByTargetEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

func (e *preventCombatDamageToAndByTargetEffect) Apply(ctx *EffectContext) error {
	if len(ctx.legacyTargets()) == 0 {
		return nil
	}
	ctx.Game.PreventCombatDamageToAndBy(ctx.legacyTargets()[0])
	return nil
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
	selector ObjectSelector
}

// PreventDamageToTarget creates an effect that prevents damage to a target.
// Chain Targeting to select a permanent without adding a rules target.
func PreventDamageToTarget(amount ValueSource) *PreventDamageEffect {
	return &PreventDamageEffect{
		amount:   amount,
		selector: ToTarget(),
	}
}

func (e *PreventDamageEffect) Targeting(selector ObjectSelector) *PreventDamageEffect {
	e.selector = selector
	return e
}

func (e *PreventDamageEffect) Text() string {
	object := "target"
	if isSourceSelector(e.selector) {
		object = "source"
	}
	if _, ok := e.amount.(xValue); ok {
		return "Prevent the next X damage to " + object
	}
	return fmt.Sprintf("Prevent the next %d damage to %s", e.amount.Resolve(NewEffectContext(nil, uuid.Nil(), uuid.Nil(), nil)), object)
}
func (e *PreventDamageEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

func (e *PreventDamageEffect) Apply(ctx *EffectContext) error {
	amount := e.amount.Resolve(ctx)
	for _, id := range currentObjectIDs(ctx, e.selector) {
		if id == uuid.Nil() {
			continue
		}
		if perm := ctx.Game.FindPermanent(id); perm != nil {
			ctx.Game.AddPreventionShield(perm.ID(), amount)
		} else if player := ctx.Game.GetPlayer(id); player != nil {
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
		playerIDs = e.sel.Select(ctx)
	case len(ctx.legacyTargets()) > 0:
		playerIDs = []uuid.UUID{ctx.legacyTargets()[0]}
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

func (e *dealDamageEffect) Apply(ctx *EffectContext) error {
	amount := e.amount.Resolve(ctx)
	if amount <= 0 {
		return nil
	}
	recipients := e.recipient.Select(ctx)
	for _, source := range e.source.Select(ctx) {
		for _, id := range recipients {
			if player := ctx.Game.GetPlayer(id); player != nil {
				ctx.Game.DealDamageToPlayer(player, amount, source)
				continue
			}
			if permanent := ctx.Game.FindPermanent(id); permanent != nil {
				ctx.Game.DealDamageToPermanent(permanent, amount, source)
			}
		}
	}
	return nil
}

// execDealDividedDamage reads the per-target damage distribution chosen at
// cast/activation time (stored on the StackObject and forwarded into ctx via
// resolvingDamageDistribution) and applies it. CR 601.2d/609.3.5: a divided-
// damage spell's distribution is chosen on announcement and is frozen; if a
// target later becomes illegal, only that target's share is wasted — the
// remaining targets still take their shares. We honor that by skipping
// ctx.legacyTargets() entries that no longer point at a valid creature/player.
func (*dealDividedDamageEffect) Apply(ctx *EffectContext) error {
	dist := ctx.DamageDistribution
	if dist == nil {
		dist = ctx.Game.ResolvingDamageDistribution()
	}
	if len(dist) == 0 {
		return nil
	}
	for _, tid := range ctx.legacyTargets() {
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
		if sourceCard != nil && perm.HasProtectionFromInGame(sourceCard, ctx.Game) {
			continue
		}
		ctx.Game.DealDamageToPermanent(perm, amt, ctx.SourceID)
	}
	return nil
}

func (e *handSizeDamageEffect) Apply(ctx *EffectContext) error {
	var p Player
	if len(ctx.legacyTargets()) > 0 {
		p = ctx.Game.GetPlayer(ctx.legacyTargets()[0])
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
