package mage

import (
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithColorTextRevision permits checked replacement of printed color protection.
func (pa *ProtectionAbility) WithColorTextRevision() *ProtectionAbility {
	pa.colorTextRevision = true
	return pa
}

func (pa *ProtectionAbility) supportsColorTextRevision() bool {
	if len(pa.FromColors) == 0 || pa.ExemptCardID != uuid.Nil {
		return false
	}
	for _, color := range pa.FromColors {
		if !slices.Contains([]core.Color{core.White, core.Blue, core.Black, core.Red, core.Green}, color) {
			return false
		}
	}
	return true
}

// RevisePrintedProtection replaces one present printed color-protection rule.
// The caller must supply text-layer order and lifetime separately.
func (g *Game) RevisePrintedProtection(source *Permanent, abilityID uuid.UUID, changes ...TextChange) error {
	if source != nil {
		if current := g.FindPermanent(source.ID()); current != nil {
			if _, layered := current.layeredProtectionOrigins[abilityID]; layered {
				return fmt.Errorf("direct protection revision cannot change a layered declaration")
			}
		}
	}
	return g.revisePrintedProtection(source, abilityID, true, changes...)
}

func (g *Game) revisePrintedProtection(source *Permanent, abilityID uuid.UUID, refresh bool, changes ...TextChange) error {
	if source == nil {
		return fmt.Errorf("text protection revision requires a present source")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("text protection revision requires the original face-up source")
	}
	base, ok := current.Card.(*BaseCard)
	if !ok {
		return fmt.Errorf("text protection revision requires a retained definition")
	}
	key, ok := base.abilityKeys[abilityID]
	if !ok {
		return fmt.Errorf("text protection revision requires a printed declaration")
	}
	index := -1
	var active *ProtectionAbility
	for i, ability := range current.RuntimeAbilities {
		if ability.AbilityID() != abilityID {
			continue
		}
		active, ok = ability.(*ProtectionAbility)
		if !ok || !active.colorTextRevision || !active.supportsColorTextRevision() {
			return fmt.Errorf("text protection revision requires eligible printed color protection")
		}
		index = i
		break
	}
	if index < 0 {
		return fmt.Errorf("text protection revision requires an active declaration")
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
		return fmt.Errorf("text protection revision lost its declaration group")
	}
	revision, err := BuildTextDefinitionRevision(definition, changes...)
	if err != nil {
		return err
	}
	replacement, ok := revision.ReplacementAbility(original.AbilityID())
	if !ok {
		return fmt.Errorf("text protection revision lost its replacement")
	}
	fresh, ok := replacement.(*ProtectionAbility)
	if !ok || !fresh.colorTextRevision || !fresh.supportsColorTextRevision() || (active.textLayerEligible && !fresh.textLayerEligible) {
		return fmt.Errorf("text protection revision changed the protection contract")
	}
	rebuilt := *fresh
	rebuilt.id = active.id
	rebuilt.source = current.ID()
	rebuilt.controller = current.ControllerID()
	rebuilt.textRevisionDefinition = revision.Definition()
	mutable := g.MutablePermanent(current.ID())
	mutable.RuntimeAbilities = slices.Clone(mutable.RuntimeAbilities)
	mutable.RuntimeAbilities[index] = &rebuilt
	mutable.revisedPrintedProtections = maps.Clone(mutable.revisedPrintedProtections)
	if mutable.revisedPrintedProtections == nil {
		mutable.revisedPrintedProtections = make(map[uuid.UUID]*ProtectionAbility)
	}
	mutable.revisedPrintedProtections[abilityID] = &rebuilt
	if refresh {
		g.ApplyContinuousEffects()
	}
	return nil
}
