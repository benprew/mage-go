package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTargetBlockHistoryQuery(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first := DeclarePermanentTarget("blocker", IsCreature)
	second := DeclarePermanentTarget("attacker", IsCreature, TargetBlockedByThisTurn(first))
	card := NewInstant("History", "", WithAbility(Spell(TapSubject(second), WithTargets(first, second))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	wall := g.PutOnBattlefield(NewCreature("Wall", "", 0, 5), b.PlayerID())
	idle := g.PutOnBattlefield(NewCreature("Idle", "", 0, 5), b.PlayerID())
	attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 5), a.PlayerID())
	g.GetCombat().AddAttacker(attacker.ID(), b.PlayerID())
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: wall.ID(), AttackerID: attacker.ID()}})
	g.GetCombat().Reset()
	specs := card.CastTargets()
	g.resolution.SetX(7)
	for range 3 {
		if got := g.PossibleTargetGroup(a.PlayerID(), card, specs, 0, nil, 0); !slices.Equal(got, []uuid.UUID{wall.ID()}) {
			t.Fatalf("first choices: %v (idle %v)", got, idle.ID())
		}
		if got := g.PossibleTargetGroup(a.PlayerID(), card, specs, 1, []uuid.UUID{wall.ID()}, 0); !slices.Equal(got, []uuid.UUID{attacker.ID()}) {
			t.Fatalf("second choices: %v", got)
		}
		if g.resolution.X() != 7 || g.resolution.targetQueryBindings != nil || g.StackSize() != 0 || attacker.Tapped || len(a.chooseQueue) != 0 {
			t.Fatal("query changed state")
		}
	}
	clone := g.Clone()
	g.RunStepWithPriority(core.Cleanup)
	if len(g.PossibleTargetGroup(a.PlayerID(), card, specs, 0, nil, 0)) != 0 {
		t.Fatal("cleanup kept history")
	}
	if len(clone.PossibleTargetGroup(a.PlayerID(), card, specs, 0, nil, 0)) != 1 {
		t.Fatal("cleanup changed clone")
	}
}

func TestTargetBlockHistoryDeclarations(t *testing.T) {
	for _, scenario := range []string{"nil", "unbound", "self", "forward", "plural"} {
		t.Run(scenario, func(t *testing.T) {
			first := DeclarePermanentTarget("first", AnyPermanent)
			ref := first
			if scenario == "nil" {
				ref = nil
			}
			if scenario == "unbound" {
				ref = DeclarePermanentTarget("other", AnyPermanent)
			}
			second := DeclarePermanentTarget("second", AnyPermanent, TargetBlockedByThisTurn(ref))
			targets := []TargetBinding{first, second}
			switch scenario {
			case "self":
				first.spec.(*PermanentTarget).blockHistoryDependencies = []targetBlockHistoryDependency{{first}}
				targets = []TargetBinding{first}
			case "forward":
				targets = []TargetBinding{second, first}
			case "plural":
				targets = []TargetBinding{first, DeclarePermanentTargets("plural", 1, 2, AnyPermanent, TargetBlockedByThisTurn(first))}
			}
			card := NewInstant("Invalid", "", WithAbility(Spell(TapSubject(first), WithTargets(targets...))))
			if err := ValidateCardDefinition(card.Name(), card); err == nil {
				t.Fatal("invalid dependency accepted")
			}
		})
	}
}

func TestTargetBlockHistoryCopyAndInstances(t *testing.T) {
	g, a, b := newSpellCopyGame()
	makeCard := func() *BaseCard {
		first := DeclarePermanentTarget("blocker", IsCreature)
		second := DeclarePermanentTarget("attacker", IsCreature, TargetBlockedByThisTurn(first))
		return NewInstant("History", "", WithAbility(Spell(TapSubject(second), WithTargets(first, second))))
	}
	wall := g.PutOnBattlefield(NewCreature("Wall", "", 0, 5), b.PlayerID())
	otherWall := g.PutOnBattlefield(NewCreature("Other wall", "", 0, 5), b.PlayerID())
	attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 5), a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other attacker", "", 2, 5), a.PlayerID())
	g.GetCombat().AddAttacker(attacker.ID(), b.PlayerID())
	g.GetCombat().AddAttacker(other.ID(), b.PlayerID())
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: wall.ID(), AttackerID: attacker.ID()}, {BlockerID: otherWall.ID(), AttackerID: other.ID()}})
	g.GetCombat().Reset()
	spell := makeCard()
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{wall.ID(), attacker.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	a.chooseQueue = [][]uuid.UUID{{otherWall.ID()}, {other.ID()}}
	copied := g.CopyStackObjectDirect(original, a.PlayerID(), true)
	if copied == nil || !slices.Equal(copied.Targets, []uuid.UUID{otherWall.ID(), other.ID()}) {
		t.Fatal("copy lost new block pair")
	}
	g.ResolveStackObject(copied)
	if attacker.Tapped || !other.Tapped {
		t.Fatal("copy followed original pair")
	}
	second := makeCard()
	second.SetOwner(a.PlayerID())
	a.AddToHand(second)
	if err := g.CastSpellByID(a.PlayerID(), second.ID(), []uuid.UUID{otherWall.ID(), other.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveTopOfStack()
	if attacker.Tapped {
		t.Fatal("factory shared targets")
	}
	g.ResolveStackObject(original)
	if !attacker.Tapped {
		t.Fatal("original lost its pair")
	}
}
