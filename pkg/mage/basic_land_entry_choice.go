package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// ChooseBasicLandTypeAsEnters saves the controller's choice before entry.
// A new entry requires a new choice. Phasing keeps the saved choice.
func ChooseBasicLandTypeAsEnters() AsEntersBattlefieldAbility {
	return &basicLandEntryChoiceAbility{BaseAbility: BaseAbility{id: uuid.New(), abilityType: core.AbilityStatic}}
}

type basicLandEntryChoiceAbility struct{ BaseAbility }

func (*basicLandEntryChoiceAbility) OnEnter(g *Game, p *Permanent) bool {
	options := []string{"Plains", "Island", "Swamp", "Mountain", "Forest"}
	p.ChosenBasicLandType = core.PlainsLandType
	if player := g.GetPlayer(p.ControllerID()); player != nil {
		answer := player.ChooseString(slices.Clone(options), "choose a basic land type")
		if index := slices.Index(options, answer); index >= 0 {
			p.ChosenBasicLandType = core.BasicLandType(index + 1)
		}
	}
	return true
}

func (*basicLandEntryChoiceAbility) inspectDefinition() definitionNode {
	return definitionNode{Role: "entry-land-type-choice", Text: "choose a basic land type", battlefieldSource: true}
}

// BecomesChosenBasicLandTypeAttachedEffect uses the source's saved land choice.
// The existing land-type service applies the mana and ability rules.
func BecomesChosenBasicLandTypeAttachedEffect() ContinuousEffect {
	return AttachedEffect(core.LayerType, func(g *Game, source, target *Permanent) error {
		if landType := source.ChosenBasicLandType.String(); landType != "" {
			applyBecomesBasicLand(g, target, []string{landType})
		}
		return nil
	})
}
