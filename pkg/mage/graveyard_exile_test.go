package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestExileGraveyardCardVisit(t *testing.T) {
	for _, change := range []string{"none", "absent", "returned", "hand", "exile"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := newObjectColorGame()
			card := NewArtifact("Card", "")
			card.SetOwner(b.PlayerID())
			b.AddToGraveyard(card)
			binding := BindCard("saved card")
			g.resolution.cardBindings = make(map[*CardBinding]capturedCard)
			g.resolution.cardBindings[binding] = capturedCard{id: card.ID(), visit: g.graveyardVisit(card.ID()), zone: ZoneGraveyard}
			switch change {
			case "absent", "returned", "hand":
				moved, ok := g.MoveFromGraveyard(b.PlayerID(), card.ID(), ZoneHand)
				if !ok {
					t.Fatal("move failed")
				}
				if change == "returned" {
					b.AddToGraveyard(moved)
				} else {
					b.AddToHand(moved)
				}
				if change == "hand" {
					g.resolution.cardBindings[binding] = capturedCard{id: card.ID(), visit: b.handVisit(card.ID()), zone: ZoneHand}
				}
			case "exile":
				moved, _ := g.MoveFromGraveyard(b.PlayerID(), card.ID(), ZoneExile)
				g.ExileCard(moved, uuid.Nil)
				g.resolution.cardBindings[binding] = capturedCard{id: card.ID(), visit: g.FindExiledCard(card.ID()).Visit, zone: ZoneExile}
			}
			clone := g.Clone()
			source := uuid.New()
			if err := ExileGraveyardCard(FromCard(binding)).Apply(&EffectContext{Game: g, SourceID: source, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			exiled := g.FindExiledCard(card.ID())
			if (exiled != nil) != (change == "none" || change == "exile") {
				t.Fatal("wrong exile result")
			}
			if change == "none" {
				if exiled.Card.Owner() != b.PlayerID() || exiled.ExiledBy != source || len(b.Graveyard()) != 0 {
					t.Fatal("owner, attribution, or removal")
				}
				if clone.FindExiledCard(card.ID()) != nil || len(clone.GetPlayer(b.PlayerID()).Graveyard()) != 1 {
					t.Fatal("clone changed")
				}
				if err := ExileGraveyardCard(FromCard(binding)).Apply(&EffectContext{Game: clone, SourceID: source, Controller: a.PlayerID()}); err != nil {
					t.Fatal(err)
				}
				if clone.FindExiledCard(card.ID()) == nil {
					t.Fatal("clone lost binding")
				}
			}
			if change == "returned" && len(b.Graveyard()) != 1 {
				t.Fatal("followed new visit")
			}
			if change == "hand" && len(b.Hand()) != 1 {
				t.Fatal("moved a hand card")
			}
		})
	}
}

func TestExileGraveyardCardDeclarations(t *testing.T) {
	for _, ref := range []CardRef{nil, FromCard(nil), FromCard(BindCard("missing"))} {
		card := NewSorcery("Invalid", "", WithAbility(Spell(ExileGraveyardCard(ref))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid reference accepted")
		}
	}
	target := DeclareGraveyardCardTarget("card", InAnyGraveyard())
	card := NewSorcery("Valid", "", WithAbility(Spell(ExileGraveyardCard(target), WithTargets(target))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestExileGraveyardCardDepartureEvents(t *testing.T) {
	g, a, b := newObjectColorGame()
	count := BindNumber("departure count")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(OneOrMoreCardsLeaveYourGraveyard().WithCount(count), GainLifeFor(You, FromNumber(count)))))
	g.PutOnBattlefield(watcher, b.PlayerID())
	card := NewArtifact("Victim", "")
	card.SetOwner(b.PlayerID())
	b.AddToGraveyard(card)
	target := DeclareGraveyardCardTarget("card", InAnyGraveyard())
	spell := NewInstant("Exile", "", WithAbility(Spell(ExileGraveyardCard(target), WithTargets(target))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{card.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveTopOfStack()
	if g.FindExiledCard(card.ID()) == nil || b.Life() != 20 {
		t.Fatal("move or trigger timing")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("wrong departure event count")
	}
	g.ResolveStack()
	if b.Life() != 21 || a.Life() != 20 {
		t.Fatal("departure count or player attribution")
	}
}
