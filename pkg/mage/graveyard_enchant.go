package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// GraveyardCreatureEnchantTarget declares an enchant creature card in a graveyard ability.
func GraveyardCreatureEnchantTarget() Target {
	target := TargetCreatureCardInAnyGraveyard().(*AnyGraveyardCardTarget)
	target.graveyardEnchant = true
	return target
}

// Read explicit targets so invalid abilities can reach definition validation.
func graveyardEnchantTarget(card Card) *AnyGraveyardCardTarget {
	var targets []Target
	if base, ok := card.(*BaseCard); ok {
		targets = base.castTargets
	} else {
		targets = card.CastTargets()
	}
	for _, target := range targets {
		if nilDefinition(target) {
			continue
		}
		if target, ok := UnwrapTarget(target).(*AnyGraveyardCardTarget); ok && target != nil && target.graveyardEnchant {
			return target
		}
	}
	return nil
}

func (g *Game) graveyardEnchantHost(perm *Permanent) (Card, bool) {
	saved := perm.enchantedCard
	if saved.id == uuid.Nil || saved.visit == uuid.Nil || g.graveyardVisit(saved.id) != saved.visit || perm.HasType(core.TypeCreature) {
		return nil, false
	}
	target := graveyardEnchantTarget(perm.Card)
	if target == nil {
		return nil, false
	}
	for _, player := range g.players {
		for _, card := range player.Graveyard() {
			if card.ID() == saved.id {
				for _, filter := range target.Filters {
					if !filter.Match(card) {
						return nil, false
					}
				}
				return card, true
			}
		}
	}
	return nil, false
}

func (g *Game) prepareGraveyardEnchant(perm *Permanent, context battlefieldEntryContext) bool {
	target := graveyardEnchantTarget(perm.Card)
	if !perm.HasSubType("Aura") || target == nil {
		return true
	}
	if perm.HasType(core.TypeCreature) {
		return false
	}
	if context.auraSpell {
		perm.enchantedCard = context.graveyardHost
	} else {
		ids := target.Possible(perm.ControllerID(), perm.Card, g)
		if len(ids) == 0 {
			return false
		}
		cards := make([]Card, 0, len(ids))
		saved := make(map[uuid.UUID]capturedCard, len(ids))
		for _, id := range ids {
			if id == perm.ID() {
				continue
			}
			cards = append(cards, g.FindCardAnywhere(id))
			saved[id] = capturedCard{id: id, visit: g.graveyardVisit(id), zone: core.ZoneGraveyard}
		}
		if len(cards) == 0 {
			return false
		}
		chooser := g.GetPlayer(perm.ControllerID())
		if chooser == nil {
			return false
		}
		chosen := chooser.ChooseCardFromGraveyard(cards, "Choose what the entering Aura enchants", g)
		if chosen == nil {
			chosen = cards[0]
		}
		subject, offered := saved[chosen.ID()]
		if !offered {
			subject = saved[cards[0].ID()]
		}
		perm.enchantedCard = subject
	}
	if _, ok := g.graveyardEnchantHost(perm); !ok {
		return false
	}
	perm.AttachedTo = perm.enchantedCard.id
	perm.attachedIncarnation = perm.enchantedCard.visit
	return true
}

type enchantedGraveyardCardRef struct{ aura PermanentRef }

// EnchantedGraveyardCard reads the exact graveyard visit enchanted by an Aura.
func EnchantedGraveyardCard(aura PermanentRef) CardRef { return enchantedGraveyardCardRef{aura} }
func (enchantedGraveyardCardRef) cardRef()             {}
func (r enchantedGraveyardCardRef) resolveCard(ctx *EffectContext) (Card, bool) {
	host, ok := (permanentHostRef{r.aura}).informationIdentity(ctx)
	if !ok || host.incarnation == uuid.Nil || ctx.Game.graveyardVisit(host.id) != host.incarnation {
		return nil, false
	}
	for _, player := range ctx.Game.players {
		for _, card := range player.Graveyard() {
			if card.ID() == host.id {
				return card, true
			}
		}
	}
	return nil, false
}

func enchantedCardReadsSource(card CardRef) bool {
	ref, ok := card.(enchantedGraveyardCardRef)
	return ok && permanentRefReadsSourcePermanent(ref.aura)
}

func (e *returnCardToHandEffect) refersToSourcePermanent() bool {
	return enchantedCardReadsSource(e.card)
}

func (r enchantedGraveyardCardRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "card-ref", Text: "enchanted graveyard card", children: []definitionEdge{{"aura", r.aura}}}
}
