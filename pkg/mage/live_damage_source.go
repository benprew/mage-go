package mage

// LiveDamageSources reads the two live-object categories permitted by CR 609.7a.
// Referred objects and command-zone objects need a separate enumeration contract.
// Each result has separate card data, so a choice cannot change the game objects.
func (g *Game) LiveDamageSources() []DamageSource {
	sources := make([]DamageSource, 0, len(g.AllBattlefield())+g.stack.Size())
	for _, permanent := range g.AllBattlefield() {
		if !permanent.PhasedOut {
			sources = append(sources, damageSourceFromPermanent(permanent))
		}
	}
	for _, object := range g.stack.objects {
		if source := g.CaptureSpellDamageSource(object.ID); source.Valid() {
			sources = append(sources, source)
		}
	}
	return sources
}

// ChooseDamageSourceFromLiveObjects chooses from current permanents and spells.
// Use this only when the caller requires these categories. General CR 609.7a
// choices also permit referred objects and face-up command-zone objects.
func ChooseDamageSourceFromLiveObjects(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, liveObjects: true}
}
