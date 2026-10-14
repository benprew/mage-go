package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type targetingAbilityDeclaration struct {
	id      uuid.UUID
	targets []Target
}

// AbilityTargetSource retains the selected ability and all its target declarations.
// The caller supplies the selected mode's declarations before target choices.
func (g *Game) AbilityTargetSource(card Card, ability Ability, declared []Target) Card {
	if card == nil || nilDefinition(ability) || ability.AbilityID() == uuid.Nil {
		return card
	}
	if ability.Type() != core.AbilityActivated && ability.Type() != core.AbilityTriggered && ability.Type() != core.AbilityMana {
		return card
	}
	if _, saved := card.(targetSourceLKI); !saved {
		obj := &StackObject{TargetSource: card, SourceID: card.ID(), IsAbility: true}
		if source := g.FindPermanentIncludingPhased(card.ID()); source != nil {
			obj.sourceIncarnation = source.incarnationID
		}
		card = g.resolutionTargetSource(obj)
	}
	return abilityTargetSource(card, ability.AbilityID(), declared)
}

func abilityTargetSource(card Card, id uuid.UUID, declared []Target) Card {
	if card == nil || id == uuid.Nil {
		return card
	}
	view, ok := card.(targetSourceLKI)
	if !ok {
		view = targetSourceLKI{Card: card, colors: cardColors(card)}
	}
	view.spell = false
	view.spellDeclarations = nil
	view.spellDeclarationsKnown = false
	view.abilityDeclaration = &targetingAbilityDeclaration{id: id, targets: slices.Clone(declared)}
	return view
}

// TargetSourceAbility returns the saved ability ID and a separate declaration list.
// A known empty list is distinct from a source with no saved ability declaration.
// The target declarations retain their existing lifetime and ownership contracts.
func TargetSourceAbility(source Card) (id uuid.UUID, declared []Target, known bool) {
	view, ok := source.(targetSourceLKI)
	if !ok || view.spell || view.abilityDeclaration == nil {
		return uuid.Nil, nil, false
	}
	return view.abilityDeclaration.id, slices.Clone(view.abilityDeclaration.targets), true
}

func (g *Game) triggerTargetSourceWithDeclarations(pt *pendingTrigger, declared []Target) Card {
	card := g.triggerTargetSource(pt)
	if pt.delayed != nil {
		return abilityTargetSource(card, pt.delayed.targetingAbilityID, declared)
	}
	return g.AbilityTargetSource(card, pt.ability, declared)
}
