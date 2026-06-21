package mage

import (
	"fmt"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
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
	KindAllTargets                    // every UUID in ctx.Targets (multi-target spells/abilities)
)

// TargetSelector describes which permanent(s) an effect applies to. Use the
// convenience constructors ToTarget(), ToSource(), ToAttached(), ToMatching(),
// ToAllMatching(), and ToGathered() to create selectors.
type TargetSelector struct {
	Kind    TargetKind
	Filter  PermanentFilter
	VarName string
}

func ToTarget() TargetSelector   { return TargetSelector{Kind: KindTarget} }
func ToSource() TargetSelector   { return TargetSelector{Kind: KindSource} }
func ToAttached() TargetSelector { return TargetSelector{Kind: KindAttached} }
func ToMatching(f PermanentFilter) TargetSelector {
	return TargetSelector{Kind: KindMatching, Filter: f}
}
func ToAllMatching(f PermanentFilter) TargetSelector {
	return TargetSelector{Kind: KindAllMatching, Filter: f}
}
func ToGathered(varName string) TargetSelector {
	return TargetSelector{Kind: KindGathered, VarName: varName}
}

// ToAllTargets selects every chosen target on a multi-target spell or ability.
// Effects using this selector are applied once per UUID in ctx.Targets, skipping
// uuid.Nil placeholders and any UUIDs that no longer resolve to a battlefield
// permanent (so a multi-target spell whose first target leaves still affects
// the surviving targets, per CR 608.2b).
func ToAllTargets() TargetSelector { return TargetSelector{Kind: KindAllTargets} }

// BoostUntilEndOfTurn is a compatibility helper for the older boost API.
func BoostUntilEndOfTurn(power, toughness ValueSource, target PermanentSelector) Effect {
	sel := ToTarget()
	if target == SelectSource {
		sel = ToSource()
	}
	return Boost(power, toughness).Targeting(sel).Until(EndOfTurn)
}

// BoostMatchingUntilEndOfTurn is a compatibility helper for the older boost API.
func BoostMatchingUntilEndOfTurn(power, toughness ValueSource, filter PermanentFilter) Effect {
	return Boost(power, toughness).Targeting(ToMatching(filter)).Until(EndOfTurn)
}

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
	case KindAllTargets:
		var out []*Permanent
		for _, id := range ctx.Targets {
			if id == uuid.Nil {
				continue
			}
			if p := ctx.Game.FindPermanent(id); p != nil {
				out = append(out, p)
			}
		}
		return out
	}
	return nil
}

// --- Boost DSL ---

// boostEffect is a composable Effect that temporarily modifies P/T.
// Use directly as an Effect.
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

// Effect interface
func (e *boostEffect) Text() string {
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

func (e *boostEffect) Properties() EffectProperties {
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

// grantKeywordEffect is a composable Effect that temporarily grants a keyword.
// Use directly as an Effect.
type grantKeywordEffect struct {
	keyword  Keyword
	selector TargetSelector
	dur      Duration
	unless   TriggerConditionData
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

// Unless suppresses the grant if the given condition is true at execution time.
func (e *grantKeywordEffect) Unless(cond TriggerConditionData) *grantKeywordEffect {
	e.unless = cond
	return e
}

// Effect interface
func (e *grantKeywordEffect) Text() string {
	switch e.selector.Kind {
	case KindSource:
		return fmt.Sprintf("~ gains %s until end of turn", e.keyword)
	default:
		return fmt.Sprintf("target creature gains %s until end of turn", e.keyword)
	}
}

func (e *grantKeywordEffect) Properties() EffectProperties {
	outcome := OutcomeBenefit
	if e.keyword == CantRegenerate {
		outcome = OutcomeDetriment
	}
	return EffectProperties{Outcome: outcome, GrantedKeyword: e.keyword}
}

func execGrantKeyword(ctx *EffectContext, e *grantKeywordEffect) error {
	if e.unless != nil && e.unless.CheckTriggerCond(&GameEvent{}, ctx.Game, ctx.SourceID, ctx.Controller) {
		return nil
	}
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

// --- RevokeKeyword DSL ---

// revokeKeywordEffect is a composable Effect that temporarily removes a
// keyword from a permanent. Mirrors grantKeywordEffect.
type revokeKeywordEffect struct {
	keyword  Keyword
	selector TargetSelector
	dur      Duration
}

// RevokeKeyword creates an effect that removes a keyword from a permanent.
// Defaults to targeting targets[0] until end of turn. Use .Targeting() and
// .Until() to override.
func RevokeKeyword(kw Keyword) *revokeKeywordEffect {
	return &revokeKeywordEffect{
		keyword:  kw,
		selector: TargetSelector{Kind: KindTarget},
		dur:      EndOfTurn,
	}
}

func (e *revokeKeywordEffect) Targeting(sel TargetSelector) *revokeKeywordEffect {
	e.selector = sel
	return e
}

func (e *revokeKeywordEffect) Until(d Duration) *revokeKeywordEffect {
	e.dur = d
	return e
}

func (e *revokeKeywordEffect) Text() string {
	switch e.selector.Kind {
	case KindSource:
		return fmt.Sprintf("~ loses %s until end of turn", e.keyword)
	default:
		return fmt.Sprintf("target creature loses %s until end of turn", e.keyword)
	}
}

func (e *revokeKeywordEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

func execRevokeKeyword(ctx *EffectContext, e *revokeKeywordEffect) error {
	perms := resolvePermanents(ctx, e.selector)
	for _, perm := range perms {
		targetID := perm.ID()
		eff := FuncContinuousEffect(LayerAbility, e.dur, func(g *Game, _ uuid.UUID) error {
			if p := g.FindPermanent(targetID); p != nil {
				g.RevokeAttr(p.ID(), Attr(e.keyword))
			}
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

// --- GrantAbility DSL ---

// grantAbilityEffect is a composable Effect that temporarily grants a
// non-keyword ability (typically a triggered ability like Rampage N) to the
// selected permanent. Mirrors grantKeywordEffect for parameterized abilities
// that don't fit into the Keyword/Attr enum.
type grantAbilityEffect struct {
	ability  Ability
	selector TargetSelector
	dur      Duration
	unless   TriggerConditionData
}

// GrantAbility creates a temporary ability grant effect. Defaults to targeting
// targets[0] until end of turn. Use .Targeting() / .Until() / .Unless().
func GrantAbility(a Ability) *grantAbilityEffect {
	return &grantAbilityEffect{
		ability:  a,
		selector: TargetSelector{Kind: KindTarget},
		dur:      EndOfTurn,
	}
}

func (e *grantAbilityEffect) Targeting(sel TargetSelector) *grantAbilityEffect {
	e.selector = sel
	return e
}

func (e *grantAbilityEffect) Until(d Duration) *grantAbilityEffect {
	e.dur = d
	return e
}

func (e *grantAbilityEffect) Unless(cond TriggerConditionData) *grantAbilityEffect {
	e.unless = cond
	return e
}

func (e *grantAbilityEffect) Text() string { return "grant ability until end of turn" }
func (e *grantAbilityEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

func execGrantAbility(ctx *EffectContext, e *grantAbilityEffect) error {
	if e.unless != nil && e.unless.CheckTriggerCond(&GameEvent{}, ctx.Game, ctx.SourceID, ctx.Controller) {
		return nil
	}
	perms := resolvePermanents(ctx, e.selector)
	for _, perm := range perms {
		targetID := perm.ID()
		ability := e.ability
		eff := FuncContinuousEffect(LayerAbility, e.dur, func(g *Game, _ uuid.UUID) error {
			p := g.FindPermanent(targetID)
			if p == nil {
				return nil
			}
			ability.SetSource(targetID)
			ability.SetController(p.Controller)
			p.RuntimeAbilities = append(p.RuntimeAbilities, WrapGrantedAbility(ability))
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

// --- GrantType DSL ---

// CardTypeAttr maps a CardType to the corresponding Attr for battlefield type identity.
func CardTypeAttr(ct CardType) Attr {
	switch ct {
	case TypeCreature:
		return AttrIsCreature
	case TypeArtifact:
		return AttrIsArtifact
	case TypeLand:
		return AttrIsLand
	case TypeEnchantment:
		return AttrIsEnchantment
	default:
		return 0
	}
}

// grantTypeEffect is a composable Effect that grants an additional card type
// to a permanent via an indefinite continuous effect at LayerType.
// Use directly as an Effect.
type grantTypeEffect struct {
	ct       CardType
	selector TargetSelector
	dur      Duration
}

// GrantType creates an effect that grants an additional card type to a permanent.
// Defaults to targeting targets[0] with Indefinite duration (lasts while target
// remains on battlefield). Use .Targeting() and .Until() to override.
func GrantType(ct CardType) *grantTypeEffect {
	return &grantTypeEffect{
		ct:       ct,
		selector: TargetSelector{Kind: KindTarget},
		dur:      Indefinite,
	}
}

func (e *grantTypeEffect) Targeting(sel TargetSelector) *grantTypeEffect {
	e.selector = sel
	return e
}

func (e *grantTypeEffect) Until(d Duration) *grantTypeEffect {
	e.dur = d
	return e
}

// Effect interface
func (e *grantTypeEffect) Text() string {
	return fmt.Sprintf("becomes a %s in addition to its other types", e.ct)
}

func (e *grantTypeEffect) Properties() EffectProperties {
	return EffectProperties{}
}

func execGrantType(ctx *EffectContext, e *grantTypeEffect) error {
	attr := CardTypeAttr(e.ct)
	if attr == 0 {
		return nil
	}
	perms := resolvePermanents(ctx, e.selector)
	for _, perm := range perms {
		eff := TargetEffect(LayerType, e.dur, perm.ID(), func(g *Game, target *Permanent) error {
			g.GrantAttr(target.ID(), attr)
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
