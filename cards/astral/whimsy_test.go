package astral

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestWhimsy_EachRandomActionCanBeSelected(t *testing.T) {
	actions := []string{
		"bounce", "untap", "tap", "damage four", "draw three", "destroy artifact and gain life",
		"destroy artifact or enchantment", "gain three", "prevent three", "destroy creature or land",
		"mill two", "Wasp", "destroy all", "Suleiman", "Pandora", "discard", "Fog", "Sindbad",
	}
	for action, name := range actions {
		t.Run(fmt.Sprintf("%02d_%s", action+1, name), func(t *testing.T) {
			g := gametest.NewTestGame(t)
			g.AddCard(core.ZoneHand, gametest.PlayerA, "Whimsy")
			g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Whimsy", 6)
			g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Whimsy", 6)
			g.SetRandomResults([]int{action, 0, 0, 0, 0})
			g.SetCoinFlipResults([]bool{true, false})
			g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Whimsy", 1)
			g.StopAt(1, core.BeginCombat)
			g.Execute()
		})
	}
}

func TestWhimsy_PerformsXIndependentActionsInOrder(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Whimsy")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Whimsy", 6)
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Whimsy", 6)
	g.SetRandomResults([]int{7, 0, 7, 1, 7, 0})
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Whimsy", 3)
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertLife(gametest.PlayerA, 26)
	g.AssertLife(gametest.PlayerB, 23)
}

func TestWhimsy_NoEligiblePermanentIsSafe(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Whimsy")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Whimsy", 3)
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Whimsy", 3)
	g.SetRandomResults([]int{0, 1, 2, 5, 6, 9})
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Whimsy", 6)
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertLife(gametest.PlayerA, 20)
	g.AssertLife(gametest.PlayerB, 20)
}

func TestWhimsyCombatPrevention(t *testing.T) {
	g := gametest.NewTestGame(t)
	source := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	controller := g.GetPlayer(gametest.PlayerA).PlayerID()
	g.SetRandomResults([]int{1})
	if err := applyWhimsyAction(g.Game, source, controller, 16); err != nil {
		t.Fatal(err)
	}
	if g.RandIntn(2) != 1 {
		t.Fatal("global prevention consumed a random choice")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source))
	g.DealDamageToPlayer(g.GetPlayer(gametest.PlayerA), 2, uuid.Nil)
	g.Attack(1, gametest.PlayerA, "Hill Giant")
	g.Attack(3, gametest.PlayerA, "Hill Giant")
	g.StopAt(3, core.EndCombat)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 18)
	g.AssertLife(gametest.PlayerB, 17)
}

func TestWhimsyTypedDestruction(t *testing.T) {
	for _, action := range []int{6, 9} {
		for _, state := range []string{"normal", "regeneration", "indestructible", "empty"} {
			t.Run(fmt.Sprintf("%d/%s", action, state), func(t *testing.T) {
				g := gametest.NewTestGame(t)
				controller := g.GetPlayer(gametest.PlayerA).PlayerID()
				var targetID uuid.UUID
				if state != "empty" {
					targetID = g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Obsianus Golem")
					if state == "regeneration" {
						g.AddRegenerationShield(targetID)
					}
					if state == "indestructible" {
						g.FindPermanent(targetID).GrantBaseAttr(core.Indestructible)
					}
					g.FindPermanent(targetID).GrantBaseAttr(core.Shroud)
				}
				g.SetRandomResults([]int{1, 0})
				if err := applyWhimsyAction(g.Game, uuid.New(), controller, action); err != nil {
					t.Fatal(err)
				}
				if state == "empty" {
					if got := g.RandIntn(2); got != 1 {
						t.Fatal("empty selection consumed randomness")
					}
					return
				}
				survives := state == "indestructible" || (state == "regeneration" && action == 6)
				if got := g.FindPermanent(targetID) != nil; got != survives {
					t.Fatalf("survived = %v, want %v", got, survives)
				}
			})
		}
	}
}

func TestWhimsyBoomerangSelection(t *testing.T) {
	t.Run("exclude enchanted permanent", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		creature := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		aura := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Holy Strength")
		g.Attach(aura, creature)
		g.FindPermanent(aura).GrantBaseAttr(core.Shroud)
		g.SetRandomResults([]int{0, 1})
		if err := applyWhimsyAction(g.Game, uuid.New(), g.GetPlayer(gametest.PlayerA).PlayerID(), 0); err != nil {
			t.Fatal(err)
		}
		if g.FindPermanent(creature) == nil {
			t.Fatal("selected an enchanted permanent")
		}
		g.AssertHandCount(gametest.PlayerB, "Holy Strength", 1)
		if g.RandIntn(2) != 1 {
			t.Fatal("selection did not consume exactly one random value")
		}
	})
	t.Run("empty", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.SetRandomResults([]int{1})
		if err := applyWhimsyAction(g.Game, uuid.New(), g.GetPlayer(gametest.PlayerA).PlayerID(), 0); err != nil {
			t.Fatal(err)
		}
		if g.RandIntn(2) != 1 {
			t.Fatal("empty selection consumed a random value")
		}
	})
}
