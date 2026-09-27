package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestAnyAttackBatchPatternBothPlayersAndCount(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	count := BindNumber("declared")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OneOrMoreCreaturesAttack().WithCount(count), GainLifeFor(You, FromNumber(count)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	first := NewCreature("First", "", 1, 1)
	second := NewCreature("Second", "", 1, 1)
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.FindPermanent(first.ID()).RevokeBaseAttr(AttrSummonSick)
	g.FindPermanent(second.ID()).RevokeBaseAttr(AttrSummonSick)
	g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{first.ID(), second.ID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack size = %d, want 1", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || clone.GetPlayer(a.PlayerID()).Life() != 22 {
		t.Fatal("clone changed original or lost attack count")
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 20 {
		t.Fatalf("life totals = %d, %d", a.Life(), b.Life())
	}
	g.FireEvent(GameEvent{Type: EvtAttackersDeclared, PlayerID: a.PlayerID(), Amount: 1})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("own attack stack size = %d, want 1", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 23 {
		t.Fatalf("life after own attack = %d, want 23", a.Life())
	}
	g.FireEvent(GameEvent{Type: EvtEntersAttacking, SourceID: first.ID(), PlayerID: b.PlayerID()})
	g.FireEvent(GameEvent{Type: EvtAttackersDeclared, PlayerID: b.PlayerID(), Amount: 0})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("an empty declaration or entering attacking triggered")
	}
}

func TestAnyAttackBatchPatternLifetimeAndDeclaration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OneOrMoreCreaturesAttack(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	phased := g.PhaseOut(watcher.ID())
	g.FireEvent(GameEvent{Type: EvtAttackersDeclared, PlayerID: b.PlayerID(), Amount: 1})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(phased)
	g.FireEvent(GameEvent{Type: EvtAttackersDeclared, PlayerID: b.PlayerID(), Amount: 1})
	g.PutTriggersOnStack()
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatal("pending trigger followed returned source")
	}
	for _, pattern := range []EventPattern{
		OneOrMoreCreaturesAttack().WithCount(nil),
		OneOrMoreCreaturesAttack().WithCount(BindNumber("count")).WithCount(BindNumber("again")),
	} {
		bad := NewEnchantment("Bad", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("accepted invalid count declaration")
		}
	}
	spell := NewInstant("Invalid", "", Spell(GainLifeFor(You, Fixed(1))))
	spell.AddAbility(WhenEvent(OneOrMoreCreaturesAttack(), GainLifeFor(You, Fixed(1))))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil || !strings.Contains(err.Error(), "battlefield") {
		t.Fatalf("spell registration = %v", err)
	}
}
