package astral

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func addGoblinPolkaBand(t *testing.T, g *gametest.TestGame) *mage.Permanent {
	t.Helper()
	player := g.GetPlayer(gametest.PlayerA)
	card, err := mage.CreateCard("Goblin Polka Band")
	if err != nil {
		t.Fatalf("create Goblin Polka Band: %v", err)
	}
	card.SetOwner(player.PlayerID())
	perm := g.PutOnBattlefield(card, player.PlayerID())
	perm.RevokeBaseAttr(core.AttrSummonSick)
	return perm
}

func addPolkaCreature(g *gametest.TestGame, player gametest.PlayerRef, name string, goblin bool) *mage.Permanent {
	owner := g.GetPlayer(player)
	var opts []mage.CardOption
	if goblin {
		opts = append(opts, mage.WithSubTypes("Goblin"))
	}
	card := mage.NewCreature(name, "{1}", 1, 1, opts...)
	card.SetOwner(owner.PlayerID())
	return g.PutOnBattlefield(card, owner.PlayerID())
}

func activateGoblinPolkaBand(t *testing.T, g *gametest.TestGame, source *mage.Permanent) error {
	t.Helper()
	return g.ActivateAbilityByIndex(g.GetPlayer(gametest.PlayerA).PlayerID(), source.ID(), 0, nil)
}

func TestGoblinPolkaBand_ZeroTargetsCostsOnlyTwoMana(t *testing.T) {
	g := gametest.NewTestGame(t)
	source := addGoblinPolkaBand(t, g)
	player := g.GetPlayer(gametest.PlayerA)
	player.ManaPool().Add(core.Colorless, 2)
	g.ChooseNumber(gametest.PlayerA, 0)

	if err := activateGoblinPolkaBand(t, g, source); err != nil {
		t.Fatalf("activate with zero targets: %v", err)
	}
	obj := g.GetStack().Peek()
	if obj == nil {
		t.Fatal("ability is not on the stack")
	}
	if len(obj.Targets) != 0 {
		t.Fatalf("targets = %v, want none", obj.Targets)
	}
	if !source.Tapped {
		t.Fatal("source tap cost was not paid")
	}
	if got := player.ManaPool().TotalMana(); got != 0 {
		t.Fatalf("mana remaining = %d, want 0", got)
	}
}

func TestGoblinPolkaBand_RandomTargetsAreDistinctAndFixedAtActivation(t *testing.T) {
	g := gametest.NewTestGame(t)
	source := addGoblinPolkaBand(t, g)
	first := addPolkaCreature(g, gametest.PlayerB, "Polka First", false)
	second := addPolkaCreature(g, gametest.PlayerB, "Polka Second", true)
	third := addPolkaCreature(g, gametest.PlayerB, "Polka Third", false)
	player := g.GetPlayer(gametest.PlayerA)
	player.ManaPool().Add(core.Colorless, 2)
	player.ManaPool().Add(core.Red, 2)
	g.ChooseNumber(gametest.PlayerA, 2)
	g.SetRandomResults([]int{2, 1})

	if err := activateGoblinPolkaBand(t, g, source); err != nil {
		t.Fatalf("activate for two targets: %v", err)
	}
	obj := g.GetStack().Peek()
	if obj == nil {
		t.Fatal("ability is not on the stack")
	}
	if want := []uuid.UUID{second.ID(), first.ID()}; !slices.Equal(obj.Targets, want) {
		t.Fatalf("targets = %v, want %v (unchosen third was %v)", obj.Targets, want, third.ID())
	}
	if got := player.ManaPool().TotalMana(); got != 0 {
		t.Fatalf("mana remaining = %d, want 0", got)
	}

	g.DestroyPermanent(first)
	g.ResolveTopOfStack()
	if !second.Tapped {
		t.Fatal("surviving frozen target was not tapped")
	}
	if third.Tapped {
		t.Fatal("unchosen creature was tapped")
	}
}

func TestGoblinPolkaBand_PerTargetManaFailureIsAtomic(t *testing.T) {
	g := gametest.NewTestGame(t)
	source := addGoblinPolkaBand(t, g)
	addPolkaCreature(g, gametest.PlayerB, "Polka First", false)
	addPolkaCreature(g, gametest.PlayerB, "Polka Second", false)
	player := g.GetPlayer(gametest.PlayerA)
	player.ManaPool().Add(core.Colorless, 2)
	player.ManaPool().Add(core.Red, 1)
	g.ChooseNumber(gametest.PlayerA, 2)
	g.SetRandomResults([]int{1, 1})

	if err := activateGoblinPolkaBand(t, g, source); err == nil {
		t.Fatal("activation succeeded without enough red mana")
	}
	if source.Tapped {
		t.Fatal("failed activation paid the source tap cost")
	}
	if got := player.ManaPool().TotalMana(); got != 3 {
		t.Fatalf("failed activation changed mana pool to %d, want 3", got)
	}
	if g.GetStack().Peek() != nil {
		t.Fatal("failed activation created a stack object")
	}
}

func TestGoblinPolkaBand_OnlyNewlyTappedGoblinsSkipOneUntap(t *testing.T) {
	g := gametest.NewTestGame(t)
	source := addGoblinPolkaBand(t, g)
	nonGoblin := addPolkaCreature(g, gametest.PlayerB, "Polka Human", false)
	freshGoblin := addPolkaCreature(g, gametest.PlayerB, "Polka Fresh Goblin", true)
	alreadyTappedGoblin := addPolkaCreature(g, gametest.PlayerB, "Polka Tapped Goblin", true)
	alreadyTappedGoblin.Tapped = true
	player := g.GetPlayer(gametest.PlayerA)
	player.ManaPool().Add(core.Colorless, 2)
	player.ManaPool().Add(core.Red, 3)
	g.ChooseNumber(gametest.PlayerA, 3)
	g.SetRandomResults([]int{1, 1, 1})

	if err := activateGoblinPolkaBand(t, g, source); err != nil {
		t.Fatalf("activate for three targets: %v", err)
	}
	g.ResolveTopOfStack()
	if !nonGoblin.Tapped || !freshGoblin.Tapped || !alreadyTappedGoblin.Tapped {
		t.Fatal("all surviving targets should be tapped after resolution")
	}

	g.SetActivePlayerIndex(1)
	g.Game.RunStepWithPriority(core.Untap)
	if nonGoblin.Tapped {
		t.Fatal("non-Goblin did not untap normally")
	}
	if !freshGoblin.Tapped {
		t.Fatal("Goblin newly tapped this way did not skip its next untap")
	}
	if alreadyTappedGoblin.Tapped {
		t.Fatal("Goblin that was already tapped incorrectly skipped its untap")
	}

	g.Game.RunStepWithPriority(core.Untap)
	if freshGoblin.Tapped {
		t.Fatal("Goblin skipped more than one untap")
	}
}

func TestGoblinPolkaBand_SkipFollowsGoblinToItsThenController(t *testing.T) {
	g := gametest.NewTestGame(t)
	source := addGoblinPolkaBand(t, g)
	goblin := addPolkaCreature(g, gametest.PlayerB, "Polka Stolen Goblin", true)
	player := g.GetPlayer(gametest.PlayerA)
	player.ManaPool().Add(core.Colorless, 2)
	player.ManaPool().Add(core.Red, 1)
	g.ChooseNumber(gametest.PlayerA, 1)
	g.SetRandomResults([]int{1})

	if err := activateGoblinPolkaBand(t, g, source); err != nil {
		t.Fatalf("activate for Goblin target: %v", err)
	}
	g.ResolveTopOfStack()
	g.AddControlEffect(mage.ControlEffectSpec{
		SourceID: source.ID(), TargetID: goblin.ID(), ControllerID: player.PlayerID(), Duration: core.Indefinite,
	})
	if goblin.ControllerID() != player.PlayerID() {
		t.Fatal("control change did not apply")
	}

	g.SetActivePlayerIndex(1)
	g.Game.RunStepWithPriority(core.Untap)
	if !goblin.Tapped {
		t.Fatal("former controller's untap step affected the Goblin")
	}
	g.SetActivePlayerIndex(0)
	g.Game.RunStepWithPriority(core.Untap)
	if !goblin.Tapped {
		t.Fatal("Goblin did not skip its then-controller's next untap")
	}
	g.Game.RunStepWithPriority(core.Untap)
	if goblin.Tapped {
		t.Fatal("Goblin did not untap on its then-controller's following untap")
	}
}
