package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// ChooseColorAsEnters saves the entry controller's color choice before entry.
// A new entry requires a new choice. Phasing keeps the saved choice.
func ChooseColorAsEnters() AsEntersBattlefieldAbility {
	return &colorEntryChoiceAbility{BaseAbility: BaseAbility{id: uuid.New(), abilityType: core.AbilityStatic}}
}

type colorEntryChoiceAbility struct{ BaseAbility }

func (*colorEntryChoiceAbility) OnEnter(g *Game, p *Permanent) bool {
	p.ChosenColor = core.White
	if player := g.GetPlayer(p.ControllerID()); player != nil {
		answer := player.ChooseManaColor("choose a color")
		if answer >= core.White && answer <= core.Green {
			p.ChosenColor = answer
		}
	}
	return true
}

func (*colorEntryChoiceAbility) inspectDefinition() definitionNode {
	return definitionNode{Role: "entry-color-choice", Text: "choose a color", battlefieldSource: true}
}
