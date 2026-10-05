package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type librarySearchPlayer struct {
	*BasePlayer
	pick       Card
	candidates []Card
	shuffles   int
	onShuffle  func()
}

func (p *librarySearchPlayer) ChooseCardFromLibrary(candidates []Card, _ string, _ GameReader) Card {
	p.candidates = append([]Card(nil), candidates...)
	return p.pick
}

func (p *librarySearchPlayer) ShuffleLibrary() {
	p.shuffles++
	if p.onShuffle != nil {
		p.onShuffle()
	}
	p.BasePlayer.ShuffleLibrary()
}

func TestLibrarySearchRequiredFindAndOrder(t *testing.T) {
	for _, destination := range []string{"hand", "battlefield"} {
		for _, choice := range []string{"valid", "decline", "foreign", "empty"} {
			t.Run(destination+"/"+choice, func(t *testing.T) {
				a := &librarySearchPlayer{BasePlayer: NewBasePlayer("A")}
				b := NewBasePlayer("B")
				g := NewGame(a, b)
				card := NewLand("Found", WithSuperTypes(core.SuperBasic))
				card.SetOwner(a.PlayerID())
				if choice != "empty" {
					a.AddToLibrary(card)
				}
				if choice == "valid" {
					a.pick = card
				}
				if choice == "foreign" {
					a.pick = NewLand("Foreign")
				}
				a.onShuffle = func() {
					want := 1
					if choice == "empty" {
						want = 0
					}
					if len(a.Library()) != 0 {
						t.Error("found card still in library at shuffle")
					}
					if destination == "hand" && len(a.Hand()) != want {
						t.Error("shuffle ran before hand movement")
					}
					if destination == "battlefield" && (g.FindPermanent(card.ID()) != nil) != (want == 1) {
						t.Error("shuffle ran before entry")
					}
				}
				effect := SearchLibraryToHand()
				if destination == "battlefield" {
					effect = SearchLibraryToBattlefield(CardFilter{})
				}
				ApplyEffect(g, effect, card.ID(), a.PlayerID(), nil)
				if a.shuffles != 1 {
					t.Fatalf("shuffles=%d", a.shuffles)
				}
				if len(g.PublicReveals()) != 0 || len(b.Library()) != 0 {
					t.Fatal("search revealed or changed another library")
				}
			})
		}
	}
}

func TestLibrarySearchQualifiedCanFail(t *testing.T) {
	for _, choice := range []string{"valid", "decline", "foreign", "no-match"} {
		t.Run(choice, func(t *testing.T) {
			a := &librarySearchPlayer{BasePlayer: NewBasePlayer("A")}
			g := NewGame(a, NewBasePlayer("B"))
			land := NewLand("Land")
			land.SetOwner(a.PlayerID())
			other := NewCreature("Other", "", 1, 1)
			a.AddToLibrary(other)
			if choice != "no-match" {
				a.AddToLibrary(land)
			}
			if choice == "valid" {
				a.pick = land
			}
			if choice == "foreign" {
				a.pick = other
			}
			filter := NewCardFilter("land", func(c Card) bool { return c.HasType(core.TypeLand) })
			ApplyEffect(g, SearchLibraryToBattlefield(filter), land.ID(), a.PlayerID(), nil)
			if (g.FindPermanent(land.ID()) != nil) != (choice == "valid") {
				t.Fatal("wrong search result")
			}
			if a.shuffles != 1 || g.FindPermanent(other.ID()) != nil {
				t.Fatal("invalid candidate or shuffle")
			}
		})
	}
}

func TestLibrarySearchProhibitedEntryKeepsCard(t *testing.T) {
	a := &librarySearchPlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	land := NewLand("Found")
	land.SetOwner(a.PlayerID())
	a.AddToLibrary(land)
	a.pick = land
	g.PutOnBattlefield(NewEnchantment("Entry prohibition", "", WithStaticAbility(WormsOfTheEarthEffect())), a.PlayerID())
	ApplyEffect(g, SearchLibraryToBattlefield(NewCardFilter("land", func(c Card) bool { return c.HasType(core.TypeLand) })), land.ID(), a.PlayerID(), nil)
	if g.FindPermanent(land.ID()) != nil || len(a.Library()) != 1 || a.Library()[0].ID() != land.ID() || a.shuffles != 1 {
		t.Fatal("prohibited entry removed the found card")
	}
}
