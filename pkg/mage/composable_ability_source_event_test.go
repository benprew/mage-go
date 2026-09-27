package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAbilityActivatedSourceCaptureAfterSacrificeAndReturn(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := BindDamageSource("activated source")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ActivatedAbilityEvent().WithSource(source),
		DealDamageFrom(FromDamageSource(source), You, Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, b.PlayerID())
	artifact := NewArtifact("Source", "", WithKeyword(core.Lifelink), WithAbility(Activate(Costs(SacrificeSourceCost()), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(artifact, a.PlayerID())
	old := g.FindPermanent(artifact.ID()).incarnationID
	if err := g.ActivateAbilityByIndex(a.PlayerID(), artifact.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(artifact, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack = %d, want 2", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(b.PlayerID()).Life() != 18 {
			t.Fatalf("life = %d, want 18", game.GetPlayer(b.PlayerID()).Life())
		}
		if game.GetPlayer(a.PlayerID()).Life() != 23 {
			t.Fatalf("source controller life = %d, want 23", game.GetPlayer(a.PlayerID()).Life())
		}
		if game.FindPermanent(artifact.ID()).incarnationID == old {
			t.Fatal("source return kept old incarnation")
		}
	}
}

func TestAbilityActivatedSourceDeclaration(t *testing.T) {
	source := BindDamageSource("source")
	good := NewEnchantment("Good", "", WithAbility(WhenEvent(ActivatedAbilityEvent().WithSource(source), DealDamageFrom(FromDamageSource(source), You, Fixed(1)))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(ActivatedAbilityEvent().WithSource(source)); got.Role != "event-pattern" || len(got.Children) != 1 {
		t.Fatal("source capture missing from inspection")
	}
	for _, pattern := range []EventPattern{ActivatedAbilityEvent().WithSource(nil), ActivatedAbilityEvent().WithSource(source).WithSource(source)} {
		bad := NewEnchantment("Bad", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("invalid source capture passed registration")
		}
	}
	bad := NewEnchantment("Bad", "", WithAbility(WhenEvent(ActivatedAbilityEvent(), DealDamageFrom(FromDamageSource(source), You, Fixed(1)))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
		t.Fatal("unbound source passed registration")
	}
}

func TestAbilityActivatedSourceSurvivesPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := BindDamageSource("source")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ActivatedAbilityEvent().WithSource(source),
		DealDamageFrom(FromDamageSource(source), You, Fixed(1)))))
	g.PutOnBattlefield(watcher, b.PlayerID())
	artifact := NewArtifact("Source", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(artifact, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), artifact.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	g.PhaseOut(artifact.ID())
	g.ResolveStack()
	if b.Life() != 19 {
		t.Fatalf("life = %d, want 19", b.Life())
	}
}
