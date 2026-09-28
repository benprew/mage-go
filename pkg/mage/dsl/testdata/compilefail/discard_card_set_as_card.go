package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DiscardCard(dsl.FromCards(dsl.BindCards("cards")))

// ERROR does not implement mage.CardRef
