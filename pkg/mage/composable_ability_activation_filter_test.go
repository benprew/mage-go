package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestActivatedAbilityArtifactOpponentFilter(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	actor := BindPlayer("actor")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ActivatedAbilityEvent().FromArtifact().ByOpponent().WithActivator(actor),
		GainLifeFor(FromPlayer(actor), Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	other := NewArtifact("Other", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(other, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), other.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	artifact := NewArtifact("Artifact", "", WithAbility(Activate(Costs(TapCost(This)), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(artifact, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), artifact.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	creature := NewCreature("Creature", "", 1, 1, WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(creature, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), creature.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 4 {
		t.Fatalf("stack = %d, want 4", got)
	}
	clone := g.Clone()
	clone.ResolveStack()
	if b.Life() != 20 {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	if b.Life() != 23 {
		t.Fatalf("life = %d, want 23", b.Life())
	}
	if a.Life() != 21 {
		t.Fatalf("life = %d, want 21", a.Life())
	}
}

func TestActivatedAbilityArtifactOpponentFilterDefinition(t *testing.T) {
	good := NewEnchantment("Good", "", WithAbility(WhenEvent(
		ActivatedAbilityEvent().FromArtifact().ByOpponent(), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	for _, pattern := range []EventPattern{
		ActivatedAbilityEvent().FromArtifact().FromArtifact(),
		ActivatedAbilityEvent().ByOpponent().ByOpponent(),
	} {
		bad := NewEnchantment("Bad", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("duplicate filter passed registration")
		}
	}
}

func TestActivatedAbilityArtifactOpponentSourceLKI(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := BindDamageSource("source")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ActivatedAbilityEvent().FromArtifact().ByOpponent().WithSource(source),
		DealDamageFrom(FromDamageSource(source), You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	artifact := NewArtifact("Artifact", "", WithKeyword(core.Lifelink), WithAbility(
		Activate(Costs(SacrificeSourceCost()), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(artifact, b.PlayerID())
	old := g.FindPermanent(artifact.ID()).incarnationID
	if err := g.ActivateAbilityByIndex(b.PlayerID(), artifact.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(artifact, a.PlayerID())
	if g.FindPermanent(artifact.ID()).incarnationID == old {
		t.Fatal("return kept old incarnation")
	}
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 2 {
		t.Fatalf("stack = %d, want 2", got)
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
	if a.Life() != 18 || b.Life() != 23 {
		t.Fatalf("life = %d/%d, want 18/23", a.Life(), b.Life())
	}
}
