package mage

import (
	"fmt"

	"github.com/google/uuid"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// addCountersEffect adds counters to the source or a target permanent.
type addCountersEffect struct {
	ct     CounterType
	amount ValueSource
	target PermanentSelector
}

// AddCounters creates an effect that adds counters to the selected permanent.
func AddCounters(ct CounterType, amount ValueSource, target PermanentSelector) Effect {
	return &addCountersEffect{ct: ct, amount: amount, target: target}
}

func (e *addCountersEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	var perm *Permanent
	if e.target == SelectSource {
		perm = g.FindPermanent(sourceID)
	} else {
		if len(targets) == 0 {
			return fmt.Errorf("no target for counters")
		}
		perm = g.FindPermanent(targets[0])
	}
	if perm == nil {
		return nil
	}
	amount := e.amount.Resolve(g, sourceID, controller)
	if amount > 0 {
		perm.AddCounter(e.ct, amount)
	}
	return nil
}

func (e *addCountersEffect) Text() string {
	if _, ok := e.amount.(xValue); ok {
		return fmt.Sprintf("put X %s counters on it", e.ct)
	}
	n := e.amount.Resolve(nil, uuid.Nil, uuid.Nil)
	if e.target == SelectSource {
		return fmt.Sprintf("put %d %s counter(s) on it", n, e.ct)
	}
	return fmt.Sprintf("put %d %s counter(s) on target", n, e.ct)
}
func (e *addCountersEffect) Properties() EffectProperties { return EffectProperties{} }

// removeCountersFromSourceEffect removes counters from the source permanent.
type removeCountersFromSourceEffect struct {
	ct     CounterType
	amount int
}

// RemoveCountersFromSource creates an effect that removes counters from the source permanent.
func RemoveCountersFromSource(ct CounterType, amount int) Effect {
	return &removeCountersFromSourceEffect{ct: ct, amount: amount}
}

func (e *removeCountersFromSourceEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.FindPermanent(sourceID)
	if p == nil {
		return nil
	}
	p.RemoveCounter(e.ct, e.amount)
	return nil
}

func (e *removeCountersFromSourceEffect) Text() string {
	return fmt.Sprintf("remove %d %s counter(s) from it", e.amount, e.ct)
}
func (e *removeCountersFromSourceEffect) Properties() EffectProperties { return EffectProperties{} }

// AddCountersUpToMax creates an effect that adds up to X +1/+0 counters on the source,
// capped so total counters don't exceed maxCounters.
func AddCountersUpToMax(ct CounterType, maxCounters int) Effect {
	return &addCountersUpToMaxEffect{ct: ct, maxCounters: maxCounters}
}

type addCountersUpToMaxEffect struct {
	ct          CounterType
	maxCounters int
}

func (e *addCountersUpToMaxEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
	perm := g.FindPermanent(sourceID)
	if perm == nil {
		return nil
	}
	x := g.XValue()
	current := perm.Counters[e.ct]
	room := max(e.maxCounters-current, 0)
	toAdd := min(x, room)
	if toAdd > 0 {
		perm.AddCounter(e.ct, toAdd)
	}
	return nil
}

func (e *addCountersUpToMaxEffect) Text() string {
	return fmt.Sprintf("put up to X %s counters on it (max %d total)", e.ct, e.maxCounters)
}

func (e *addCountersUpToMaxEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// tapTargetEffect taps a target permanent.
type tapTargetEffect struct{}

// TapTarget creates an effect that taps a target permanent.
func TapTarget() Effect {
	return &tapTargetEffect{}
}

func (e *tapTargetEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for tap")
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	g.TapPermanent(perm)
	return nil
}

func (e *tapTargetEffect) Text() string { return "tap target permanent" }
func (e *tapTargetEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// untapTargetEffect untaps a target permanent.
type untapTargetEffect struct{}

// UntapTarget creates an effect that untaps a target permanent.
func UntapTarget() Effect {
	return &untapTargetEffect{}
}

func (e *untapTargetEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for untap")
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	if perm.Tapped {
		perm.Tapped = false
		g.FireEvent(GameEvent{Type: EvtBecameUntapped, SourceID: perm.ID()})
	}
	return nil
}

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

func (e *untapSourceEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	perm := g.FindPermanent(sourceID)
	if perm != nil && perm.Tapped {
		perm.Tapped = false
		g.FireEvent(GameEvent{Type: EvtBecameUntapped, SourceID: perm.ID()})
	}
	return nil
}

func (e *untapSourceEffect) Text() string { return "Untap this permanent" }
func (e *untapSourceEffect) Properties() EffectProperties { return EffectProperties{} }

// tapAttachedCreatureEffect taps the creature attached to the source aura.
type tapAttachedCreatureEffect struct{}

// TapAttachedCreature creates an effect that taps the creature the source aura is attached to.
func TapAttachedCreature() Effect {
	return &tapAttachedCreatureEffect{}
}

func (e *tapAttachedCreatureEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	src := g.FindPermanent(sourceID)
	if src == nil || !src.IsAttached() {
		return nil
	}
	target := g.FindPermanent(src.AttachedTo)
	if target != nil {
		g.TapPermanent(target)
	}
	return nil
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

func (e *tapOrUntapTargetEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	caster := g.GetPlayer(controller)
	if caster == nil {
		return nil
	}
	mode := caster.ChooseMode([]string{"Tap", "Untap"}, "Twiddle")
	if mode == 0 {
		g.TapPermanent(perm)
	} else {
		perm.Tapped = false
	}
	return nil
}

func (e *tapOrUntapTargetEffect) Text() string {
	return "Tap or untap target permanent"
}
func (e *tapOrUntapTargetEffect) Properties() EffectProperties { return EffectProperties{} }

// tapAllLandsEffect taps all lands target player controls.
type tapAllLandsEffect struct{}

// TapAllLands creates an effect that taps all lands a target player controls (e.g. Mana Short).
func TapAllLands() Effect { return &tapAllLandsEffect{} }

func (e *tapAllLandsEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	playerID := targets[0]
	for _, p := range g.FilterBattlefield(And(ControlledBy(playerID), IsLand)) {
		g.TapPermanent(p)
	}
	return nil
}
func (e *tapAllLandsEffect) Text() string { return "Tap all lands target player controls" }
func (e *tapAllLandsEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// removeFromCombatEffect removes a target creature from combat.
type removeFromCombatEffect struct{}

// RemoveFromCombat creates an effect that removes a target creature from combat.
func RemoveFromCombat() Effect { return &removeFromCombatEffect{} }

func (e *removeFromCombatEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	g.RemoveFromCombat(perm.ID())
	return nil
}
func (e *removeFromCombatEffect) Text() string { return "Remove target creature from combat" }
func (e *removeFromCombatEffect) Properties() EffectProperties { return EffectProperties{} }

// makeUnblockableUntilEndOfTurnEffect makes a target creature unblockable until end of turn.
type makeUnblockableUntilEndOfTurnEffect struct{}

// MakeUnblockableUntilEndOfTurn creates an effect that makes a target creature unblockable until end of turn.
func MakeUnblockableUntilEndOfTurn() Effect {
	return &makeUnblockableUntilEndOfTurnEffect{}
}

func (e *makeUnblockableUntilEndOfTurnEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	eff := TemporaryKeyword(perm.ID(), UnblockableKW)
	eff.SetSourceID(sourceID)
	g.AddContinuousEffect(eff)
	g.ApplyContinuousEffects()
	return nil
}

func (e *makeUnblockableUntilEndOfTurnEffect) Text() string {
	return "Target creature can't be blocked this turn"
}
func (e *makeUnblockableUntilEndOfTurnEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// boostUntilEndOfTurnEffect boosts a creature's P/T until end of turn.
type boostUntilEndOfTurnEffect struct {
	power     ValueSource
	toughness ValueSource
	target    PermanentSelector
}

// BoostUntilEndOfTurn creates an effect that boosts the selected creature's P/T until end of turn.
func BoostUntilEndOfTurn(power, toughness ValueSource, target PermanentSelector) Effect {
	return &boostUntilEndOfTurnEffect{power: power, toughness: toughness, target: target}
}

func (e *boostUntilEndOfTurnEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	var perm *Permanent
	if e.target == SelectSource {
		perm = g.FindPermanent(sourceID)
	} else {
		if len(targets) == 0 {
			return fmt.Errorf("no target for boost")
		}
		perm = g.FindPermanent(targets[0])
	}
	if perm == nil {
		return nil
	}
	p := e.power.Resolve(g, sourceID, controller)
	t := e.toughness.Resolve(g, sourceID, controller)
	eff := TemporaryBoost(perm.ID(), p, t)
	eff.SetSourceID(sourceID)
	g.AddContinuousEffect(eff)
	g.ApplyContinuousEffects()
	return nil
}

func (e *boostUntilEndOfTurnEffect) Text() string {
	_, pIsX := e.power.(xValue)
	_, tIsX := e.toughness.(xValue)
	if pIsX || tIsX {
		return "Target creature gets +X/+0 until end of turn"
	}
	p := e.power.Resolve(nil, uuid.Nil, uuid.Nil)
	t := e.toughness.Resolve(nil, uuid.Nil, uuid.Nil)
	if e.target == SelectSource {
		return fmt.Sprintf("this creature gets +%d/+%d until end of turn", p, t)
	}
	return fmt.Sprintf("target creature gets +%d/+%d until end of turn", p, t)
}
func (e *boostUntilEndOfTurnEffect) Properties() EffectProperties {
	var pb, tb int
	if _, ok := e.power.(xValue); !ok {
		pb = e.power.Resolve(nil, uuid.Nil, uuid.Nil)
	}
	if _, ok := e.toughness.(xValue); !ok {
		tb = e.toughness.Resolve(nil, uuid.Nil, uuid.Nil)
	}
	return EffectProperties{Outcome: OutcomeBenefit, PowerBoost: pb, ToughnessBoost: tb}
}

// boostMatchingUntilEndOfTurnEffect boosts the P/T of all creatures matching a predicate until end of turn
type boostMatchingUntilEndOfTurnEffect struct {
	power     ValueSource
	toughness ValueSource
	predicate PermanentFilter
}

// BoostMatchingUntilEndOfTurn creates an effect that gives +P/+T until end of turn to all
// creatures the controller owns that match the predicate (e.g. Crusade, Bad Moon).
func BoostMatchingUntilEndOfTurn(power, toughness ValueSource, predicate PermanentFilter) Effect {
	return &boostMatchingUntilEndOfTurnEffect{power: power, toughness: toughness, predicate: predicate}
}

func (e *boostMatchingUntilEndOfTurnEffect) Apply(g GameMutator, sourceID uuid.UUID, controller uuid.UUID, targets []uuid.UUID) error {
	for _, perm := range g.FilterBattlefield(And(ControlledBy(controller), IsCreature, e.predicate)) {
		p := e.power.Resolve(g, sourceID, controller)
		t := e.toughness.Resolve(g, sourceID, controller)
		eff := TemporaryBoost(perm.ID(), p, t)
		eff.SetSourceID(sourceID)
		g.AddContinuousEffect(eff)
	}
	g.ApplyContinuousEffects()
	return nil
}

func (e *boostMatchingUntilEndOfTurnEffect) Text() string {
	return "XXX populate filter predicate text"
}
func (e *boostMatchingUntilEndOfTurnEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit, Mass: true}
}

// boostAllMatchingUntilEndOfTurnEffect boosts the P/T of all creatures matching a predicate until end of turn,
// regardless of controller (e.g. Piety, Army of Allah).
type boostAllMatchingUntilEndOfTurnEffect struct {
	power     ValueSource
	toughness ValueSource
	predicate PermanentFilter
}

// BoostAllMatchingUntilEndOfTurn creates an effect that gives +P/+T until end of turn to all
// creatures on the battlefield that match the predicate, regardless of controller.
func BoostAllMatchingUntilEndOfTurn(power, toughness ValueSource, predicate PermanentFilter) Effect {
	return &boostAllMatchingUntilEndOfTurnEffect{power: power, toughness: toughness, predicate: predicate}
}

func (e *boostAllMatchingUntilEndOfTurnEffect) Apply(g GameMutator, sourceID uuid.UUID, controller uuid.UUID, targets []uuid.UUID) error {
	for _, perm := range g.FilterBattlefield(And(IsCreature, e.predicate)) {
		p := e.power.Resolve(g, sourceID, controller)
		t := e.toughness.Resolve(g, sourceID, controller)
		eff := TemporaryBoost(perm.ID(), p, t)
		eff.SetSourceID(sourceID)
		g.AddContinuousEffect(eff)
	}
	g.ApplyContinuousEffects()
	return nil
}

func (e *boostAllMatchingUntilEndOfTurnEffect) Text() string {
	return "matching creatures get a boost until end of turn"
}
func (e *boostAllMatchingUntilEndOfTurnEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit, Mass: true}
}

// doubleSourcePowerEffect doubles the source creature's power until end of turn.
type doubleSourcePowerEffect struct{}

// DoubleTargetPower creates an effect that doubles a target creature's power until end of turn (e.g. Berserk).
func DoubleTargetPower() Effect {
	return &doubleSourcePowerEffect{}
}

func (e *doubleSourcePowerEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	currentPower := perm.CurrentPower(g)
	eff := TemporaryBoost(perm.ID(), currentPower, 0)
	eff.SetSourceID(sourceID)
	g.AddContinuousEffect(eff)
	g.ApplyContinuousEffects()
	return nil
}

func (e *doubleSourcePowerEffect) Text() string {
	return "Target creature's power is doubled until end of turn"
}
func (e *doubleSourcePowerEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// grantKeywordUntilEndOfTurnEffect grants a keyword to the source or a target
// creature until end of turn.
type grantKeywordUntilEndOfTurnEffect struct {
	keyword Keyword
	target  PermanentSelector
}

// GrantKeywordUntilEndOfTurn creates an effect that grants a keyword to the selected creature until end of turn.
func GrantKeywordUntilEndOfTurn(kw Keyword, target PermanentSelector) Effect {
	return &grantKeywordUntilEndOfTurnEffect{keyword: kw, target: target}
}

func (e *grantKeywordUntilEndOfTurnEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	var perm *Permanent
	if e.target == SelectSource {
		perm = g.FindPermanent(sourceID)
	} else {
		if len(targets) == 0 {
			return fmt.Errorf("no target for keyword grant")
		}
		perm = g.FindPermanent(targets[0])
	}
	if perm == nil {
		return nil
	}
	eff := TemporaryKeyword(perm.ID(), e.keyword)
	eff.SetSourceID(sourceID)
	g.AddContinuousEffect(eff)
	g.ApplyContinuousEffects()
	return nil
}

func (e *grantKeywordUntilEndOfTurnEffect) Text() string {
	if e.target == SelectSource {
		return fmt.Sprintf("~ gains %s until end of turn", e.keyword)
	}
	return fmt.Sprintf("target creature gains %s until end of turn", e.keyword)
}
func (e *grantKeywordUntilEndOfTurnEffect) Properties() EffectProperties {
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

func (e *replaceKeywordEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	// Remove the old keyword and add the new one as a continuous effect
	eff := KeywordReplacement(perm.ID(), e.from, e.to)
	eff.SetSourceID(sourceID)
	g.AddContinuousEffect(eff)
	g.ApplyContinuousEffects()
	return nil
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

func (e *regenerateSourceEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	perm := g.FindPermanent(sourceID)
	if perm == nil {
		return nil
	}
	g.AddRegenerationShield(sourceID)
	return nil
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

func (e *regenerateTargetEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	g.AddRegenerationShield(perm.ID())
	return nil
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

func (e *markDestroyAtEOTAfterNActivationsEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	perm := g.FindPermanent(sourceID)
	if perm == nil {
		return nil
	}
	perm.AddCounter(Charge, 1)
	if perm.Counters[Charge] >= e.threshold {
		g.RegisterDelayedTrigger(&DelayedTrigger{
			EventType:  EvtEndStep,
			TargetID:   perm.ID(),
			Effects:    []Effect{DestroyTarget()},
			SourceID:   sourceID,
			Controller: controller,
		})
	}
	return nil
}

func (e *markDestroyAtEOTAfterNActivationsEffect) Text() string {
	return fmt.Sprintf("if activated %d+ times, destroy at end of turn", e.threshold)
}
func (e *markDestroyAtEOTAfterNActivationsEffect) Properties() EffectProperties {
	return EffectProperties{}
}

// destroyTargetAtEndOfTurnEffect marks a creature for destruction at end of turn.
type destroyTargetAtEndOfTurnEffect struct{}

// DestroyTargetAtEndOfTurn creates an effect that registers a delayed trigger to destroy
// the target creature at the next end step.
func DestroyTargetAtEndOfTurn() Effect {
	return &destroyTargetAtEndOfTurnEffect{}
}

func (e *destroyTargetAtEndOfTurnEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	g.RegisterDelayedTrigger(&DelayedTrigger{
		EventType:  EvtEndStep,
		TargetID:   perm.ID(),
		Effects:    []Effect{DestroyTarget()},
		SourceID:   sourceID,
		Controller: controller,
	})
	return nil
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

func (e *setBasePTUntilEndOfTurnEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for set P/T")
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	eff := SetBasePT(perm.ID(), e.power, e.toughness)
	eff.SetSourceID(sourceID)
	g.AddContinuousEffect(eff)
	g.ApplyContinuousEffects()
	return nil
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

func (e *setBasePowerUntilEndOfTurnEffect) Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for set power")
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	eff := SetBasePower(perm.ID(), e.power)
	eff.SetSourceID(sourceID)
	g.AddContinuousEffect(eff)
	g.ApplyContinuousEffects()
	return nil
}

func (e *setBasePowerUntilEndOfTurnEffect) Text() string {
	return fmt.Sprintf("target creature has base power %d until end of turn", e.power)
}
func (e *setBasePowerUntilEndOfTurnEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
