package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestAttackDefendingPlayerCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	defender := BindPlayer("defender")
	card := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(
		Attacks(This).WithDefendingPlayer(defender), GainLifeFor(FromPlayer(defender), Fixed(2)))))
	card.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(card, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtDeclaredAttacker, SourceID: card.ID(), PlayerID: a.PlayerID(), TargetID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("attack trigger count = %d", g.StackSize())
	}
	clone := g.Clone()
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	g.PutOnBattlefield(card, b.PlayerID())
	g.ResolveStack()
	if g.GetPlayer(b.PlayerID()).Life() != 22 || g.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("defending player changed after attack")
	}
	if clone.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("clone shared trigger state")
	}
	clone.ResolveStack()
	if clone.GetPlayer(b.PlayerID()).Life() != 22 {
		t.Fatal("clone lost event capture")
	}
}

func TestAttackDefendingPlayerDeclaration(t *testing.T) {
	defender := BindPlayer("defender")
	for _, pattern := range []EventPattern{Attacks(This).WithDefendingPlayer(nil), Attacks(This).WithDefendingPlayer(defender).WithDefendingPlayer(defender)} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid defender binding")
		}
	}
	card := NewCreature("Missing", "", 1, 1, WithAbility(WhenEvent(Attacks(This), GainLifeFor(FromPlayer(defender), Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "binding") {
		t.Fatalf("missing binding validation = %v", err)
	}
}

func TestAttackDefendingPlayerRealDeclarationAndAbsence(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	defender := BindPlayer("defender")
	card := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(
		Attacks(This).WithDefendingPlayer(defender), GainLifeFor(FromPlayer(defender), Fixed(1)))))
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	g.FindPermanent(card.ID()).RevokeBaseAttr(AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{card.ID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("actual attack trigger count = %d", g.StackSize())
	}
	g.ResolveStack()
	if g.GetPlayer(b.PlayerID()).Life() != 21 {
		t.Fatal("actual attack did not capture defender")
	}
	phase := g.PhaseOut(card.ID())
	g.FireEvent(GameEvent{Type: EvtDeclaredAttacker, SourceID: card.ID(), PlayerID: a.PlayerID(), TargetID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(phase)
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	g.FireEvent(GameEvent{Type: EvtDeclaredAttacker, SourceID: card.ID(), PlayerID: a.PlayerID(), TargetID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("absent source triggered")
	}
}
