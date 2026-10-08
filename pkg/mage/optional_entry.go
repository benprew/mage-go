package mage

import (
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// MayEntryOperations lets the entry controller accept a sequence before entry.
// A declined sequence does not prevent the permanent from entering.
func MayEntryOperations(operations ...AsEntersBattlefieldAbility) AsEntersBattlefieldAbility {
	return &optionalEntryAbility{
		BaseAbility: BaseAbility{id: uuid.New(), abilityType: core.AbilityStatic},
		operations:  slices.Clone(operations),
	}
}

type optionalEntryAbility struct {
	BaseAbility
	operations []AsEntersBattlefieldAbility
}

func (a *optionalEntryAbility) OnEnter(g *Game, p *Permanent) bool {
	player := g.GetPlayer(p.ControllerID())
	if player == nil || !player.ChooseMayAbility("use this permanent's optional entry ability") {
		return true
	}
	for _, operation := range a.operations {
		if !operation.OnEnter(g, p) {
			return false
		}
	}
	return true
}

func (a *optionalEntryAbility) inspectDefinition() definitionNode {
	n := definitionNode{Role: "optional-entry-operations", Text: "optional entry operations", battlefieldSource: true}
	if len(a.operations) == 0 {
		n.problem = "supply at least one entry operation"
	}
	for i, operation := range a.operations {
		n.children = append(n.children, definitionEdge{fmt.Sprintf("operations[%d]", i), operation})
	}
	return n
}

// EnterTappedAsEnters sets tapped status before entry observers run.
// Use this entry operation when tapped entry is part of an optional sequence.
func EnterTappedAsEnters() AsEntersBattlefieldAbility {
	return &tappedEntryAbility{BaseAbility: BaseAbility{id: uuid.New(), abilityType: core.AbilityStatic}}
}

type tappedEntryAbility struct{ BaseAbility }

func (*tappedEntryAbility) OnEnter(_ *Game, p *Permanent) bool {
	p.Tapped = true
	return true
}

func (*tappedEntryAbility) inspectDefinition() definitionNode {
	return definitionNode{Role: "tapped-entry-operation", Text: "enter tapped", battlefieldSource: true}
}
