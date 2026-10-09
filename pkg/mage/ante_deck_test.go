package mage

import (
	"slices"
	"testing"
)

func TestAnteDeckPreparation(t *testing.T) {
	restricted := NewArtifact("Restricted", "", WithAnteDeckRestriction())
	ordinary := NewArtifact("Ordinary", "")
	cards := []Card{restricted, ordinary, restricted.Copy()}
	for _, enabled := range []bool{false, true} {
		kept, removed := PrepareCardsForAnte(cards, enabled)
		wantKept, wantRemoved := 1, 2
		if enabled {
			wantKept, wantRemoved = 3, 0
		}
		if len(kept) != wantKept || len(removed) != wantRemoved {
			t.Fatalf("enabled=%v: kept=%d removed=%d", enabled, len(kept), len(removed))
		}
		if !enabled && (kept[0] != ordinary || removed[0] != restricted || removed[1] != cards[2]) {
			t.Fatal("card order or identity changed")
		}
		kept[0] = ordinary.Copy()
		if cards[0] != restricted {
			t.Fatal("result shares input storage")
		}
	}
	if !restricted.RequiresAnte() || !restricted.Copy().(*BaseCard).RequiresAnte() || ordinary.RequiresAnte() {
		t.Fatal("restriction did not survive physical copy")
	}
	restricted.CloneFrom(ordinary)
	if !restricted.RequiresAnte() {
		t.Fatal("rule replacement changed physical deck eligibility")
	}
	kept, removed := PrepareCardsForAnte(nil, false)
	if len(kept) != 0 || len(removed) != 0 {
		t.Fatal("empty deck changed")
	}
}

func TestAnteDeckGameSetup(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		restricted := NewArtifact("Restricted", "", WithAnteDeckRestriction())
		ordinary := NewArtifact("Ordinary", "")
		restricted.SetOwner(a.PlayerID())
		ordinary.SetOwner(a.PlayerID())
		a.SetLibrary([]Card{restricted, ordinary})
		other := restricted.Copy()
		other.SetOwner(b.PlayerID())
		b.SetLibrary([]Card{other})
		var g *Game
		if enabled {
			var err error
			g, err = NewGameWithAnte(a, b, []Card{restricted}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Contains(a.Ante(), Card(restricted)) || len(b.Library()) != 1 {
				t.Fatal("ante setup excluded eligible cards")
			}
		} else {
			g = NewGame(a, b)
			if len(a.Library()) != 1 || a.Library()[0] != ordinary || len(b.Library()) != 0 {
				t.Fatal("non-ante setup retained restricted cards")
			}
			if g.FindCardAnywhere(restricted.ID()) != nil {
				t.Fatal("excluded card entered a game zone")
			}
		}
		a.DrawCard()
		if len(a.Hand()) != 1 || a.Hand()[0] != ordinary {
			t.Fatal("setup did not precede draw")
		}
		clone := g.Clone()
		if clone.AnteEnabled() != enabled || len(clone.GetPlayer(b.PlayerID()).Library()) != len(b.Library()) {
			t.Fatal("clone changed prepared deck")
		}
	}
}

func TestAnteDeckEntryCopyKeepsPhysicalRestriction(t *testing.T) {
	for _, restricted := range []bool{false, true} {
		physicalPlan, err := NewCardDefinitionPlan(func() Card {
			var options []CardOption
			if restricted {
				options = append(options, WithAnteDeckRestriction())
			}
			return NewArtifact("Physical", "", options...)
		})
		if err != nil {
			t.Fatal(err)
		}
		copiedPlan, err := NewCardDefinitionPlan(func() Card {
			var options []CardOption
			if !restricted {
				options = append(options, WithAnteDeckRestriction())
			}
			return NewArtifact("Copied", "", options...)
		})
		if err != nil {
			t.Fatal(err)
		}
		physical, err := physicalPlan.Build()
		if err != nil {
			t.Fatal(err)
		}
		result, err := BuildEntryCopy(physical, copiedPlan.CopiableValues())
		if err != nil {
			t.Fatal(err)
		}
		if result.(*BaseCard).RequiresAnte() != restricted {
			t.Fatal("entry copy replaced physical deck restriction")
		}
	}
}

func TestAnteDeckSideboardPreparation(t *testing.T) {
	restricted := NewArtifact("Restricted", "", WithAnteDeckRestriction())
	ordinary := NewArtifact("Ordinary", "")
	sideboard := []Card{ordinary, restricted, ordinary.Copy(), restricted.Copy()}
	kept, removed := PrepareCardsForAnte(sideboard, false)
	if len(kept) != 2 || len(removed) != 2 || kept[0] != ordinary || kept[1] != sideboard[2] || removed[0] != restricted || removed[1] != sideboard[3] {
		t.Fatal("sideboard filtering changed membership or order")
	}
	removed[0] = ordinary
	if sideboard[1] != restricted {
		t.Fatal("excluded slice changed input")
	}
	if restricted.Owner() != sideboard[3].Owner() {
		t.Fatal("preparation changed ownership")
	}
}
