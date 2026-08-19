package astral

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func addPowerStruggle(t *testing.T, g *gametest.TestGame) *mage.Permanent {
	t.Helper()
	owner := g.GetPlayer(gametest.PlayerA)
	card, err := mage.CreateCard("Power Struggle")
	if err != nil {
		t.Fatalf("create Power Struggle: %v", err)
	}
	card.SetOwner(owner.PlayerID())
	return g.PutOnBattlefield(card, owner.PlayerID())
}

func addPowerStruggleCreature(g *gametest.TestGame, player gametest.PlayerRef, name string, artifact bool) *mage.Permanent {
	owner := g.GetPlayer(player)
	var opts []mage.CardOption
	if artifact {
		opts = append(opts, mage.WithCardType(core.TypeArtifact))
	}
	card := mage.NewCreature(name, "{1}", 1, 1, opts...)
	card.SetOwner(owner.PlayerID())
	return g.PutOnBattlefield(card, owner.PlayerID())
}

func addPowerStruggleArtifact(g *gametest.TestGame, player gametest.PlayerRef, name string) *mage.Permanent {
	owner := g.GetPlayer(player)
	card := mage.NewArtifact(name, "{1}")
	card.SetOwner(owner.PlayerID())
	return g.PutOnBattlefield(card, owner.PlayerID())
}

func addPowerStruggleLand(g *gametest.TestGame, player gametest.PlayerRef, name string) *mage.Permanent {
	owner := g.GetPlayer(player)
	card := mage.NewLand(name)
	card.SetOwner(owner.PlayerID())
	return g.PutOnBattlefield(card, owner.PlayerID())
}

func putPowerStruggleUpkeepTrigger(t *testing.T, g *gametest.TestGame, active gametest.PlayerRef) {
	t.Helper()
	activeIndex := 0
	if active == gametest.PlayerB {
		activeIndex = 1
	}
	g.SetActivePlayerIndex(activeIndex)
	activeID := g.GetPlayer(active).PlayerID()
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: activeID})
	g.PutTriggersOnStack()
}

func TestPowerStruggle_TriggersForEachPlayersUpkeepUsingTheActivePlayer(t *testing.T) {
	for _, tc := range []struct {
		name   string
		active gametest.PlayerRef
	}{
		{name: "Player A", active: gametest.PlayerA},
		{name: "Player B", active: gametest.PlayerB},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			addPowerStruggle(t, g)
			active := tc.active
			opponent := gametest.PlayerB
			if active == gametest.PlayerB {
				opponent = gametest.PlayerA
			}
			activeCreature := addPowerStruggleCreature(g, active, "Power Active Creature", false)
			opposingCreature := addPowerStruggleCreature(g, opponent, "Power Opposing Creature", false)
			g.SetRandomResults([]int{0, 0})

			putPowerStruggleUpkeepTrigger(t, g, active)
			obj := g.GetStack().Peek()
			if obj == nil {
				t.Fatal("upkeep trigger is not on the stack")
			}
			if want := []uuid.UUID{activeCreature.ID(), opposingCreature.ID()}; !slices.Equal(obj.Targets, want) {
				t.Fatalf("targets = %v, want active-player pair %v", obj.Targets, want)
			}
			g.ResolveTopOfStack()
			if activeCreature.ControllerID() != g.GetPlayer(opponent).PlayerID() || opposingCreature.ControllerID() != g.GetPlayer(active).PlayerID() {
				t.Fatal("upkeep permanents did not exchange controllers")
			}
		})
	}
}

func TestPowerStruggle_ArtifactCreatureCanPairByEitherSharedType(t *testing.T) {
	g := gametest.NewTestGame(t)
	addPowerStruggle(t, g)
	active := addPowerStruggleCreature(g, gametest.PlayerA, "Power Artifact Creature", true)
	creaturePartner := addPowerStruggleCreature(g, gametest.PlayerB, "Power Creature Partner", false)
	artifactPartner := addPowerStruggleArtifact(g, gametest.PlayerB, "Power Artifact Partner")
	g.SetRandomResults([]int{0, 1})

	putPowerStruggleUpkeepTrigger(t, g, gametest.PlayerA)
	obj := g.GetStack().Peek()
	if obj == nil {
		t.Fatal("upkeep trigger is not on the stack")
	}
	if want := []uuid.UUID{active.ID(), artifactPartner.ID()}; !slices.Equal(obj.Targets, want) {
		t.Fatalf("targets = %v, want %v (other compatible partner %v)", obj.Targets, want, creaturePartner.ID())
	}
	g.ResolveTopOfStack()
	if active.ControllerID() != g.GetPlayer(gametest.PlayerB).PlayerID() || artifactPartner.ControllerID() != g.GetPlayer(gametest.PlayerA).PlayerID() {
		t.Fatal("artifact-shared pair did not exchange controllers")
	}
}

func TestPowerStruggle_NoCompatiblePairHasNoTargets(t *testing.T) {
	g := gametest.NewTestGame(t)
	addPowerStruggle(t, g)
	land := addPowerStruggleLand(g, gametest.PlayerA, "Power Only Land")
	creature := addPowerStruggleCreature(g, gametest.PlayerB, "Power Only Creature", false)

	putPowerStruggleUpkeepTrigger(t, g, gametest.PlayerA)
	obj := g.GetStack().Peek()
	if obj == nil {
		t.Fatal("upkeep trigger is not on the stack")
	}
	if len(obj.Targets) != 0 {
		t.Fatalf("targets = %v, want none", obj.Targets)
	}
	g.ResolveTopOfStack()
	if land.ControllerID() != g.GetPlayer(gametest.PlayerA).PlayerID() || creature.ControllerID() != g.GetPlayer(gametest.PlayerB).PlayerID() {
		t.Fatal("no-pair trigger changed control")
	}
}

func TestPowerStruggle_MissingTargetPreventsPartialExchange(t *testing.T) {
	g := gametest.NewTestGame(t)
	addPowerStruggle(t, g)
	activeLand := addPowerStruggleLand(g, gametest.PlayerA, "Power Active Land")
	opposingLand := addPowerStruggleLand(g, gametest.PlayerB, "Power Opposing Land")
	g.SetRandomResults([]int{0, 0})

	putPowerStruggleUpkeepTrigger(t, g, gametest.PlayerA)
	g.DestroyPermanent(opposingLand)
	g.ResolveTopOfStack()
	if activeLand.ControllerID() != g.GetPlayer(gametest.PlayerA).PlayerID() {
		t.Fatal("one missing target caused a partial exchange")
	}
}

func TestPowerStruggle_SourceLeavingDoesNotStopStackedExchange(t *testing.T) {
	g := gametest.NewTestGame(t)
	source := addPowerStruggle(t, g)
	activeCreature := addPowerStruggleCreature(g, gametest.PlayerA, "Power Active Creature", false)
	opposingCreature := addPowerStruggleCreature(g, gametest.PlayerB, "Power Opposing Creature", false)
	g.SetRandomResults([]int{0, 0})

	putPowerStruggleUpkeepTrigger(t, g, gametest.PlayerA)
	g.DestroyPermanent(source)
	g.ResolveTopOfStack()
	if activeCreature.ControllerID() != g.GetPlayer(gametest.PlayerB).PlayerID() || opposingCreature.ControllerID() != g.GetPlayer(gametest.PlayerA).PlayerID() {
		t.Fatal("source leaving stopped the stacked exchange")
	}
}
