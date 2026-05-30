package jumpstart

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage"
	. "github.com/benprew/mage-go/pkg/mage/core"
)

var basicLandNames = map[string]bool{
	"Plains": true, "Island": true, "Swamp": true, "Mountain": true, "Forest": true,
}

func isBasicLand(c Card) bool { return basicLandNames[c.Name()] }

func isLandCard(c Card) bool { return c.HasType(TypeLand) }

func isCreatureCardLocal(c Card) bool { return c.HasType(TypeCreature) }

// searchBasicLandToBattlefieldTapped searches the controller's library for a
// basic land card and puts it onto the battlefield tapped, then shuffles.
func searchBasicLandToBattlefieldTapped(g *Game, controller uuid.UUID) {
	p := g.GetPlayer(controller)
	if p == nil {
		return
	}
	lib := p.Library()
	var candidates []Card
	for _, c := range lib {
		if isBasicLand(c) {
			candidates = append(candidates, c)
		}
	}
	if len(candidates) == 0 {
		p.ShuffleLibrary()
		return
	}
	chosen := p.ChooseCardFromLibrary(candidates, "search for basic land", g)
	if chosen == nil {
		p.ShuffleLibrary()
		return
	}
	newLib := make([]Card, 0, len(lib)-1)
	for _, c := range lib {
		if c.ID() != chosen.ID() {
			newLib = append(newLib, c)
		}
	}
	p.SetLibrary(newLib)
	p.ShuffleLibrary()
	perm := g.PutOnBattlefield(chosen, controller)
	if perm != nil {
		g.TapPermanent(perm)
	}
}

// searchLandToBattlefieldTapped searches for any land card.
func searchLandToBattlefieldTapped(g *Game, controller uuid.UUID, tapped bool) {
	p := g.GetPlayer(controller)
	if p == nil {
		return
	}
	lib := p.Library()
	var candidates []Card
	for _, c := range lib {
		if isLandCard(c) {
			candidates = append(candidates, c)
		}
	}
	if len(candidates) == 0 {
		p.ShuffleLibrary()
		return
	}
	chosen := p.ChooseCardFromLibrary(candidates, "search for land", g)
	if chosen == nil {
		p.ShuffleLibrary()
		return
	}
	newLib := make([]Card, 0, len(lib)-1)
	for _, c := range lib {
		if c.ID() != chosen.ID() {
			newLib = append(newLib, c)
		}
	}
	p.SetLibrary(newLib)
	p.ShuffleLibrary()
	perm := g.PutOnBattlefield(chosen, controller)
	if perm != nil && tapped {
		g.TapPermanent(perm)
	}
}

// searchBasicLandToHand searches for a basic land card and puts it into hand, optional reveal.
func searchBasicLandToHand(g *Game, controller uuid.UUID) {
	p := g.GetPlayer(controller)
	if p == nil {
		return
	}
	lib := p.Library()
	var candidates []Card
	for _, c := range lib {
		if isBasicLand(c) {
			candidates = append(candidates, c)
		}
	}
	if len(candidates) == 0 {
		p.ShuffleLibrary()
		return
	}
	chosen := p.ChooseCardFromLibrary(candidates, "search for basic land", g)
	if chosen == nil {
		p.ShuffleLibrary()
		return
	}
	newLib := make([]Card, 0, len(lib)-1)
	for _, c := range lib {
		if c.ID() != chosen.ID() {
			newLib = append(newLib, c)
		}
	}
	p.SetLibrary(newLib)
	p.ShuffleLibrary()
	p.AddToHand(chosen)
}

// fightTargets makes source and target deal damage equal to their power to each other.
func fightTargets(g *Game, source, target *Permanent) {
	if source == nil || target == nil {
		return
	}
	srcPower := source.CurrentPower(g)
	tgtPower := target.CurrentPower(g)
	if srcPower > 0 {
		g.DealDamageToPermanent(target, srcPower, source.ID())
	}
	if tgtPower > 0 {
		g.DealDamageToPermanent(source, tgtPower, target.ID())
	}
}
