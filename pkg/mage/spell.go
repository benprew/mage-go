package mage

// SpellAbility is kept as a compatibility name for spell actions.
type SpellAbility = ActionDefinition

// NewSpell creates a spell action. Arguments may be Effect values or ActionOption
// values, allowing card code to use one builder for effects and targets.
func NewSpell(parts ...any) *ActionDefinition {
	return NewAction(ActionSpell, actionPartsToOptions(parts...)...)
}

// NewSpellAbility creates a spell ability with the given effects and no targets.
func NewSpellAbility(effects ...Effect) *SpellAbility {
	parts := make([]any, 0, len(effects))
	for _, effect := range effects {
		parts = append(parts, effect)
	}
	return NewSpell(parts...)
}

// NewTargetedSpell creates a spell ability that requires a target.
func NewTargetedSpell(target Target, effects ...Effect) *SpellAbility {
	parts := make([]any, 0, len(effects)+1)
	for _, effect := range effects {
		parts = append(parts, effect)
	}
	parts = append(parts, WithTarget(target))
	return NewSpell(parts...)
}
