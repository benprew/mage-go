package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type staticDamageRedirectRule struct {
	player     PlayerRef
	subject    PermanentRef
	from       PermanentFilter
	sourceType core.CardType
	byType     bool
	condition  StateCondition
}

// RedirectDamageToPermanentIf redirects matching permanent-source damage from a player.
// The condition and references use the current source controller.
func RedirectDamageToPermanentIf(player PlayerRef, subject PermanentRef, from PermanentFilter, condition StateCondition) StaticRule {
	return staticDamageRedirectRule{player: player, subject: subject, from: from, condition: condition}
}

// RedirectDamageFromCardTypeToPermanentIf includes sources outside the battlefield.
func RedirectDamageFromCardTypeToPermanentIf(player PlayerRef, subject PermanentRef, kind core.CardType, condition StateCondition) StaticRule {
	return staticDamageRedirectRule{player: player, subject: subject, sourceType: kind, byType: true, condition: condition}
}
func (staticDamageRedirectRule) staticRule() {}
func (r staticDamageRedirectRule) inspectDefinition() definitionNode {
	n := definitionNode{Role: "static-rule", Text: "redirect player damage", battlefieldSource: true, children: []definitionEdge{{"player", r.player}, {"subject", r.subject}, {"condition", r.condition}}}
	if nilDefinition(r.player) || nilDefinition(r.subject) || nilDefinition(r.condition) {
		n.problem = "supply player, permanent and condition references"
	}
	if r.byType && (r.sourceType < core.TypeCreature || r.sourceType > core.TypePlaneswalker) {
		n.problem = "supply a valid damage source card type"
	}
	return n
}
func (r staticDamageRedirectRule) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{&staticDamageRedirectEffect{funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil }).(*funcContinuousEffect), rule: r}}
}

type staticDamageRedirectEffect struct {
	*funcContinuousEffect
	rule staticDamageRedirectRule
}

func (e *staticDamageRedirectEffect) inspectDefinition() definitionNode {
	return e.rule.inspectDefinition()
}
func (e *staticDamageRedirectEffect) cloneEffect() ContinuousEffect {
	c := *e
	inner := *e.funcContinuousEffect
	c.funcContinuousEffect = &inner
	return &c
}
func (e *staticDamageRedirectEffect) ApplyContinuous(ctx *EffectContext) error {
	source := ctx.Game.FindPermanent(e.SourceID())
	if source != nil {
		ctx.Game.effects.AddCycleReplacement(&staticDamageRedirectReplacement{replacementBase: replacementBase{sourceID: source.ID()}, sourceIncarnation: source.incarnationID, rule: e.rule})
	}
	return nil
}

type staticDamageRedirectReplacement struct {
	replacementBase
	sourceIncarnation uuid.UUID
	rule              staticDamageRedirectRule
}

func (r *staticDamageRedirectReplacement) context(g *Game) (*EffectContext, *Permanent) {
	source := g.FindPermanent(r.sourceID)
	if source == nil || source.incarnationID != r.sourceIncarnation {
		return nil, nil
	}
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: source.ControllerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	if !r.rule.condition.checkState(g, StateContext{SourceID: source.ID(), ControllerID: source.ControllerID(), SourceIncarnation: source.incarnationID}) {
		return nil, nil
	}
	subject, ok := r.rule.subject.resolvePermanent(ctx)
	if !ok || (!subject.HasType(core.TypeCreature) && !subject.HasType(core.TypePlaneswalker)) {
		return nil, nil
	}
	return ctx, subject
}
func (r *staticDamageRedirectReplacement) Matches(action Action, reader GameReader) bool {
	damage, ok := action.(*DamageToPlayerAction)
	if !ok {
		return false
	}
	g := reader.(*Game)
	ctx, subject := r.context(g)
	if subject == nil {
		return false
	}
	player, ok := r.rule.player.resolvePlayer(ctx)
	if !ok || player != damage.PlayerID() {
		return false
	}
	if r.rule.byType {
		return damageSourceHasCardType(action, g, r.rule.sourceType)
	}
	source := g.damageFilterSource(action.ActionSource(), damage.DamageSource())
	return source != nil && (r.rule.from.IsZero() || r.rule.from.Match(source, g))
}
func (r *staticDamageRedirectReplacement) Replace(action Action, g *Game) Action {
	_, subject := r.context(g)
	if subject == nil {
		return action
	}
	damage := action.(*DamageToPlayerAction)
	return NewDamageToCreatureAction(damage.ActionSource(), subject.ID(), damage.Amount(), damage.IsCombatDamage()).withDamageSource(damage.DamageSource())
}
func (*staticDamageRedirectReplacement) IsActive(GameReader) bool   { return true }
func (r *staticDamageRedirectReplacement) Clone() ReplacementEffect { c := *r; return &c }

// DamageSourceIsUnblocked reads the status set after blockers are declared.
// Absent sources use the status saved before departure or phasing.
func DamageSourceIsUnblocked() PermanentFilter {
	return NewPermanentFilter("unblocked damage source", func(p *Permanent, g *Game) bool {
		if current := g.FindPermanentIncludingPhased(p.ID()); current != nil && current.incarnationID == p.incarnationID {
			if !current.PhasedOut {
				return g.combat.IsUnblockedIncarnation(p.ID(), p.incarnationID)
			}
			information := g.zones.phaseInformation[p.incarnationID]
			return information != nil && information.Unblocked
		}
		information := g.zones.LKIByIncarnation(p.incarnationID)
		return information != nil && information.ID == p.ID() && information.Unblocked
	})
}
