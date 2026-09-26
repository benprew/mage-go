package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestArtifactAbilityWithoutTapCostEvent(t *testing.T) {
	for _, tc := range []struct {
		name string
		cost Cost
		want int
	}{
		{"without tap", ManaCostOf("{0}"), 1},
		{"with tap", Tap(), 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			artifact := BindPermanent("activated artifact")
			controller := BindPlayer("artifact controller")
			watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
				ArtifactAbilityActivatedWithoutTapCost().WithArtifact(artifact).WithController(controller),
				Sequence(AddCountersTo(From(artifact), core.Charge, Fixed(1)), GainLifeFor(FromPlayer(controller), Fixed(1))))))
			if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(watcher, a.PlayerID())
			source := NewArtifact("Source", "", WithActivatedAbility(GainLifeFor(You, Fixed(1)), tc.cost))
			g.PutOnBattlefield(source, b.PlayerID())
			if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.PutTriggersOnStack()
			if got := g.StackSize(); got != 1+tc.want {
				t.Fatalf("stack=%d, want %d", got, 1+tc.want)
			}
			clone := g.Clone()
			clone.ResolveStack()
			if b.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			if got := g.FindPermanent(source.ID()).Counters[core.Charge]; int(got) != tc.want {
				t.Fatalf("counters=%d, want %d", got, tc.want)
			}
			if b.Life() != 21+tc.want {
				t.Fatalf("life=%d, want %d", b.Life(), 21+tc.want)
			}
		})
	}
}

func TestArtifactAbilityWithoutTapCostFiltersAndLKI(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	controller := BindPlayer("controller")
	watcher := NewAura("Watcher", "", WithAbility(WhenEvent(
		ArtifactAbilityActivatedWithoutTapCost().AttachedToSource().WithController(controller),
		GainLifeFor(FromPlayer(controller), Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	attached := NewArtifact("Attached", "", WithAbility(Activate(Costs(SacrificeSourceCost()), GainLifeFor(You, Fixed(1)))))
	attached.SetOwner(b.PlayerID())
	g.PutOnBattlefield(attached, b.PlayerID())
	g.Attach(watcher.ID(), attached.ID())
	other := NewArtifact("Other", "", WithActivatedAbility(GainLifeFor(You, Fixed(1)), ManaCostOf("{0}")))
	g.PutOnBattlefield(other, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), other.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), attached.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("stack=%d, want 3", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	g.ResolveStack()
	if b.Life() != 24 {
		t.Fatalf("life=%d, want 24", b.Life())
	}
}

func TestArtifactAbilityWithoutTapCostDeclaration(t *testing.T) {
	artifact := BindPermanent("artifact")
	controller := BindPlayer("controller")
	valid := NewEnchantment("Valid", "", WithAbility(WhenEvent(
		ArtifactAbilityActivatedWithoutTapCost().WithArtifact(artifact).WithController(controller),
		GainLifeFor(FromPlayer(controller), Fixed(1)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, pattern := range []EventPattern{
		ArtifactAbilityActivatedWithoutTapCost().WithArtifact(nil),
		ArtifactAbilityActivatedWithoutTapCost().WithController(nil),
		ArtifactAbilityActivatedWithoutTapCost().WithArtifact(artifact).WithArtifact(artifact),
		ArtifactAbilityActivatedWithoutTapCost().WithController(controller).WithController(controller),
	} {
		card := NewEnchantment("Invalid", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid pattern passed registration")
		}
	}
	missing := NewEnchantment("Missing", "", WithAbility(WhenEvent(
		ArtifactAbilityActivatedWithoutTapCost(), GainLifeFor(FromPlayer(controller), Fixed(1)))))
	if err := ValidateCardDefinition(missing.Name(), missing); err == nil {
		t.Fatal("unbound controller passed registration")
	}
}

func TestArtifactAbilityWithoutTapCostExcludesManaTap(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ArtifactAbilityActivatedWithoutTapCost(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	artifact := NewArtifact("Mana artifact", "", WithAbility(NewManaAbility(core.Blue)))
	g.PutOnBattlefield(artifact, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), artifact.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("tap-cost mana activation matched no-tap event")
	}
	planned := NewArtifact("Planned mana", "", WithActivatedAbility(AddMana(core.Green, 1), Tap()))
	g.PutOnBattlefield(planned, b.PlayerID())
	if err := g.AutoTapForCost(b.PlayerID(), core.ManaCost{Green: 1}); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("planned tap-cost mana activation matched no-tap event")
	}
}

func TestArtifactAbilityWithoutTapCostFailedPayment(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ArtifactAbilityActivatedWithoutTapCost(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	artifact := NewArtifact("Costly", "", WithActivatedAbility(GainLifeFor(You, Fixed(1)), ManaCostOf("{9}")))
	g.PutOnBattlefield(artifact, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), artifact.ID(), 0, nil); err == nil {
		t.Fatal("unpayable activation succeeded")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("failed payment published an activation event")
	}
}
