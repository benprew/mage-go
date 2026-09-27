package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestActivatedAbilityWithoutTapCostFilter(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	actor := BindPlayer("actor")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ActivatedAbilityEvent().FromArtifact().ByOpponent().WithoutTapCost().WithActivator(actor),
		GainLifeFor(FromPlayer(actor), Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	plain := NewArtifact("Plain", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)))))
	tapped := NewArtifact("Tapped", "", WithAbility(Activate(Costs(TapCost(This)), GainLifeFor(You, Fixed(1)))))
	creature := NewCreature("Creature", "", 1, 1, WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)))))
	failed := NewArtifact("Failed", "", WithAbility(Activate(Costs(LifePayCost(30)), GainLifeFor(You, Fixed(1)))))
	for _, card := range []Card{plain, tapped, creature, failed} {
		g.PutOnBattlefield(card, b.PlayerID())
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), plain.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), tapped.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), creature.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), failed.ID(), 0, nil); err == nil {
		t.Fatal("unpaid activation succeeded")
	}
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 4 {
		t.Fatalf("stack = %d, want 4", got)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 25 {
		t.Fatalf("life = %d/%d, want 20/25", a.Life(), b.Life())
	}
}

func TestActivatedAbilityWithoutTapCostSourceAndPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := BindDamageSource("source")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ActivatedAbilityEvent().FromArtifact().ByOpponent().WithoutTapCost().WithSource(source),
		DealDamageFrom(FromDamageSource(source), You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	artifact := NewArtifact("Artifact", "", WithKeyword(core.Lifelink), WithAbility(
		Activate(Costs(SacrificeSourceCost()), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(artifact, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), artifact.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(artifact, a.PlayerID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 22 {
		t.Fatalf("life = %d/%d, want 19/22", a.Life(), b.Life())
	}
	g.PhaseOut(watcher.ID())
	other := NewArtifact("Other", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(other, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), other.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 1 {
		t.Fatalf("phased watcher created trigger: stack = %d", got)
	}
}

func TestActivatedAbilityWithoutTapCostManaAbilities(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ActivatedAbilityEvent().WithoutTapCost(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	plain := NewArtifact("Plain mana", "", WithAbility(Activate(Costs(), AddManaAmount(core.Blue, Fixed(1)))))
	tapped := NewArtifact("Tap mana", "", WithAbility(NewManaAbility(core.Blue)))
	g.PutOnBattlefield(plain, b.PlayerID())
	g.PutOnBattlefield(tapped, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), plain.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), tapped.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 1 {
		t.Fatalf("stack = %d, want 1", got)
	}
	g.ResolveStack()
	if got := a.Life(); got != 21 {
		t.Fatalf("life = %d, want 21", got)
	}
}

func TestActivatedAbilityWithoutTapCostDefinition(t *testing.T) {
	pattern := ActivatedAbilityEvent().FromArtifact().ByOpponent().WithoutTapCost()
	if got := InspectDefinition(pattern); got.Role != "event-pattern" {
		t.Fatalf("role = %q", got.Role)
	}
	good := NewEnchantment("Good", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	bad := NewEnchantment("Bad", "", WithAbility(WhenEvent(pattern.WithoutTapCost(), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
		t.Fatal("repeated filter passed registration")
	}
}
