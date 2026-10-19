package mage

func damageReplacementProhibited(action Action, g *Game, replacement ReplacementEffect) bool {
	if !damageReplacementBlocked(action, g) {
		return false
	}
	if additional, ok := replacement.(interface{ appliesToUnpreventableDamage() bool }); ok && additional.appliesToUnpreventableDamage() {
		return false
	}
	if prevention, ok := replacement.(PreventionEffect); ok && prevention.IsPreventionEffect() {
		return true
	}
	_, redirect := replacement.(interface{ redirectsDamage() })
	return redirect
}

func (*bodyguardReplacement) redirectsDamage()               {}
func (*playerDamageRedirectReplacement) redirectsDamage()    {}
func (*artifactDamageRedirectReplacement) redirectsDamage()  {}
func (*creatureDamageRedirectReplacement) redirectsDamage()  {}
func (*attackerDamageRedirectReplacement) redirectsDamage()  {}
func (*permanentDamageRedirectReplacement) redirectsDamage() {}
func (*staticDamageRedirectReplacement) redirectsDamage()    {}
func (*selectedSourceRedirectReplacement) redirectsDamage()  {}
func (*finiteDamageRedirectReplacement) redirectsDamage()    {}

func (*instructionDamagePrevention) IsPreventionEffect() bool { return true }
func (*protectionDamageReplacement) IsPreventionEffect() bool { return true }
func (*reverseDamageReplacement) IsPreventionEffect() bool    { return true }
func (*playerDamageRedirectThisTurn) redirectsDamage()        {}
