package mage

import (
	"fmt"
	"slices"

	"github.com/google/uuid"
)

// cloneInstanceAbility copies ability context. Compiled effects and costs are immutable.
func cloneInstanceAbility(ability Ability) Ability {
	return newGameCloneContext().ability(ability)
}

// freshPermanentAbilities clears execution state from another zone.
// Ability IDs stay equal to the card's IDs for printed-ability restoration.
func freshPermanentAbilities(abilities []Ability) []Ability {
	ctx := newGameCloneContext()
	cloned := ctx.abilityList(abilities)
	for _, ability := range ctx.abilities {
		ability.SetSource(uuid.Nil)
		ability.SetController(uuid.Nil)
		if action, ok := ability.(interface{ ResetActivation() }); ok {
			action.ResetActivation()
		}
	}
	for _, target := range ctx.targets {
		target.Reset()
	}
	return cloned
}

func (ctx *gameCloneContext) ability(ability Ability) Ability {
	if ability == nil {
		return nil
	}
	if cloned, ok := ctx.abilities[ability]; ok {
		return cloned
	}
	cloned := ctx.copyAbility(ability)
	if ctx.abilities == nil {
		ctx.abilities = make(map[Ability]Ability)
	}
	ctx.abilities[ability] = cloned
	return cloned
}

func (ctx *gameCloneContext) copyAbility(ability Ability) Ability {
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
			cloned.Effects[i] = ctx.continuous(effect)
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
		ctx.action(&cloned)
		return &cloned
	case *GraveyardActivatedAbility:
		cloned := *a
		cloned.SimpleActivatedAbility = ctx.ability(a.SimpleActivatedAbility).(*ActionDefinition)
		return &cloned
	case *EquipAbility:
		cloned := *a
		ctx.action(&cloned.ActionDefinition)
		return &cloned
	case *GenericTriggered:
		cloned := *a
		cloned.effects = slices.Clone(a.effects)
		cloned.targets = ctx.targetList(a.targets)
		cloned.zones = slices.Clone(a.zones)
		cloned.modes = ctx.modes(a.modes)
		return &cloned
	case *ModalSpellAbility:
		cloned := *a
		ctx.action(&cloned.SpellAbility)
		cloned.modes = ctx.modes(a.modes)
		return &cloned
	case *grantedByEffect:
		cloned := *a
		cloned.Ability = ctx.ability(a.Ability)
		return &cloned
	default:
		panic(fmt.Sprintf("mage: ability %T needs an instance clone operation", ability))
	}
}

func (ctx *gameCloneContext) action(action *ActionDefinition) {
	action.effects = slices.Clone(action.effects)
	action.costs = slices.Clone(action.costs)
	action.targets = ctx.targetList(action.targets)
	action.activationConds = slices.Clone(action.activationConds)
	action.aiHints = slices.Clone(action.aiHints)
	for i := range action.aiHints {
		action.aiHints[i].Roles = slices.Clone(action.aiHints[i].Roles)
	}
	action.modes = ctx.modes(action.modes)
}

func (ctx *gameCloneContext) modes(modes []Mode) []Mode {
	result := slices.Clone(modes)
	for i := range result {
		result[i].Targets = ctx.targetList(modes[i].Targets)
		result[i].Effects = slices.Clone(modes[i].Effects)
	}
	return result
}

func cloneInstanceTargets(targets []Target) []Target {
	return newGameCloneContext().targetList(targets)
}

func (ctx *gameCloneContext) targetList(targets []Target) []Target {
	if targets == nil {
		return nil
	}
	result := make([]Target, len(targets))
	for i, target := range targets {
		result[i] = ctx.target(target)
	}
	return result
}

func (ctx *gameCloneContext) target(target Target) Target {
	if target == nil {
		return nil
	}
	if cloned, ok := ctx.targets[target]; ok {
		return cloned
	}
	cloned := ctx.copyTarget(target)
	if ctx.targets == nil {
		ctx.targets = make(map[Target]Target)
	}
	ctx.targets[target] = cloned
	return cloned
}

func (ctx *gameCloneContext) copyTarget(target Target) Target {
	switch t := target.(type) {
	case interface{ CloneTarget() Target }:
		return t.CloneTarget()
	case *countBoundTarget:
		cloned := *t
		cloned.inner = ctx.target(t.inner)
		return &cloned
	case *randomTarget:
		cloned := *t
		cloned.inner = ctx.target(t.inner)
		return &cloned
	case *randomActivePlayerExchangePairTarget:
		cloned := *t
		cloned.chosen = slices.Clone(t.chosen)
		return &cloned
	case *opponentChosenTarget:
		cloned := *t
		cloned.inner = ctx.target(t.inner)
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

func (ctx *gameCloneContext) continuous(effect ContinuousEffect) ContinuousEffect {
	if cloned, ok := ctx.effects[effect]; ok {
		return cloned
	}
	cloned := ctx.copyContinuous(effect)
	if ctx.effects == nil {
		ctx.effects = make(map[ContinuousEffect]ContinuousEffect)
	}
	ctx.effects[effect] = cloned
	return cloned
}

func (ctx *gameCloneContext) copyContinuous(effect ContinuousEffect) ContinuousEffect {
	switch e := effect.(type) {
	case *funcContinuousEffect:
		cloned := *e
		return &cloned
	case *attachedActivatedAbilityEffect:
		cloned := *e
		cloned.template = ctx.ability(e.template).(*SimpleActivatedAbility)
		cloned.abilities = make(map[uuid.UUID]*SimpleActivatedAbility, len(e.abilities))
		for id, ability := range e.abilities {
			cloned.abilities[id] = ctx.ability(ability).(*SimpleActivatedAbility)
		}
		return &cloned
	case *attachedEffect:
		cloned := *e
		return &cloned
	case *attachedControlEffect:
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
		return e.cloneEffect()
	case *objectColorEffect:
		cloned := *e
		cloned.colors = slices.Clone(e.colors)
		return &cloned
	case *printedAbilityContinuousEffect:
		return &printedAbilityContinuousEffect{ContinuousEffect: ctx.continuous(e.ContinuousEffect)}
	default:
		panic(fmt.Sprintf("mage: continuous effect %T needs an instance clone operation", effect))
	}
}

func (ctx *gameCloneContext) activeEffect(effect ContinuousEffect) ContinuousEffect {
	if cloned, ok := ctx.effects[effect]; ok {
		return cloned
	}
	cloned := effect
	switch e := effect.(type) {
	case *printedAbilityContinuousEffect:
		cloned = &printedAbilityContinuousEffect{ContinuousEffect: ctx.activeEffect(e.ContinuousEffect)}
	case *attachedActivatedAbilityEffect:
		cloned = ctx.continuous(e)
	case deepCloneableEffect:
		cloned = e.cloneEffect()
	}
	if ctx.effects == nil {
		ctx.effects = make(map[ContinuousEffect]ContinuousEffect)
	}
	ctx.effects[effect] = cloned
	return cloned
}
