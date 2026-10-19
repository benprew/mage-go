package mage

import "github.com/google/uuid"

type castSpellChoice struct {
	player, caster PlayerRef
	filter         DamageSourceFilter
	binding        *DamageSourceBinding
}

// ChooseSpellCastThisTurn saves one exact cast from the current turn.
// Copies that were not cast cannot supply a choice. Departure does not remove a cast.
func ChooseSpellCastThisTurn(player, caster PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &castSpellChoice{player, caster, filter, binding}
}
func (*castSpellChoice) Text() string                 { return "choose a spell cast this turn" }
func (*castSpellChoice) Properties() EffectProperties { return EffectProperties{} }
func (e *castSpellChoice) inspectDefinition() definitionNode {
	return definitionNode{Role: "cast-spell-choice", bindingWrite: e.binding, children: []definitionEdge{{"player", e.player}, {"caster", e.caster}, {"filter", e.filter}, {"binding", e.binding}}}
}
func (e *castSpellChoice) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || playerRefReadsSourcePermanent(e.caster)
}
func (e *castSpellChoice) Apply(ctx *EffectContext) error {
	delete(ctx.Game.resolution.damageSourceBindings, e.binding)
	player, ok := e.player.resolvePlayer(ctx)
	if !ok || ctx.Game.GetPlayer(player) == nil {
		return nil
	}
	caster, ok := e.caster.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	var candidates []DamageSource
	for _, source := range ctx.Game.trackers.Turn.castSpells {
		if source.Controller() == caster {
			candidates = append(candidates, source)
		}
	}
	choice := damageSourceChoice{filter: e.filter, binding: e.binding}
	return choice.chooseCandidates(ctx, ctx.Game.GetPlayer(player), candidates)
}

type sourceDamageHistoryValue struct{ source DamageSourceRef }

// DamageDealtThisTurnBySource reads actual damage from one exact spell identity.
// Prevention and redirection use the shared damage pipeline before this total changes.
func DamageDealtThisTurnBySource(source DamageSourceRef) ResolutionValue {
	return sourceDamageHistoryValue{source}
}
func (sourceDamageHistoryValue) Text() string { return "damage dealt by a spell this turn" }
func (v sourceDamageHistoryValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value", Text: v.Text(), children: []definitionEdge{{"source", damageSourceDefinition(v)}}}
}
func (v sourceDamageHistoryValue) resolveValue(ctx *EffectContext) int {
	source := v.source.resolveDamageSource(ctx)
	if !source.hasExactIdentity() || source.StackID() == uuid.Nil {
		return 0
	}
	return ctx.Game.damage.spellDamageAmounts[source.StackID()]
}
func (ds *DamageSystem) recordSpellDamage(source DamageSource, amount int) {
	if !source.hasExactIdentity() || source.StackID() == uuid.Nil || amount <= 0 {
		return
	}
	if ds.spellDamageAmounts == nil {
		ds.spellDamageAmounts = make(map[uuid.UUID]int)
	}
	ds.spellDamageAmounts[source.StackID()] += amount
}
