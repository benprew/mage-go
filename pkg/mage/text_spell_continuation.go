package mage

import (
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithSpellTextContinuation permits fresh rules for an ordinary pending spell.
// Casting results need a separate binding transfer contract.
func WithSpellTextContinuation() ActionOption {
	return func(action *ActionDefinition) { action.spellTextContinuation = true }
}

func (a *ActionDefinition) supportsSpellTextContinuation() bool {
	return a.kind == ActionSpell && !a.staticGrant && len(a.costs) == 0 && len(a.modes) == 0 &&
		len(a.dslCaptures) == 0 && a.activationHistory == nil && a.activationManaX == nil
}

func spellTextAction(card Card) (*ActionDefinition, error) {
	base, ok := card.(*BaseCard)
	if !ok || base == nil || (!base.HasType(core.TypeInstant) && !base.HasType(core.TypeSorcery)) ||
		len(base.abilities) != 1 || len(base.additionalCosts) != 0 || len(base.alternateCosts) != 0 || len(base.castTargets) != 0 {
		return nil, fmt.Errorf("spell text continuation requires one ordinary instant or sorcery declaration")
	}
	action, ok := base.abilities[0].(*ActionDefinition)
	if !ok || !action.spellTextContinuation || !action.supportsSpellTextContinuation() {
		return nil, fmt.Errorf("spell text continuation requires an eligible printed action")
	}
	return action, nil
}

// RevisePrintedSpellText changes rules for one exact stack visit.
// The physical card keeps its printed rules when the spell leaves the stack.
func (g *Game) RevisePrintedSpellText(stackID uuid.UUID, changes ...TextChange) error {
	obj := g.stack.FindByID(stackID)
	if obj == nil || obj.IsAbility || obj.Card == nil || obj.spellTextDefinition == nil {
		return fmt.Errorf("spell text continuation requires a retained spell on the stack")
	}
	if len(changes) == 0 {
		return fmt.Errorf("spell text continuation requires word changes")
	}
	if _, modal := getModalSpellAbility(obj.spellTextDefinition); modal {
		return g.reviseModalSpellText(obj, changes)
	}
	old, err := spellTextAction(obj.spellTextDefinition)
	if err != nil {
		return err
	}
	if len(obj.capturedNumbers)+len(obj.capturedPlayers)+len(obj.capturedCards)+len(obj.capturedCardSets)+len(obj.capturedPermanents) != 0 {
		return fmt.Errorf("spell text continuation cannot transfer casting bindings")
	}
	revision, err := BuildTextDefinitionRevision(obj.spellTextDefinition, changes...)
	if err != nil {
		return err
	}
	fresh, err := spellTextAction(revision.Definition())
	if err != nil {
		return err
	}
	if !sameActivatedTextContext(old, fresh) || len(old.castingConditions) != len(fresh.castingConditions) || len(old.effects) != len(fresh.effects) {
		return fmt.Errorf("spell text continuation changed its declaration context")
	}
	if _, ok := revision.ReplacementAbility(old.AbilityID()); !ok {
		return fmt.Errorf("spell text continuation lost its declaration group")
	}
	if len(obj.Targets) != len(obj.TargetSpecs) {
		return fmt.Errorf("spell text continuation requires aligned target declarations")
	}
	for i, target := range old.targets {
		if !sameSpellTextTarget(target, fresh.targets[i]) {
			return fmt.Errorf("spell text continuation changed target group %d", i+1)
		}
	}
	specs := make([]Target, len(obj.TargetSpecs))
	for i, chosen := range obj.TargetSpecs {
		index := slices.Index(old.targets, chosen)
		if index < 0 {
			return fmt.Errorf("spell text continuation lost an announced target group")
		}
		specs[i] = fresh.targets[index]
	}
	rebuilt := cloneStackObject(obj)
	rebuilt.spellTextDefinition = revision.Definition()
	rebuilt.Effects = slices.Clone(fresh.effects)
	rebuilt.TargetSpecs = specs
	for i, current := range g.stack.objects {
		if current.ID == stackID {
			g.stack.objects[i] = rebuilt
			return nil
		}
	}
	return fmt.Errorf("spell text continuation lost the stack visit")
}

func sameSpellTextTarget(a, b Target) bool {
	first, firstOK := a.(*targetDeclaration)
	second, secondOK := b.(*targetDeclaration)
	if !firstOK || !secondOK || first.name != second.name || first.optionalGraveyardCount != second.optionalGraveyardCount {
		return false
	}
	return sameSpellTextTargetSpec(first.spec, second.spec)
}

func sameSpellTextTargetSpec(a, b Target) bool {
	if a.Min() != b.Min() || a.Max() != b.Max() {
		return false
	}
	switch first := a.(type) {
	case *countBoundTarget:
		second, ok := b.(*countBoundTarget)
		return ok && first.count == second.count && first.usesX == second.usesX && first.upToX == second.upToX && first.oneToX == second.oneToX && sameSpellTextTargetSpec(first.inner, second.inner)
	case *PermanentTarget:
		second, ok := b.(*PermanentTarget)
		return ok && first.manaValueEqualX == second.manaValueEqualX && first.excludeSource == second.excludeSource &&
			first.attachedToControllerOnly == second.attachedToControllerOnly && first.toughnessLessThanSourcePower == second.toughnessLessThanSourcePower &&
			first.controllerOnly == second.controllerOnly && first.controllerOwned == second.controllerOwned &&
			first.opponentOnly == second.opponentOnly && first.opponentOwned == second.opponentOwned && first.opponentChooses == second.opponentChooses &&
			len(first.Filters) == len(second.Filters) && len(first.controllerRefs) == len(second.controllerRefs) && slices.Equal(first.combatRelations, second.combatRelations)
	case *PlayerTarget:
		second, ok := b.(*PlayerTarget)
		return ok && first.attackedThisTurn == second.attackedThisTurn
	case *OpponentTarget:
		second, ok := b.(*OpponentTarget)
		return ok && first.attackedThisTurn == second.attackedThisTurn
	case *DamageAnyTarget:
		_, ok := b.(*DamageAnyTarget)
		return ok
	case *SpellOnStackTarget:
		second, ok := b.(*SpellOnStackTarget)
		return ok && first.controlledByYou == second.controlledByYou && len(first.Filters) == len(second.Filters)
	case *typedSpellTarget:
		second, ok := b.(*typedSpellTarget)
		return ok && first.manaValueEqualX == second.manaValueEqualX && first.targetsControlledPermanent == second.targetsControlledPermanent && len(first.filters) == len(second.filters)
	default:
		return false
	}
}
