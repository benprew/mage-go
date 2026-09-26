package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.OpponentDiscardsCard(dsl.BindPermanent("discarder")) // ERROR *mage.PlayerBinding
