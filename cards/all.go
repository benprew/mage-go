// Package cards imports all card set packages to register every card.
// Import this package for its side effects:
//
//	import _ "github.com/mage/mage/cards"
package cards

import (
	_ "github.com/mage/mage/cards/antiquities"
	_ "github.com/mage/mage/cards/arabian"
	_ "github.com/mage/mage/cards/custom"
	_ "github.com/mage/mage/cards/legends"
	_ "github.com/mage/mage/cards/limited"
)
