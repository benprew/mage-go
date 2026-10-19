package mage

// HasRampage reads current abilities, including temporary grants.
var HasRampage = PermanentFilter{label: "has rampage", query: &permanentCollectionFilterNode{operation: collectionFilterRampage}, fn: permanentHasRampage}

func permanentHasRampage(p *Permanent, _ *Game) bool {
	for _, ability := range p.RuntimeAbilities {
		trigger, ok := UnwrapAbility(ability).(*GenericTriggered)
		if !ok {
			continue
		}
		for _, effect := range trigger.Effects() {
			if _, ok := effect.(*RampageEffectData); ok {
				return true
			}
		}
	}
	return false
}
