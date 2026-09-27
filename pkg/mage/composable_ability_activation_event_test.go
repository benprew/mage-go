package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAbilityActivatedCapturesActivator(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	actor := BindPlayer("actor")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ActivatedAbilityEvent().WithActivator(actor), GainLifeFor(FromPlayer(actor), Fixed(2)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	one := NewArtifact("One", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)))))
	two := NewArtifact("Two", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(one, a.PlayerID())
	g.PutOnBattlefield(two, b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), one.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), two.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 4 {
		t.Fatalf("stack size = %d, want 4", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed original players")
	}
	g.ResolveStack()
	if a.Life() != 25 || b.Life() != 23 {
		t.Fatalf("life totals = %d, %d, want 25 and 23", a.Life(), b.Life())
	}
}

func TestAbilityActivatedAvailabilityAndPayment(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(ActivatedAbilityEvent(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	costly := NewArtifact("Costly", "", WithActivatedAbility(GainLifeFor(You, Fixed(1)), ManaCostOf("{9}")))
	g.PutOnBattlefield(costly, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), costly.ID(), 0, nil); err == nil {
		t.Fatal("unpaid activation succeeded")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("failed activation created a trigger")
	}
	mana := NewArtifact("Mana", "", WithAbility(NewManaAbility(core.Blue)))
	g.PutOnBattlefield(mana, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), mana.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("mana activation did not create one trigger")
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("pending trigger lost its controller")
	}
	g.PhaseOut(watcher.ID())
	otherMana := NewArtifact("Other mana", "", WithAbility(NewManaAbility(core.Blue)))
	g.PutOnBattlefield(otherMana, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), otherMana.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased watcher triggered")
	}
}

func TestAbilityActivatedDeclaration(t *testing.T) {
	actor := BindPlayer("actor")
	if got := InspectDefinition(ActivatedAbilityEvent().WithActivator(actor)); got.Role != "event-pattern" || len(got.Children) != 1 {
		t.Fatal("activation pattern inspection lost its binding")
	}
	good := NewEnchantment("Good", "", WithAbility(WhenEvent(ActivatedAbilityEvent().WithActivator(actor), GainLifeFor(FromPlayer(actor), Fixed(1)))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	for _, pattern := range []EventPattern{ActivatedAbilityEvent().WithActivator(nil), ActivatedAbilityEvent().WithActivator(actor).WithActivator(actor)} {
		bad := NewEnchantment("Bad", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("invalid activation pattern passed registration")
		}
	}
	bad := NewEnchantment("Bad", "", WithAbility(WhenEvent(ActivatedAbilityEvent(), GainLifeFor(FromPlayer(actor), Fixed(1)))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
		t.Fatal("unbound activator passed registration")
	}
	spell := NewInstant("Bad spell", "", WithAbility(WhenEvent(ActivatedAbilityEvent(), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil {
		t.Fatal("non-battlefield source passed registration")
	}
	modal := NewEnchantment("Modal", "", WithAbility(WhenEventModes(
		ActivatedAbilityEvent().WithActivator(actor),
		ModeOf("Gain", GainLifeFor(FromPlayer(actor), Fixed(1))),
		ModeOf("Draw", DrawFor(FromPlayer(actor), Fixed(1))))))
	if err := ValidateCardDefinition(modal.Name(), modal); err != nil {
		t.Fatal(err)
	}
}
