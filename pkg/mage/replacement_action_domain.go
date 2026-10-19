package mage

// ReplacementActionDomains is a set of proposal categories a native rule can match.
// Membership does not prove that the rule is active or matches a particular event.
type ReplacementActionDomains uint8

const (
	// ReplacementDamageActions includes damage to players and permanents.
	ReplacementDamageActions ReplacementActionDomains = 1 << iota
	// ReplacementDestructionActions includes direct destruction proposals.
	ReplacementDestructionActions
	// ReplacementTapActions includes permanent tap proposals.
	ReplacementTapActions
	// ReplacementDamageAssignmentActions includes combat damage assignment proposals.
	ReplacementDamageAssignmentActions
	// ReplacementDamageLifeLossActions includes life loss that results from damage.
	ReplacementDamageLifeLossActions
	// ReplacementDrawActions includes individual card draw proposals.
	ReplacementDrawActions
	// ReplacementLifeGainActions includes positive life gain proposals.
	ReplacementLifeGainActions
)

// InspectReplacementActionDomains reads complete domains for supported native rules.
// Unknown and nil rules return false. No rule methods or operand callbacks run.
// The closed type list prevents an opaque rule from supplying an unchecked proof.
func InspectReplacementActionDomains(replacement ReplacementEffect) (ReplacementActionDomains, bool) {
	if nilDefinition(replacement) {
		return 0, false
	}
	switch replacement.(type) {
	case *preventionShieldReplacement, *sourcePreventionShieldReplacement, *halfDamagePreventionShieldReplacement,
		*fogReplacement, *creatureCombatDamagePreventionReplacement, *combatSourcePreventionReplacement,
		*playerDamageFromFlyingCreaturesPreventionReplacement, *forcefieldReplacement, *colorPreventionReplacement,
		*sourcePreventionReplacement, *typePreventionReplacement, *reverseDamageReplacement, *bodyguardReplacement,
		*playerDamageRedirectReplacement, *artifactDamageRedirectReplacement, *creatureDamageRedirectReplacement,
		*attackerDamageRedirectReplacement, *damagePreventionRuleReplacement, *counterDamagePreventionReplacement,
		*instructionDamagePrevention, *continuedDamageReplacement, *selectedSourceRedirectReplacement,
		*finiteDamageRedirectReplacement, *staticDamagePreventionReplacement, *staticDamageRedirectReplacement,
		*protectionDamageReplacement, *permanentDamageRedirectReplacement, *playerDamageRedirectThisTurn:
		return ReplacementDamageActions, true
	case *staticPlayerDamageCapReplacement:
		return ReplacementDamageActions, true
	case *scopedDrawReplacement, *futureDrawReplacement:
		return ReplacementDrawActions, true
	case *lifeGainDrawReplacement:
		return ReplacementLifeGainActions, true
	case *damageLifeFloorReplacement:
		return ReplacementDamageLifeLossActions, true
	case *faceUpActionReplacement:
		return ReplacementDamageActions | ReplacementTapActions | ReplacementDamageAssignmentActions, true
	case *regenerationReplacement, *staticRegenerationReplacement, *nextDestructionReplacement:
		return ReplacementDestructionActions, true
	default:
		return 0, false
	}
}
