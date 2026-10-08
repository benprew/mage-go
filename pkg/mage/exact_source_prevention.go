package mage

import (
	"github.com/benprew/mage-go/pkg/mage/core"
)

// PreventNextDamageFromExactSource prevents one source event to a player this turn.
// It keeps the exact object and checks its properties before damage.
// A spell and the permanent it becomes require a separate continuity contract.
func PreventNextDamageFromExactSource(source DamageSourceRef, recipient PlayerRef, filter DamageSourceFilter) ResolvingEffect {
	return &exactSourcePreventionInstallation{source: source, recipient: recipient, filter: filter}
}

type exactSourcePreventionInstallation struct {
	source                 DamageSourceRef
	recipient              PlayerRef
	filter                 DamageSourceFilter
	continuePermanentSpell bool
}

func (*exactSourcePreventionInstallation) Text() string {
	return "prevent next damage from an exact source this turn"
}
func (*exactSourcePreventionInstallation) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *exactSourcePreventionInstallation) inspectDefinition() definitionNode {
	return definitionNode{Role: "install-exact-source-prevention", Text: e.Text(), children: []definitionEdge{{"source", damageSourceDefinition{e.source}}, {"recipient", e.recipient}, {"filter", e.filter}}}
}
func (e *exactSourcePreventionInstallation) Apply(ctx *EffectContext) error {
	source := e.source.resolveDamageSource(ctx)
	if !source.hasExactIdentity() {
		return nil
	}
	player, ok := e.recipient.resolvePlayer(ctx)
	if !ok || ctx.Game.GetPlayer(player) == nil {
		return nil
	}
	information := ctx.Game.damageSourceRuleInformation(source)
	if !information.SameObject(source) {
		information = source
	}
	if !e.filter.Match(information) {
		return nil
	}
	ctx.Game.effects.AddReplacement(&sourcePreventionReplacement{
		replacementBase: replacementBase{duration: core.EndOfTurn}, playerID: player, dmgSource: source.ID(),
		exactSource: cloneDamageSource(information), sourceFilter: e.filter,
		continuePermanentSpell: e.continuePermanentSpell,
	})
	return nil
}

func (r *sourcePreventionReplacement) matchesExactSource(source DamageSource, g *Game) bool {
	if !r.exactSource.SameObject(source) && !r.continuedSource.SameObject(source) {
		return false
	}
	information := g.damageSourceRuleInformation(source)
	if !information.SameObject(source) {
		information = source
	}
	return r.sourceFilter.Match(information)
}
