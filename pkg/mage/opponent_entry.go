package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// ChooseOpponentAsEnters saves the entry controller's opponent before entry.
// Each permanent keeps its choice through control changes and phasing.
func ChooseOpponentAsEnters() AsEntersBattlefieldAbility {
	return &opponentEntryChoiceAbility{BaseAbility: BaseAbility{id: uuid.New(), abilityType: core.AbilityStatic}}
}

type opponentEntryChoiceAbility struct{ BaseAbility }

func (*opponentEntryChoiceAbility) OnEnter(g *Game, p *Permanent) bool {
	if opponent := g.GetOpponent(p.ControllerID()); opponent != nil {
		p.ChosenPlayer = opponent.PlayerID()
	}
	return true
}

func (*opponentEntryChoiceAbility) inspectDefinition() definitionNode {
	return definitionNode{Role: "entry-player-choice", Text: "choose an opponent", battlefieldSource: true}
}
