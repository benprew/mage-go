package mage

// isolateCards preserves references to the same instance within a clone.
// Legacy cards retain their existing copy-on-write contract.
func (g *Game) isolateCards() {
	instances := make(map[*Card]*Card)
	abilities := make(map[Ability]Ability)
	cloneCard := func(card *Card) *Card {
		instance := card
		if instance == nil || instance.definition == nil {
			return card
		}
		if cloned, found := instances[instance]; found {
			return cloned
		}
		cloned := instance.CloneInstance()
		instances[instance] = cloned
		for i, ability := range instance.abilities {
			abilities[ability] = cloned.abilities[i]
		}
		return cloned
	}
	cloneCards := func(cards []*Card) []*Card {
		var result []*Card
		for i, card := range cards {
			cloned := cloneCard(card)
			if cloned != card {
				if result == nil {
					result = append([]*Card(nil), cards...)
				}
				result[i] = cloned
			}
		}
		if result != nil {
			return result
		}
		return cards
	}
	for _, player := range g.players {
		bp := player.(*SearchPlayer).BasePlayer
		bp.hand = cloneCards(bp.hand)
		bp.library = cloneCards(bp.library)
		bp.graveyard = cloneCards(bp.graveyard)
		bp.ante = cloneCards(bp.ante)
	}
	for i, exiled := range g.zones.exile {
		g.zones.exile[i].Card = cloneCard(exiled.Card)
	}
	permanents := make(map[*Permanent]*Permanent)
	clonePermanent := func(permanent *Permanent) *Permanent {
		if permanent == nil {
			return nil
		}
		if cloned, ok := permanents[permanent]; ok {
			return cloned
		}
		card := cloneCard(permanent.Card)
		if card == permanent.Card {
			return permanent
		}
		cloned := &Permanent{}
		clonePermanentInto(cloned, permanent)
		cloned.Card = card
		cloned.RuntimeAbilities = make([]Ability, len(permanent.RuntimeAbilities))
		for i, ability := range permanent.RuntimeAbilities {
			mapped, ok := abilities[ability]
			if !ok {
				mapped = cloneInstanceAbility(ability)
				abilities[ability] = mapped
			}
			cloned.RuntimeAbilities[i] = mapped
		}
		permanents[permanent] = cloned
		return cloned
	}
	for i, permanent := range g.zones.battlefield {
		cloned := clonePermanent(permanent)
		if cloned != permanent {
			g.zones.ensureBattlefieldSliceOwned()
			g.zones.battlefield[i] = cloned
			g.zones.addOwnedPermanent(cloned)
		}
	}
	g.zones.enteringPermanent = clonePermanent(g.zones.enteringPermanent)
	cloneContext := func(context *CastContext) *CastContext {
		if context == nil {
			return nil
		}
		cloned := cloneCastContext(context)
		cloned.RevealedAtCast = cloneCards(context.RevealedAtCast)
		return cloned
	}
	for _, object := range g.stack.objects {
		object.Card = cloneCard(object.Card)
		object.TargetSource = cloneCard(object.TargetSource)
		modernCard := object.Card != nil && object.Card.definition != nil
		modernSource := object.TargetSource != nil && object.TargetSource.definition != nil
		if modernCard || modernSource {
			object.TargetSpecs = cloneInstanceTargets(object.TargetSpecs)
		}
		object.CastContext = cloneContext(object.CastContext)
	}
	g.resolution.resolvingCard = cloneCard(g.resolution.resolvingCard)
	g.resolution.lastCostReveal = cloneCard(g.resolution.lastCostReveal)
	g.resolution.lastExiledCard = cloneCard(g.resolution.lastExiledCard)
	g.resolution.resolvingCastContext = cloneContext(g.resolution.resolvingCastContext)
	for _, trigger := range g.triggers.pending {
		if mapped, ok := abilities[trigger.ability]; ok {
			trigger.ability = mapped.(TriggeredAbility)
		}
	}
}
