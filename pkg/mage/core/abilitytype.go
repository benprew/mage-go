package core

//go:generate enumer -type=AbilityType -trimprefix=Ability -output=abilitytype_enumer.go

// AbilityType classifies abilities.
type AbilityType int

const (
	AbilitySpell AbilityType = iota
	AbilityActivated
	AbilityTriggered
	AbilityMana
	AbilityStatic
)
