package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestBlockEachAttackerRequirements(t *testing.T) {
	for _, capacity := range []bool{false, true} {
		for _, menace := range []bool{false, true} {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			first := g.PutOnBattlefield(NewCreature("First", "", 1, 8), a.PlayerID())
			second := g.PutOnBattlefield(NewCreature("Second", "", 1, 8), a.PlayerID())
			flyer := g.PutOnBattlefield(NewCreature("Flying", "", 1, 8, WithKeyword(core.Flying)), a.PlayerID())
			blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 1, 8), b.PlayerID())
			if capacity {
				if err := ApplyEffect(g, Until(core.EndOfTurn, BlockAnyNumber(This)), blocker.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			if menace {
				first.GrantBaseAttr(core.Menace)
			}
			for _, p := range []*Permanent{first, second, flyer} {
				g.combat.AddAttackerIncarnation(p.ID(), p.incarnationID, b.PlayerID())
			}
			if err := ApplyEffect(g, Until(core.EndOfTurn, BlockEachAttacker(This)), blocker.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			want := 1
			if capacity && !menace {
				want = 2
			}
			got := g.CompleteBlockAssignments(b.PlayerID(), nil)
			if len(got) != want {
				t.Fatalf("capacity=%v menace=%v got %v", capacity, menace, got)
			}
			for _, assignment := range got {
				if assignment.AttackerID == flyer.ID() || menace && assignment.AttackerID == first.ID() {
					t.Fatal("illegal block")
				}
			}
			if menace {
				helper := g.PutOnBattlefield(NewCreature("Helper", "", 1, 8), b.PlayerID())
				got = g.CompleteBlockAssignments(b.PlayerID(), nil)
				want = 2
				if capacity {
					want = 3
				}
				if len(got) != want {
					t.Fatalf("helper %s: got %v", helper.ID(), got)
				}
			}
			clone := g.Clone()
			clone.GetEffects().RemoveEndOfTurn()
			clone.ApplyContinuousEffects()
			if len(clone.CompleteBlockAssignments(b.PlayerID(), nil)) != 0 || len(g.CompleteBlockAssignments(b.PlayerID(), nil)) == 0 {
				t.Fatal("clone expiry")
			}
		}
	}
}

func TestBlockEachAttackerIdentity(t *testing.T) {
	for _, change := range []string{"live", "phase", "cycle", "returned", "abilities", "control", "cleanup"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 1, 8), a.PlayerID())
			card := NewCreature("Blocker", "", 1, 8)
			p := g.PutOnBattlefield(card, b.PlayerID())
			if err := ApplyEffect(g, Until(core.EndOfTurn, BlockEachAttacker(This)), p.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "phase":
				g.PhaseOut(p.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "returned":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(card, b.PlayerID())
			case "abilities":
				g.AddContinuousEffect(RemoveAllAbilities(p.ID(), core.Indefinite))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "cleanup":
				g.RunStepWithPriority(core.Cleanup)
			}
			g.combat.AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, b.PlayerID())
			want := change == "live" || change == "cycle" || change == "abilities"
			if (len(g.CompleteBlockAssignments(b.PlayerID(), nil)) == 1) != want {
				t.Fatal(change)
			}
		})
	}
	binding := DeclarePermanentTarget("missing", IsCreature)
	card := NewInstant("Invalid", "", WithAbility(Spell(Until(core.EndOfTurn, BlockEachAttacker(binding)))))
	if ValidateCardDefinition(card.Name(), card) == nil {
		t.Fatal("unbound declaration")
	}
}
