package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLibrarySearchInstructionDeclaration(t *testing.T) {
	for _, kind := range []LibrarySearchKind{SearchAnyCardToHand, SearchBasicLandToBattlefield} {
		effect := (LibrarySearchInstruction{Kind: kind}).Effect()
		got, ok := InspectLibrarySearchInstruction(effect)
		if !ok || got.Kind != kind {
			t.Fatal("lost search declaration")
		}
	}
	for _, kind := range []LibrarySearchKind{0, 255} {
		if (LibrarySearchInstruction{Kind: kind}).Effect() != nil {
			t.Fatal("accepted invalid search")
		}
	}
	for _, effect := range []Effect{nil, (*librarySearchInstructionEffect)(nil), &librarySearchInstructionEffect{}, &librarySearchInstructionEffect{kind: 255}, (*searchLibraryEffect)(nil), SearchLibraryToBattlefield(NewCardFilter("opaque", func(Card) bool { panic("callback") })), Sequence(SearchLibraryToHand(), destructionInspectionTrap{})} {
		if _, ok := InspectLibrarySearchInstruction(effect); ok {
			t.Fatal("accepted incomplete search")
		}
	}
}

func TestLibrarySearchInstructionOrder(t *testing.T) {
	for _, kind := range []LibrarySearchKind{SearchAnyCardToHand, SearchBasicLandToBattlefield} {
		for _, state := range []string{"alone", "prefix", "suffix", "unknown suffix", "clone", "copy", "all illegal"} {
			t.Run(string(rune('A'+kind))+"/"+state, func(t *testing.T) {
				g, p, obj, target := targetQueryFixture(t)
				search := (LibrarySearchInstruction{Kind: kind}).Effect()
				destroy := Destroy(target)
				obj.Effects = []Effect{search}
				want, known := false, true
				switch state {
				case "prefix":
					obj.Effects = []Effect{Sequence(search, destroy)}
					known = false
				case "suffix", "clone", "copy":
					obj.Effects = []Effect{Sequence(destroy, search)}
					want = true
				case "unknown suffix":
					obj.Effects = []Effect{Sequence(destroy, search, destructionInspectionTrap{})}
					known = false
				case "all illegal":
					p.GrantBaseAttr(core.Shroud)
					obj.Effects = []Effect{Sequence(search, destroy)}
				}
				if state == "clone" {
					g = g.Clone()
					obj = g.StackPeek()
				}
				if state == "copy" {
					obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
				}
				for range 2 {
					got, k := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}})
					if got != want || k != known {
						t.Fatalf("certainty=%v/%v want=%v/%v", got, k, want, known)
					}
				}
				if g.FindPermanent(p.ID()) == nil {
					t.Fatal("inspection destroyed permanent")
				}
			})
		}
	}
}

type librarySearchInspectionTrapPlayer struct{ *BasePlayer }

func (*librarySearchInspectionTrapPlayer) Library() []Card { panic("inspection read hidden library") }
func (*librarySearchInspectionTrapPlayer) ShuffleLibrary() { panic("inspection shuffled library") }
func (*librarySearchInspectionTrapPlayer) ChooseCardFromLibrary([]Card, string, GameReader) Card {
	panic("inspection made a choice")
}

func TestLibrarySearchInstructionPurity(t *testing.T) {
	for _, kind := range []LibrarySearchKind{SearchAnyCardToHand, SearchBasicLandToBattlefield} {
		g, p, obj, target := targetQueryFixture(t)
		controller := g.GetPlayer(obj.Controller).(*BasePlayer)
		for i, player := range g.players {
			if player.PlayerID() == controller.PlayerID() {
				g.players[i] = &librarySearchInspectionTrapPlayer{controller}
			}
		}
		search := (LibrarySearchInstruction{Kind: kind}).Effect()
		obj.Effects = []Effect{Sequence(search, Destroy(target))}
		if certain, known := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}}); certain || known {
			t.Fatal("hidden search prefix supplied a proof")
		}
		obj.Effects = []Effect{Sequence(search)}
		if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
			t.Fatal("search alone is unknown")
		}
	}
}
