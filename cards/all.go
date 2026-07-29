// Package cards imports all card set packages to register every card.
// Import this package for its side effects:
//
//	import _ "github.com/benprew/mage-go/cards"
package cards

import (
	// Register every card set through its init function.
	_ "github.com/benprew/mage-go/cards/antiquities"
	_ "github.com/benprew/mage-go/cards/arabian"
	_ "github.com/benprew/mage-go/cards/custom"
	_ "github.com/benprew/mage-go/cards/fallen_empires"
	_ "github.com/benprew/mage-go/cards/fourthedition"
	_ "github.com/benprew/mage-go/cards/legends"
	_ "github.com/benprew/mage-go/cards/limited"
)
