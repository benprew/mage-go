package mage

import (
	"strings"
	"testing"
)

func TestUnblockedAttackAfterLegalBlockers(t *testing.T) {
	for _, simulated := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		defender := BindPlayer("defender")
		watcher := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(AttacksUnblocked(This).WithDefendingPlayer(defender), GainLifeFor(FromPlayer(defender), Fixed(1)))))
		watcher.SetOwner(a.PlayerID())
		other := NewCreature("Other", "", 2, 2)
		other.SetOwner(a.PlayerID())
		blocker := NewCreature("Blocker", "", 2, 2)
		blocker.SetOwner(b.PlayerID())
		if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
			t.Fatal(err)
		}
		g.PutOnBattlefield(watcher, a.PlayerID())
		g.PutOnBattlefield(other, a.PlayerID())
		g.PutOnBattlefield(blocker, b.PlayerID())
		g.combat.AddAttacker(watcher.ID(), b.PlayerID())
		g.combat.AddAttacker(other.ID(), b.PlayerID())
		if simulated {
			g.ExecuteBlockers([]BlockAssignment{{BlockerID: blocker.ID(), AttackerID: other.ID()}})
		} else {
			g.combat.AddBlocker(blocker.ID(), other.ID())
			g.doDeclareBlockers()
		}
		if simulated {
			if g.GetPlayer(b.PlayerID()).Life() != 21 {
				t.Fatal("simulated declaration missed unblocked attacker")
			}
		} else {
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatalf("triggers = %d", g.StackSize())
			}
			clone := g.Clone()
			if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("could not copy trigger")
			}
			g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
			g.PutOnBattlefield(watcher, b.PlayerID())
			g.ResolveStack()
			clone.ResolveStack()
			if g.GetPlayer(b.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 21 {
				t.Fatal("defender capture or clone failed")
			}
		}
	}
}

func TestUnblockedAttackRejectsEarlierBlockAndAbsentSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(AttacksUnblocked(This), GainLifeFor(You, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	blocker := NewCreature("Blocker", "", 2, 2)
	blocker.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(blocker, b.PlayerID())
	g.combat.AddAttacker(watcher.ID(), b.PlayerID())
	g.combat.AddBlocker(blocker.ID(), watcher.ID())
	g.combat.Groups[0].BlockerIDs = nil
	g.doDeclareBlockers()
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("blocked attacker triggered after blocker departed")
	}
	g.combat.Reset()
	g.combat.AddAttacker(watcher.ID(), b.PlayerID())
	g.PhaseOut(watcher.ID())
	g.doDeclareBlockers()
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased attacker triggered")
	}
}

func TestUnblockedAttackDeclarationValidation(t *testing.T) {
	defender := BindPlayer("defender")
	for _, pattern := range []EventPattern{AttacksUnblocked(This).WithDefendingPlayer(nil), AttacksUnblocked(This).WithDefendingPlayer(defender).WithDefendingPlayer(defender)} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid binding")
		}
	}
	card := NewCreature("Missing", "", 1, 1, WithAbility(WhenEvent(AttacksUnblocked(This), GainLifeFor(FromPlayer(defender), Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "binding") {
		t.Fatalf("missing binding error = %v", err)
	}
}
