package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type handInspectionPlayer struct {
	*BasePlayer
	observed []Card
}

func (p *handInspectionPlayer) ObserveHandLook(cards []Card, _ GameReader) { p.observed = cards }

func TestHandInspectionPrivacyAndVisits(t *testing.T) {
	a := &handInspectionPlayer{BasePlayer: NewBasePlayer("A")}
	b := &handInspectionPlayer{BasePlayer: NewBasePlayer("B")}
	g := NewGame(a, b)
	g.SetActivePlayerIndex(1)
	c := NewLand("Private")
	c.SetOwner(b.PlayerID())
	b.AddToHand(c)
	visit := g.handVisit(c.ID())
	if err := ApplyEffect(g, LookAtHand(You, ActivePlayer), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.observed) != 1 || len(b.observed) != 0 || len(g.HandLooks(a.PlayerID())) != 1 || len(g.HandLooks(b.PlayerID())) != 0 || len(g.PublicReveals()) != 0 || g.IsHandRevealed(b.PlayerID()) {
		t.Fatal("inspection was not private")
	}
	a.observed[0].SetOwner(a.PlayerID())
	seen := g.HandLooks(a.PlayerID())
	seen[0].Card.SetOwner(a.PlayerID())
	seen[0].Types[0] = core.TypeCreature
	if c.Owner() != b.PlayerID() || g.handVisit(c.ID()) != visit || g.HandLooks(a.PlayerID())[0].Types[0] != core.TypeLand || g.HandLooks(a.PlayerID())[0].Card.Owner() != b.PlayerID() {
		t.Fatal("inspection changed the hand or shared snapshots")
	}
	clone := g.Clone()
	b.RemoveFromHand(c.ID())
	b.AddToHand(NewLand("Later"))
	if err := ApplyEffect(g, LookAtHand(You, ActivePlayer), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(clone.HandLooks(a.PlayerID())) != 1 || clone.HandLooks(a.PlayerID())[0].Name != "Private" || g.HandLooks(a.PlayerID())[1].Name != "Later" {
		t.Fatal("history followed current cards or shared clone storage")
	}
	if err := ApplyEffect(clone, LookAtHand(You, ActivePlayer), uuid.Nil, b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.HandLooks(b.PlayerID())) != 0 {
		t.Fatal("clone changed original history")
	}
}

func TestHandInspectionInvalidAndMissingPlayers(t *testing.T) {
	for _, effect := range []Effect{LookAtHand(nil, You), LookAtHand(You, nil), LookAtHand(You, FromPlayer(BindPlayer("missing"))), LookAtHand(You, DeclarePlayerTarget("missing"))} {
		c := NewInstant("Invalid", "", WithAbility(Spell(effect)))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("invalid inspection declaration accepted")
		}
	}
	a := &handInspectionPlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	for _, effect := range []Effect{LookAtHand(You, ActivePlayer), LookAtHand(ControllerOf(This), You), LookAtHand(You, ControllerOf(This))} {
		if err := ApplyEffect(g, effect, uuid.New(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(a.observed) != 0 || len(g.HandLooks(a.PlayerID())) != 0 {
		t.Fatal("empty or missing hand caused inspection")
	}
}
