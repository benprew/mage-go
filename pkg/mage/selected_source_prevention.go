package mage

// PreventNextDamageFromSelectedSource prevents one source event this turn.
// A shield on a permanent spell also applies to the permanent it becomes.
// Source enumeration remains the caller's responsibility.
func PreventNextDamageFromSelectedSource(source DamageSourceRef, recipient PlayerRef, filter DamageSourceFilter) ResolvingEffect {
	return &exactSourcePreventionInstallation{source: source, recipient: recipient, filter: filter, continuePermanentSpell: true}
}

func (g *Game) continueSpellDamageReplacements(spell DamageSource, permanent *Permanent) {
	for _, list := range [2][]ReplacementEffect{g.effects.replacements, g.effects.cycleReplacements} {
		for _, replacement := range list {
			if redirect, ok := replacement.(*selectedSourceRedirectReplacement); ok && redirect.IsActive(g) && redirect.source.SameObject(spell) {
				redirect.continued = damageSourceFromPermanent(permanent)
			}
			shield, ok := replacement.(*sourcePreventionReplacement)
			if ok && shield.continuePermanentSpell && shield.IsActive(g) && shield.exactSource.SameObject(spell) {
				shield.continuedSource = damageSourceFromPermanent(permanent)
			}
		}
	}
}

func (g *Game) putPermanentSpellOnBattlefield(obj *StackObject, card Card, colorsSpent int) *Permanent {
	permanent := g.prepareBattlefieldEntry(card, obj.Controller, obj.ColorOverride, colorsSpent, false)
	if permanent == nil {
		return nil
	}
	g.registerBattlefieldEntry(permanent, obj.ColorOverride, obj.SourceID)
	g.continueSpellDamageReplacements(damageSourceFromSpellObject(obj), permanent)
	g.completeBattlefieldEntry(permanent, obj.Controller, obj.Controller, colorsSpent)
	return permanent
}
