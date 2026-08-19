package astral

import (
	"slices"
	"testing"

	_ "github.com/benprew/mage-go/cards/limited"
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func activateFaerieDragon(t *testing.T, g *gametest.TestGame, sourceID string, rolls ...int) {
	t.Helper()
	g.SetRandomResults(rolls)
	player := g.GetPlayer(gametest.PlayerA)
	player.ManaPool().Add(core.Green, 3)
	id := g.FindPermanentByName(sourceID, player.PlayerID()).ID()
	if err := g.ActivateAbilityByIndex(player.PlayerID(), id, 0, nil); err != nil {
		t.Fatalf("activate Faerie Dragon: %v", err)
	}
	g.ResolveStack()
}

func TestFaerieDragonRandomCombatEffects(t *testing.T) {
	t.Run("berserk", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Faerie Dragon")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		activateFaerieDragon(t, g, "Faerie Dragon", 0, 1)
		g.AssertPowerToughness(gametest.PlayerB, "Grizzly Bears", 4, 2)
		g.AssertHasAbility(gametest.PlayerB, "Grizzly Bears", core.Trample, true)
	})

	t.Run("blood lust", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Faerie Dragon")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Stone")
		activateFaerieDragon(t, g, "Faerie Dragon", 2, 1)
		g.AssertPowerToughness(gametest.PlayerB, "Wall of Stone", 4, 4)
	})

	t.Run("blood lust below five toughness", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Faerie Dragon")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		activateFaerieDragon(t, g, "Faerie Dragon", 2, 1)
		g.AssertPowerToughness(gametest.PlayerB, "Grizzly Bears", 6, 1)
	})

	for _, tc := range []struct {
		name   string
		action int
		power  int
		tough  int
	}{
		{"plus three", 8, 5, 5},
		{"minus power", 14, 0, 2},
		{"base zero two", 17, 0, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Faerie Dragon")
			g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
			activateFaerieDragon(t, g, "Faerie Dragon", tc.action, 1)
			g.AssertPowerToughness(gametest.PlayerB, "Grizzly Bears", tc.power, tc.tough)
		})
	}
}

func TestFaerieDragonBerserkDestroysCreatureThatAttacked(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetRandomResults([]int{0, 1})
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Faerie Dragon")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.Attack(2, gametest.PlayerB, "Grizzly Bears")
	g.ActivateAbility(2, core.PostcombatMain, gametest.PlayerA, "Faerie Dragon")
	g.StopAt(3, core.PrecombatMain)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}

func TestFaerieDragonRandomKeywordEffects(t *testing.T) {
	for _, tc := range []struct {
		name    string
		action  int
		keyword core.Attr
	}{
		{"flying", 7, core.Flying},
		{"banding", 9, core.Banding},
		{"cant regenerate", 12, core.CantRegenerate},
		{"unblockable", 13, core.UnblockableKW},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Faerie Dragon")
			g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
			activateFaerieDragon(t, g, "Faerie Dragon", tc.action, 1)
			g.AssertHasAbility(gametest.PlayerB, "Grizzly Bears", tc.keyword, true)
		})
	}
}

func TestFaerieDragonRandomColorEffects(t *testing.T) {
	for _, tc := range []struct {
		action int
		color  core.Color
	}{
		{3, core.Green}, {4, core.White}, {5, core.Red}, {10, core.Black}, {11, core.Blue},
	} {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Faerie Dragon")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Sol Ring")
		activateFaerieDragon(t, g, "Faerie Dragon", tc.action, 1)
		g.AssertHasColor(gametest.PlayerB, "Sol Ring", tc.color, true)
	}
}

func TestFaerieDragonRandomColorEffectChangesSpell(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Prismatic Dragon")
	player := g.GetPlayer(gametest.PlayerA)
	g.SetStep(core.PrecombatMain)
	player.ManaPool().Add(core.White, 2)
	player.ManaPool().Add(core.Colorless, 2)
	if err := g.CastSpellByName(player.PlayerID(), "Prismatic Dragon", nil); err != nil {
		t.Fatalf("cast Prismatic Dragon: %v", err)
	}
	spellID := g.GetStack().Peek().SourceID
	g.SetRandomResults([]int{0})
	if err := mage.ApplyEffect(g.Game, mage.ApplyToRandomSpellOrPermanent(mage.LaceEffect(core.Blue)), spellID, player.PlayerID(), nil); err != nil {
		t.Fatalf("change spell color: %v", err)
	}
	if got := g.EffectiveColors(spellID); !slices.Equal(got, []core.Color{core.Blue}) {
		t.Fatalf("spell colors = %v, want Blue", got)
	}
	g.ResolveTopOfStack()
	g.AssertHasColor(gametest.PlayerA, "Prismatic Dragon", core.Blue, true)
}

func TestFaerieDragonRandomZoneAndDamageEffects(t *testing.T) {
	t.Run("three damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Faerie Dragon")
		activateFaerieDragon(t, g, "Faerie Dragon", 6, 1)
		g.AssertLife(gametest.PlayerA, 17)
	})

	t.Run("one damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Faerie Dragon")
		activateFaerieDragon(t, g, "Faerie Dragon", 16, 2)
		g.AssertLife(gametest.PlayerB, 19)
	})

	t.Run("bounce", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Faerie Dragon")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		activateFaerieDragon(t, g, "Faerie Dragon", 15, 1)
		g.AssertHandCount(gametest.PlayerB, "Grizzly Bears", 1)
	})

	t.Run("exile and gain life", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Faerie Dragon")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		activateFaerieDragon(t, g, "Faerie Dragon", 18, 1)
		g.AssertExileCount("Hill Giant", 1)
		g.AssertLife(gametest.PlayerB, 23)
	})

	t.Run("one minus zero minus one counter", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Faerie Dragon")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		activateFaerieDragon(t, g, "Faerie Dragon", 19, 1)
		g.AssertCounterCount(gametest.PlayerB, "Hill Giant", core.M0M1, 1)
	})
}

func TestFaerieDragonMayTapOrUntapRandomPermanent(t *testing.T) {
	t.Run("tap", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Faerie Dragon")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Sol Ring")
		g.GetPlayer(gametest.PlayerA).QueueMayAbilityChoices(true)
		g.ChooseMode(gametest.PlayerA, 0)
		activateFaerieDragon(t, g, "Faerie Dragon", 1, 1)
		g.AssertTapped(gametest.PlayerB, "Sol Ring", true)
	})

	t.Run("decline", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Faerie Dragon")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Sol Ring")
		g.GetPlayer(gametest.PlayerA).QueueMayAbilityChoices(false)
		activateFaerieDragon(t, g, "Faerie Dragon", 1, 1)
		g.AssertTapped(gametest.PlayerB, "Sol Ring", false)
	})

	t.Run("untap", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Faerie Dragon")
		ringID := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Sol Ring")
		g.FindPermanent(ringID).Tapped = true
		g.GetPlayer(gametest.PlayerA).QueueMayAbilityChoices(true)
		g.ChooseMode(gametest.PlayerA, 1)
		activateFaerieDragon(t, g, "Faerie Dragon", 1, 1)
		g.AssertTapped(gametest.PlayerB, "Sol Ring", false)
	})
}
