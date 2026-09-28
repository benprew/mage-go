package mage

import (
	"strings"
	"testing"
)

func TestBecomesBlockedCapturesDefendingPlayer(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	defender := BindPlayer("defender")
	attacker := NewCreature("Attacker", "", 2, 2, WithAbility(WhenEvent(
		BecomesBlocked(This).WithDefendingPlayer(defender), Discard(FromPlayer(defender), Fixed(1)))))
	attacker.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(attacker.Name(), attacker); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(attacker, a.PlayerID())
	blocker := NewCreature("Blocker", "", 1, 3)
	g.PutOnBattlefield(blocker, b.PlayerID())
	b.AddToHand(NewArtifact("Card", ""))
	g.combat.AddAttacker(attacker.ID(), b.PlayerID())
	g.combat.AddBlockerIncarnation(blocker.ID(), g.FindPermanent(blocker.ID()).incarnationID, attacker.ID())
	g.fireCreatureBecomesBlocked(attacker.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d, want 1", g.StackSize())
	}
	clone := g.Clone()
	g.RemoveFromBattlefield(g.FindPermanent(attacker.ID()))
	g.PutOnBattlefield(attacker, b.PlayerID())
	g.ResolveStack()
	if len(b.Hand()) != 0 || len(a.Hand()) != 0 {
		t.Fatal("defending player's card was not discarded")
	}
	if len(clone.GetPlayer(b.PlayerID()).Hand()) != 1 {
		t.Fatal("clone shared trigger result")
	}
	clone.ResolveStack()
	if len(clone.GetPlayer(b.PlayerID()).Hand()) != 0 {
		t.Fatal("clone lost defender capture")
	}
}

func TestBecomesBlockedDefenderDeclaration(t *testing.T) {
	defender := BindPlayer("defender")
	for _, pattern := range []EventPattern{
		BecomesBlocked(This).WithDefendingPlayer(nil),
		BecomesBlocked(This).WithDefendingPlayer(defender).WithDefendingPlayer(defender),
	} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid defender capture")
		}
	}
	card := NewCreature("Missing", "", 1, 1, WithAbility(WhenEvent(BecomesBlocked(This), GainLifeFor(FromPlayer(defender), Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "binding") {
		t.Fatalf("validation = %v", err)
	}
}

func TestBecomesBlockedDefenderRealCombatEvent(t *testing.T) {
	for _, entering := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		defender := BindPlayer("defender")
		attacker := NewCreature("Attacker", "", 2, 2, WithAbility(WhenEvent(
			BecomesBlocked(This).WithDefendingPlayer(defender), GainLifeFor(FromPlayer(defender), Fixed(1)))))
		g.PutOnBattlefield(attacker, a.PlayerID())
		g.combat.AddAttacker(attacker.ID(), b.PlayerID())
		if entering {
			g.PutOnBattlefieldBlocking(NewCreature("Entering blocker", "", 1, 1), b.PlayerID(), attacker.ID())
			g.PutTriggersOnStack()
			g.ResolveStack()
		} else {
			blocker := NewCreature("Blocker", "", 1, 1)
			g.PutOnBattlefield(blocker, b.PlayerID())
			g.ExecuteBlockers([]BlockAssignment{{BlockerID: blocker.ID(), AttackerID: attacker.ID()}})
		}
		if b.Life() != 21 || a.Life() != 20 {
			t.Fatalf("entering=%t: defender life = %d, attacker life = %d", entering, b.Life(), a.Life())
		}
	}
}

func TestBecomesBlockedDefenderPermanentController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	defender := BindPlayer("defender")
	attacker := NewCreature("Attacker", "", 2, 2, WithAbility(WhenEvent(
		BecomesBlocked(This).WithDefendingPlayer(defender), GainLifeFor(FromPlayer(defender), Fixed(1)))))
	g.PutOnBattlefield(attacker, a.PlayerID())
	walker := g.PutOnBattlefield(NewPlaneswalker("Walker", "", 3), b.PlayerID())
	g.combat.AddAttacker(attacker.ID(), walker.ID())
	g.PutOnBattlefieldBlocking(NewCreature("Blocker", "", 1, 1), b.PlayerID(), attacker.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if b.Life() != 21 || a.Life() != 20 {
		t.Fatalf("defending player life = %d, attacker player life = %d", b.Life(), a.Life())
	}
}

func TestBecomesBlockedDefenderStackCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	defender := BindPlayer("defender")
	attacker := NewCreature("Attacker", "", 2, 2, WithAbility(WhenEvent(
		BecomesBlocked(This).WithDefendingPlayer(defender), GainLifeFor(FromPlayer(defender), Fixed(1)))))
	g.PutOnBattlefield(attacker, a.PlayerID())
	g.combat.AddAttacker(attacker.ID(), b.PlayerID())
	g.fireCreatureBecomesBlocked(attacker.ID())
	g.PutTriggersOnStack()
	g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	g.ResolveStack()
	if b.Life() != 22 || a.Life() != 20 {
		t.Fatalf("copied defender life = %d, attacker life = %d", b.Life(), a.Life())
	}
}
