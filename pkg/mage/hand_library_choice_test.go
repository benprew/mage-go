package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type handLibraryPlayer struct {
	*BasePlayer
	pick   Card
	change func()
	calls  int
}

func (p *handLibraryPlayer) ChooseCardFromHand(cards []Card, _ string, _ GameReader) Card {
	p.calls++
	if p.change != nil {
		p.change()
	}
	return p.pick
}

func TestHandLibraryChoiceIdentity(t *testing.T) {
	for _, state := range []string{"chosen", "second", "nil", "foreign", "forged", "returned", "removed", "empty"} {
		t.Run(state, func(t *testing.T) {
			a := &handLibraryPlayer{BasePlayer: NewBasePlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			chosen, spare, top := NewLand("Chosen"), NewLand("Spare"), NewLand("Top")
			chosen.SetOwner(a.PlayerID())
			spare.SetOwner(a.PlayerID())
			top.SetOwner(a.PlayerID())
			a.SetLibrary([]Card{top})
			topVisit := a.libraryVisit(top.ID())
			if state != "empty" {
				a.AddToHand(chosen)
				a.AddToHand(spare)
			}
			a.pick = chosen
			switch state {
			case "second":
				a.pick = spare
			case "nil":
				a.pick = nil
			case "foreign":
				a.pick = NewLand("Foreign")
			case "forged":
				forged := NewLand("Forged")
				forged.SetID(chosen.ID())
				a.pick = forged
			case "returned":
				a.change = func() { a.RemoveFromHand(chosen.ID()); a.AddToHand(chosen) }
			case "removed":
				a.change = func() { a.RemoveFromHand(chosen.ID()); a.AddToGraveyard(chosen) }
			}
			zones, discards := 0, 0
			observer := NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
				if e.FromZone == core.ZoneHand && e.ToZone == core.ZoneLibrary {
					zones++
					if len(a.Library()) != 2 || a.Library()[0].ID() != e.SourceID || a.handVisit(e.SourceID) != uuid.Nil {
						t.Error("event observed incomplete move")
					}
				}
				return false
			})
			discard := NewTriggered(core.EvtDiscard, false).SetCondition(func(*core.GameEvent, GameReader, uuid.UUID, uuid.UUID) bool { discards++; return false })
			g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(observer), WithAbility(discard)), b.PlayerID())
			if err := ApplyEffect(g, PutChosenHandCardOnLibrary(You), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if state == "empty" {
				if a.calls != 0 || zones != 0 || len(a.Library()) != 1 {
					t.Fatal("empty hand made a choice")
				}
				return
			}
			want := chosen
			if state == "returned" || state == "removed" || state == "second" {
				want = spare
			}
			if a.calls != 1 || zones != 1 || a.Library()[0] != want || a.Library()[1] != top || a.libraryVisit(top.ID()) != topVisit || discards != 0 || len(g.PublicReveals()) != 0 || a.LastDrawnCardID() != uuid.Nil {
				t.Fatal("choice, identity, order or events", state)
			}
		})
	}
}

func TestHandLibraryChoicePlayerAndDeclarations(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewLand("Owner card")
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	c.SetOwner(b.PlayerID())
	clone := g.Clone()
	if err := ApplyEffect(g, PutChosenHandCardOnLibrary(You), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 0 || len(a.Library()) != 0 || len(b.Library()) != 1 || b.Library()[0].ID() != c.ID() || len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 {
		t.Fatal("owner destination or clone isolation")
	}
	for _, ref := range []PlayerRef{nil, FromPlayer(BindPlayer("unbound"))} {
		if ValidateCardDefinition("Bad", NewSorcery("Bad", "", Spell(PutChosenHandCardOnLibrary(ref)))) == nil {
			t.Fatal("invalid player accepted")
		}
	}
	if err := ApplyEffect(g, PutChosenHandCardOnLibrary(FromPlayer(BindPlayer("absent"))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
}

func TestHandLibraryChoiceReturnedOnlyAndWaiting(t *testing.T) {
	a := &handLibraryPlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewLand("Returned")
	a.AddToHand(card)
	a.pick = card
	a.change = func() { a.RemoveFromHand(card.ID()); a.AddToHand(card) }
	if err := ApplyEffect(g, PutChosenHandCardOnLibrary(You), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 1 || len(a.Library()) != 0 {
		t.Fatal("returned visit replaced original choice")
	}
	a.change = nil
	sourceCard := NewArtifact("Source", "")
	sourceCard.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(sourceCard, a.PlayerID())
	if err := ApplyEffect(g, AtNext(EachEndStep, PutChosenHandCardOnLibrary(ControllerOfWithLKI(This))), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	saved := g.CaptureDamageSource(source.ID())
	g.DestroyPermanent(source)
	g.PutOnBattlefield(source.Card, b.PlayerID())
	for range 2 {
		refs := g.WaitingDelayedPermanentSources()
		if len(refs) != 1 || !refs[0].SameObject(saved) || a.calls != 1 || len(a.Hand()) != 1 {
			t.Fatal("waiting query made choice or lost source identity")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if len(a.Hand()) != 0 || len(a.Library()) != 1 || len(b.Library()) != 0 {
		t.Fatal("delayed action lost source LKI player")
	}
}
