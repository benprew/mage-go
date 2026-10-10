package mage

import (
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithTextContinuation permits fresh rules for a pending choose-one spell.
// The selected mode and its announced targets remain fixed.
func (m *ModalSpellAbility) WithTextContinuation() *ModalSpellAbility {
	m.textContinuation = true
	return m
}

func (m *ModalSpellAbility) supportsTextContinuation() bool {
	return len(m.modes) >= 2 && len(m.dslOptions) == 0 && len(m.targets) == 0 && len(m.effects) == 0 &&
		m.supportsSpellTextContinuation() && !m.spellTextContinuation && len(m.castingConditions) == 0
}

func modalSpellTextDeclaration(card Card) (*ModalSpellAbility, error) {
	base, ok := card.(*BaseCard)
	if !ok || base == nil || (!base.HasType(core.TypeInstant) && !base.HasType(core.TypeSorcery)) ||
		len(base.abilities) != 1 || len(base.additionalCosts) != 0 || len(base.alternateCosts) != 0 || len(base.castTargets) != 0 {
		return nil, fmt.Errorf("modal spell text continuation requires one instant or sorcery declaration")
	}
	modal, ok := base.abilities[0].(*ModalSpellAbility)
	if !ok || !modal.textContinuation || !modal.supportsTextContinuation() {
		return nil, fmt.Errorf("modal spell text continuation requires an eligible printed declaration")
	}
	return modal, nil
}

func (g *Game) reviseModalSpellText(obj *StackObject, changes []TextChange) error {
	old, err := modalSpellTextDeclaration(obj.spellTextDefinition)
	if err != nil {
		return err
	}
	if len(obj.capturedNumbers)+len(obj.capturedPlayers)+len(obj.capturedCards)+len(obj.capturedCardSets)+len(obj.capturedPermanents) != 0 {
		return fmt.Errorf("modal spell text continuation cannot transfer casting bindings")
	}
	if obj.ModeChoice < 0 || obj.ModeChoice >= len(old.modes) || len(obj.ModalTargets) != len(old.modes) || len(obj.Targets) != len(obj.TargetSpecs) {
		return fmt.Errorf("modal spell text continuation requires an aligned selected mode")
	}
	for i, targets := range obj.ModalTargets {
		if (i != obj.ModeChoice && len(targets) != 0) || (i == obj.ModeChoice && !slices.Equal(targets, obj.Targets)) {
			return fmt.Errorf("modal spell text continuation changed its selected targets")
		}
	}
	revision, err := BuildTextDefinitionRevision(obj.spellTextDefinition, changes...)
	if err != nil {
		return err
	}
	fresh, err := modalSpellTextDeclaration(revision.Definition())
	if err != nil {
		return err
	}
	if !sameActivatedTextContext(&old.SpellAbility, &fresh.SpellAbility) || len(old.modes) != len(fresh.modes) {
		return fmt.Errorf("modal spell text continuation changed its declaration context")
	}
	replacement, ok := revision.ReplacementAbility(old.AbilityID())
	if !ok || replacement != fresh {
		return fmt.Errorf("modal spell text continuation lost its declaration group")
	}
	for i, mode := range old.modes {
		next := fresh.modes[i]
		if len(mode.Targets) != len(next.Targets) || len(mode.Effects) != len(next.Effects) {
			return fmt.Errorf("modal spell text continuation changed mode %d", i+1)
		}
		for j, target := range mode.Targets {
			if !sameSpellTextTarget(target, next.Targets[j]) {
				return fmt.Errorf("modal spell text continuation changed mode %d target group %d", i+1, j+1)
			}
		}
	}
	oldMode, newMode := old.modes[obj.ModeChoice], fresh.modes[obj.ModeChoice]
	specs := make([]Target, len(obj.TargetSpecs))
	for i, chosen := range obj.TargetSpecs {
		index := slices.Index(oldMode.Targets, chosen)
		if index < 0 {
			return fmt.Errorf("modal spell text continuation lost an announced target group")
		}
		specs[i] = newMode.Targets[index]
	}
	rebuilt := cloneStackObject(obj)
	rebuilt.spellTextDefinition = revision.Definition()
	rebuilt.Effects = slices.Clone(newMode.Effects)
	rebuilt.TargetSpecs = specs
	for i, current := range g.stack.objects {
		if current.ID == obj.ID {
			g.stack.objects[i] = rebuilt
			return nil
		}
	}
	return fmt.Errorf("modal spell text continuation lost the stack visit")
}
