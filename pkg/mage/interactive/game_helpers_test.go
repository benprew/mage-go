package interactive

import (
	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

// makePerm creates a creature permanent with cleared summoning sickness.
func makePerm(name, cost string, power, toughness int, owner uuid.UUID, opts ...mage.CardOption) *mage.Permanent {
	card := mage.NewCreature(name, cost, power, toughness, opts...)
	card.SetOwner(owner)
	perm := mage.NewPermanent(card, owner)
	perm.RevokeBaseAttr(core.AttrSummonSick)
	return perm
}

// makeGame creates a two-player game with BasePlayer instances.
func makeGame() (*mage.Game, *mage.BasePlayer, *mage.BasePlayer) {
	pa := mage.NewBasePlayer("Alice")
	pb := mage.NewBasePlayer("Bob")
	g := mage.NewGame(pa, pb)
	return g, pa, pb
}
