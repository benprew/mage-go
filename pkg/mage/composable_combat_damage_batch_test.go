package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCombatDamageBatchRecipientAndController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	recipient := BindPlayer("damaged player")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OneOrMoreCreaturesYouControlDealCombatDamageToPlayer().WithPlayer(recipient),
		GainLifeFor(FromPlayer(recipient), Fixed(2)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtCombatDamageDealt, PlayerID: b.PlayerID(), TargetID: a.PlayerID(), Amount: 3})
	g.FireEvent(core.GameEvent{Type: core.EvtCombatDamageDealt, PlayerID: a.PlayerID(), TargetID: b.PlayerID(), Amount: 3})
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 1 {
		t.Fatalf("triggers = %d, want 1", got)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if b.Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 24 {
		t.Fatal("clone lost captured recipient or changed original")
	}
	g.ResolveStack()
	if b.Life() != 24 {
		t.Fatalf("recipient life = %d, want 24", b.Life())
	}
}

func TestCombatDamageBatchRealStepAndPrevention(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OneOrMoreCreaturesYouControlDealCombatDamageToPlayer(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 2), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 2), a.PlayerID())
	g.combat.AddAttacker(first.ID(), b.PlayerID())
	g.combat.AddAttacker(second.ID(), b.PlayerID())
	g.AddPreventionShield(b.PlayerID(), 2)
	g.ExecuteCombatDamage()
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("two sources gave %d triggers, want 1", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 18 {
		t.Fatalf("post-combat life = %d, %d", a.Life(), b.Life())
	}
	g.AddPreventionShield(b.PlayerID(), 4)
	g.ExecuteCombatDamage()
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("fully prevented combat damage triggered")
	}
}

func TestCombatDamageBatchSourceLifetimeAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OneOrMoreCreaturesYouControlDealCombatDamageToPlayer(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	phase := g.PhaseOut(watcher.ID())
	g.FireEvent(core.GameEvent{Type: core.EvtCombatDamageDealt, PlayerID: a.PlayerID(), TargetID: b.PlayerID(), Amount: 2})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(phase)
	g.FireEvent(core.GameEvent{Type: core.EvtCombatDamageDealt, PlayerID: a.PlayerID(), TargetID: b.PlayerID(), Amount: 2})
	g.PutTriggersOnStack()
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatal("pending trigger changed controller")
	}
	for _, pattern := range []EventPattern{
		OneOrMoreCreaturesYouControlDealCombatDamageToPlayer().WithPlayer(nil),
		OneOrMoreCreaturesYouControlDealCombatDamageToPlayer().WithPlayer(BindPlayer("")),
		OneOrMoreCreaturesYouControlDealCombatDamageToPlayer().WithPlayer(BindPlayer("first")).WithPlayer(BindPlayer("second")),
	} {
		bad := NewEnchantment("Bad", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("invalid event declaration accepted")
		}
	}
	missing := BindPlayer("missing")
	bad := NewEnchantment("Bad", "", WithAbility(WhenEvent(
		OneOrMoreCreaturesYouControlDealCombatDamageToPlayer(), GainLifeFor(FromPlayer(missing), Fixed(1)))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("missing producer: %v", err)
	}
}
