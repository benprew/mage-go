package mage

import "fmt"

type copyTriggeredAddition struct {
	build func() *GenericTriggered
}

// CopyAddTriggeredAbility adds a triggered rule to the copiable values.
// The pure builder must supply fresh declarations without saved game state.
func CopyAddTriggeredAbility(build func() *GenericTriggered) CopyModification {
	return copyTriggeredAddition{build: build}
}

func (m copyTriggeredAddition) validateCopyModification() error {
	if m.build == nil {
		return fmt.Errorf("copy ability addition requires a builder")
	}
	_, err := NewCardDefinitionPlan(func() Card {
		return NewCreature("Copy ability declaration", "", 1, 1, WithAbility(m.build()))
	})
	return err
}

func (m copyTriggeredAddition) applyCopyModification(card *BaseCard) {
	card.AddAbility(m.build())
}
