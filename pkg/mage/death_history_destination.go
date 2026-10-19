package mage

import "github.com/benprew/mage-go/pkg/mage/core"

func (g *Game) inspectDeathHistory(source DamageSource, filter PermanentFilter, creaturesOnly bool, p *Permanent) (matches, known bool) {
	predicate, known := InspectPermanentCardTypeFilter(filter)
	if !known {
		return false, false
	}
	var types []core.CardType
	for _, kind := range []core.CardType{core.TypeCreature, core.TypeInstant, core.TypeSorcery, core.TypeLand, core.TypeArtifact, core.TypeEnchantment, core.TypePlaneswalker} {
		if p.HasType(kind) {
			types = append(types, kind)
		}
	}
	if !predicate.Matches(types...) {
		return false, true
	}
	recipient := capturedPermanent{p.ID(), p.incarnationID}
	if creaturesOnly {
		return g.damage.turnDamageToCreatures[capturedPermanent{source.ID(), source.Incarnation()}][recipient], true
	}
	return g.damage.dealtDamageToPermanentThisTurn(source, recipient), true
}

func (g *Game) inspectDeathHistoryRegeneration(p *Permanent) (allowed, known bool) {
	prohibited, unknown := false, false
	for _, effect := range g.effects.effects {
		rule, ok := effect.(*liveDamageRegenerationRule)
		if !ok {
			continue
		}
		if rule == nil || rule.funcContinuousEffect == nil || rule.active != nil || rule.duration != core.EndOfTurn {
			unknown = true
			continue
		}
		matches, known := g.inspectDeathHistory(rule.source, rule.filter, rule.creaturesOnly, p)
		if !known {
			unknown = true
		}
		if matches {
			prohibited = true
		}
	}
	if unknown {
		return false, false
	}
	if prohibited {
		return false, true
	}
	return true, true
}
