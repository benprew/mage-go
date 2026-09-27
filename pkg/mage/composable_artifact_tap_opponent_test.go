package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestArtifactBecomesTappedByOpponent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	artifact := BindPermanent("artifact")
	controller := BindPlayer("controller")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ArtifactBecomesTapped().ByOpponent().WithArtifact(artifact).WithController(controller),
		Sequence(AddCountersTo(From(artifact), core.Charge, Fixed(1)), GainLifeFor(FromPlayer(controller), Fixed(1))))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	own := NewArtifact("Own", "")
	own.SetOwner(b.PlayerID())
	g.PutOnBattlefield(own, a.PlayerID())
	g.TapPermanent(g.FindPermanent(own.ID()))
	opponent := NewArtifact("Opponent", "")
	opponent.SetOwner(a.PlayerID())
	g.PutOnBattlefield(opponent, b.PlayerID())
	g.TapPermanent(g.FindPermanent(opponent.ID()))
	g.TapPermanent(g.FindPermanent(opponent.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack=%d, want 1", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(opponent.ID()))
	g.PutOnBattlefield(opponent, a.PlayerID())
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("could not copy trigger")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if b.Life() != 20 {
		t.Fatal("clone changed original life")
	}
	g.ResolveStack()
	if b.Life() != 22 || a.Life() != 20 {
		t.Fatalf("life A=%d B=%d", a.Life(), b.Life())
	}
	if got := g.FindPermanent(opponent.ID()).Counters[core.Charge]; got != 0 {
		t.Fatalf("returned artifact has %d counters", got)
	}
}

func TestArtifactBecomesTappedByOpponentPhasingAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ArtifactBecomesTapped().ByOpponent(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	artifact := NewArtifact("Artifact", "")
	g.PutOnBattlefield(artifact, b.PlayerID())
	g.PhaseOut(watcher.ID())
	g.TapPermanent(g.FindPermanent(artifact.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased watcher triggered")
	}
	if err := ValidateCardDefinition("Valid", watcher); err != nil {
		t.Fatal(err)
	}
	bad := NewEnchantment("Bad", "", WithAbility(WhenEvent(
		ArtifactBecomesTapped().ByOpponent().ByOpponent(), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil || !strings.Contains(err.Error(), "filter") {
		t.Fatalf("duplicate filter validation=%v", err)
	}
}
