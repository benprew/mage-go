package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"
)

func TestRemoveFromLibrary(t *testing.T) {
	for _, index := range []int{0, 1, 2} {
		t.Run([]string{"top", "middle", "bottom"}[index], func(t *testing.T) {
			p := NewBasePlayer("Alice")
			cards := []*Card{NewLand("First"), NewLand("Second"), NewLand("Third")}
			p.SetLibrary(cards)
			var player Player = p
			got, ok := player.RemoveFromLibrary(cards[index].ID())
			if !ok || got != cards[index] {
				t.Fatalf("removal = (%p, %v), want (%p, true)", got, ok, cards[index])
			}
			want := slices.Delete(slices.Clone(cards), index, index+1)
			if !slices.Equal(p.Library(), want) {
				t.Fatal("removal changed the order of the remaining cards")
			}
			if len(cards) != 3 || cards[0].Name() != "First" || cards[1].Name() != "Second" || cards[2].Name() != "Third" {
				t.Fatal("removal changed shared library storage")
			}
			if got, ok := player.RemoveFromLibrary(cards[index].ID()); ok || got != nil {
				t.Fatal("a second removal must fail")
			}
		})
	}
}

func TestRemoveFromLibraryMissing(t *testing.T) {
	for _, cards := range [][]*Card{nil, {}, {NewLand("Only")}} {
		p := NewBasePlayer("Alice")
		p.SetLibrary(cards)
		if got, ok := p.RemoveFromLibrary(uuid.New()); ok || got != nil {
			t.Fatal("removal of a missing card must fail")
		}
		if !slices.Equal(p.Library(), cards) {
			t.Fatal("failed removal changed the library")
		}
	}
}

func TestRemoveFromLibraryLastCard(t *testing.T) {
	p := NewBasePlayer("Alice")
	card := NewLand("Only")
	p.SetLibrary([]*Card{card})
	if got, ok := p.RemoveFromLibrary(card.ID()); !ok || got != card || len(p.Library()) != 0 {
		t.Fatal("removal must return the last card and leave an empty library")
	}
}

type librarySearchPlayer struct {
	*BasePlayer
	choice *Card
}

func (p *librarySearchPlayer) ChooseCardFromLibrary(_ []*Card, _ string, _ GameReader) *Card {
	return p.choice
}

func TestLibrarySearchRequiresStoredCard(t *testing.T) {
	for _, destination := range []string{"hand", "top", "battlefield"} {
		for _, present := range []bool{false, true} {
			t.Run(destination+map[bool]string{false: "/absent", true: "/present"}[present], func(t *testing.T) {
				p := &librarySearchPlayer{BasePlayer: NewBasePlayer("Alice")}
				g := NewGame(p, NewBasePlayer("Bob"))
				stored := NewLand("Plains")
				stored.SetOwner(p.PlayerID())
				other := NewLand("Forest")
				other.SetOwner(p.PlayerID())
				p.SetLibrary([]*Card{other, stored})
				p.choice = stored.CloneInstance()
				if !present {
					p.choice = NewLand("Absent")
					p.choice.SetOwner(p.PlayerID())
				}
				var effect Effect
				switch destination {
				case "hand":
					effect = SearchLibraryToHand()
				case "top":
					effect = SearchLibraryToTop()
				case "battlefield":
					effect = SearchLibraryToBattlefield(CardFilter{})
				}
				if err := effect.Apply(&EffectContext{Game: g, Controller: p.PlayerID()}); err != nil {
					t.Fatal(err)
				}
				if !present {
					if len(p.Library()) != 2 || len(p.Hand()) != 0 || g.FindPermanent(p.choice.ID()) != nil {
						t.Fatal("an absent selection must not move a card")
					}
					return
				}
				switch destination {
				case "hand":
					if len(p.Library()) != 1 || len(p.Hand()) != 1 || p.Hand()[0] != stored {
						t.Fatal("search must move the stored card to hand")
					}
				case "top":
					if !slices.Equal(p.Library(), []*Card{stored, other}) {
						t.Fatal("search must put the stored card on top")
					}
				case "battlefield":
					perm := g.FindPermanent(stored.ID())
					if len(p.Library()) != 1 || perm == nil || perm.Card != stored {
						t.Fatal("search must move the stored card to the battlefield")
					}
				}
			})
		}
	}
}
