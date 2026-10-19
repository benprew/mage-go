package mage

import (
	"maps"
	"slices"

	"github.com/google/uuid"
)

// IfSpellTargetsPlayer checks current targets without last known information.
func IfSpellTargetsPlayer(spell StackObjectRef, player PlayerRef, body ResolvingEffect) ResolvingEffect {
	return &spellTargetsPlayerEffect{spell: spell, player: player, body: body}
}

type spellTargetsPlayerEffect struct {
	spell  StackObjectRef
	player PlayerRef
	body   ResolvingEffect
}

func (*spellTargetsPlayerEffect) Text() string                 { return "if spell targets player" }
func (*spellTargetsPlayerEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *spellTargetsPlayerEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"spell", e.spell}, {"player", e.player}, {"body", e.body}}}
}
func (e *spellTargetsPlayerEffect) Apply(ctx *EffectContext) error {
	obj, ok := e.spell.resolveStackObject(ctx)
	if !ok || obj.leftStack || obj.IsAbility || obj.Card == nil {
		return nil
	}
	player, ok := e.player.resolvePlayer(ctx)
	if !ok || !slices.Contains(obj.Targets, player) {
		return nil
	}
	return e.body.Apply(ctx)
}

// ChangeSingleSpellTargetToPlayer changes one target to another legal player.
// The spell keeps its controller, mode, X and announced allocation.
func ChangeSingleSpellTargetToPlayer(spell StackObjectRef) ResolvingEffect {
	return &spellPlayerRetargetEffect{spell: spell}
}

type spellPlayerRetargetEffect struct{ spell StackObjectRef }

func (*spellPlayerRetargetEffect) Text() string                 { return "change spell target to a player" }
func (*spellPlayerRetargetEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *spellPlayerRetargetEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"spell", e.spell}}}
}
func (e *spellPlayerRetargetEffect) Apply(ctx *EffectContext) error {
	g := ctx.Game
	obj, ok := e.spell.resolveStackObject(ctx)
	if !ok || obj.leftStack || obj.IsAbility || obj.Card == nil || len(obj.Targets) != 1 {
		return nil
	}
	specs := obj.TargetSpecs
	if len(specs) != 1 {
		return nil
	}
	old := obj.Targets[0]
	query := g.targetEventQuery(obj.capturedPlayers, obj.capturedPermanents).targetBindingsQuery(specs, obj.Targets, obj.targetIncarnations, obj.XValue)
	possible := query.possibleTargetsWithX(specs[0], obj.Controller, g.resolutionTargetSource(obj), obj.XValue)
	possible = slices.DeleteFunc(possible, func(id uuid.UUID) bool { return id == old || g.GetPlayer(id) == nil })
	chooser := g.GetPlayer(ctx.Controller)
	if len(possible) == 0 || chooser == nil {
		return nil
	}
	chosen := chooser.ChooseTargets(possible, 1, 1, g)
	if len(chosen) != 1 || !slices.Contains(possible, chosen[0]) {
		return nil
	}
	next := chosen[0]
	obj.Targets = []uuid.UUID{next}
	if len(obj.ModalTargets) > 0 && obj.ModeChoice >= 0 && obj.ModeChoice < len(obj.ModalTargets) {
		obj.ModalTargets = slices.Clone(obj.ModalTargets)
		obj.ModalTargets[obj.ModeChoice] = slices.Clone(obj.Targets)
	}
	obj.TargetZones = maps.Clone(obj.TargetZones)
	delete(obj.TargetZones, old)
	obj.DamageDistribution = moveSpellAllocation(obj.DamageDistribution, old, next)
	obj.CounterDistribution = moveSpellAllocation(obj.CounterDistribution, old, next)
	g.fireBecomesTargetEvents(obj, false)
	return nil
}
func moveSpellAllocation(values map[uuid.UUID]int, old, next uuid.UUID) map[uuid.UUID]int {
	if len(values) == 0 {
		return values
	}
	values = maps.Clone(values)
	amount, ok := values[old]
	delete(values, old)
	if ok {
		values[next] = amount
	}
	return values
}
