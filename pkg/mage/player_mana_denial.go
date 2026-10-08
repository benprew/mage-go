package mage

import . "github.com/benprew/mage-go/pkg/mage/core"

type loseUnspentManaEffect struct{ player PlayerRef }

func (e *loseUnspentManaEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player)
}

// LoseUnspentManaFor removes all unspent mana from the selected player's pool.
func LoseUnspentManaFor(player PlayerRef) ResolvingEffect {
	return &loseUnspentManaEffect{player: player}
}
func (*loseUnspentManaEffect) Text() string { return "lose all unspent mana" }
func (*loseUnspentManaEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *loseUnspentManaEffect) Apply(ctx *EffectContext) error {
	id, ok := e.player.resolvePlayer(ctx)
	if ok {
		ctx.Game.GetPlayer(id).ManaPool().Clear()
	}
	return nil
}
func (e *loseUnspentManaEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "lose-unspent-mana", Text: e.Text(), children: []definitionEdge{{"player", e.player}}}
}

func isTriggeredManaAbility(ability *GenericTriggered) bool {
	return ability.eventType == EvtTappedForMana && len(ability.Modes()) == 0 && canProduceMana(ability.Effects(), ability.Targets())
}

// HasManaAbility checks current abilities without checking payment or output.
var HasManaAbility = NewPermanentFilter("has a mana ability", func(p *Permanent, _ *Game) bool {
	for _, ability := range p.RuntimeAbilities {
		if IsActivatedManaAbility(ability) {
			return true
		}
		if triggered, ok := UnwrapAbility(ability).(*GenericTriggered); ok && isTriggeredManaAbility(triggered) {
			return true
		}
	}
	return false
})
