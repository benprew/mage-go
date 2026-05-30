package heuristic

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
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

// landColor maps basic land names to their mana color.
var landColor = map[string]core.Color{
	"Forest":   core.Green,
	"Mountain": core.Red,
	"Plains":   core.White,
	"Island":   core.Blue,
	"Swamp":    core.Black,
}

func addLands(g *mage.Game, p *mage.BasePlayer, name string, count int) {
	color := landColor[name]
	for range count {
		land := mage.NewLand(name, mage.WithManaAbility(color))
		land.SetOwner(p.PlayerID())
		perm := mage.NewPermanent(land, p.PlayerID())
		perm.RevokeBaseAttr(core.AttrSummonSick)
		g.AddToBattlefield(perm)
	}
}
