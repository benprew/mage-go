package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// PreventNextDamageFromSelectedSource prevents one source event this turn.
// A shield on a permanent spell also applies to the permanent it becomes.
// Source enumeration remains the caller's responsibility.
func PreventNextDamageFromSelectedSource(source DamageSourceRef, recipient PlayerRef, filter DamageSourceFilter) ResolvingEffect {
	return &exactSourcePreventionInstallation{source: source, recipient: recipient, filter: filter, continuePermanentSpell: true}
}

func (g *Game) continueSpellDamageReplacements(spell DamageSource, permanent *Permanent) {
	for _, list := range [2][]ReplacementEffect{g.effects.replacements, g.effects.cycleReplacements} {
		for _, replacement := range list {
			if rule, ok := replacement.(*continuedDamageReplacement); ok && rule.IsActive(g) && rule.source.SameObject(spell) {
				rule.continued = damageSourceFromPermanent(permanent)
			}
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

func (g *Game) putPermanentSpellOnBattlefield(obj *StackObject, card Card, colorsSpent int, hostID uuid.UUID) *Permanent {
	context := battlefieldEntryContext{announcedX: obj.XValue, castColorsSpent: colorsSpent, auraSpell: true}
	if hostID != uuid.Nil && obj.TargetZones[hostID] == ZoneGraveyard {
		context.graveyardHost = capturedCard{id: hostID, visit: obj.targetIncarnations[hostID], zone: ZoneGraveyard}
	}
	permanent := g.prepareBattlefieldEntryWithContext(card, obj.Controller, obj.ColorOverride, context, false)
	if permanent == nil {
		if graveyardEnchantTarget(card) != nil && g.graveyardVisit(card.ID()) == uuid.Nil {
			g.cleanupSpell(obj)
		}
		return nil
	}
	g.registerBattlefieldEntry(permanent, obj.ColorOverride, obj.SourceID)
	g.continueSpellDamageReplacements(damageSourceFromSpellObject(obj), permanent)
	g.continueSpellExileReplacement(damageSourceFromSpellObject(obj), permanent)
	if hostID != uuid.Nil && permanent.enchantedCard.id == uuid.Nil {
		g.Attach(permanent.ID(), hostID)
	}
	g.completeBattlefieldEntry(permanent, obj.Controller, obj.Controller, colorsSpent)
	return permanent
}
