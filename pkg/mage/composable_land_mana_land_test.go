package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLandTappedForManaEventKeepsIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := NewLand("Land", WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(core.Blue, Fixed(1)))))
	g.PutOnBattlefield(land, a.PlayerID())
	var observed int
	watcher := NewEnchantment("Observer", "", WithAbility(NewTriggered(core.EvtTappedForMana, false,
		GainLifeFor(You, Fixed(1))).SetCondition(func(event *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if event.SourceID == land.ID() && event.SourceIncarnation == g.FindPermanent(land.ID()).incarnationID {
			observed++
		}
		return false
	})))
	g.PutOnBattlefield(watcher, b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if observed != 1 {
		t.Fatalf("observed matching incarnation %d times, want 1", observed)
	}
}

func TestLandTappedForManaLandBindingIdentityAndCopies(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := NewLand("Land", WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(core.Blue, Fixed(1)))))
	bound := BindPermanent("producing land")
	player := BindPlayer("activator")
	amount := BindNumber("amount")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		LandTappedForMana().WithLand(bound).WithPlayer(player).WithAmount(amount),
		AddCountersTo(From(bound), core.P1P1, FromNumber(amount)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.PutOnBattlefield(land, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack size = %d, want 1", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if got := clone.FindPermanent(land.ID()).Counters[core.P1P1]; got != 1 {
		t.Fatalf("clone counters = %d, want 1", got)
	}
	if got := g.FindPermanent(land.ID()).Counters[core.P1P1]; got != 0 {
		t.Fatalf("original counters before resolution = %d", got)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true) == nil {
		t.Fatal("trigger copy failed")
	}
	g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
	g.PutOnBattlefield(land, b.PlayerID())
	g.ResolveStack()
	if got := g.FindPermanent(land.ID()).Counters[core.P1P1]; got != 0 {
		t.Fatalf("returned land counters = %d, want 0", got)
	}
}

func TestLandTappedForManaLandBindingDeclaration(t *testing.T) {
	land := BindPermanent("land")
	for _, pattern := range []landTappedForManaPattern{
		LandTappedForMana().WithLand(nil),
		LandTappedForMana().WithLand(BindPermanent("")),
		LandTappedForMana().WithLand(land).WithLand(land),
	} {
		card := NewEnchantment("Bad", "", WithAbility(WhenEvent(pattern, AddCountersTo(From(land), core.P1P1, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid land binding")
		}
	}
	card := NewEnchantment("Unbound", "", WithAbility(WhenEvent(LandTappedForMana(),
		AddCountersTo(From(land), core.P1P1, Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("accepted unbound land")
	}
}

func TestLandTappedForManaLandBindingPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := NewLand("Land", WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(core.Blue, Fixed(1)))))
	bound := BindPermanent("producing land")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		LandTappedForMana().WithLand(bound), AddCountersTo(From(bound), core.P1P1, Fixed(1)))))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.PutOnBattlefield(land, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	g.PhaseOut(land.ID())
	g.ResolveStack()
	g.PhaseIn([]uuid.UUID{land.ID()})
	if got := g.FindPermanent(land.ID()).Counters[core.P1P1]; got != 0 {
		t.Fatalf("phased land counters = %d, want 0", got)
	}
}
