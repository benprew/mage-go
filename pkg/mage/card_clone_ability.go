package mage

import (
	"fmt"
	"slices"
)

// cloneInstanceAbility copies ability context. Compiled effects and costs are immutable.
func cloneInstanceAbility(ability Ability) Ability {
	switch a := ability.(type) {
	case *AttributeAbility:
		cloned := *a
		return &cloned
	case *ProtectionAbility:
		cloned := *a
		cloned.FromColors = slices.Clone(a.FromColors)
		return &cloned
	case *StaticAbilityHolder:
		cloned := *a
		cloned.Effects = make([]ContinuousEffect, len(a.Effects))
		for i, effect := range a.Effects {
			cloned.Effects[i] = cloneInstanceContinuous(effect)
		}
		return &cloned
	case *SacrificeUnlessLandAbility:
		cloned := *a
		return &cloned
	case *SacrificeIfControlsAbility:
		cloned := *a
		return &cloned
	case *EntersWithXCountersAbility:
		cloned := *a
		return &cloned
	case *EntersWithNCountersAbility:
		cloned := *a
		return &cloned
	case *EntersWithComputedCountersAbility:
		cloned := *a
		return &cloned
	case *CopyCreatureOnETBAbility:
		cloned := *a
		return &cloned
	case *ETBWithTargetsAbility:
		cloned := *a
		return &cloned
	case *ETBEffectAbility:
		cloned := *a
		return &cloned
	case *frankensteinsMonsterAbility:
		cloned := *a
		return &cloned
	case *namelessRaceAbility:
		cloned := *a
		return &cloned
	case *ManaAbility:
		cloned := *a
		cloned.Productions = slices.Clone(a.Productions)
		cloned.postProduction = slices.Clone(a.postProduction)
		return &cloned
	case *ManaBonusAbility:
		cloned := *a
		return &cloned
	case *SelfCostReductionAbility:
		cloned := *a
		return &cloned
	case *ActionDefinition:
		cloned := *a
		cloneInstanceAction(&cloned)
		return &cloned
	case *EquipAbility:
		cloned := *a
		cloneInstanceAction(&cloned.ActionDefinition)
		return &cloned
	case *GenericTriggered:
		cloned := *a
		cloned.effects = slices.Clone(a.effects)
		cloned.targets = cloneInstanceTargets(a.targets)
		cloned.zones = slices.Clone(a.zones)
		cloned.modes = cloneInstanceModes(a.modes)
		return &cloned
	case *ModalSpellAbility:
		cloned := *a
		cloneInstanceAction(&cloned.SpellAbility)
		cloned.modes = cloneInstanceModes(a.modes)
		return &cloned
	case *grantedByEffect:
		cloned := *a
		cloned.Ability = cloneInstanceAbility(a.Ability)
		return &cloned
	default:
		panic(fmt.Sprintf("mage: ability %T needs an instance clone operation", ability))
	}
}

func cloneInstanceAction(action *ActionDefinition) {
	action.effects = slices.Clone(action.effects)
	action.costs = slices.Clone(action.costs)
	action.targets = cloneInstanceTargets(action.targets)
	action.activationConds = slices.Clone(action.activationConds)
	action.aiHints = slices.Clone(action.aiHints)
	for i := range action.aiHints {
		action.aiHints[i].Roles = slices.Clone(action.aiHints[i].Roles)
	}
	action.modes = cloneInstanceModes(action.modes)
}

func cloneInstanceModes(modes []Mode) []Mode {
	result := slices.Clone(modes)
	for i := range result {
		result[i].Targets = cloneInstanceTargets(modes[i].Targets)
		result[i].Effects = slices.Clone(modes[i].Effects)
	}
	return result
}

func cloneInstanceTargets(targets []Target) []Target {
	result := make([]Target, len(targets))
	for i, target := range targets {
		result[i] = cloneInstanceTarget(target)
	}
	return result
}

func cloneInstanceTarget(target Target) Target {
	switch t := target.(type) {
	case *countBoundTarget:
		cloned := *t
		cloned.inner = cloneInstanceTarget(t.inner)
		return &cloned
	case *randomTarget:
		cloned := *t
		cloned.inner = cloneInstanceTarget(t.inner)
		return &cloned
	case *randomActivePlayerExchangePairTarget:
		cloned := *t
		cloned.chosen = slices.Clone(t.chosen)
		return &cloned
	case *opponentChosenTarget:
		cloned := *t
		cloned.inner = cloneInstanceTarget(t.inner)
		return &cloned
	case *CreatureTarget:
		cloned := *t
		cloned.chosen = slices.Clone(t.chosen)
		cloned.Filters = slices.Clone(t.Filters)
		return &cloned
	case *PlayerTarget:
		cloned := *t
		cloned.chosen = slices.Clone(t.chosen)
		return &cloned
	case *ControllerTarget:
		cloned := *t
		cloned.chosen = slices.Clone(t.chosen)
		return &cloned
	case *DamageAnyTarget:
		cloned := *t
		cloned.chosen = slices.Clone(t.chosen)
		return &cloned
	case *PlayerOrPlaneswalkerTarget:
		cloned := *t
		cloned.chosen = slices.Clone(t.chosen)
		return &cloned
	case *ControlledCreatureTarget:
		cloned := *t
		cloned.chosen = slices.Clone(t.chosen)
		return &cloned
	case *PermanentTarget:
		cloned := *t
		cloned.chosen = slices.Clone(t.chosen)
		cloned.Filters = slices.Clone(t.Filters)
		return &cloned
	case *auraAttachedToCreatureYouControlTarget:
		cloned := *t
		cloned.chosen = slices.Clone(t.chosen)
		return &cloned
	case *artifactWithManaValueXTarget:
		cloned := *t
		cloned.chosen = slices.Clone(t.chosen)
		return &cloned
	case *SpellOrPermanentTarget:
		cloned := *t
		cloned.chosen = slices.Clone(t.chosen)
		return &cloned
	case *sourceChosenSubtypeCreatureTarget:
		cloned := *t
		cloned.chosen = slices.Clone(t.chosen)
		return &cloned
	case *DamageSourceTarget:
		cloned := *t
		cloned.chosen = slices.Clone(t.chosen)
		return &cloned
	case *SpellOnStackTarget:
		cloned := *t
		cloned.chosen = slices.Clone(t.chosen)
		cloned.Filters = slices.Clone(t.Filters)
		return &cloned
	case *GraveyardCardTarget:
		cloned := *t
		cloned.chosen = slices.Clone(t.chosen)
		cloned.Filters = slices.Clone(t.Filters)
		return &cloned
	case *AnyGraveyardCardTarget:
		cloned := *t
		cloned.chosen = slices.Clone(t.chosen)
		cloned.Filters = slices.Clone(t.Filters)
		return &cloned
	case *HandCardTarget:
		cloned := *t
		cloned.chosen = slices.Clone(t.chosen)
		cloned.Filters = slices.Clone(t.Filters)
		return &cloned
	case *ControlledPermanentTarget:
		cloned := *t
		cloned.chosen = slices.Clone(t.chosen)
		return &cloned
	case *OpponentTarget:
		cloned := *t
		cloned.chosen = slices.Clone(t.chosen)
		return &cloned
	case *PowerLESourceCreatureTarget:
		cloned := *t
		cloned.chosen = slices.Clone(t.chosen)
		return &cloned
	case *BlockingOrBlockedBySourceTarget:
		cloned := *t
		cloned.chosen = slices.Clone(t.chosen)
		return &cloned
	case *cardYouOwnInAnteTarget:
		cloned := *t
		cloned.chosen = slices.Clone(t.chosen)
		return &cloned
	default:
		panic(fmt.Sprintf("mage: target %T needs an instance clone operation", target))
	}
}

func cloneInstanceContinuous(effect ContinuousEffect) ContinuousEffect {
	switch e := effect.(type) {
	case *funcContinuousEffect:
		cloned := *e
		return &cloned
	case *attachedEffect:
		cloned := *e
		return &cloned
	case *targetEffect:
		return e.cloneEffect()
	case *controlContinuousEffect:
		return e.cloneEffect()
	case *preventDamageRuleContinuous:
		cloned := *e
		return &cloned
	case *preventNoncombatDamageToControllerContinuous:
		cloned := *e
		return &cloned
	case *doppelgangerCopyEffect:
		cloned := *e
		cloned.keywords = slices.Clone(e.keywords)
		return &cloned
	case *objectColorEffect:
		cloned := *e
		cloned.colors = slices.Clone(e.colors)
		return &cloned
	case *printedAbilityContinuousEffect:
		return &printedAbilityContinuousEffect{ContinuousEffect: cloneInstanceContinuous(e.ContinuousEffect)}
	default:
		panic(fmt.Sprintf("mage: continuous effect %T needs an instance clone operation", effect))
	}
}
