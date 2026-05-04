package mage

import (
	"fmt"

	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// --- Add counters ---

// addCountersEffect is a composable effect that adds counters to a permanent.
// It implements both EffectData (for pipelines) and Effect (for direct use).
// Defaults to targeting targets[0]; use .Targeting() to override.
type addCountersEffect struct {
	ct       CounterType
	amount   ValueSource
	selector TargetSelector
	maxTotal int // 0 = no cap
}

// AddCounters creates an effect that adds counters to a permanent.
// Defaults to targeting targets[0]. Use .Targeting() to override.
func AddCounters(ct CounterType, amount ValueSource) *addCountersEffect {
	return &addCountersEffect{
		ct:       ct,
		amount:   amount,
		selector: TargetSelector{Kind: KindTarget},
	}
}

func (e *addCountersEffect) Targeting(sel TargetSelector) *addCountersEffect {
	e.selector = sel
	return e
}

// Max caps the total counters of this type on the permanent.
func (e *addCountersEffect) Max(n int) *addCountersEffect {
	e.maxTotal = n
	return e
}

// EffectData interface
func (e *addCountersEffect) Text() string {
	if _, ok := e.amount.(xValue); ok {
		if e.maxTotal > 0 {
			return fmt.Sprintf("put up to X %s counters on it (max %d total)", e.ct, e.maxTotal)
		}
		return fmt.Sprintf("put X %s counters on it", e.ct)
	}
	n := e.amount.Resolve(nil, uuid.Nil, uuid.Nil, nil)
	switch e.selector.Kind {
	case KindSource:
		return fmt.Sprintf("put %d %s counter(s) on it", n, e.ct)
	case KindAttached:
		return "add counter to enchanted permanent"
	default:
		return fmt.Sprintf("put %d %s counter(s) on target", n, e.ct)
	}
}

func (e *addCountersEffect) Properties() EffectProperties {
	if e.maxTotal > 0 {
		return EffectProperties{Outcome: OutcomeBenefit}
	}
	return EffectProperties{}
}

func execAddCounters(ctx *EffectContext, e *addCountersEffect) error {
	perms := resolvePermanents(ctx, e.selector)
	for _, perm := range perms {
		amount := e.amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
		if e.maxTotal > 0 {
			current := int(perm.Counters[e.ct])
			room := max(e.maxTotal-current, 0)
			amount = min(amount, room)
		}
		if amount > 0 {
			ctx.Game.AddCountersWithReplacement(perm, e.ct, amount, ctx.SourceID, false)
		}
	}
	return nil
}

// --- Remove counters ---

// removeCountersEffect is a composable effect that removes counters from a
// permanent. It implements both EffectData (for pipelines) and Effect (for
// direct use). Defaults to targeting the source; use .Targeting() to override.
type removeCountersEffect struct {
	ct       CounterType
	amount   int
	selector TargetSelector
}

// RemoveCounters creates an effect that removes counters from a permanent.
// Defaults to targeting the source. Use .Targeting() to override.
func RemoveCounters(ct CounterType, amount int) *removeCountersEffect {
	return &removeCountersEffect{
		ct:       ct,
		amount:   amount,
		selector: TargetSelector{Kind: KindSource},
	}
}

func (e *removeCountersEffect) Targeting(sel TargetSelector) *removeCountersEffect {
	e.selector = sel
	return e
}

// EffectData interface
func (e *removeCountersEffect) Text() string {
	switch e.selector.Kind {
	case KindSource:
		return fmt.Sprintf("remove %d %s counter(s) from it", e.amount, e.ct)
	default:
		return fmt.Sprintf("remove %d %s counter(s) from target", e.amount, e.ct)
	}
}
func (e *removeCountersEffect) Properties() EffectProperties { return EffectProperties{} }

func execRemoveCounters(ctx *EffectContext, e *removeCountersEffect) error {
	perms := resolvePermanents(ctx, e.selector)
	for _, p := range perms {
		p.RemoveCounter(e.ct, e.amount)
	}
	return nil
}

// MoveCountersFromSourceToTarget creates an effect that prompts the
// controller to move up to N counters of the given type from the source
// permanent to the resolving target permanent. N is capped at the
// number of counters currently on the source. Used by Scrounging
// Bandar / Power Conduit / Crystalline Crawler-shape upkeep triggers
// ("you may move any number of +1/+1 counters from this creature onto
// another target creature"). The choice is delegated to
// Player.ChooseNumber(0, available, reason) so AI/TUI can override.
func MoveCountersFromSourceToTarget(ct CounterType) Effect {
	return FuncEffect(
		"move any number of counters from source to target",
		EffectProperties{Outcome: OutcomeBenefit},
		func(g *Game, sourceID, controllerID uuid.UUID, targets []uuid.UUID) error {
			if len(targets) == 0 {
				return nil
			}
			source := g.FindPermanent(sourceID)
			target := g.FindPermanent(targets[0])
			if source == nil || target == nil {
				return nil
			}
			available := int(source.Counters[ct])
			if available <= 0 {
				return nil
			}
			player := g.GetPlayer(controllerID)
			if player == nil {
				return nil
			}
			n := player.ChooseNumber(0, available, fmt.Sprintf("Move how many %s counters?", ct))
			if n <= 0 {
				return nil
			}
			if n > available {
				n = available
			}
			source.RemoveCounter(ct, n)
			target.AddCounter(ct, n)
			g.ApplyContinuousEffects()
			return nil
		},
	)
}

// --- Snapshot source counter (pipeline) ---

// SnapshotSourceCounterData reads a counter value from the source permanent.
type SnapshotSourceCounterData struct {
	CounterType CounterType
	StoreAs     string
}

// SnapshotSourceCounter reads a counter count from the source permanent.
func SnapshotSourceCounter(ct CounterType, storeAs string) EffectData {
	return &SnapshotSourceCounterData{CounterType: ct, StoreAs: storeAs}
}

func (e *SnapshotSourceCounterData) Text() string                 { return "" }
func (e *SnapshotSourceCounterData) Properties() EffectProperties { return EffectProperties{} }

func execSnapshotSourceCounter(ctx *EffectContext, e *SnapshotSourceCounterData) error {
	perm := ctx.Game.FindPermanent(ctx.SourceID)
	if perm == nil {
		ctx.SetInt(e.StoreAs, 0)
		return nil
	}
	ctx.SetInt(e.StoreAs, int(perm.Counters[e.CounterType]))
	return nil
}

// --- Source has counter condition ---

// SourceHasCounterCond checks if the source permanent has at least MinCount of a counter type.
type SourceHasCounterCond struct {
	CounterType CounterType
	MinCount    int
}

func (c *SourceHasCounterCond) Check(ctx *EffectContext) bool {
	perm := ctx.Game.FindPermanent(ctx.SourceID)
	if perm == nil {
		return false
	}
	return int(perm.Counters[c.CounterType]) >= c.MinCount
}
