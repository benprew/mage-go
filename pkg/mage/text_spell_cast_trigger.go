package mage

import (
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithSpellCastTextRevision permits checked replacement of a printed trigger.
// The declaration must use a typed spell-cast event on the battlefield.
func (t *GenericTriggered) WithSpellCastTextRevision() *GenericTriggered {
	t.spellCastTextRevision = true
	return t
}

func (t *GenericTriggered) supportsSpellCastTextRevision() bool {
	_, typed := t.eventDefinition.(spellCastPattern)
	return typed && t.eventType == core.EvtSpellCast && t.sourceZone == core.ZoneBattlefield && len(t.zones) == 0 && !t.isStateTrigger
}

// RevisePrintedSpellCastTrigger replaces one present printed spell-cast trigger.
// The caller must supply the text layer's order and lifetime separately.
func (g *Game) RevisePrintedSpellCastTrigger(source *Permanent, abilityID uuid.UUID, changes ...TextChange) error {
	marked := false
	if source != nil {
		if current := g.FindPermanent(source.ID()); current != nil {
			for _, a := range current.RuntimeAbilities {
				if a.AbilityID() == abilityID {
					marked = layeredSpellCastEligible(a)
				}
			}
			if _, exists := current.layeredSpellCastOrigins[abilityID]; exists {
				return fmt.Errorf("direct spell-cast revision cannot change a layered result")
			}
		}
	}
	return g.revisePrintedTrigger(source, abilityID, func(t *GenericTriggered) bool {
		return t.spellCastTextRevision && t.supportsSpellCastTextRevision() && (!marked || t.spellCastTextLayer)
	}, changes...)
}

func (g *Game) revisePrintedTrigger(source *Permanent, abilityID uuid.UUID, eligible func(*GenericTriggered) bool, changes ...TextChange) error {
	if source == nil {
		return fmt.Errorf("text trigger revision requires a present source")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("text trigger revision requires the original face-up source")
	}
	base, ok := current.Card.(*BaseCard)
	if !ok {
		return fmt.Errorf("text trigger revision requires a retained definition")
	}
	key, ok := base.abilityKeys[abilityID]
	if !ok {
		return fmt.Errorf("text trigger revision requires a printed declaration")
	}
	index := -1
	var active *GenericTriggered
	for i, ability := range current.RuntimeAbilities {
		if ability.AbilityID() != abilityID {
			continue
		}
		active, ok = ability.(*GenericTriggered)
		if !ok || !eligible(active) {
			return fmt.Errorf("text trigger revision requires an eligible printed trigger")
		}
		index = i
		break
	}
	if index < 0 {
		return fmt.Errorf("text trigger revision requires an active declaration")
	}
	definition := current.Card
	if active.textRevisionDefinition != nil {
		definition = active.textRevisionDefinition
	}
	declarations, err := definitionAbilitiesByKey(definition.(*BaseCard), true)
	if err != nil {
		return err
	}
	original, ok := declarations[key]
	if !ok {
		return fmt.Errorf("text trigger revision lost its declaration group")
	}
	revision, err := BuildTextDefinitionRevision(definition, changes...)
	if err != nil {
		return err
	}
	replacement, ok := revision.ReplacementAbility(original.AbilityID())
	if !ok {
		return fmt.Errorf("text trigger revision lost its replacement")
	}
	fresh, ok := replacement.(*GenericTriggered)
	if !ok || !eligible(fresh) {
		return fmt.Errorf("text trigger revision changed the trigger contract")
	}
	rebuilt := *fresh
	rebuilt.id = active.id
	rebuilt.source = current.ID()
	rebuilt.controller = current.ControllerID()
	rebuilt.textRevisionDefinition = revision.Definition()
	mutable := g.MutablePermanent(current.ID())
	mutable.RuntimeAbilities = slices.Clone(mutable.RuntimeAbilities)
	mutable.RuntimeAbilities[index] = &rebuilt
	mutable.revisedPrintedTriggers = maps.Clone(mutable.revisedPrintedTriggers)
	if mutable.revisedPrintedTriggers == nil {
		mutable.revisedPrintedTriggers = make(map[uuid.UUID]*GenericTriggered)
	}
	mutable.revisedPrintedTriggers[abilityID] = &rebuilt
	return nil
}
