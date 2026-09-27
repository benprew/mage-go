package mage

import (
	"fmt"
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDealsDamageCapturesPermanentRecipient(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	recipient := BindPermanent("recipient")
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(
		DealsDamage(This, BindNumber("amount")).WithPermanent(recipient),
		AddCountersTo(From(recipient), core.P1P1, Fixed(1)))))
	source.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	victim := NewCreature("Victim", "", 3, 3)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack = %d", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(victim.ID()))
	g.PutOnBattlefield(victim, b.PlayerID())
	clone := g.Clone()
	clone.ResolveStack()
	g.ResolveStack()
	if g.FindPermanent(victim.ID()).Counters[core.P1P1] != 0 || clone.FindPermanent(victim.ID()).Counters[core.P1P1] != 0 {
		t.Fatal("returned recipient used an old binding")
	}
}

func TestDealsDamagePermanentRecipientPresentAndAbsent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	recipient := BindPermanent("recipient")
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(
		DealsDamage(This, BindNumber("amount")).WithPermanent(recipient),
		AddCountersTo(From(recipient), core.P1P1, Fixed(1)))))
	source.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 5, 5)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, source.ID())
	g.PutTriggersOnStack()
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if got := g.FindPermanent(victim.ID()).Counters[core.P1P1]; got != 2 {
		t.Fatalf("counters = %d", got)
	}
	g.PhaseOut(source.ID())
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseOut(victim.ID())
	g.DealDamageToPermanent(g.FindPermanentIncludingPhased(victim.ID()), 1, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased recipient was damaged")
	}
}

func TestDealsDamagePermanentRecipientDeclaration(t *testing.T) {
	bound := BindPermanent("recipient")
	good := NewCreature("Good", "", 1, 1, WithAbility(WhenEvent(
		DealsDamage(This, BindNumber("amount")).WithPermanent(bound),
		AddCountersTo(From(bound), core.P1P1, Fixed(1)))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	if meta := InspectDefinition(good.Abilities()[0]); !strings.Contains(fmt.Sprintf("%+v", meta), "event-permanent-capture") {
		t.Fatalf("missing capture: %+v", meta)
	}
	for _, pattern := range []EventPattern{
		DealsDamage(This, BindNumber("amount")).WithPermanent(nil),
		DealsDamage(This, BindNumber("amount")).WithPermanent(BindPermanent("")),
		DealsDamage(This, BindNumber("amount")).WithPermanent(bound).WithPermanent(bound),
		DealsDamageToPlayer(This, BindPlayer("player")).WithPermanent(bound),
	} {
		bad := NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("accepted invalid capture")
		}
	}
	unbound := NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(
		DealsDamage(This, BindNumber("amount")), AddCountersTo(From(bound), core.P1P1, Fixed(1)))))
	if err := ValidateCardDefinition(unbound.Name(), unbound); err == nil {
		t.Fatal("accepted unbound recipient")
	}
}

func TestDealsDamagePermanentRecipientFollowsFinalDamage(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	recipient := BindPermanent("recipient")
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(
		DealsDamage(This, BindNumber("amount")).WithPermanent(recipient),
		AddCountersTo(From(recipient), core.P1P1, Fixed(1)))))
	source.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 5, 5)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.AddPreventionShield(victim.ID(), 1)
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("prevented damage triggered")
	}
	g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: victim.ID()})
	g.DealDamageToPlayer(b, 1, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("redirected damage stack = %d", g.StackSize())
	}
	g.ResolveStack()
	if got := g.FindPermanent(victim.ID()).Counters[core.P1P1]; got != 1 {
		t.Fatalf("redirected recipient counters = %d", got)
	}
}

func TestDealsDamagePermanentRecipientsInOneBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	recipient := BindPermanent("recipient")
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(
		DealsDamage(This, BindNumber("amount")).WithPermanent(recipient),
		AddCountersTo(From(recipient), core.P1P1, Fixed(1)))))
	source.SetOwner(a.PlayerID())
	first := NewCreature("First", "", 5, 5)
	second := NewCreature("Second", "", 5, 5)
	first.SetOwner(b.PlayerID())
	second.SetOwner(b.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	if err := ApplyEffect(g, DealDamageEach(This,
		DamageRecipients(EachPlayer, WithoutPermanent(MatchingPermanents(IsCreature), This)), Fixed(1)),
		source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 4 {
		t.Fatalf("stack = %d", g.StackSize())
	}
	g.ResolveStack()
	for _, card := range []Card{first, second} {
		if got := g.FindPermanent(card.ID()).Counters[core.P1P1]; got != 1 {
			t.Fatalf("%s counters = %d", card.Name(), got)
		}
	}
}
