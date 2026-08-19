package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestRandomActivePlayerExchangePairHandlesOverlappingTypes(t *testing.T) {
	g, a, b := newRandomTargetGame()
	source := NewEnchantment("Pair Trigger", "{3}", WithAbility(
		NewTriggered(EvtUpkeep, false, ExchangeControlOfTargetsSharingPermanentType()).
			AddTarget(TargetRandomActivePlayerExchangePair()),
	))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	artifactCreature := NewCreature("Artifact Creature", "{2}", 2, 2, WithCardType(TypeArtifact))
	artifactCreature.SetOwner(a.PlayerID())
	first := g.PutOnBattlefield(artifactCreature, a.PlayerID())
	land := NewLand("Land")
	land.SetOwner(a.PlayerID())
	g.PutOnBattlefield(land, a.PlayerID())
	opposingCreature := addRandomTargetCreature(g, b, "Opposing Creature")
	opposingArtifactCard := NewArtifact("Opposing Artifact", "{1}")
	opposingArtifactCard.SetOwner(b.PlayerID())
	opposingArtifact := g.PutOnBattlefield(opposingArtifactCard, b.PlayerID())
	g.SetRandomResults([]int{0, 1})

	g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	obj := g.Stack().Peek()
	if obj == nil {
		t.Fatal("expected trigger on stack")
	}
	if got, want := obj.Targets, []uuid.UUID{first.ID(), opposingArtifact.ID()}; !slices.Equal(got, want) {
		t.Fatalf("random exchange pair = %v, want %v (other partner %v)", got, want, opposingCreature.ID())
	}
	if len(obj.TargetSpecs) != 2 {
		t.Fatalf("target specs = %d, want 2", len(obj.TargetSpecs))
	}
	g.DestroyPermanent(opposingArtifact)
	g.ResolveTopOfStack()
	if got := g.FindPermanent(first.ID()).ControllerID(); got != a.PlayerID() {
		t.Fatalf("lost partner caused partial exchange: controller = %s, want %s", got, a.PlayerID())
	}
}

func TestRandomActivePlayerExchangePairWithNoCompatiblePair(t *testing.T) {
	g, a, b := newRandomTargetGame()
	source := NewEnchantment("Pair Trigger", "{3}", WithAbility(
		NewTriggered(EvtUpkeep, false, ExchangeControlOfTargetsSharingPermanentType()).
			AddTarget(TargetRandomActivePlayerExchangePair()),
	))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	land := NewLand("Only Land")
	land.SetOwner(a.PlayerID())
	g.PutOnBattlefield(land, a.PlayerID())
	addRandomTargetCreature(g, b, "Only Creature")

	g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	obj := g.Stack().Peek()
	if obj == nil {
		t.Fatal("expected no-pair trigger on stack")
	}
	if len(obj.Targets) != 0 {
		t.Fatalf("no-pair trigger targets = %v, want none", obj.Targets)
	}
}
