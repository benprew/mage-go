package mage

import (
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithManaTapTextRevision permits checked replacement of a printed trigger.
// The declaration must use a typed land mana event on the battlefield.
func (t *GenericTriggered) WithManaTapTextRevision() *GenericTriggered {
	t.manaTapTextRevision = true
	return t
}

func (t *GenericTriggered) supportsManaTapTextRevision() bool {
	_, typed := t.eventDefinition.(landTappedForManaPattern)
	return typed && t.eventType == core.EvtTappedForMana && t.sourceZone == core.ZoneBattlefield && len(t.zones) == 0 && !t.isStateTrigger
}

// WithManaTapTextRevision permits checked replacement of a printed mana bonus.
// The shared mana service keeps payment planning and production together.
func (a *ManaBonusAbility) WithManaTapTextRevision() *ManaBonusAbility {
	a.manaTapTextRevision = true
	return a
}

func (a *ManaBonusAbility) supportsManaTapTextRevision() bool {
	return a.abilityType == core.AbilityStatic &&
		a.BonusMana >= core.Colorless && a.BonusMana < core.AnyColor &&
		(!a.AttachedOnly || !a.MatchProduced && a.Filter.IsZero())
}

// RevisePrintedManaTapTrigger replaces one present printed mana-tap declaration.
// The caller must supply text-layer order and lifetime separately.
func (g *Game) RevisePrintedManaTapTrigger(source *Permanent, abilityID uuid.UUID, changes ...TextChange) error {
	if source == nil {
		return fmt.Errorf("text mana-tap revision requires a present source")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("text mana-tap revision requires the original face-up source")
	}
	for _, ability := range current.RuntimeAbilities {
		if ability.AbilityID() != abilityID {
			continue
		}
		if _, ok := ability.(*GenericTriggered); ok {
			return g.revisePrintedTrigger(source, abilityID, func(t *GenericTriggered) bool {
				return t.manaTapTextRevision && t.supportsManaTapTextRevision()
			}, changes...)
		}
		active, ok := ability.(*ManaBonusAbility)
		if !ok || !active.manaTapTextRevision || !active.supportsManaTapTextRevision() {
			return fmt.Errorf("text mana-tap revision requires an eligible printed declaration")
		}
		return g.revisePrintedManaBonus(current, active, changes...)
	}
	return fmt.Errorf("text mana-tap revision requires an active declaration")
}

func (g *Game) revisePrintedManaBonus(current *Permanent, active *ManaBonusAbility, changes ...TextChange) error {
	base, ok := current.Card.(*BaseCard)
	if !ok {
		return fmt.Errorf("text mana-tap revision requires a retained definition")
	}
	key, ok := base.abilityKeys[active.AbilityID()]
	if !ok {
		return fmt.Errorf("text mana-tap revision requires a printed declaration")
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
		return fmt.Errorf("text mana-tap revision lost its declaration group")
	}
	revision, err := BuildTextDefinitionRevision(definition, changes...)
	if err != nil {
		return err
	}
	replacement, ok := revision.ReplacementAbility(original.AbilityID())
	if !ok {
		return fmt.Errorf("text mana-tap revision lost its replacement")
	}
	fresh, ok := replacement.(*ManaBonusAbility)
	if !ok || !fresh.manaTapTextRevision || !fresh.supportsManaTapTextRevision() ||
		fresh.AttachedOnly != active.AttachedOnly || fresh.MatchProduced != active.MatchProduced || fresh.controllerRecipient != active.controllerRecipient {
		return fmt.Errorf("text mana-tap revision changed the bonus contract")
	}
	rebuilt := *fresh
	rebuilt.id = active.id
	rebuilt.source = current.ID()
	rebuilt.controller = current.ControllerID()
	rebuilt.textRevisionDefinition = revision.Definition()
	mutable := g.MutablePermanent(current.ID())
	mutable.RuntimeAbilities = slices.Clone(mutable.RuntimeAbilities)
	for i, ability := range mutable.RuntimeAbilities {
		if ability.AbilityID() == active.AbilityID() {
			mutable.RuntimeAbilities[i] = &rebuilt
			break
		}
	}
	mutable.revisedPrintedManaBonuses = maps.Clone(mutable.revisedPrintedManaBonuses)
	if mutable.revisedPrintedManaBonuses == nil {
		mutable.revisedPrintedManaBonuses = make(map[uuid.UUID]*ManaBonusAbility)
	}
	mutable.revisedPrintedManaBonuses[active.AbilityID()] = &rebuilt
	return nil
}
