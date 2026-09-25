package jumpstart

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestAffaGuardHoundTypedDefinition(t *testing.T) {
	card, err := mage.CreateCard("Affa Guard Hound")
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if len(card.CastTargets()) != 0 {
		t.Fatal("entry target belongs to the trigger")
	}
	var trigger mage.TriggeredAbility
	for _, ability := range card.Abilities() {
		if candidate, ok := mage.UnwrapAbility(ability).(mage.TriggeredAbility); ok {
			trigger = candidate
		}
	}
	if trigger == nil || trigger.IsOptional() || len(trigger.Targets()) != 1 {
		t.Fatal("expected a mandatory trigger with one target")
	}
	if _, ok := trigger.Targets()[0].(mage.TargetBinding); !ok {
		t.Fatal("missing typed creature target")
	}
	meta := mage.InspectDefinition(trigger.Effects()[0])
	if meta.Role != "until" || meta.Properties.PowerBoost != 0 || meta.Properties.ToughnessBoost != 3 || meta.Properties.Mass || meta.Properties.Outcome != mage.OutcomeBenefit {
		t.Fatalf("boost metadata = %+v", meta)
	}
	g := gametest.NewTestGame(t)
	a := g.GetPlayer(gametest.PlayerA)
	own := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	opposing := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Sol Ring")
	possible := trigger.Targets()[0].Possible(a.PlayerID(), card, g.Game)
	if len(possible) != 2 || !slices.Contains(possible, own) || !slices.Contains(possible, opposing) {
		t.Fatalf("creature candidates = %v, want both creatures only", possible)
	}
}

func TestAffaGuardHoundWithoutCasting(t *testing.T) {
	g := gametest.NewTestGame(t)
	a := g.GetPlayer(gametest.PlayerA)
	card, err := mage.CreateCard("Affa Guard Hound")
	if err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	g.ChooseTarget(gametest.PlayerA, card.Name())
	g.PutOnBattlefield(card, a.PlayerID())
	g.PutTriggersOnStack()
	g.AssertPowerToughness(gametest.PlayerA, card.Name(), 2, 2)
	if g.StackSize() != 1 {
		t.Fatalf("entry trigger stack size = %d, want 1", g.StackSize())
	}
	g.ResolveStack()
	g.AssertPowerToughness(gametest.PlayerA, card.Name(), 2, 5)
}

func TestAffaGuardHoundEntryTarget(t *testing.T) {
	for _, scenario := range []string{"self", "opponent", "source leaves", "target returns", "target phases"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
			card, err := mage.CreateCard("Affa Guard Hound")
			if err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.White, 3)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatalf("cast without an entry target: %v", err)
			}
			if len(g.StackPeek().Targets) != 0 {
				t.Fatal("creature spell has a target")
			}
			targetID := card.ID()
			targetName := card.Name()
			owner := gametest.PlayerA
			if scenario != "self" {
				targetID = g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
				targetName = "Grizzly Bears"
				owner = gametest.PlayerB
			}
			g.ChooseTarget(gametest.PlayerA, targetName)
			g.ResolveTopOfStack()
			g.AssertPermanentCount(gametest.PlayerA, card.Name(), 1)
			g.AssertPowerToughness(owner, targetName, 2, 2)
			if g.StackSize() != 1 {
				t.Fatalf("entry trigger stack size = %d, want 1", g.StackSize())
			}
			trigger := g.StackPeek()
			if trigger.Controller != a.PlayerID() || len(trigger.Targets) != 1 || trigger.Targets[0] != targetID {
				t.Fatal("entry trigger has the wrong controller or target")
			}
			wantToughness := 5
			switch scenario {
			case "source leaves":
				g.DestroyPermanent(g.FindPermanent(card.ID()))
			case "target returns":
				targetCard := g.FindPermanent(targetID).Card
				g.RemoveFromBattlefield(g.FindPermanent(targetID))
				g.PutOnBattlefield(targetCard, b.PlayerID())
				wantToughness = 2
			case "target phases":
				g.PhaseOut(targetID)
				wantToughness = 2
			}
			g.ResolveStack()
			if scenario == "target phases" {
				g.PhaseIn([]uuid.UUID{targetID})
			}
			g.AssertPowerToughness(owner, targetName, 2, wantToughness)
			g.StopAt(2, core.Upkeep)
			g.Execute()
			g.AssertPowerToughness(owner, targetName, 2, 2)
		})
	}
}
