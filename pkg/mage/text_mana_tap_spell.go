package mage

import (
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithManaTapSpellTextContinuation permits one land mana event rule at entry.
// Other rules and casting results require separate contracts.
func WithManaTapSpellTextContinuation() CardOption {
	return func(card *BaseCard) { card.manaTapSpellText++ }
}

func manaTapSpellTextContext(card Card) (Ability, error) {
	base, ok := card.(*BaseCard)
	if !ok || base == nil || base.manaTapSpellText != 1 || !isEntryCopyPermanent(base) || base.HasType(core.TypeLand) ||
		base.HasType(core.TypeInstant) || base.HasType(core.TypeSorcery) || base.HasSubType("Aura") || len(base.attrSeeds) != 0 ||
		len(base.abilities) != 1 || len(base.modes) != 0 || len(base.additionalCosts) != 0 || len(base.alternateCosts) != 0 || len(base.castTargets) != 0 ||
		base.spellCastSpellText != 0 || base.globalStaticSpellText != 0 || base.intrinsicSpellText != 0 || base.landAuraSpellText != 0 || base.copyEntrySpellText != 0 {
		return nil, fmt.Errorf("land-mana-event spell text requires one marked permanent rule without casting costs or targets")
	}
	rule := base.abilities[0]
	if !manaTapSpellRulePresent(rule) || !layeredManaTapEligible(rule) {
		return nil, fmt.Errorf("land-mana-event spell text requires one marked typed battlefield rule")
	}
	switch ability := rule.(type) {
	case *GenericTriggered:
		if ability.eventDefinition.(landTappedForManaPattern).attachedOnly {
			return nil, fmt.Errorf("land-mana-event spell text requires a global event")
		}
	case *ManaBonusAbility:
		if ability.AttachedOnly {
			return nil, fmt.Errorf("land-mana-event spell text requires a global bonus")
		}
	}
	return rule, nil
}

func manaTapSpellRulePresent(rule Ability) bool {
	switch ability := rule.(type) {
	case *GenericTriggered:
		return ability != nil
	case *ManaBonusAbility:
		return ability != nil
	}
	return false
}

func sameManaTapSpellTextContext(old, fresh Ability) bool {
	switch first := old.(type) {
	case *GenericTriggered:
		second, ok := fresh.(*GenericTriggered)
		if !ok || !sameManaTapTextContext(first.eventDefinition.(landTappedForManaPattern), second.eventDefinition.(landTappedForManaPattern)) ||
			len(first.targets) != len(second.targets) || len(first.effects) != len(second.effects) {
			return false
		}
		for i, target := range first.targets {
			if !sameSpellTextTarget(target, second.targets[i]) {
				return false
			}
		}
		return true
	case *ManaBonusAbility:
		second, ok := fresh.(*ManaBonusAbility)
		return ok && first.AttachedOnly == second.AttachedOnly && first.MatchProduced == second.MatchProduced && first.controllerRecipient == second.controllerRecipient
	}
	return false
}

func sameManaTapTextContext(a, b landTappedForManaPattern) bool {
	a.player, b.player = nil, nil
	a.amount, b.amount = nil, nil
	a.land, b.land = nil, nil
	a.types, b.types = nil, nil
	a.manaType, b.manaType = core.Colorless, core.Colorless
	return a == b
}

func (g *Game) reviseManaTapSpellText(obj *StackObject, changes []TextChange) error {
	old, err := manaTapSpellTextContext(obj.spellTextDefinition)
	if err != nil {
		return err
	}
	if len(obj.Effects) != 0 || len(obj.Targets) != 0 || len(obj.TargetSpecs) != 0 || len(obj.capturedNumbers) != 0 ||
		len(obj.capturedPlayers) != 0 || len(obj.capturedPermanents) != 0 || len(obj.capturedCards) != 0 || len(obj.capturedCardSets) != 0 {
		return fmt.Errorf("land-mana-event spell text requires no announced targets, bodies or casting results")
	}
	revision, err := BuildTextDefinitionRevision(obj.spellTextDefinition, changes...)
	if err != nil {
		return err
	}
	fresh, err := manaTapSpellTextContext(revision.Definition())
	if err != nil {
		return err
	}
	oldBase, newBase := obj.spellTextDefinition.(*BaseCard), revision.Definition().(*BaseCard)
	replacement, paired := revision.ReplacementAbility(old.AbilityID())
	if !paired || replacement != fresh || !slices.Equal(oldBase.subTypes, newBase.subTypes) || !sameManaTapSpellTextContext(old, fresh) {
		return fmt.Errorf("land-mana-event spell text changed its declaration context")
	}
	rebuilt := cloneStackObject(obj)
	rebuilt.spellTextDefinition = revision.Definition()
	for i, current := range g.stack.objects {
		if current.ID == obj.ID {
			g.stack.objects[i] = rebuilt
			return nil
		}
	}
	return fmt.Errorf("land-mana-event spell text lost its stack visit")
}

func buildManaTapSpellEntry(obj *StackObject) (Card, error) {
	if _, err := manaTapSpellTextContext(obj.spellTextDefinition); err != nil {
		return nil, err
	}
	fresh, err := BuildTextChangedDefinition(obj.spellTextDefinition)
	if err != nil {
		return nil, err
	}
	base := fresh.(*BaseCard)
	base.id, base.owner, base.isToken = obj.Card.ID(), obj.Card.Owner(), obj.Card.IsToken()
	base.departureCard = obj.Card
	return base, nil
}
