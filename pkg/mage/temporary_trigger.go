package mage

// HasRampage reads current abilities, including temporary grants.
var HasRampage = NewPermanentFilter("has rampage", func(p *Permanent, _ *Game) bool {
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
})
