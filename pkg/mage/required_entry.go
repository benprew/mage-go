package mage

import (
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// RequiredEntryOperations runs one required sequence before battlefield entry.
// A rejected child stops the sequence and prevents entry.
func RequiredEntryOperations(operations ...AsEntersBattlefieldAbility) AsEntersBattlefieldAbility {
	return &requiredEntryAbility{
		BaseAbility: BaseAbility{id: uuid.New(), abilityType: core.AbilityStatic},
		operations:  slices.Clone(operations),
	}
}

type requiredEntryAbility struct {
	BaseAbility
	operations []AsEntersBattlefieldAbility
}

func (a *requiredEntryAbility) OnEnter(g *Game, p *Permanent) bool {
	for _, operation := range a.operations {
		if !operation.OnEnter(g, p) {
			return false
		}
	}
	return true
}

func (a *requiredEntryAbility) inspectDefinition() definitionNode {
	n := definitionNode{Role: "required-entry-operations", Text: "required entry operations", battlefieldSource: true}
	if len(a.operations) == 0 {
		n.problem = "supply at least one entry operation"
	}
	for i, operation := range a.operations {
		n.children = append(n.children, definitionEdge{fmt.Sprintf("operations[%d]", i), operation})
	}
	return n
}
