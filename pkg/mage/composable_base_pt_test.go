package mage

import (
	"slices"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestTypedBasePowerToughness(t *testing.T) {
	for _, change := range []string{"none", "phase", "return", "source leaves", "control", "overwrite"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			card := NewCreature("Subject", "", 4, 5)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			p.AddCounter(P1P1, 1)
			saved := BindPermanent("subject")
			body := Sequence(ChooseRandomPermanent(MatchingPermanents(IsCreature), saved), Until(EndOfTurn,
				ModifyPT(From(saved), Fixed(3), Fixed(3)),
				SetBasePowerToughness(From(saved), Fixed(0), Fixed(2))))
			if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			check := func(game *Game, power, toughness int) {
				t.Helper()
				game.ApplyContinuousEffects()
				got := game.FindPermanent(card.ID())
				if got.CurrentPower(game) != power || got.CurrentToughness(game) != toughness {
					t.Fatalf("P/T = %d/%d, want %d/%d", got.CurrentPower(game), got.CurrentToughness(game), power, toughness)
				}
			}
			check(g, 4, 6)
			clone := g.Clone()
			clone.effects.RemoveEndOfTurn()
			check(clone, 5, 6)
			check(g, 4, 6)
			switch change {
			case "phase":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
				check(g, 4, 5)
				return
			case "source leaves":
				g.RemoveFromBattlefield(source)
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			case "overwrite":
				if err := ApplyEffect(g, Until(EndOfTurn, SetBasePowerToughness(This, Fixed(7), Fixed(8))), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				check(g, 11, 12)
				return
			}
			check(g, 4, 6)
		})
	}
}

func TestTypedOtherThanSourceTargets(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Same name", "", 1, 1)
	g.PutOnBattlefield(source, a.PlayerID())
	own := g.PutOnBattlefield(NewCreature("Same name", "", 2, 2), a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 3, 3), b.PlayerID())
	target := DeclarePermanentTarget("creature", IsCreature, TargetOtherThanSource())
	choices := target.Possible(a.PlayerID(), source, g)
	if len(choices) != 2 || !slices.Contains(choices, own.ID()) || !slices.Contains(choices, other.ID()) {
		t.Fatalf("choices = %v", choices)
	}
	g.PhaseOut(other.ID())
	choices = target.Possible(a.PlayerID(), source, g)
	if len(choices) != 1 || choices[0] != own.ID() {
		t.Fatalf("choices = %v", choices)
	}
}
