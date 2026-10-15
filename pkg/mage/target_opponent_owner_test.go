package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"
)

func TestTargetOpponentOwner(t *testing.T) {
	g, a, b := newSpellCopyGame()
	target := DeclarePermanentTarget("opponent owned", AnyPermanent, TargetOpponentOwns())
	card := NewInstant("Owner test", "", Spell(TapSubject(target), WithTargets(target)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	own := NewArtifact("Own but stolen", "")
	own.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(own, b.PlayerID())
	other := NewArtifact("Opponent but controlled", "")
	other.SetOwner(b.PlayerID())
	q := g.PutOnBattlefield(other, a.PlayerID())
	for range 2 {
		if got := target.Possible(a.PlayerID(), card, g); !slices.Equal(got, []uuid.UUID{q.ID()}) {
			t.Fatalf("owner candidates %v", got)
		}
		if got := target.Possible(b.PlayerID(), card, g); !slices.Equal(got, []uuid.UUID{p.ID()}) {
			t.Fatalf("reverse owner candidates %v", got)
		}
	}
	g.PhaseOut(q.ID())
	if len(target.Possible(a.PlayerID(), card, g)) != 0 {
		t.Fatal("phased target selected")
	}
}
