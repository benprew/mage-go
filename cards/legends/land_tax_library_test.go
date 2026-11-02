package legends

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
)

type landTaxLibraryPlayer struct {
	*mage.BasePlayer
	choices []*mage.Card
}

func (p *landTaxLibraryPlayer) ChooseCardsFromLibrary(_ []*mage.Card, _ int, _ string, _ mage.GameReader) []*mage.Card {
	return p.choices
}

func TestLandTaxRejectsDuplicateAndAbsentLibraryCards(t *testing.T) {
	p := &landTaxLibraryPlayer{BasePlayer: mage.NewBasePlayer("Alice")}
	opp := mage.NewBasePlayer("Bob")
	g := mage.NewGame(p, opp)
	land := mage.NewLand("Plains")
	land.SetOwner(p.PlayerID())
	absent := mage.NewLand("Forest")
	absent.SetOwner(p.PlayerID())
	p.SetLibrary([]*mage.Card{land})
	p.choices = []*mage.Card{land, land, absent}
	opponentLand := mage.NewLand("Mountain")
	opponentLand.SetOwner(opp.PlayerID())
	g.PutOnBattlefield(opponentLand, opp.PlayerID())
	tax, err := mage.CreateCard("Land Tax")
	if err != nil {
		t.Fatal(err)
	}
	tax.SetOwner(p.PlayerID())
	for _, ability := range tax.AllAbilities() {
		trigger, ok := ability.(mage.TriggeredAbility)
		if !ok {
			continue
		}
		for _, effect := range trigger.Effects() {
			if err := effect.Apply(&mage.EffectContext{Game: g, SourceID: tax.ID(), Controller: p.PlayerID()}); err != nil {
				t.Fatal(err)
			}
		}
		if len(p.Hand()) != 1 || p.Hand()[0] != land || len(p.Library()) != 0 {
			t.Fatal("Land Tax must move each library card at most once")
		}
		return
	}
	t.Fatal("Land Tax has no triggered ability")
}
