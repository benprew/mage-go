package mage

import (
	"fmt"
	"slices"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// TargetKind describes which permanent(s) an effect applies to.
type TargetKind int

const (
	KindTarget                    TargetKind = iota // targets[0] from spell/ability targeting
	KindSource                                      // the source permanent
	KindAttached                                    // the permanent source is attached to
	KindMatching                                    // controlled creatures matching a filter
	KindAllMatching                                 // all creatures on battlefield matching a filter
	KindOtherMatching                               // all creatures on battlefield matching a filter, except source
	KindGathered                                    // permanent ID from a pipeline context variable
	KindAllTargets                                  // every UUID in ctx.legacyTargets() (multi-target spells/abilities)
	KindBlockingOrBlockedBySource                   // creatures blocking or blocked by the source permanent
)

// TargetSelector describes which permanent(s) an effect applies to. Use the
// convenience constructors ToTarget(), ToSource(), ToAttached(), ToMatching(),
// ToAllMatching(), ToOtherMatching(), and ToGathered() to create selectors.
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
func ToOtherMatching(f PermanentFilter) TargetSelector {
	return TargetSelector{Kind: KindOtherMatching, Filter: f}
}
func ToGathered(varName string) TargetSelector {
	return TargetSelector{Kind: KindGathered, VarName: varName}
}

// ToAllTargets selects every chosen target on a multi-target spell or ability.
// Effects using this selector are applied once per UUID in ctx.legacyTargets(), skipping
// uuid.Nil() placeholders and any UUIDs that no longer resolve to a battlefield
// permanent (so a multi-target spell whose first target leaves still affects
// the surviving targets, per CR 608.2b).
func ToAllTargets() TargetSelector { return TargetSelector{Kind: KindAllTargets} }
func ToBlockingOrBlockedBySource() TargetSelector {
	return TargetSelector{Kind: KindBlockingOrBlockedBySource}
}

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

// resolvePermanents resolves the permanent(s) identified by an ObjectSelector.
func resolvePermanents(ctx *EffectContext, sel ObjectSelector) []*Permanent {
	if sel == nil || ctx == nil || ctx.Game == nil {
		return nil
	}
	var out []*Permanent
	for _, id := range currentObjectIDs(ctx, sel) {
		if id == uuid.Nil() {
			continue
		}
		if p := ctx.Game.FindPermanent(id); p != nil {
			out = append(out, p)
		}
	}
	return out
}

func (sel TargetSelector) Text() string {
	switch sel.Kind {
	case KindSource:
		return "it"
	case KindAttached:
		return "enchanted permanent"
	case KindMatching, KindAllMatching:
		return "matching creatures"
	case KindOtherMatching:
		return "other matching creatures"
	case KindBlockingOrBlockedBySource:
		return "blocking or blocked creature"
	default:
		return "target"
	}
}

func (sel TargetSelector) Select(ctx *EffectContext) []uuid.UUID {
	if ctx == nil || ctx.Game == nil {
		return nil
	}
	switch sel.Kind {
	case KindTarget:
		if len(ctx.legacyTargets()) == 0 {
			return nil
		}
		return []uuid.UUID{ctx.legacyTargets()[0]}
	case KindSource:
		if ctx.SourceID == uuid.Nil() {
			return nil
		}
		return []uuid.UUID{ctx.SourceID}
	case KindAttached:
		src := ctx.Game.FindPermanent(ctx.SourceID)
		if src != nil && src.AttachedTo != uuid.Nil() {
			return []uuid.UUID{src.AttachedTo}
		}
	case KindMatching:
		perms := ctx.Game.FilterBattlefield(And(ControlledBy(ctx.Controller), IsCreature, sel.Filter))
		ids := make([]uuid.UUID, len(perms))
		for i, p := range perms {
			ids[i] = p.ID()
		}
		return ids
	case KindAllMatching:
		perms := ctx.Game.FilterBattlefield(And(IsCreature, sel.Filter))
		ids := make([]uuid.UUID, len(perms))
		for i, p := range perms {
			ids[i] = p.ID()
		}
		return ids
	case KindOtherMatching:
		var ids []uuid.UUID
		for _, p := range ctx.Game.zones.battlefield {
			if p.ID() != ctx.SourceID && p.HasType(TypeCreature) && sel.Filter.Match(p, ctx.Game) {
				ids = append(ids, p.ID())
			}
		}
		return ids
	case KindGathered:
		id := ctx.TryGetUUID(sel.VarName)
		if id != uuid.Nil() {
			return []uuid.UUID{id}
		}
	case KindAllTargets:
		var ids []uuid.UUID
		for _, id := range ctx.legacyTargets() {
			if id != uuid.Nil() {
				ids = append(ids, id)
			}
		}
		return ids
	case KindBlockingOrBlockedBySource:
		var ids []uuid.UUID
		for _, cg := range ctx.Game.CombatGroups() {
			if cg.AttackerID == ctx.SourceID {
				for _, bid := range cg.BlockerIDs {
					if p := ctx.Game.FindPermanent(bid); p != nil {
						ids = append(ids, p.ID())
					}
				}
			} else {
				for _, bid := range cg.BlockerIDs {
					if bid == ctx.SourceID {
						if p := ctx.Game.FindPermanent(cg.AttackerID); p != nil {
							ids = append(ids, p.ID())
						}
					}
				}
			}
		}
		return ids
	}
	return nil
}

func isSourceSelector(sel ObjectSelector) bool {
	if ts, ok := sel.(TargetSelector); ok {
		return ts.Kind == KindSource
	}
	if ref, ok := sel.(objectReference); ok {
		return ref.kind == "source"
	}
	return false
}

func isAttachedSelector(sel ObjectSelector) bool {
	if ts, ok := sel.(TargetSelector); ok {
		return ts.Kind == KindAttached
	}
	return false
}

func isMassSelector(sel ObjectSelector) bool {
	if ts, ok := sel.(TargetSelector); ok {
		return ts.Kind == KindMatching || ts.Kind == KindAllMatching || ts.Kind == KindOtherMatching
	}
	_, ok := sel.(battlefieldObjects)
	return ok
}

// --- Boost DSL ---

// boostEffect is a composable Effect that temporarily modifies P/T.
// Use directly as an Effect.
type boostEffect struct {
	power         ValueSource
	toughness     ValueSource
	selector      ObjectSelector
	dur           Duration
	active        ActiveCondition
	tapMaintained bool
}

// Boost creates a temporary P/T modification effect. Defaults to targeting
// targets[0] until end of turn. Use .Targeting() and .Until() to override.
func Boost(power, toughness ValueSource) *boostEffect {
	return &boostEffect{
		power:     power,
		toughness: toughness,
		selector:  ToTarget(),
		dur:       EndOfTurn,
	}
}

func (e *boostEffect) Targeting(sel ObjectSelector) *boostEffect {
	e.selector = sel
	return e
}

func (e *boostEffect) Until(d Duration) *boostEffect {
	e.dur = d
	return e
}

// While applies the effect only while the source satisfies cond.
func (e *boostEffect) While(cond ActiveCondition) *boostEffect {
	e.active = cond
	return e
}

// WhileSourceTapped applies the boost only while the source remains tapped and
// marks it tap-maintained, so the untap step untaps the source automatically
// once the boosted creature leaves the battlefield (Tawnos's Weaponry).
func (e *boostEffect) WhileSourceTapped() *boostEffect {
	e.active = SourceTapped
	e.tapMaintained = true
	return e
}

// Effect interface
func (e *boostEffect) Text() string {
	_, pIsX := e.power.(xValue)
	_, tIsX := e.toughness.(xValue)
	if pIsX || tIsX {
		return "gets +X/+0 until end of turn"
	}
	p := e.power.Resolve(NewEffectContext(nil, uuid.Nil(), uuid.Nil(), nil))
	t := e.toughness.Resolve(NewEffectContext(nil, uuid.Nil(), uuid.Nil(), nil))
	switch {
	case isSourceSelector(e.selector):
		return fmt.Sprintf("this creature gets %+d/%+d until end of turn", p, t)
	case isMassSelector(e.selector):
		return fmt.Sprintf("matching creatures get %+d/%+d until end of turn", p, t)
	default:
		return fmt.Sprintf("target creature gets %+d/%+d until end of turn", p, t)
	}
}

func (e *boostEffect) Properties() EffectProperties {
	var pb, tb int
	if _, ok := e.power.(xValue); !ok {
		pb = e.power.Resolve(NewEffectContext(nil, uuid.Nil(), uuid.Nil(), nil))
	}
	if _, ok := e.toughness.(xValue); !ok {
		tb = e.toughness.Resolve(NewEffectContext(nil, uuid.Nil(), uuid.Nil(), nil))
	}
	return EffectProperties{Outcome: OutcomeBenefit, PowerBoost: pb, ToughnessBoost: tb, Mass: isMassSelector(e.selector)}
}

func (e *boostEffect) Apply(ctx *EffectContext) error {
	if !activeConditionStarted(ctx, e.active) {
		return nil
	}
	perms := resolvePermanents(ctx, e.selector)
	for _, perm := range perms {
		p := e.power.Resolve(ctx)
		t := e.toughness.Resolve(ctx)
		applyBoost := func(g *Game, target *Permanent) error {
			target.powerBonus += p
			target.toughBonus += t
			return nil
		}
		var eff ContinuousEffect
		if e.tapMaintained {
			eff = TapMaintainedTargetEffect(LayerPT, e.dur, perm.ID(), applyBoost, e.active)
		} else {
			eff = TargetEffectWhen(LayerPT, e.dur, perm.ID(), applyBoost, e.active)
		}
		ctx.Game.AddContinuousEffect(eff, ctx.SourceID)
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
	selector ObjectSelector
	dur      Duration
	unless   TriggerConditionData
	active   ActiveCondition
}

// GrantKeyword creates a temporary keyword grant effect. Defaults to targeting
// targets[0] until end of turn. Use .Targeting() and .Until() to override.
func GrantKeyword(kw Keyword) *grantKeywordEffect {
	return &grantKeywordEffect{
		keyword:  kw,
		selector: ToTarget(),
		dur:      EndOfTurn,
	}
}

func (e *grantKeywordEffect) Targeting(sel ObjectSelector) *grantKeywordEffect {
	e.selector = sel
	return e
}

func (e *grantKeywordEffect) Until(d Duration) *grantKeywordEffect {
	e.dur = d
	return e
}

// While applies the grant only while the source satisfies cond.
func (e *grantKeywordEffect) While(cond ActiveCondition) *grantKeywordEffect {
	e.active = cond
	return e
}

// KeywordGrantTarget exposes the recipient selector to AI policy without
// exposing the concrete effect implementation.
func (e *grantKeywordEffect) KeywordGrantTarget() ObjectSelector {
	return e.selector
}

// Unless suppresses the grant if the given condition is true at execution time.
func (e *grantKeywordEffect) Unless(cond TriggerConditionData) *grantKeywordEffect {
	e.unless = cond
	return e
}

// Effect interface
func (e *grantKeywordEffect) Text() string {
	if isSourceSelector(e.selector) {
		return fmt.Sprintf("~ gains %s until end of turn", e.keyword)
	}
	return fmt.Sprintf("target creature gains %s until end of turn", e.keyword)
}

func (e *grantKeywordEffect) Properties() EffectProperties {
	outcome := OutcomeBenefit
	if e.keyword == CantRegenerate {
		outcome = OutcomeDetriment
	}
	return EffectProperties{Outcome: outcome, GrantedKeyword: e.keyword}
}

func (e *grantKeywordEffect) Apply(ctx *EffectContext) error {
	if e.unless != nil && e.unless.CheckTriggerCond(&GameEvent{}, ctx.Game, ctx.SourceID, ctx.Controller) {
		return nil
	}
	if !activeConditionStarted(ctx, e.active) {
		return nil
	}
	perms := resolvePermanents(ctx, e.selector)
	for _, perm := range perms {
		eff := TargetEffectWhen(LayerAbility, e.dur, perm.ID(), func(g *Game, target *Permanent) error {
			g.GrantAttr(target.ID(), e.keyword)
			return nil
		}, e.active)
		ctx.Game.AddContinuousEffect(eff, ctx.SourceID)
	}
	if len(perms) > 0 {
		ctx.Game.ApplyContinuousEffects()
	}
	return nil
}

// --- RevokeKeyword DSL ---

// RevokeKeywordEffect is a composable Effect that temporarily removes a
// keyword from a permanent. Mirrors grantKeywordEffect.
type RevokeKeywordEffect struct {
	keywords []Keyword
	selector ObjectSelector
	dur      Duration
	active   ActiveCondition
}

// RevokeKeyword creates an effect that removes a keyword from a permanent.
// Defaults to targeting targets[0] until end of turn. Use .Targeting() and
// .Until() to override.
func RevokeKeyword(kw Keyword) *RevokeKeywordEffect {
	return RevokeKeywords(kw)
}

// RevokeKeywords creates an effect that removes one or more keywords from a
// permanent. Defaults to targeting targets[0] until end of turn.
func RevokeKeywords(kws ...Keyword) *RevokeKeywordEffect {
	return &RevokeKeywordEffect{
		keywords: kws,
		selector: ToTarget(),
		dur:      EndOfTurn,
	}
}

// RevokeLandwalk creates an effect that removes all landwalk abilities from a
// permanent. Defaults to targeting targets[0] until end of turn.
func RevokeLandwalk() *RevokeKeywordEffect {
	keywords := make([]Keyword, 0, len(LandwalkAttrs()))
	for kw := range LandwalkAttrs() {
		keywords = append(keywords, kw)
	}
	return RevokeKeywords(keywords...)
}

func (e *RevokeKeywordEffect) Targeting(sel ObjectSelector) *RevokeKeywordEffect {
	e.selector = sel
	return e
}

func (e *RevokeKeywordEffect) Until(d Duration) *RevokeKeywordEffect {
	e.dur = d
	return e
}

// While applies the revocation only while the source satisfies cond.
func (e *RevokeKeywordEffect) While(cond ActiveCondition) *RevokeKeywordEffect {
	e.active = cond
	return e
}

func (e *RevokeKeywordEffect) Text() string {
	if len(e.keywords) != 1 {
		if isSourceSelector(e.selector) {
			return "~ loses keywords until end of turn"
		}
		return "target creature loses keywords until end of turn"
	}
	if isSourceSelector(e.selector) {
		return fmt.Sprintf("~ loses %s until end of turn", e.keywords[0])
	}
	return fmt.Sprintf("target creature loses %s until end of turn", e.keywords[0])
}

func (e *RevokeKeywordEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

func (e *RevokeKeywordEffect) Apply(ctx *EffectContext) error {
	if !activeConditionStarted(ctx, e.active) {
		return nil
	}
	perms := resolvePermanents(ctx, e.selector)
	for _, perm := range perms {
		keywords := slices.Clone(e.keywords)
		eff := TargetEffectWhen(LayerAbility, e.dur, perm.ID(), func(g *Game, target *Permanent) error {
			for _, kw := range keywords {
				g.RevokeAttr(target.ID(), kw)
			}
			return nil
		}, e.active)
		ctx.Game.AddContinuousEffect(eff, ctx.SourceID)
	}
	if len(perms) > 0 {
		ctx.Game.ApplyContinuousEffects()
	}
	return nil
}

func activeConditionStarted(ctx *EffectContext, condition ActiveCondition) bool {
	return condition == nil || condition(ctx.Game, ctx.SourceID)
}

// --- GrantAbility DSL ---

// grantAbilityEffect is a composable Effect that temporarily grants a
// non-keyword ability (typically a triggered ability like Rampage N) to the
// selected permanent. Mirrors grantKeywordEffect for parameterized abilities
// that don't fit into the Keyword/Attr enum.
type grantAbilityEffect struct {
	ability  Ability
	selector ObjectSelector
	dur      Duration
	unless   TriggerConditionData
}

// GrantAbility creates a temporary ability grant effect. Defaults to targeting
// targets[0] until end of turn. Use .Targeting() / .Until() / .Unless().
func GrantAbility(a Ability) *grantAbilityEffect {
	return &grantAbilityEffect{
		ability:  a,
		selector: ToTarget(),
		dur:      EndOfTurn,
	}
}

func (e *grantAbilityEffect) Targeting(sel ObjectSelector) *grantAbilityEffect {
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

func (e *grantAbilityEffect) Apply(ctx *EffectContext) error {
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

			p.RuntimeAbilities = append(p.RuntimeAbilities, WrapGrantedAbility(ability))
			return nil
		})
		ctx.Game.AddContinuousEffect(eff, ctx.SourceID)
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
	selector ObjectSelector
	dur      Duration
}

// GrantType creates an effect that grants an additional card type to a permanent.
// Defaults to targeting targets[0] with Indefinite duration (lasts while target
// remains on battlefield). Use .Targeting() and .Until() to override.
func GrantType(ct CardType) *grantTypeEffect {
	return &grantTypeEffect{
		ct:       ct,
		selector: ToTarget(),
		dur:      Indefinite,
	}
}

func (e *grantTypeEffect) Targeting(sel ObjectSelector) *grantTypeEffect {
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

func (e *grantTypeEffect) Apply(ctx *EffectContext) error {
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
		ctx.Game.AddContinuousEffect(eff, ctx.SourceID)
	}
	if len(perms) > 0 {
		ctx.Game.ApplyContinuousEffects()
	}
	return nil
}
