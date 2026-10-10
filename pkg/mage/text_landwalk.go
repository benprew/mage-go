package mage

import (
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// PrintedBasicLandwalkAbility keeps one printed keyword declaration separate.
// Its keyword contribution uses the shared attribute and combat services.
type PrintedBasicLandwalkAbility struct {
	BaseAbility
	keyword                core.Attr
	textLayerEligible      bool
	textRevisionDefinition Card
}

// WithPrintedBasicLandwalk adds a checked printed basic-landwalk declaration.
// Granted keywords must use their separate ability-layer services.
func WithPrintedBasicLandwalk(keyword core.Attr) CardOption {
	return func(card *BaseCard) {
		if isBasicLandwalk(keyword) {
			WithKeyword(keyword)(card)
		}
		card.AddAbility(&PrintedBasicLandwalkAbility{
			BaseAbility: BaseAbility{id: uuid.New(), abilityType: core.AbilityStatic},
			keyword:     keyword,
		})
	}
}

func isBasicLandwalk(keyword core.Attr) bool {
	return slices.Contains([]core.Attr{core.Plainswalk, core.Islandwalk, core.Swampwalk, core.Mountainwalk, core.Forestwalk}, keyword)
}

// RevisePrintedBasicLandwalk replaces one present printed basic-landwalk rule.
// The caller must supply text-layer order and lifetime separately.
func (g *Game) RevisePrintedBasicLandwalk(source *Permanent, abilityID uuid.UUID, changes ...TextChange) error {
	if source != nil {
		if current := g.FindPermanent(source.ID()); current != nil {
			if _, layered := current.layeredLandwalkOrigins[abilityID]; layered {
				return fmt.Errorf("direct landwalk revision cannot change a layered declaration")
			}
		}
	}
	return g.revisePrintedBasicLandwalk(source, abilityID, true, changes...)
}

func (g *Game) revisePrintedBasicLandwalk(source *Permanent, abilityID uuid.UUID, refresh bool, changes ...TextChange) error {
	if source == nil {
		return fmt.Errorf("text landwalk revision requires a present source")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("text landwalk revision requires the original face-up source")
	}
	base, ok := current.Card.(*BaseCard)
	if !ok {
		return fmt.Errorf("text landwalk revision requires a retained definition")
	}
	key, ok := base.abilityKeys[abilityID]
	if !ok {
		return fmt.Errorf("text landwalk revision requires a printed declaration")
	}
	index := -1
	var active *PrintedBasicLandwalkAbility
	for i, ability := range current.RuntimeAbilities {
		if ability.AbilityID() != abilityID {
			continue
		}
		active, ok = ability.(*PrintedBasicLandwalkAbility)
		if !ok || !isBasicLandwalk(active.keyword) {
			return fmt.Errorf("text landwalk revision requires eligible printed basic landwalk")
		}
		index = i
		break
	}
	if index < 0 {
		return fmt.Errorf("text landwalk revision requires an active declaration")
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
		return fmt.Errorf("text landwalk revision lost its declaration group")
	}
	revision, err := BuildTextDefinitionRevision(definition, changes...)
	if err != nil {
		return err
	}
	replacement, ok := revision.ReplacementAbility(original.AbilityID())
	if !ok {
		return fmt.Errorf("text landwalk revision lost its replacement")
	}
	fresh, ok := replacement.(*PrintedBasicLandwalkAbility)
	if !ok || !isBasicLandwalk(fresh.keyword) || (active.textLayerEligible && !fresh.textLayerEligible) {
		return fmt.Errorf("text landwalk revision changed the landwalk contract")
	}
	rebuilt := *fresh
	rebuilt.id = active.id
	rebuilt.source = current.ID()
	rebuilt.controller = current.ControllerID()
	rebuilt.textRevisionDefinition = revision.Definition()
	mutable := g.MutablePermanent(current.ID())
	mutable.baseAttrs[active.keyword]--
	mutable.printedAbilityAttrs[active.keyword]--
	mutable.baseAttrs[fresh.keyword]++
	mutable.printedAbilityAttrs[fresh.keyword]++
	mutable.RuntimeAbilities = slices.Clone(mutable.RuntimeAbilities)
	mutable.RuntimeAbilities[index] = &rebuilt
	mutable.revisedPrintedLandwalks = maps.Clone(mutable.revisedPrintedLandwalks)
	if mutable.revisedPrintedLandwalks == nil {
		mutable.revisedPrintedLandwalks = make(map[uuid.UUID]*PrintedBasicLandwalkAbility)
	}
	mutable.revisedPrintedLandwalks[abilityID] = &rebuilt
	if refresh {
		g.ApplyContinuousEffects()
	}
	return nil
}
