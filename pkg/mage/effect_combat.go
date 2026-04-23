package mage

import (
	"fmt"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// tapTargetEffect taps a target permanent.
type tapTargetEffect struct{}

// TapTarget creates an effect that taps a target permanent.
func TapTarget() Effect {
	return DataEffect(&tapTargetEffect{})
}

// TapTargetStep returns the EffectData for use as a pipeline/ForEach inner step.
func TapTargetStep() EffectData { return &tapTargetEffect{} }

func (e *tapTargetEffect) EffectText() string { return "tap target permanent" }
func (e *tapTargetEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// untapTargetEffect untaps a target permanent.
type untapTargetEffect struct{}

// UntapTarget creates an effect that untaps a target permanent.
func UntapTarget() Effect {
	return DataEffect(&untapTargetEffect{})
}

// UntapTargetStep returns the EffectData for use as a pipeline/ForEach inner step.
func UntapTargetStep() EffectData { return &untapTargetEffect{} }

func (e *untapTargetEffect) EffectText() string { return "untap target permanent" }
func (e *untapTargetEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// untapSourceEffect untaps the source permanent.
type untapSourceEffect struct{}

// UntapSource creates an effect that untaps the source permanent.
func UntapSource() Effect {
	return DataEffect(&untapSourceEffect{})
}

func (e *untapSourceEffect) EffectText() string          { return "Untap this permanent" }
func (e *untapSourceEffect) EffectProps() EffectProperties { return EffectProperties{} }

// tapAttachedCreatureEffect taps the creature attached to the source aura.
type tapAttachedCreatureEffect struct{}

// TapAttachedCreature creates an effect that taps the creature the source aura is attached to.
func TapAttachedCreature() Effect {
	return DataEffect(&tapAttachedCreatureEffect{})
}

func (e *tapAttachedCreatureEffect) EffectText() string { return "Tap enchanted creature" }
func (e *tapAttachedCreatureEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// tapOrUntapTargetEffect lets you tap or untap a target permanent.
type tapOrUntapTargetEffect struct{}

// TapOrUntapTarget creates an effect that toggles a target permanent's tapped state (e.g. Twiddle).
func TapOrUntapTarget() Effect {
	return DataEffect(&tapOrUntapTargetEffect{})
}

func (e *tapOrUntapTargetEffect) EffectText() string {
	return "Tap or untap target permanent"
}
func (e *tapOrUntapTargetEffect) EffectProps() EffectProperties { return EffectProperties{} }

// tapAllLandsEffect taps all lands target player controls.
type tapAllLandsEffect struct{}

// TapAllLands creates an effect that taps all lands a target player controls (e.g. Mana Short).
func TapAllLands() Effect { return DataEffect(&tapAllLandsEffect{}) }

func (e *tapAllLandsEffect) EffectText() string { return "Tap all lands target player controls" }
func (e *tapAllLandsEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// removeFromCombatEffect removes a target creature from combat.
type removeFromCombatEffect struct{}

// RemoveFromCombat creates an effect that removes a target creature from combat.
func RemoveFromCombat() Effect { return DataEffect(&removeFromCombatEffect{}) }

func (e *removeFromCombatEffect) EffectText() string { return "Remove target creature from combat" }
func (e *removeFromCombatEffect) EffectProps() EffectProperties { return EffectProperties{} }

// makeUnblockableUntilEndOfTurnEffect makes a target creature unblockable until end of turn.
type makeUnblockableUntilEndOfTurnEffect struct{}

// MakeUnblockableUntilEndOfTurn creates an effect that makes a target creature unblockable until end of turn.
func MakeUnblockableUntilEndOfTurn() Effect {
	return DataEffect(&makeUnblockableUntilEndOfTurnEffect{})
}

func (e *makeUnblockableUntilEndOfTurnEffect) EffectText() string {
	return "Target creature can't be blocked this turn"
}
func (e *makeUnblockableUntilEndOfTurnEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// doubleSourcePowerEffect doubles the source creature's power until end of turn.
type doubleSourcePowerEffect struct{}

// DoubleTargetPower creates an effect that doubles a target creature's power until end of turn (e.g. Berserk).
func DoubleTargetPower() Effect {
	return DataEffect(&doubleSourcePowerEffect{})
}

func (e *doubleSourcePowerEffect) EffectText() string {
	return "Target creature's power is doubled until end of turn"
}
func (e *doubleSourcePowerEffect) EffectProps() EffectProperties {
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
	return DataEffect(&replaceKeywordEffect{from: from, to: to})
}

func (e *replaceKeywordEffect) EffectText() string {
	return fmt.Sprintf("Replace %s with %s", e.from, e.to)
}
func (e *replaceKeywordEffect) EffectProps() EffectProperties { return EffectProperties{} }

// regenerateSourceEffect sets a regeneration shield on the source.
type regenerateSourceEffect struct{}

// RegenerateSource creates an effect that sets a regeneration shield on the source permanent.
func RegenerateSource() Effect {
	return DataEffect(&regenerateSourceEffect{})
}

func (e *regenerateSourceEffect) EffectText() string { return "Regenerate ~" }
func (e *regenerateSourceEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// regenerateTargetEffect sets a regeneration shield on the target.
type regenerateTargetEffect struct{}

// RegenerateTarget creates an effect that sets a regeneration shield on a target creature.
func RegenerateTarget() Effect {
	return DataEffect(&regenerateTargetEffect{})
}

func (e *regenerateTargetEffect) EffectText() string { return "Regenerate target creature" }
func (e *regenerateTargetEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// markDestroyAtEOTAfterNActivationsEffect tracks pump activations using Charge
// counters. When the count reaches the threshold, sets DestroyAtEndOfTurn.
type markDestroyAtEOTAfterNActivationsEffect struct {
	threshold int
}

// MarkDestroyAtEOTAfterNActivations creates an effect that tracks activations via Charge counters.
// When the count reaches the threshold, the source is destroyed at end of turn (e.g. Basalt Monolith variant).
func MarkDestroyAtEOTAfterNActivations(threshold int) Effect {
	return DataEffect(&markDestroyAtEOTAfterNActivationsEffect{threshold: threshold})
}

func (e *markDestroyAtEOTAfterNActivationsEffect) EffectText() string {
	return fmt.Sprintf("if activated %d+ times, destroy at end of turn", e.threshold)
}
func (e *markDestroyAtEOTAfterNActivationsEffect) EffectProps() EffectProperties {
	return EffectProperties{}
}

// destroyTargetAtEndOfTurnEffect marks a creature for destruction at end of turn.
type destroyTargetAtEndOfTurnEffect struct{}

// DestroyTargetAtEndOfTurn creates an effect that registers a delayed trigger to destroy
// the target creature at the next end step.
func DestroyTargetAtEndOfTurn() Effect {
	return DataEffect(&destroyTargetAtEndOfTurnEffect{})
}

func (e *destroyTargetAtEndOfTurnEffect) EffectText() string {
	return "Destroy target creature at end of turn"
}
func (e *destroyTargetAtEndOfTurnEffect) EffectProps() EffectProperties {
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
	return DataEffect(&setBasePTUntilEndOfTurnEffect{power: power, toughness: toughness})
}

func (e *setBasePTUntilEndOfTurnEffect) EffectText() string {
	return fmt.Sprintf("target creature has base power and toughness %d/%d until end of turn", e.power, e.toughness)
}
func (e *setBasePTUntilEndOfTurnEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// setBasePowerUntilEndOfTurnEffect sets a target creature's base power until end of turn.
type setBasePowerUntilEndOfTurnEffect struct {
	power int
}

// SetPowerUntilEndOfTurn creates an effect that sets the target creature's base power
// until end of turn (e.g. Singing Tree, Island of Wak-Wak: power becomes 0).
func SetPowerUntilEndOfTurn(power int, target PermanentSelector) Effect {
	return DataEffect(&setBasePowerUntilEndOfTurnEffect{power: power})
}

func (e *setBasePowerUntilEndOfTurnEffect) EffectText() string {
	return fmt.Sprintf("target creature has base power %d until end of turn", e.power)
}
func (e *setBasePowerUntilEndOfTurnEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// --- Combat effect executors ---

func execTapTarget(ctx *EffectContext, _ *tapTargetEffect) error {
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

func execUntapTarget(ctx *EffectContext, _ *untapTargetEffect) error {
	if len(ctx.Targets) == 0 {
		return fmt.Errorf("no target for untap")
	}
	perm := ctx.Game.FindPermanent(ctx.Targets[0])
	if perm == nil {
		return nil
	}
	if perm.Tapped {
		perm.Tapped = false
		ctx.Game.FireEvent(GameEvent{Type: EvtBecameUntapped, SourceID: perm.ID()})
	}
	return nil
}

func execUntapSource(ctx *EffectContext, _ *untapSourceEffect) error {
	perm := ctx.Game.FindPermanent(ctx.SourceID)
	if perm != nil && perm.Tapped {
		perm.Tapped = false
		ctx.Game.FireEvent(GameEvent{Type: EvtBecameUntapped, SourceID: perm.ID()})
	}
	return nil
}

func execTapAttachedCreature(ctx *EffectContext, _ *tapAttachedCreatureEffect) error {
	src := ctx.Game.FindPermanent(ctx.SourceID)
	if src == nil || !src.IsAttached() {
		return nil
	}
	target := ctx.Game.FindPermanent(src.AttachedTo)
	if target != nil {
		ctx.Game.TapPermanent(target)
	}
	return nil
}

func execTapOrUntapTarget(ctx *EffectContext, _ *tapOrUntapTargetEffect) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	perm := ctx.Game.FindPermanent(ctx.Targets[0])
	if perm == nil {
		return nil
	}
	caster := ctx.Game.GetPlayer(ctx.Controller)
	if caster == nil {
		return nil
	}
	mode := caster.ChooseMode([]string{"Tap", "Untap"}, "Twiddle")
	if mode == 0 {
		ctx.Game.TapPermanent(perm)
	} else {
		perm.Tapped = false
	}
	return nil
}

func execTapAllLands(ctx *EffectContext, _ *tapAllLandsEffect) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	playerID := ctx.Targets[0]
	for _, p := range ctx.Game.FilterBattlefield(And(ControlledBy(playerID), IsLand)) {
		ctx.Game.TapPermanent(p)
	}
	return nil
}

func execRemoveFromCombat(ctx *EffectContext, _ *removeFromCombatEffect) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	perm := ctx.Game.FindPermanent(ctx.Targets[0])
	if perm == nil {
		return nil
	}
	ctx.Game.RemoveFromCombat(perm.ID())
	return nil
}

func execMakeUnblockableUntilEndOfTurn(ctx *EffectContext, _ *makeUnblockableUntilEndOfTurnEffect) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	perm := ctx.Game.FindPermanent(ctx.Targets[0])
	if perm == nil {
		return nil
	}
	eff := TemporaryKeyword(perm.ID(), UnblockableKW)
	eff.SetSourceID(ctx.SourceID)
	ctx.Game.AddContinuousEffect(eff)
	ctx.Game.ApplyContinuousEffects()
	return nil
}

func execDoubleTargetPower(ctx *EffectContext, _ *doubleSourcePowerEffect) error {
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

func execReplaceKeyword(ctx *EffectContext, e *replaceKeywordEffect) error {
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

func execRegenerateSource(ctx *EffectContext, _ *regenerateSourceEffect) error {
	perm := ctx.Game.FindPermanent(ctx.SourceID)
	if perm == nil {
		return nil
	}
	ctx.Game.AddRegenerationShield(ctx.SourceID)
	return nil
}

func execRegenerateTarget(ctx *EffectContext, _ *regenerateTargetEffect) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	perm := ctx.Game.FindPermanent(ctx.Targets[0])
	if perm == nil {
		return nil
	}
	ctx.Game.AddRegenerationShield(perm.ID())
	return nil
}

func execMarkDestroyAtEOTAfterNActivations(ctx *EffectContext, e *markDestroyAtEOTAfterNActivationsEffect) error {
	perm := ctx.Game.FindPermanent(ctx.SourceID)
	if perm == nil {
		return nil
	}
	perm.AddCounter(Charge, 1)
	if int(perm.Counters[Charge]) >= e.threshold {
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

func execDestroyTargetAtEndOfTurn(ctx *EffectContext, _ *destroyTargetAtEndOfTurnEffect) error {
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

func execSetPTUntilEndOfTurn(ctx *EffectContext, e *setBasePTUntilEndOfTurnEffect) error {
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

func execSetPowerUntilEndOfTurn(ctx *EffectContext, e *setBasePowerUntilEndOfTurnEffect) error {
	if len(ctx.Targets) == 0 {
		return fmt.Errorf("no target for set power")
	}
	perm := ctx.Game.FindPermanent(ctx.Targets[0])
	if perm == nil {
		return nil
	}
	eff := SetBasePower(perm.ID(), e.power)
	eff.SetSourceID(ctx.SourceID)
	ctx.Game.AddContinuousEffect(eff)
	ctx.Game.ApplyContinuousEffects()
	return nil
}
