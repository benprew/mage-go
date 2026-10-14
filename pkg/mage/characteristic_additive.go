package mage

// PTEqualsCountPlus adds a fixed value to a battlefield count in layer 7a.
// CardPowerToughness uses the same declaration in other zones.
func PTEqualsCountPlus(filter PermanentFilter, offset int) ContinuousEffect {
	return &countCharacteristicPT{filter: filter, offset: offset, power: true, toughness: true}
}

// PTEqualsControlledCountPlus counts the source controller's matching permanents.
// A card without a controller uses its owner.
func PTEqualsControlledCountPlus(filter PermanentFilter, offset int) ContinuousEffect {
	return &countCharacteristicPT{filter: filter, offset: offset, controlled: true, power: true, toughness: true}
}

// PTEqualsOpponentCountPlus counts the source controller's opponent's permanents.
// An absent controller supplies no count. The fixed value still applies.
func PTEqualsOpponentCountPlus(filter PermanentFilter, offset int) ContinuousEffect {
	return &countCharacteristicPT{filter: filter, offset: offset, opponents: true, power: true, toughness: true}
}
