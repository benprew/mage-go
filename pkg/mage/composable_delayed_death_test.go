package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestWhenDiesThisTurnIncarnationAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 2), a.PlayerID())
	saved := BindPermanent("victim")
	effect := Sequence(Snapshot(This, saved), WhenDiesThisTurn(From(saved), GainLifeFor(You, Fixed(3))))
	if err := ApplyEffect(g, effect, victim.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clone.PutPermanentIntoGraveyard(clone.FindPermanent(victim.ID()))
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 23 || a.Life() != 20 {
		t.Fatal("clone lost the delayed trigger or changed the original")
	}
	g.PutPermanentIntoGraveyard(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 23 {
		t.Fatal("death did not trigger")
	}
	g.PutOnBattlefield(victim.Card, a.PlayerID())
	g.PutPermanentIntoGraveyard(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 23 {
		t.Fatal("trigger followed a returned card")
	}
}

func TestWhenDiesThisTurnValidation(t *testing.T) {
	saved := BindPermanent("saved")
	valid := NewCreature("Valid", "", 1, 1, WithAbility(Activate(Costs(), Sequence(Snapshot(This, saved), WhenDiesThisTurn(From(saved), GainLifeFor(You, Fixed(1)))))))
	if err := ValidateCardDefinition("Valid", valid); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{WhenDiesThisTurn(From(saved), GainLifeFor(You, Fixed(1))), WhenDiesThisTurn(nil, GainLifeFor(You, Fixed(1))), WhenDiesThisTurn(This, nil)} {
		card := NewInstant("Invalid", "", Spell(effect))
		if err := ValidateCardDefinition("Invalid", card); err == nil || !strings.Contains(err.Error(), "delayed") && !strings.Contains(err.Error(), "producer") {
			t.Fatalf("unexpected validation: %v", err)
		}
	}
}

func TestWhenDiesThisTurnTargetSourceAndExpiry(t *testing.T) {
	for _, tc := range []struct {
		name string
		act  func(*Game, uuid.UUID, uuid.UUID)
		want int
	}{
		{"dies", func(g *Game, _, victim uuid.UUID) { g.PutPermanentIntoGraveyard(g.FindPermanent(victim)) }, 23},
		{"source leaves", func(g *Game, source, victim uuid.UUID) {
			g.PutPermanentIntoGraveyard(g.FindPermanent(source))
			g.PutPermanentIntoGraveyard(g.FindPermanent(victim))
		}, 23},
		{"returned before death", func(g *Game, _, victim uuid.UUID) {
			p := g.FindPermanent(victim)
			g.BouncePermanentToHand(p)
			g.PutOnBattlefield(p.Card, p.ControllerID())
			g.PutPermanentIntoGraveyard(g.FindPermanent(victim))
		}, 20},
		{"expires", func(g *Game, _, victim uuid.UUID) {
			g.triggers.ClearEndOfTurn()
			g.PutPermanentIntoGraveyard(g.FindPermanent(victim))
		}, 20},
		{"phased", func(g *Game, _, victim uuid.UUID) { g.PhaseOut(victim) }, 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 2), b.PlayerID())
			target := DeclarePermanentTarget("victim", IsCreature)
			ability := NewInstant("Watch", "", Spell(WhenDiesThisTurn(target, GainLifeFor(You, Fixed(3))), WithTargets(target)))
			if err := ValidateCardDefinition("Watch", ability); err != nil {
				t.Fatal(err)
			}
			g.resolution.targetSpecs = []Target{target.targetDeclaration}
			if err := ApplyEffect(g, WhenDiesThisTurn(target, GainLifeFor(You, Fixed(3))), source.ID(), a.PlayerID(), []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			tc.act(g, source.ID(), victim.ID())
			g.PutTriggersOnStack()
			g.ResolveStack()
			if a.Life() != tc.want {
				t.Fatalf("life = %d, want %d", a.Life(), tc.want)
			}
		})
	}
}

func TestWhenDiesThisTurnSimultaneousAndReplacement(t *testing.T) {
	for _, replaced := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		source := g.PutOnBattlefield(NewCreature("Source", "", 2, 2), a.PlayerID())
		victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 2), b.PlayerID())
		target := DeclarePermanentTarget("victim", IsCreature)
		g.resolution.targetSpecs = []Target{target.targetDeclaration}
		if err := ApplyEffect(g, WhenDiesThisTurn(target, GainLifeFor(You, Fixed(3))), source.ID(), a.PlayerID(), []uuid.UUID{victim.ID()}); err != nil {
			t.Fatal(err)
		}
		if replaced {
			g.AddExileIfWouldGoToGraveyardThisTurn(victim.ID(), source.ID())
		}
		g.DestroyPermanents([]*Permanent{source, victim})
		g.PutTriggersOnStack()
		g.ResolveStack()
		want := 23
		if replaced {
			want = 20
		}
		if a.Life() != want {
			t.Fatalf("replaced=%t: life=%d, want %d", replaced, a.Life(), want)
		}
	}
}

func TestWhenDiesThisTurnAbsentTargetDoesNotSubscribe(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 2), b.PlayerID())
	target := DeclarePermanentTarget("victim", IsCreature)
	g.resolution.targetSpecs = []Target{target.targetDeclaration}
	g.PhaseOut(victim.ID())
	if err := ApplyEffect(g, WhenDiesThisTurn(target, GainLifeFor(You, Fixed(3))), uuid.Nil, a.PlayerID(), []uuid.UUID{victim.ID()}); err != nil {
		t.Fatal(err)
	}
	if len(g.DelayedTriggers()) != 0 {
		t.Fatal("absent target created a subscription")
	}
	g.PhaseIn([]uuid.UUID{victim.ID()})
	g.PutPermanentIntoGraveyard(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("a later death matched an absent target")
	}
}
