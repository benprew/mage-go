package astral

import (
	"maps"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func castOrcishCatapult(t *testing.T, g *gametest.TestGame, x int) {
	t.Helper()
	player := g.GetPlayer(gametest.PlayerA)
	card, err := mage.CreateCard("Orcish Catapult")
	if err != nil {
		t.Fatalf("create Orcish Catapult: %v", err)
	}
	card.SetOwner(player.PlayerID())
	player.AddToHand(card)
	player.ManaPool().Add(core.Red, 2)
	player.ManaPool().Add(core.Colorless, x)
	if err := g.CastSpellByName(player.PlayerID(), "Orcish Catapult", nil, x); err != nil {
		t.Fatalf("cast Orcish Catapult with X=%d: %v", x, err)
	}
}

func addCatapultCreature(g *gametest.TestGame, player gametest.PlayerRef, name string) *mage.Permanent {
	p := g.GetPlayer(player)
	card := mage.NewCreature(name, "{1}", 4, 4)
	card.SetOwner(p.PlayerID())
	return g.PutOnBattlefield(card, p.PlayerID())
}

func TestOrcishCatapult_FixesRandomTargetsAndPositiveDistributionAtCast(t *testing.T) {
	g := gametest.NewTestGame(t)
	first := addCatapultCreature(g, gametest.PlayerB, "Catapult First")
	addCatapultCreature(g, gametest.PlayerB, "Catapult Second")
	third := addCatapultCreature(g, gametest.PlayerB, "Catapult Third")
	g.SetRandomResults([]int{1, 2, 0, 1, 1})

	castOrcishCatapult(t, g, 4)

	obj := g.GetStack().Peek()
	if obj == nil {
		t.Fatal("Orcish Catapult is not on the stack")
	}
	if want := []uuid.UUID{third.ID(), first.ID()}; !slices.Equal(obj.Targets, want) {
		t.Fatalf("targets = %v, want %v", obj.Targets, want)
	}
	if want := map[uuid.UUID]int{third.ID(): 1, first.ID(): 3}; !maps.Equal(obj.CounterDistribution, want) {
		t.Fatalf("distribution = %v, want %v", obj.CounterDistribution, want)
	}

	g.DestroyPermanent(first)
	g.ResolveTopOfStack()
	if got := third.Counters[core.M0M1]; got != 1 {
		t.Fatalf("surviving target counters = %d, want its frozen assignment 1", got)
	}
}

func TestOrcishCatapult_ZeroXHasNoTargetsOrCounters(t *testing.T) {
	g := gametest.NewTestGame(t)
	addCatapultCreature(g, gametest.PlayerB, "Catapult Target")
	castOrcishCatapult(t, g, 0)

	obj := g.GetStack().Peek()
	if obj == nil || len(obj.Targets) != 0 || len(obj.CounterDistribution) != 0 {
		t.Fatalf("X=0 stack state: targets=%v distribution=%v", obj.Targets, obj.CounterDistribution)
	}
	g.ResolveTopOfStack()
}

func TestOrcishCatapult_CounterPlacementUsesReplacementPipeline(t *testing.T) {
	g := gametest.NewTestGame(t)
	target := addCatapultCreature(g, gametest.PlayerB, "Catapult Doubled")
	g.AddCounterDoubler(target.ID(), core.M0M1, mage.PermanentFilter{})
	g.SetRandomResults([]int{0, 0})

	castOrcishCatapult(t, g, 1)
	g.ResolveTopOfStack()

	if got := target.Counters[core.M0M1]; got != 2 {
		t.Fatalf("replacement-adjusted counters = %d, want 2", got)
	}
}

func TestOrcishCatapult_AllTargetsDisappearFizzle(t *testing.T) {
	g := gametest.NewTestGame(t)
	target := addCatapultCreature(g, gametest.PlayerB, "Catapult Vanishing")
	g.SetRandomResults([]int{0, 0})
	castOrcishCatapult(t, g, 1)

	g.DestroyPermanent(target)
	g.ResolveTopOfStack()

	if g.FindPermanent(target.ID()) != nil {
		t.Fatal("destroyed target unexpectedly returned")
	}
	if got := g.GetPlayer(gametest.PlayerA).Graveyard(); len(got) != 1 || got[0].Name() != "Orcish Catapult" {
		t.Fatalf("resolved spell graveyard = %v", got)
	}
}
