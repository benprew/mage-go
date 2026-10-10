package mage

import (
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithSpellCastSpellTextContinuation permits one cast-event rule at entry.
// Other rules and casting results require separate contracts.
func WithSpellCastSpellTextContinuation() CardOption {
	return func(card *BaseCard) { card.spellCastSpellText++ }
}

func spellCastSpellTextContext(card Card) (*GenericTriggered, error) {
	base, ok := card.(*BaseCard)
	if !ok || base == nil || base.spellCastSpellText != 1 || !isEntryCopyPermanent(base) || base.HasType(core.TypeLand) ||
		base.HasType(core.TypeInstant) || base.HasType(core.TypeSorcery) || base.HasSubType("Aura") || len(base.attrSeeds) != 0 ||
		len(base.abilities) != 1 || len(base.modes) != 0 || len(base.additionalCosts) != 0 || len(base.alternateCosts) != 0 || len(base.castTargets) != 0 ||
		base.globalStaticSpellText != 0 || base.intrinsicSpellText != 0 || base.landAuraSpellText != 0 || base.copyEntrySpellText != 0 {
		return nil, fmt.Errorf("cast-event spell text requires one marked permanent rule without casting costs or targets")
	}
	rule, ok := base.abilities[0].(*GenericTriggered)
	if !ok || rule == nil || !layeredSpellCastEligible(rule) {
		return nil, fmt.Errorf("cast-event spell text requires one marked typed battlefield trigger")
	}
	return rule, nil
}

func (g *Game) reviseSpellCastSpellText(obj *StackObject, changes []TextChange) error {
	old, err := spellCastSpellTextContext(obj.spellTextDefinition)
	if err != nil {
		return err
	}
	if len(obj.Effects) != 0 || len(obj.Targets) != 0 || len(obj.TargetSpecs) != 0 || len(obj.capturedNumbers) != 0 ||
		len(obj.capturedPlayers) != 0 || len(obj.capturedPermanents) != 0 || len(obj.capturedCards) != 0 || len(obj.capturedCardSets) != 0 {
		return fmt.Errorf("cast-event spell text requires no announced targets, bodies or casting results")
	}
	revision, err := BuildTextDefinitionRevision(obj.spellTextDefinition, changes...)
	if err != nil {
		return err
	}
	fresh, err := spellCastSpellTextContext(revision.Definition())
	if err != nil {
		return err
	}
	oldBase, newBase := obj.spellTextDefinition.(*BaseCard), revision.Definition().(*BaseCard)
	replacement, paired := revision.ReplacementAbility(old.AbilityID())
	if !paired || replacement != fresh || !slices.Equal(oldBase.subTypes, newBase.subTypes) ||
		!sameSpellCastTextContext(old.eventDefinition.(spellCastPattern), fresh.eventDefinition.(spellCastPattern)) ||
		len(old.targets) != len(fresh.targets) || len(old.effects) != len(fresh.effects) {
		return fmt.Errorf("cast-event spell text changed its declaration context")
	}
	for i, target := range old.targets {
		if !sameSpellTextTarget(target, fresh.targets[i]) {
			return fmt.Errorf("cast-event spell text changed target group %d", i+1)
		}
	}
	rebuilt := cloneStackObject(obj)
	rebuilt.spellTextDefinition = revision.Definition()
	for i, current := range g.stack.objects {
		if current.ID == obj.ID {
			g.stack.objects[i] = rebuilt
			return nil
		}
	}
	return fmt.Errorf("cast-event spell text lost its stack visit")
}

func buildSpellCastSpellEntry(obj *StackObject) (Card, error) {
	if _, err := spellCastSpellTextContext(obj.spellTextDefinition); err != nil {
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
