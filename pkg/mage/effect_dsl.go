package mage

import (
	"fmt"

	"github.com/google/uuid"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// TargetKind describes which permanent(s) an effect applies to.
type TargetKind int

const (
	KindTarget      TargetKind = iota // targets[0] from spell/ability targeting
	KindSource                        // the source permanent
	KindAttached                      // the permanent source is attached to
	KindMatching                      // controlled creatures matching a filter
	KindAllMatching                   // all creatures on battlefield matching a filter
	KindGathered                      // permanent ID from a pipeline context variable
)

// TargetSelector describes which permanent(s) an effect applies to. Use the
// convenience constructors ToTarget(), ToSource(), ToAttached(), ToMatching(),
// ToAllMatching(), and ToGathered() to create selectors.
type TargetSelector struct {
	Kind    TargetKind
	Filter  PermanentFilter
	VarName string
}

func ToTarget() TargetSelector                       { return TargetSelector{Kind: KindTarget} }
func ToSource() TargetSelector                       { return TargetSelector{Kind: KindSource} }
func ToAttached() TargetSelector                     { return TargetSelector{Kind: KindAttached} }
func ToMatching(f PermanentFilter) TargetSelector    { return TargetSelector{Kind: KindMatching, Filter: f} }
func ToAllMatching(f PermanentFilter) TargetSelector { return TargetSelector{Kind: KindAllMatching, Filter: f} }
func ToGathered(varName string) TargetSelector       { return TargetSelector{Kind: KindGathered, VarName: varName} }

// resolvePermanents resolves the permanent(s) identified by a TargetSelector.
func resolvePermanents(ctx *EffectContext, sel TargetSelector) []*Permanent {
	switch sel.Kind {
	case KindTarget:
		if len(ctx.Targets) == 0 {
			return nil
		}
		if p := ctx.Game.FindPermanent(ctx.Targets[0]); p != nil {
			return []*Permanent{p}
		}
	case KindSource:
		if p := ctx.Game.FindPermanent(ctx.SourceID); p != nil {
			return []*Permanent{p}
		}
	case KindAttached:
		src := ctx.Game.FindPermanent(ctx.SourceID)
		if src != nil && src.AttachedTo != uuid.Nil {
			if p := ctx.Game.FindPermanent(src.AttachedTo); p != nil {
				return []*Permanent{p}
			}
		}
	case KindMatching:
		return ctx.Game.FilterBattlefield(And(ControlledBy(ctx.Controller), IsCreature, sel.Filter))
	case KindAllMatching:
		return ctx.Game.FilterBattlefield(And(IsCreature, sel.Filter))
	case KindGathered:
		id := ctx.TryGetUUID(sel.VarName)
		if id == uuid.Nil {
			return nil
		}
		if p := ctx.Game.FindPermanent(id); p != nil {
			return []*Permanent{p}
		}
	}
	return nil
}

// --- Boost DSL ---

// boostEffect is a composable effect that temporarily modifies P/T. It
// implements both EffectData (for pipelines) and Effect (for direct use).
type boostEffect struct {
	power     ValueSource
	toughness ValueSource
	selector  TargetSelector
	dur       Duration
}

// Boost creates a temporary P/T modification effect. Defaults to targeting
// targets[0] until end of turn. Use .Targeting() and .Until() to override.
func Boost(power, toughness ValueSource) *boostEffect {
	return &boostEffect{
		power:     power,
		toughness: toughness,
		selector:  TargetSelector{Kind: KindTarget},
		dur:       EndOfTurn,
	}
}

func (e *boostEffect) Targeting(sel TargetSelector) *boostEffect {
	e.selector = sel
	return e
}

func (e *boostEffect) Until(d Duration) *boostEffect {
	e.dur = d
	return e
}

// EffectData interface
func (e *boostEffect) EffectText() string {
	_, pIsX := e.power.(xValue)
	_, tIsX := e.toughness.(xValue)
	if pIsX || tIsX {
		return "gets +X/+0 until end of turn"
	}
	p := e.power.Resolve(nil, uuid.Nil, uuid.Nil, nil)
	t := e.toughness.Resolve(nil, uuid.Nil, uuid.Nil, nil)
	switch e.selector.Kind {
	case KindSource:
		return fmt.Sprintf("this creature gets %+d/%+d until end of turn", p, t)
	case KindMatching, KindAllMatching:
		return fmt.Sprintf("matching creatures get %+d/%+d until end of turn", p, t)
	default:
		return fmt.Sprintf("target creature gets %+d/%+d until end of turn", p, t)
	}
}

func (e *boostEffect) EffectProps() EffectProperties {
	var pb, tb int
	if _, ok := e.power.(xValue); !ok {
		pb = e.power.Resolve(nil, uuid.Nil, uuid.Nil, nil)
	}
	if _, ok := e.toughness.(xValue); !ok {
		tb = e.toughness.Resolve(nil, uuid.Nil, uuid.Nil, nil)
	}
	mass := e.selector.Kind == KindMatching || e.selector.Kind == KindAllMatching
	return EffectProperties{Outcome: OutcomeBenefit, PowerBoost: pb, ToughnessBoost: tb, Mass: mass}
}

// Effect interface — allows direct use without DataEffect() wrapper.
func (e *boostEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	ctx := &EffectContext{
		Game:       g,
		SourceID:   sourceID,
		Controller: controller,
		Targets:    targets,
		Vars:       make(map[string]any),
	}
	return execBoost(ctx, e)
}

func (e *boostEffect) Text() string            { return e.EffectText() }
func (e *boostEffect) Properties() EffectProperties { return e.EffectProps() }

func execBoost(ctx *EffectContext, e *boostEffect) error {
	perms := resolvePermanents(ctx, e.selector)
	for _, perm := range perms {
		p := e.power.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
		t := e.toughness.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
		eff := TargetEffect(LayerPT, e.dur, perm.ID(), func(g *Game, target *Permanent) error {
			target.powerBonus += p
			target.toughBonus += t
			return nil
		})
		eff.SetSourceID(ctx.SourceID)
		ctx.Game.AddContinuousEffect(eff)
	}
	if len(perms) > 0 {
		ctx.Game.ApplyContinuousEffects()
	}
	return nil
}

// --- GrantKeyword DSL ---

// grantKeywordEffect is a composable effect that temporarily grants a keyword.
// It implements both EffectData (for pipelines) and Effect (for direct use).
type grantKeywordEffect struct {
	keyword  Keyword
	selector TargetSelector
	dur      Duration
}

// GrantKeyword creates a temporary keyword grant effect. Defaults to targeting
// targets[0] until end of turn. Use .Targeting() and .Until() to override.
func GrantKeyword(kw Keyword) *grantKeywordEffect {
	return &grantKeywordEffect{
		keyword:  kw,
		selector: TargetSelector{Kind: KindTarget},
		dur:      EndOfTurn,
	}
}

func (e *grantKeywordEffect) Targeting(sel TargetSelector) *grantKeywordEffect {
	e.selector = sel
	return e
}

func (e *grantKeywordEffect) Until(d Duration) *grantKeywordEffect {
	e.dur = d
	return e
}

// EffectData interface
func (e *grantKeywordEffect) EffectText() string {
	switch e.selector.Kind {
	case KindSource:
		return fmt.Sprintf("~ gains %s until end of turn", e.keyword)
	default:
		return fmt.Sprintf("target creature gains %s until end of turn", e.keyword)
	}
}

func (e *grantKeywordEffect) EffectProps() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// Effect interface — allows direct use without DataEffect() wrapper.
func (e *grantKeywordEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	ctx := &EffectContext{
		Game:       g,
		SourceID:   sourceID,
		Controller: controller,
		Targets:    targets,
		Vars:       make(map[string]any),
	}
	return execGrantKeyword(ctx, e)
}

func (e *grantKeywordEffect) Text() string            { return e.EffectText() }
func (e *grantKeywordEffect) Properties() EffectProperties { return e.EffectProps() }

func execGrantKeyword(ctx *EffectContext, e *grantKeywordEffect) error {
	perms := resolvePermanents(ctx, e.selector)
	for _, perm := range perms {
		eff := TargetEffect(LayerAbility, e.dur, perm.ID(), func(g *Game, target *Permanent) error {
			g.GrantAttr(target.ID(), e.keyword)
			return nil
		})
		eff.SetSourceID(ctx.SourceID)
		ctx.Game.AddContinuousEffect(eff)
	}
	if len(perms) > 0 {
		ctx.Game.ApplyContinuousEffects()
	}
	return nil
}
