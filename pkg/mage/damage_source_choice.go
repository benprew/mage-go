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
	filter                        DamageSourceFilter
	player                        PlayerRef
	liveObjects                   bool
	commandObjects                bool
	stackAbilities                bool
	stackTargets                  bool
	stackSpellTargets             bool
	stackBindings                 bool
	stackExileBindings            bool
	stackGraveyardReferences      bool
	stackHandReferences           bool
	inspectHandSources            bool
	stackDamageBindings           bool
	stackSpellBindings            bool
	waitingDelayedCards           bool
	waitingDelayedGraveyard       bool
	waitingDelayedBindings        bool
	waitingDelayedSpells          bool
	waitingDelayedDamage          bool
	waitingCommands               bool
	waitingPrecastExile           bool
	waitingPrecastHand            bool
	waitingModifiers              bool
	waitingEmptyLibrary           bool
	waitingPersistentReplacements bool
	waitingUntapSkip              bool
	waitingDiscard                bool
	waitingExile                  bool
	waitingPrevention             bool
	waitingPreventionSpells       bool
	waitingRedirection            bool
	waitingRegeneration           bool
	waitingDrawStepLife           bool
	waitingCounterPlacement       bool
	sources                       []DamageSourceRef
	binding                       *DamageSourceBinding
}

func (*damageSourceChoice) Text() string                 { return "choose a damage source" }
func (*damageSourceChoice) Properties() EffectProperties { return EffectProperties{} }
func (e *damageSourceChoice) inspectDefinition() definitionNode {
	n := definitionNode{Role: "damage-source-choice", Text: e.Text(), bindingWrite: e.binding,
		children: []definitionEdge{{"player", e.player}, {"binding", e.binding}, {"filter", e.filter}}}
	if len(e.sources) == 0 && !e.liveObjects && !e.waitingCommands && !e.waitingPrecastExile && !e.waitingPrecastHand && !e.waitingModifiers && !e.waitingEmptyLibrary && !e.waitingPersistentReplacements && !e.waitingUntapSkip && !e.waitingDiscard && !e.waitingExile && !e.commandObjects && !e.stackAbilities && !e.stackTargets && !e.stackSpellTargets && !e.stackBindings && !e.stackExileBindings && !e.stackGraveyardReferences && !e.stackHandReferences && !e.stackDamageBindings && !e.stackSpellBindings && !e.waitingDelayedGraveyard && !e.waitingDelayedCards && !e.waitingDelayedBindings && !e.waitingDelayedSpells && !e.waitingDelayedDamage && !e.waitingPrevention && !e.waitingPreventionSpells && !e.waitingRedirection && !e.waitingRegeneration && !e.waitingDrawStepLife && !e.waitingCounterPlacement {
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
	if e.commandObjects {
		candidates = append(candidates, ctx.Game.CommandDamageSources()...)
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
	if e.stackHandReferences {
		if e.inspectHandSources {
			candidates = append(candidates, ctx.Game.StackHandSourcesFor(playerID)...)
		} else {
			candidates = append(candidates, ctx.Game.StackHandSources()...)
		}
	}
	if e.stackGraveyardReferences {
		candidates = append(candidates, ctx.Game.StackGraveyardSources()...)
	}
	if e.stackExileBindings {
		candidates = append(candidates, ctx.Game.StackBoundExileSources()...)
	}
	if e.stackDamageBindings {
		candidates = append(candidates, ctx.Game.StackBoundDamageSources()...)
	}
	if e.stackSpellBindings {
		candidates = append(candidates, ctx.Game.StackBoundSpellSources()...)
	}
	if e.waitingCounterPlacement {
		candidates = append(candidates, ctx.Game.WaitingCounterPlacementPermanentSources()...)
	}
	if e.waitingDrawStepLife {
		candidates = append(candidates, ctx.Game.WaitingDrawStepLifePermanentSources()...)
	}
	if e.waitingRegeneration {
		candidates = append(candidates, ctx.Game.WaitingRegenerationPermanentSources()...)
	}
	if e.waitingRedirection {
		candidates = append(candidates, ctx.Game.WaitingRedirectionPermanentSources()...)
		candidates = append(candidates, ctx.Game.WaitingRedirectionSpellSources()...)
	}
	if e.waitingExile {
		candidates = append(candidates, ctx.Game.WaitingExileSources()...)
	}
	if e.waitingDiscard {
		candidates = append(candidates, ctx.Game.WaitingDiscardPermanentSources()...)
	}
	if e.waitingUntapSkip {
		candidates = append(candidates, ctx.Game.WaitingUntapSkipPermanentSources()...)
	}
	if e.waitingPersistentReplacements {
		candidates = append(candidates, ctx.Game.WaitingPersistentReplacementPermanentSources()...)
	}
	if e.waitingEmptyLibrary {
		candidates = append(candidates, ctx.Game.WaitingEmptyLibraryPermanentSources()...)
	}
	if e.waitingModifiers {
		candidates = append(candidates, ctx.Game.WaitingModifierPermanentSources()...)
	}
	if e.waitingPrecastHand {
		candidates = append(candidates, ctx.Game.WaitingPreventionHandSourcesFor(playerID)...)
		candidates = append(candidates, ctx.Game.WaitingRedirectionHandSourcesFor(playerID)...)
	}
	if e.waitingPrecastExile {
		candidates = append(candidates, ctx.Game.WaitingPreventionExileSources()...)
		candidates = append(candidates, ctx.Game.WaitingRedirectionExileSources()...)
	}
	if e.waitingCommands {
		candidates = append(candidates, ctx.Game.WaitingPreventionCommandSources()...)
		candidates = append(candidates, ctx.Game.WaitingRedirectionCommandSources()...)
	}
	if e.waitingPreventionSpells {
		candidates = append(candidates, ctx.Game.WaitingPreventionSpellSources()...)
	}
	if e.waitingPrevention {
		candidates = append(candidates, ctx.Game.WaitingPreventionPermanentSources()...)
	}
	if e.waitingDelayedGraveyard {
		candidates = append(candidates, ctx.Game.WaitingDelayedGraveyardSources()...)
	}
	if e.waitingDelayedCards {
		candidates = append(candidates, ctx.Game.WaitingDelayedHandSourcesFor(playerID)...)
		candidates = append(candidates, ctx.Game.WaitingDelayedExileSources()...)
	}
	if e.waitingDelayedDamage {
		candidates = append(candidates, ctx.Game.WaitingDelayedDamageSources()...)
	}
	if e.waitingDelayedSpells {
		candidates = append(candidates, ctx.Game.WaitingDelayedSpellSources()...)
	}
	if e.waitingDelayedBindings {
		candidates = append(candidates, ctx.Game.WaitingDelayedPermanentSources()...)
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
