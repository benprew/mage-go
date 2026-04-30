package mage

import (
	"fmt"

	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// untapTargetEffect untaps a target permanent.
type untapTargetEffect struct{}

// UntapTarget creates an effect that untaps a target permanent.
func UntapTarget() Effect {
	return &untapTargetEffect{}
}

// UntapTargetStep returns the EffectData for use as a pipeline/ForEach inner step.
func UntapTargetStep() EffectData { return &untapTargetEffect{} }

func (e *untapTargetEffect) Text() string { return "untap target permanent" }
func (e *untapTargetEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// untapSourceEffect untaps the source permanent.
type untapSourceEffect struct{}

// UntapSource creates an effect that untaps the source permanent.
func UntapSource() Effect {
	return &untapSourceEffect{}
}

func (e *untapSourceEffect) Text() string                 { return "Untap this permanent" }
func (e *untapSourceEffect) Properties() EffectProperties { return EffectProperties{} }

// tapAttachedCreatureEffect taps the creature attached to the source aura.
type tapAttachedCreatureEffect struct{}

// TapAttachedCreature creates an effect that taps the creature the source aura is attached to.
func TapAttachedCreature() Effect {
	return &tapAttachedCreatureEffect{}
}

func (e *tapAttachedCreatureEffect) Text() string { return "Tap enchanted creature" }
func (e *tapAttachedCreatureEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// tapOrUntapTargetEffect lets you tap or untap a target permanent.
type tapOrUntapTargetEffect struct{}

// TapOrUntapTarget creates an effect that toggles a target permanent's tapped state (e.g. Twiddle).
func TapOrUntapTarget() Effect {
	return &tapOrUntapTargetEffect{}
}

func (e *tapOrUntapTargetEffect) Text() string {
	return "Tap or untap target permanent"
}
func (e *tapOrUntapTargetEffect) Properties() EffectProperties { return EffectProperties{} }

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

// makeUnblockableUntilEndOfTurnEffect makes a target creature unblockable until end of turn.
type makeUnblockableUntilEndOfTurnEffect struct{}

// MakeUnblockableUntilEndOfTurn creates an effect that makes a target creature unblockable until end of turn.
func MakeUnblockableUntilEndOfTurn() Effect {
	return &makeUnblockableUntilEndOfTurnEffect{}
}

func (e *makeUnblockableUntilEndOfTurnEffect) Text() string {
	return "Target creature can't be blocked this turn"
}
func (e *makeUnblockableUntilEndOfTurnEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

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

func execPreventAttackingTargetUntilEndOfTurn(ctx *EffectContext, _ *preventAttackingUntilEndOfTurnEffect) error {
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

// regenerateSourceEffect sets a regeneration shield on the source.
type regenerateSourceEffect struct{}

// RegenerateSource creates an effect that sets a regeneration shield on the source permanent.
func RegenerateSource() Effect {
	return &regenerateSourceEffect{}
}

func (e *regenerateSourceEffect) Text() string { return "Regenerate ~" }
func (e *regenerateSourceEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// regenerateTargetEffect sets a regeneration shield on the target.
type regenerateTargetEffect struct{}

// RegenerateTarget creates an effect that sets a regeneration shield on a target creature.
func RegenerateTarget() Effect {
	return &regenerateTargetEffect{}
}

func (e *regenerateTargetEffect) Text() string { return "Regenerate target creature" }
func (e *regenerateTargetEffect) Properties() EffectProperties {
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
	return &markDestroyAtEOTAfterNActivationsEffect{threshold: threshold}
}

func (e *markDestroyAtEOTAfterNActivationsEffect) Text() string {
	return fmt.Sprintf("if activated %d+ times, destroy at end of turn", e.threshold)
}
func (e *markDestroyAtEOTAfterNActivationsEffect) Properties() EffectProperties {
	return EffectProperties{}
}

// stunEffect causes the selected permanent to skip its next own untap step
// (i.e. its controller's next untap step), then expires. Implemented as a
// turn-bounded continuous effect that grants AttrDoesNotUntap; no counter is
// placed on the permanent.
type stunEffect struct {
	selector TargetSelector
}

// Stun creates an effect that causes the targeted permanent to skip its next
// untap step. Defaults to targeting the resolved target; chain .Targeting(...)
// to apply to the source.
func Stun() *stunEffect {
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

// setBasePowerUntilEndOfTurnEffect sets a target creature's base power until end of turn.
type setBasePowerUntilEndOfTurnEffect struct {
	power int
}

// SetPowerUntilEndOfTurn creates an effect that sets the target creature's base power
// until end of turn (e.g. Singing Tree, Island of Wak-Wak: power becomes 0).
func SetPowerUntilEndOfTurn(power int, target PermanentSelector) Effect {
	return &setBasePowerUntilEndOfTurnEffect{power: power}
}

func (e *setBasePowerUntilEndOfTurnEffect) Text() string {
	return fmt.Sprintf("target creature has base power %d until end of turn", e.power)
}
func (e *setBasePowerUntilEndOfTurnEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// --- Combat effect executors ---

func execTap(ctx *EffectContext, _ *tap) error {
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

func execStun(ctx *EffectContext, e *stunEffect) error {
	perms := resolvePermanents(ctx, e.selector)
	if len(perms) == 0 {
		return nil
	}
	for _, perm := range perms {
		permID := perm.ID()
		// Compute the turn number on which the permanent's controller will
		// next have an untap step. In a 2-player game that's currentTurn+2 if
		// the permanent's controller is currently active, else currentTurn+1.
		expiryTurn := ctx.Game.CurrentTurn() + 1
		if ctx.Game.ActivePlayerObj().PlayerID() == perm.Controller {
			expiryTurn = ctx.Game.CurrentTurn() + 2
		}
		ce := FuncContinuousEffect(LayerAbility, Indefinite, func(g *Game, _ uuid.UUID) error {
			p := g.FindPermanent(permID)
			if p != nil {
				g.GrantAttr(p.ID(), AttrDoesNotUntap)
			}
			return nil
		}, func(g *Game, _ uuid.UUID) bool {
			return g.CurrentTurn() <= expiryTurn && g.FindPermanent(permID) != nil
		})
		ce.SetSourceID(ctx.SourceID)
		ctx.Game.AddContinuousEffect(ce)
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
