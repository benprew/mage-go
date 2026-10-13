package mage

import (
	"fmt"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// tapAllLandsEffect taps all lands target player controls.
type tapAllLandsEffect struct{}

// TapAllLands creates an effect that taps all lands a target player controls (e.g. Mana Short).
func TapAllLands() Effect { return &tapAllLandsEffect{} }

func (e *tapAllLandsEffect) Text() string { return "Tap all lands target player controls" }
func (e *tapAllLandsEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// removeFromCombatEffect removes a target creature from combat.
type removeFromCombatEffect struct{}

// RemoveFromCombat creates an effect that removes a target creature from combat.
func RemoveFromCombat() Effect { return &removeFromCombatEffect{} }

func (e *removeFromCombatEffect) Text() string                 { return "Remove target creature from combat" }
func (e *removeFromCombatEffect) Properties() EffectProperties { return EffectProperties{} }

// preventAttackingUntilEndOfTurnEffect makes a target creature unable to attack
// for the rest of this turn. Used for instant-speed "target creature can't
// attack this turn" cards. Per CR 506.4a, if applied after the target has
// already been declared as an attacker, the creature stays in combat — the
// effect only prevents future declarations this turn.
type preventAttackingUntilEndOfTurnEffect struct{}

// PreventAttackingTargetUntilEndOfTurn creates an effect that prevents a target
// creature from attacking this turn (CR 506.4a).
func PreventAttackingTargetUntilEndOfTurn() Effect {
	return &preventAttackingUntilEndOfTurnEffect{}
}

func (*preventAttackingUntilEndOfTurnEffect) Apply(ctx *EffectContext) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	perm := ctx.Game.FindPermanent(ctx.Targets[0])
	if perm == nil {
		return nil
	}
	eff := PreventAttackingUntilEndOfTurn(perm.ID())
	eff.SetSourceID(ctx.SourceID)
	ctx.Game.AddContinuousEffect(eff)
	ctx.Game.ApplyContinuousEffects()
	return nil
}

func (e *preventAttackingUntilEndOfTurnEffect) Text() string {
	return "Target creature can't attack this turn"
}
func (e *preventAttackingUntilEndOfTurnEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// doubleSourcePowerEffect doubles the source creature's power until end of turn.
type doubleSourcePowerEffect struct{}

// DoubleTargetPower creates an effect that doubles a target creature's power until end of turn (e.g. Berserk).
func DoubleTargetPower() Effect {
	return &doubleSourcePowerEffect{}
}

func (e *doubleSourcePowerEffect) Text() string {
	return "Target creature's power is doubled until end of turn"
}
func (e *doubleSourcePowerEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// replaceKeywordEffect replaces one keyword with another on a target permanent.
type replaceKeywordEffect struct {
	from Keyword
	to   Keyword
}

// ReplaceKeywordEffect creates an effect that replaces one keyword with another on a target permanent
// as a continuous effect (e.g. replacing Flying with a different evasion).
func ReplaceKeywordEffect(from, to Keyword) Effect {
	return &replaceKeywordEffect{from: from, to: to}
}

func (e *replaceKeywordEffect) Text() string {
	return fmt.Sprintf("Replace %s with %s", e.from, e.to)
}
func (e *replaceKeywordEffect) Properties() EffectProperties { return EffectProperties{} }

// markDestroyAtEOTAfterNActivationsEffect tracks pump activations using Charge
// counters. When the count reaches the threshold, sets DestroyAtEndOfTurn.
type markDestroyAtEOTAfterNActivationsEffect struct {
	threshold int
}

// MarkDestroyAtEOTAfterNActivations creates an effect that tracks activations via Charge counters.
// When the count reaches the threshold, the source is destroyed at end of turn (e.g. Basalt Monolith variant).
func MarkDestroyAtEOTAfterNActivations(threshold int) Effect {
	return &markDestroyAtEOTAfterNActivationsEffect{threshold: threshold}
}

func (e *markDestroyAtEOTAfterNActivationsEffect) Text() string {
	return fmt.Sprintf("if activated %d+ times, destroy at end of turn", e.threshold)
}
func (e *markDestroyAtEOTAfterNActivationsEffect) Properties() EffectProperties {
	return EffectProperties{}
}

// stunEffect causes the selected permanent to skip its next untap attempt
// during its then-controller's untap step. No counter is placed.
type stunEffect struct {
	selector TargetSelector
}

// StunCreature creates an effect that causes the targeted permanent to skip its
// next untap step. Defaults to targeting the resolved target; chain
// .Targeting(...) to apply to the source.
func StunCreature() *stunEffect {
	return &stunEffect{selector: TargetSelector{Kind: KindTarget}}
}

// Targeting sets which permanent the stun applies to.
func (e *stunEffect) Targeting(sel TargetSelector) *stunEffect {
	e.selector = sel
	return e
}

func (e *stunEffect) Text() string { return "stun" }
func (e *stunEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// destroyTargetAtEndOfTurnEffect marks a creature for destruction at end of turn.
type destroyTargetAtEndOfTurnEffect struct{}

// DestroyTargetAtEndOfTurn creates an effect that registers a delayed trigger to destroy
// the target creature at the next end step.
func DestroyTargetAtEndOfTurn() Effect {
	return &destroyTargetAtEndOfTurnEffect{}
}

func (e *destroyTargetAtEndOfTurnEffect) Text() string {
	return "Destroy target creature at end of turn"
}
func (e *destroyTargetAtEndOfTurnEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// setBasePTUntilEndOfTurnEffect sets a target creature's base P/T until end of turn.
type setBasePTUntilEndOfTurnEffect struct {
	power     int
	toughness int
}

// SetPTUntilEndOfTurn creates an effect that sets the target creature's base P/T
// until end of turn (e.g. Sorceress Queen: 0/2).
func SetPTUntilEndOfTurn(power, toughness int, target PermanentSelector) Effect {
	return &setBasePTUntilEndOfTurnEffect{power: power, toughness: toughness}
}

func (e *setBasePTUntilEndOfTurnEffect) Text() string {
	return fmt.Sprintf("target creature has base power and toughness %d/%d until end of turn", e.power, e.toughness)
}
func (e *setBasePTUntilEndOfTurnEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// --- Combat effect executors ---

func (*tap) Apply(ctx *EffectContext) error {
	if len(ctx.Targets) == 0 {
		return fmt.Errorf("no target for tap")
	}
	perm := ctx.Game.FindPermanent(ctx.Targets[0])
	if perm == nil {
		return nil
	}
	ctx.Game.TapPermanent(perm)
	return nil
}

func (*tapAllLandsEffect) Apply(ctx *EffectContext) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	playerID := ctx.Targets[0]
	for _, p := range ctx.Game.FilterBattlefield(And(ControlledBy(playerID), IsLand)) {
		ctx.Game.TapPermanent(p)
	}
	return nil
}

func (*removeFromCombatEffect) Apply(ctx *EffectContext) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	perm := ctx.Game.FindPermanent(ctx.Targets[0])
	if perm == nil {
		return nil
	}
	ctx.Game.combat.removeFromCombatAndUnblock(perm.ID())
	return nil
}

func (*doubleSourcePowerEffect) Apply(ctx *EffectContext) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	perm := ctx.Game.FindPermanent(ctx.Targets[0])
	if perm == nil {
		return nil
	}
	currentPower := perm.CurrentPower(ctx.Game)
	eff := TemporaryBoost(perm.ID(), currentPower, 0)
	eff.SetSourceID(ctx.SourceID)
	ctx.Game.AddContinuousEffect(eff)
	ctx.Game.ApplyContinuousEffects()
	return nil
}

func (e *replaceKeywordEffect) Apply(ctx *EffectContext) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	perm := ctx.Game.FindPermanent(ctx.Targets[0])
	if perm == nil {
		return nil
	}
	eff := KeywordReplacement(perm.ID(), e.from, e.to)
	eff.SetSourceID(ctx.SourceID)
	ctx.Game.AddContinuousEffect(eff)
	ctx.Game.ApplyContinuousEffects()
	return nil
}

func (e *markDestroyAtEOTAfterNActivationsEffect) Apply(ctx *EffectContext) error {
	perm := ctx.Game.FindPermanent(ctx.SourceID)
	if perm == nil {
		return nil
	}
	perm.AddCounter(Charge, 1)
	if perm.Counters[Charge] >= e.threshold {
		ctx.Game.RegisterDelayedTrigger(&DelayedTrigger{
			EventType:  EvtEndStep,
			TargetID:   perm.ID(),
			Effects:    []Effect{DestroyTarget()},
			SourceID:   ctx.SourceID,
			Controller: ctx.Controller,
		})
	}
	return nil
}

func (e *stunEffect) Apply(ctx *EffectContext) error {
	perms := resolvePermanents(ctx, e.selector)
	if len(perms) == 0 {
		return nil
	}
	for _, perm := range perms {
		ctx.Game.SkipNextUntap(perm.ID())
	}
	return nil
}

func (*destroyTargetAtEndOfTurnEffect) Apply(ctx *EffectContext) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	perm := ctx.Game.FindPermanent(ctx.Targets[0])
	if perm == nil {
		return nil
	}
	ctx.Game.RegisterDelayedTrigger(&DelayedTrigger{
		EventType:  EvtEndStep,
		TargetID:   perm.ID(),
		Effects:    []Effect{DestroyTarget()},
		SourceID:   ctx.SourceID,
		Controller: ctx.Controller,
	})
	return nil
}

func (e *setBasePTUntilEndOfTurnEffect) Apply(ctx *EffectContext) error {
	if len(ctx.Targets) == 0 {
		return fmt.Errorf("no target for set P/T")
	}
	perm := ctx.Game.FindPermanent(ctx.Targets[0])
	if perm == nil {
		return nil
	}
	eff := SetBasePT(perm.ID(), e.power, e.toughness)
	eff.SetSourceID(ctx.SourceID)
	ctx.Game.AddContinuousEffect(eff)
	ctx.Game.ApplyContinuousEffects()
	return nil
}
