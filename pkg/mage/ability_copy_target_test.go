package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestAbilityCopyRetargetsStoredSpecs(t *testing.T) {
	for _, withCard := range []bool{false, true} {
		for _, change := range []bool{false, true} {
			g, a, b := newSpellCopyGame()
			target := DeclarePlayerTarget("player")
			ability := Activate(Costs(), Mill(target, Fixed(2)), WithTargets(target))
			source := NewCreature("Mill source", "", 1, 1, WithAbility(ability))
			original := &StackObject{ID: uuid.New(), SourceID: source.ID(), TargetSource: source, Controller: a.PlayerID(), IsAbility: true, Effects: ability.Effects(), Targets: []uuid.UUID{b.PlayerID()}, TargetSpecs: ability.Targets()}
			if withCard {
				original.Card = source
			}
			g.pushStack(original)
			b.chooseQueue = [][]uuid.UUID{{a.PlayerID()}}
			cp := g.CopyStackObjectDirect(original, b.PlayerID(), change)
			want := b.PlayerID()
			if change {
				want = a.PlayerID()
			}
			if cp.Targets[0] != want || original.Targets[0] != b.PlayerID() {
				t.Fatalf("withCard=%v change=%v: wrong targets", withCard, change)
			}
			if cp.SourceID != original.SourceID {
				t.Fatal("ability copy changed source")
			}
			g.ResolveTopOfStack()
			if len(g.GetPlayer(want).Graveyard()) != 2 {
				t.Fatal("copy lost target binding")
			}
		}
	}
}

func TestAbilityCopyRejectsInvalidNewTarget(t *testing.T) {
	for _, choice := range [][]uuid.UUID{nil, {uuid.New()}} {
		g, a, b := newSpellCopyGame()
		target := DeclarePlayerTarget("player")
		ability := Activate(Costs(), Mill(target, Fixed(2)), WithTargets(target))
		source := NewCreature("Source", "", 1, 1, WithAbility(ability))
		original := &StackObject{ID: uuid.New(), SourceID: source.ID(), Card: source, Controller: a.PlayerID(), IsAbility: true, Effects: ability.Effects(), Targets: []uuid.UUID{b.PlayerID()}, TargetSpecs: ability.Targets()}
		g.pushStack(original)
		b.chooseQueue = [][]uuid.UUID{choice}
		cp := g.CopyStackObjectDirect(original, b.PlayerID(), true)
		if cp.Targets[0] != b.PlayerID() || cp.TargetSource != source || cp.SourceID != source.ID() {
			t.Fatal("invalid choice changed the target or source")
		}
		if len(b.calls) != 1 || b.calls[0].min != 1 || b.calls[0].max != 1 {
			t.Fatal("copy used the wrong target bounds")
		}
		g.ResolveTopOfStack()
		if len(b.Graveyard()) != 2 || len(a.Graveyard()) != 0 {
			t.Fatal("copy did not retain its original target")
		}
	}
}
