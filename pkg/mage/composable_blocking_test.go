package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestBlockingProhibitionSurvivesAbilityRemoval(t *testing.T) {
	for _, removalFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "removal after", true: "removal before"}[removalFirst], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 2), a.PlayerID())
			blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 2, 2), b.PlayerID())
			remove := func() { g.AddContinuousEffect(RemoveAllAbilities(attacker.ID(), EndOfTurn)) }
			if removalFirst {
				remove()
			}
			if err := ApplyEffect(g, MakeUnblockableUntilEndOfTurn(), source.ID(), a.PlayerID(), []uuid.UUID{attacker.ID()}); err != nil {
				t.Fatal(err)
			}
			if !removalFirst {
				remove()
			}
			g.ApplyContinuousEffects()
			if CanBlock(blocker, attacker, g) {
				t.Fatal("ability removal erased the blocking prohibition")
			}
			g.RemoveFromBattlefield(source)
			g.ApplyContinuousEffects()
			if CanBlock(blocker, attacker, g) {
				t.Fatal("source departure erased the blocking prohibition")
			}
			clone := g.Clone()
			clone.effects.RemoveEndOfTurn()
			clone.ApplyContinuousEffects()
			if !CanBlock(clone.FindPermanent(blocker.ID()), clone.FindPermanent(attacker.ID()), clone) || CanBlock(blocker, attacker, g) {
				t.Fatal("expiry did not stay in the clone")
			}
		})
	}
}

func TestTypedBlockingProhibitionLifetime(t *testing.T) {
	for _, change := range []string{"none", "target returns", "phased", "phase cycle", "control", "power", "ability removal"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Attacker", "", 2, 2)
			attacker := g.PutOnBattlefield(card, a.PlayerID())
			blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 2, 2), b.PlayerID())
			body := Until(EndOfTurn, CantBeBlocked(This))
			if err := ApplyEffect(g, body, attacker.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "target returns":
				g.RemoveFromBattlefield(attacker)
				attacker = g.PutOnBattlefield(card, a.PlayerID())
			case "phased":
				attacker.PhasedOut = true
			case "phase cycle":
				attacker.PhasedOut = true
				g.ApplyContinuousEffects()
				attacker.PhasedOut = false
			case "control":
				if err := ApplyEffect(g, GainControl().Until(EndOfTurn), b.PlayerID(), b.PlayerID(), []uuid.UUID{attacker.ID()}); err != nil {
					t.Fatal(err)
				}
			case "power":
				g.AddContinuousEffect(TemporaryBoost(attacker.ID(), 4, 4))
			case "ability removal":
				g.AddContinuousEffect(RemoveAllAbilities(attacker.ID(), EndOfTurn))
			}
			g.ApplyContinuousEffects()
			wantBlock := change == "target returns" || change == "phased"
			if got := CanBlock(blocker, attacker, g); got != wantBlock {
				t.Fatalf("CanBlock = %v, want %v", got, wantBlock)
			}
			late := g.PutOnBattlefield(NewCreature("Late blocker", "", 2, 2), b.PlayerID())
			g.ApplyContinuousEffects()
			if CanBlock(late, attacker, g) != wantBlock {
				t.Fatal("restriction missed a later blocker")
			}
			clone := g.Clone()
			clone.effects.RemoveEndOfTurn()
			clone.ApplyContinuousEffects()
			if !CanBlock(clone.FindPermanentIncludingPhased(blocker.ID()), clone.FindPermanentIncludingPhased(attacker.ID()), clone) {
				t.Fatal("restriction did not expire")
			}
		})
	}
}

func TestBlockingProhibitionPreservesPrintedAbilityRemoval(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 2, WithKeyword(UnblockableKW)), a.PlayerID())
	blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 2, 2), b.PlayerID())
	g.AddContinuousEffect(RemoveAllAbilities(attacker.ID(), EndOfTurn))
	g.ApplyContinuousEffects()
	if !CanBlock(blocker, attacker, g) {
		t.Fatal("printed ability survived ability removal")
	}
}
