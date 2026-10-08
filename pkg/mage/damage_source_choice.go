package mage

import (
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
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
	allSources                    bool
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
	if len(e.sources) == 0 && !e.allSources && !e.liveObjects && !e.waitingCommands && !e.waitingPrecastExile && !e.waitingPrecastHand && !e.waitingModifiers && !e.waitingEmptyLibrary && !e.waitingPersistentReplacements && !e.waitingUntapSkip && !e.waitingDiscard && !e.waitingExile && !e.commandObjects && !e.stackAbilities && !e.stackTargets && !e.stackSpellTargets && !e.stackBindings && !e.stackExileBindings && !e.stackGraveyardReferences && !e.stackHandReferences && !e.stackDamageBindings && !e.stackSpellBindings && !e.waitingDelayedGraveyard && !e.waitingDelayedCards && !e.waitingDelayedBindings && !e.waitingDelayedSpells && !e.waitingDelayedDamage && !e.waitingPrevention && !e.waitingPreventionSpells && !e.waitingRedirection && !e.waitingRegeneration && !e.waitingDrawStepLife && !e.waitingCounterPlacement {
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
	var candidates []DamageSource
	if e.allSources {
		candidates = ctx.Game.DamageSourcesFor(playerID)
	} else {
		candidates = e.candidates(ctx.Game, playerID)
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

func (e *damageSourceChoice) candidates(g *Game, playerID uuid.UUID) []DamageSource {
	var candidates []DamageSource
	if e.allSources || e.liveObjects {
		candidates = g.LiveDamageSources()
	}
	if e.allSources || e.commandObjects {
		candidates = append(candidates, g.CommandDamageSources()...)
	}
	if e.allSources || e.stackAbilities {
		candidates = append(candidates, g.StackAbilityPermanentSources()...)
	}
	if e.allSources || e.stackTargets {
		candidates = append(candidates, g.StackTargetPermanentSources()...)
	}
	if e.allSources || e.stackSpellTargets {
		candidates = append(candidates, g.StackTargetSpellSources()...)
	}
	if e.allSources || e.stackBindings {
		candidates = append(candidates, g.StackBoundPermanentSources()...)
	}
	if e.allSources || e.stackHandReferences {
		if e.allSources || e.inspectHandSources {
			candidates = append(candidates, g.StackHandSourcesFor(playerID)...)
		} else {
			candidates = append(candidates, g.StackHandSources()...)
		}
	}
	if e.allSources || e.stackGraveyardReferences {
		candidates = append(candidates, g.StackGraveyardSources()...)
	}
	if e.allSources || e.stackExileBindings {
		candidates = append(candidates, g.StackBoundExileSources()...)
	}
	if e.allSources || e.stackDamageBindings {
		candidates = append(candidates, g.StackBoundDamageSources()...)
	}
	if e.allSources || e.stackSpellBindings {
		candidates = append(candidates, g.StackBoundSpellSources()...)
	}
	if e.allSources || e.waitingCounterPlacement {
		candidates = append(candidates, g.WaitingCounterPlacementPermanentSources()...)
	}
	if e.allSources || e.waitingDrawStepLife {
		candidates = append(candidates, g.WaitingDrawStepLifePermanentSources()...)
	}
	if e.allSources || e.waitingRegeneration {
		candidates = append(candidates, g.WaitingRegenerationPermanentSources()...)
	}
	if e.allSources || e.waitingRedirection {
		candidates = append(candidates, g.WaitingRedirectionPermanentSources()...)
		candidates = append(candidates, g.WaitingRedirectionSpellSources()...)
	}
	if e.allSources || e.waitingExile {
		candidates = append(candidates, g.WaitingExileSources()...)
	}
	if e.allSources || e.waitingDiscard {
		candidates = append(candidates, g.WaitingDiscardPermanentSources()...)
	}
	if e.allSources || e.waitingUntapSkip {
		candidates = append(candidates, g.WaitingUntapSkipPermanentSources()...)
	}
	if e.allSources || e.waitingPersistentReplacements {
		candidates = append(candidates, g.WaitingPersistentReplacementPermanentSources()...)
	}
	if e.allSources || e.waitingEmptyLibrary {
		candidates = append(candidates, g.WaitingEmptyLibraryPermanentSources()...)
	}
	if e.allSources || e.waitingModifiers {
		candidates = append(candidates, g.WaitingModifierPermanentSources()...)
	}
	if e.allSources || e.waitingPrecastHand {
		candidates = append(candidates, g.WaitingPreventionHandSourcesFor(playerID)...)
		candidates = append(candidates, g.WaitingRedirectionHandSourcesFor(playerID)...)
	}
	if e.allSources || e.waitingPrecastExile {
		candidates = append(candidates, g.WaitingPreventionExileSources()...)
		candidates = append(candidates, g.WaitingRedirectionExileSources()...)
	}
	if e.allSources || e.waitingCommands {
		candidates = append(candidates, g.WaitingPreventionCommandSources()...)
		candidates = append(candidates, g.WaitingRedirectionCommandSources()...)
	}
	if e.allSources || e.waitingPreventionSpells {
		candidates = append(candidates, g.WaitingPreventionSpellSources()...)
	}
	if e.allSources || e.waitingPrevention {
		candidates = append(candidates, g.WaitingPreventionPermanentSources()...)
	}
	if e.allSources || e.waitingDelayedGraveyard {
		candidates = append(candidates, g.WaitingDelayedGraveyardSources()...)
	}
	if e.allSources || e.waitingDelayedCards {
		candidates = append(candidates, g.WaitingDelayedHandSourcesFor(playerID)...)
		candidates = append(candidates, g.WaitingDelayedExileSources()...)
	}
	if e.allSources || e.waitingDelayedDamage {
		candidates = append(candidates, g.WaitingDelayedDamageSources()...)
	}
	if e.allSources || e.waitingDelayedSpells {
		candidates = append(candidates, g.WaitingDelayedSpellSources()...)
	}
	if e.allSources || e.waitingDelayedBindings {
		candidates = append(candidates, g.WaitingDelayedPermanentSources()...)
	}
	return candidates
}
