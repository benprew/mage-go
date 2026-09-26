package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestEachControlledAttackerCapturesIndependentEvents(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	attacker := BindPermanent("attacker")
	defender := BindPlayer("defender")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		EachCreatureYouControlAttacks().WithAttacker(attacker).WithDefendingPlayer(defender),
		Sequence(AddCountersTo(From(attacker), P1P1, Fixed(1)), GainLifeFor(FromPlayer(defender), Fixed(1))))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	meta := InspectDefinition(watcher.Abilities()[0])
	if meta.Role != "event-trigger" || len(meta.Children) == 0 || len(meta.Children[0].Children) != 2 {
		t.Fatalf("attack capture inspection = %+v", meta)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	first := NewCreature("First", "", 2, 2)
	second := NewCreature("Second", "", 2, 2)
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	g.FindPermanent(first.ID()).RevokeBaseAttr(AttrSummonSick)
	g.FindPermanent(second.ID()).RevokeBaseAttr(AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{first.ID(), second.ID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("got %d per-attacker triggers", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 || g.FindPermanent(first.ID()).Counters[P1P1] != 0 {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	if b.Life() != 22 || g.FindPermanent(first.ID()).Counters[P1P1] != 1 || g.FindPermanent(second.ID()).Counters[P1P1] != 1 {
		t.Fatal("per-attacker event values were mixed")
	}
}

func TestEachControlledAttackerIdentityAndAbsence(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	attacker := BindPermanent("attacker")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		EachCreatureYouControlAttacks().WithAttacker(attacker), AddCountersTo(From(attacker), P1P1, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	creature := NewCreature("Creature", "", 1, 1)
	g.PutOnBattlefield(creature, a.PlayerID())
	event := GameEvent{Type: EvtDeclaredAttacker, SourceID: creature.ID(), PlayerID: a.PlayerID(), TargetID: b.PlayerID()}
	g.FireEvent(event)
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("missing attack trigger")
	}
	g.RemoveFromBattlefield(g.FindPermanent(creature.ID()))
	g.PutOnBattlefield(creature, a.PlayerID())
	g.ResolveStack()
	if g.FindPermanent(creature.ID()).Counters[P1P1] != 0 {
		t.Fatal("captured attacker followed a returned card")
	}
	g.FireEvent(GameEvent{Type: EvtDeclaredAttacker, SourceID: creature.ID(), PlayerID: b.PlayerID(), TargetID: a.PlayerID()})
	g.FireEvent(GameEvent{Type: EvtEntersAttacking, SourceID: creature.ID(), PlayerID: a.PlayerID(), TargetID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("wrong controller or nondeclaration triggered")
	}
	phase := g.PhaseOut(creature.ID())
	g.FireEvent(event)
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased attacker triggered")
	}
	g.PhaseIn(phase)
}

func TestEachControlledAttackerDeclaration(t *testing.T) {
	attacker := BindPermanent("attacker")
	defender := BindPlayer("defender")
	for _, pattern := range []EventPattern{
		EachCreatureYouControlAttacks().WithAttacker(nil),
		EachCreatureYouControlAttacks().WithDefendingPlayer(nil),
		EachCreatureYouControlAttacks().WithAttacker(attacker).WithAttacker(attacker),
		EachCreatureYouControlAttacks().WithDefendingPlayer(defender).WithDefendingPlayer(defender),
	} {
		card := NewEnchantment("Invalid", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid event capture")
		}
	}
	card := NewEnchantment("Missing", "", WithAbility(WhenEvent(EachCreatureYouControlAttacks(), AddCountersTo(From(attacker), P1P1, Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "binding") {
		t.Fatalf("scope check = %v", err)
	}
}

func TestEachControlledAttackerSourceLifecycleAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	defender := BindPlayer("defender")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		EachCreatureYouControlAttacks().WithDefendingPlayer(defender), GainLifeFor(FromPlayer(defender), Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	attacker := NewCreature("Attacker", "", 2, 2)
	g.PutOnBattlefield(attacker, a.PlayerID())
	event := GameEvent{Type: EvtDeclaredAttacker, SourceID: attacker.ID(), PlayerID: a.PlayerID(), TargetID: b.PlayerID()}
	phase := g.PhaseOut(watcher.ID())
	g.FireEvent(event)
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased watcher triggered")
	}
	g.PhaseIn(phase)
	g.FireEvent(event)
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("present watcher did not trigger")
	}
	original := g.StackPeek()
	if copied := g.CopyStackObjectDirect(original, b.PlayerID(), false); copied == nil {
		t.Fatal("trigger copy failed")
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	clone := g.Clone()
	clone.ResolveStack()
	if b.Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 22 {
		t.Fatal("clone lost or shared event capture")
	}
	g.ResolveStack()
	if b.Life() != 22 {
		t.Fatal("pending source return or copy changed defender")
	}
}
