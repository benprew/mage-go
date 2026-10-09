package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// PreventNextDamageFromExactSource prevents one source event to a player this turn.
// It keeps the exact object and checks its properties before damage.
// A spell and the permanent it becomes require a separate continuity contract.
func PreventNextDamageFromExactSource(source DamageSourceRef, recipient PlayerRef, filter DamageSourceFilter) ResolvingEffect {
	return &exactSourcePreventionInstallation{source: source, recipient: recipient, filter: filter}
}

type exactSourcePreventionInstallation struct {
	amount                 ProposedDamageValue
	gainLifeRecipient      PlayerRef
	operationProblem       string
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
	n := definitionNode{Role: "install-exact-source-prevention", Text: e.Text(), problem: e.operationProblem, children: []definitionEdge{{"source", damageSourceDefinition{e.source}}, {"recipient", e.recipient}, {"filter", e.filter}}}
	if e.amount != nil {
		n.children = append(n.children, definitionEdge{"amount", e.amount})
	}
	if e.gainLifeRecipient != nil {
		n.children = append(n.children, definitionEdge{"life gain recipient", e.gainLifeRecipient})
	}
	return n
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
	gainLifePlayer := uuid.Nil
	if e.gainLifeRecipient != nil {
		if recipient, ok := e.gainLifeRecipient.resolvePlayer(ctx); ok && ctx.Game.GetPlayer(recipient) != nil {
			gainLifePlayer = recipient
		}
	}
	ctx.Game.effects.AddReplacement(&sourcePreventionReplacement{
		replacementBase: replacementBase{duration: core.EndOfTurn}, playerID: player, dmgSource: source.ID(),
		exactSource: cloneDamageSource(information), sourceFilter: e.filter,
		continuePermanentSpell: e.continuePermanentSpell, amount: e.amount, gainLifePlayerID: gainLifePlayer,
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
