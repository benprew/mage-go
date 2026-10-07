package mage

import (
	"fmt"
	"slices"
	"strings"
)

// DamageSourceChooser permits a player to distinguish exact source objects.
// The returned index selects one member of sources.
type DamageSourceChooser interface {
	ChooseDamageSource(sources []DamageSource, g *Game) int
}

// ChooseDamageSourceFrom saves one non-targeted choice from explicit references.
// The caller must supply references that the rules permit as damage sources.
// This operation does not enumerate all sources permitted by CR 609.7a.
// Missing sources or an absent player clear the result without a choice.
func ChooseDamageSourceFrom(player PlayerRef, sources []DamageSourceRef, binding *DamageSourceBinding) ResolvingEffect {
	return ChooseDamageSourceMatchingFrom(player, sources, DamageSourceFilter{}, binding)
}

// ChooseDamageSourceMatchingFrom saves a choice from matching explicit references.
// The caller must supply the complete legal source list for the operation.
func ChooseDamageSourceMatchingFrom(player PlayerRef, sources []DamageSourceRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, sources: slices.Clone(sources), filter: filter, binding: binding}
}

type damageSourceChoice struct {
	filter              DamageSourceFilter
	player              PlayerRef
	liveObjects         bool
	stackAbilities      bool
	stackTargets        bool
	stackSpellTargets   bool
	stackBindings       bool
	stackDamageBindings bool
	stackSpellBindings  bool
	sources             []DamageSourceRef
	binding             *DamageSourceBinding
}

func (*damageSourceChoice) Text() string                 { return "choose a damage source" }
func (*damageSourceChoice) Properties() EffectProperties { return EffectProperties{} }
func (e *damageSourceChoice) inspectDefinition() definitionNode {
	n := definitionNode{Role: "damage-source-choice", Text: e.Text(), bindingWrite: e.binding,
		children: []definitionEdge{{"player", e.player}, {"binding", e.binding}, {"filter", e.filter}}}
	if len(e.sources) == 0 && !e.liveObjects && !e.stackAbilities && !e.stackTargets && !e.stackSpellTargets && !e.stackBindings && !e.stackDamageBindings && !e.stackSpellBindings {
		n.problem = "supply at least one source reference"
	}
	for i, source := range e.sources {
		n.children = append(n.children, definitionEdge{fmt.Sprintf("sources[%d]", i), damageSourceDefinition{source}})
	}
	return n
}
func (e *damageSourceChoice) Apply(ctx *EffectContext) error {
	playerID, ok := e.player.resolvePlayer(ctx)
	if !ok {
		delete(ctx.Game.resolution.damageSourceBindings, e.binding)
		return nil
	}
	player := ctx.Game.GetPlayer(playerID)
	if player == nil {
		delete(ctx.Game.resolution.damageSourceBindings, e.binding)
		return nil
	}
	candidates := make([]DamageSource, 0, len(e.sources))
	if e.liveObjects {
		candidates = ctx.Game.LiveDamageSources()
	}
	if e.stackAbilities {
		candidates = append(candidates, ctx.Game.StackAbilityPermanentSources()...)
	}
	if e.stackTargets {
		candidates = append(candidates, ctx.Game.StackTargetPermanentSources()...)
	}
	if e.stackSpellTargets {
		candidates = append(candidates, ctx.Game.StackTargetSpellSources()...)
	}
	if e.stackBindings {
		candidates = append(candidates, ctx.Game.StackBoundPermanentSources()...)
	}
	if e.stackDamageBindings {
		candidates = append(candidates, ctx.Game.StackBoundDamageSources()...)
	}
	if e.stackSpellBindings {
		candidates = append(candidates, ctx.Game.StackBoundSpellSources()...)
	}
	for _, ref := range e.sources {
		candidates = append(candidates, ref.resolveDamageSource(ctx))
	}
	sources := make([]DamageSource, 0, len(candidates))
	for _, source := range candidates {
		if !e.filter.Match(source) || slices.ContainsFunc(sources, source.SameObject) {
			continue
		}
		sources = append(sources, cloneDamageSource(source))
	}
	delete(ctx.Game.resolution.damageSourceBindings, e.binding)
	if len(sources) == 0 {
		return nil
	}
	index := 0
	if chooser, ok := player.(DamageSourceChooser); ok {
		// Choices must not expose the saved source's mutable card data.
		options := make([]DamageSource, len(sources))
		for i, source := range sources {
			options[i] = cloneDamageSource(source)
		}
		index = chooser.ChooseDamageSource(options, ctx.Game)
	} else if len(sources) > 1 {
		labels := make([]string, len(sources))
		for i, source := range sources {
			labels[i] = fmt.Sprintf("%d: %s", i+1, source.Name())
		}
		index = player.ChooseNumber(1, len(sources), "Choose "+e.binding.name+" ("+strings.Join(labels, ", ")+")") - 1
	}
	if index < 0 || index >= len(sources) {
		index = 0
	}
	if ctx.Game.resolution.damageSourceBindings == nil {
		ctx.Game.resolution.damageSourceBindings = make(map[*DamageSourceBinding]DamageSource)
	}
	ctx.Game.resolution.damageSourceBindings[e.binding] = sources[index]
	return nil
}
