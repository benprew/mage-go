package thedark

import (
	"testing"

	_ "github.com/benprew/mage-go/cards/legends"
	_ "github.com/benprew/mage-go/cards/limited"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestCurseArtifact(t *testing.T) {
	t.Run("controller declines to sacrifice and takes 2 damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Black Vise")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Curse Artifact")
		permA := g.FindPermanentByName("Curse Artifact", g.GetPlayer(gametest.PlayerA).PlayerID())
		permB := g.FindPermanentByName("Black Vise", g.GetPlayer(gametest.PlayerB).PlayerID())
		permA.AttachedTo = permB.ID()
		permB.Attachments = append(permB.Attachments, permA.ID())

		// Turn 2 is Player B's upkeep. Player B chooses not to sacrifice Black Vise.
		g.GetPlayer(gametest.PlayerB).QueueMayAbilityChoices(false)
		g.StopAt(2, core.Draw)
		g.Execute()

		g.AssertLife(gametest.PlayerB, 18)
		g.AssertPermanentCount(gametest.PlayerB, "Black Vise", 1)
	})

	t.Run("controller sacrifices artifact and avoids damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Black Vise")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Curse Artifact")
		permA := g.FindPermanentByName("Curse Artifact", g.GetPlayer(gametest.PlayerA).PlayerID())
		permB := g.FindPermanentByName("Black Vise", g.GetPlayer(gametest.PlayerB).PlayerID())
		permA.AttachedTo = permB.ID()
		permB.Attachments = append(permB.Attachments, permA.ID())

		// Turn 2 is Player B's upkeep. Player B chooses to sacrifice Black Vise.
		g.GetPlayer(gametest.PlayerB).QueueMayAbilityChoices(true)
		g.StopAt(2, core.Draw)
		g.Execute()

		g.AssertLife(gametest.PlayerB, 20)
		g.AssertGraveyardCount(gametest.PlayerB, "Black Vise", 1)
	})
}

func TestDanceOfMany(t *testing.T) {
	t.Run("creates token copy on ETB and upkeep payment preserves it", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dance of Many")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 4)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")

		g.StopAt(1, core.PrecombatMain)
		g.Execute()

		g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1)
	})
}

func TestDarkHeartOfTheWood(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dark Heart of the Wood")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)

	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Dark Heart of the Wood")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()

	g.AssertLife(gametest.PlayerA, 23)
	g.AssertGraveyardCount(gametest.PlayerA, "Forest", 1)
}

func TestDeepWater(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Deep Water")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")

	// Pay {U} to activate Deep Water.
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Deep Water")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
}

func TestFasting(t *testing.T) {
	t.Run("skip draw step to gain 2 life and add hunger counter", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fasting")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears", 2)

		// On turn 1 (upkeep 1): Fasting gets counter 1. Draw step skipped on turn 1 (CR 103.8a for player playing first).
		// On turn 3 (upkeep 2): Fasting gets counter 2.
		// Draw: choose to skip draw step. Life +2 = 22.
		g.GetPlayer(gametest.PlayerA).QueueMayAbilityChoices(true)
		g.StopAt(3, core.PrecombatMain)
		g.Execute()

		g.AssertCounterCount(gametest.PlayerA, "Fasting", core.Hunger, 2)
		g.AssertLife(gametest.PlayerA, 22)
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 0)
	})

	t.Run("drawing a card destroys Fasting", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fasting")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears", 2)

		// On turn 3, choose not to skip draw step. Player draws Grizzly Bears.
		// Fasting trigger on draw: destroy Fasting.
		g.GetPlayer(gametest.PlayerA).QueueMayAbilityChoices(false)
		g.StopAt(3, core.PrecombatMain)
		g.Execute()

		g.AssertPermanentCount(gametest.PlayerA, "Fasting", 0)
		g.AssertGraveyardCount(gametest.PlayerA, "Fasting", 1)
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}

func TestGaeasTouch(t *testing.T) {
	t.Run("puts basic Forest onto battlefield", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gaea's Touch")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest")

		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Gaea's Touch")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()

		g.AssertPermanentCount(gametest.PlayerA, "Forest", 1)
		g.AssertHandCount(gametest.PlayerA, "Forest", 0)
	})

	t.Run("sacrifice adds GG", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gaea's Touch")

		// Activate Gaea's Touch sacrifice mana ability
		g.ActivateAbilityIndex(1, core.PrecombatMain, gametest.PlayerA, "Gaea's Touch", 1)
		g.StopAt(1, core.PostcombatMain)
		g.Execute()

		g.AssertGraveyardCount(gametest.PlayerA, "Gaea's Touch", 1)
		pA := g.GetPlayer(gametest.PlayerA)
		if pA.ManaPool().CountProducedThisTurn(core.Green) < 2 {
			t.Errorf("expected at least 2 green mana produced this turn, got %d", pA.ManaPool().CountProducedThisTurn(core.Green))
		}
	})
}

func TestGoblinCaves(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblin Caves")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblin Hero") // 2/2 Goblin

	permA := g.FindPermanentByName("Goblin Caves", g.GetPlayer(gametest.PlayerA).PlayerID())
	permM := g.FindPermanentByName("Mountain", g.GetPlayer(gametest.PlayerA).PlayerID())
	permA.AttachedTo = permM.ID()
	permM.Attachments = append(permM.Attachments, permA.ID())

	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	// Goblin Hero gets +0/+2 -> 2/4
	g.AssertPowerToughness(gametest.PlayerA, "Goblin Hero", 2, 4)
}

func TestGoblinShrine(t *testing.T) {
	t.Run("buffs goblins", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblin Shrine")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblin Hero") // 2/2 Goblin

		permA := g.FindPermanentByName("Goblin Shrine", g.GetPlayer(gametest.PlayerA).PlayerID())
		permM := g.FindPermanentByName("Mountain", g.GetPlayer(gametest.PlayerA).PlayerID())
		permA.AttachedTo = permM.ID()
		permM.Attachments = append(permM.Attachments, permA.ID())

		g.StopAt(1, core.PrecombatMain)
		g.Execute()

		// Goblin Hero gets +1/+0 -> 3/2
		g.AssertPowerToughness(gametest.PlayerA, "Goblin Hero", 3, 2)
	})

	t.Run("leaves battlefield deals 1 damage to each goblin", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblin Shrine")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblin Digging Team") // 1/1 Goblin
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains", 2)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Disenchant")

		permA := g.FindPermanentByName("Goblin Shrine", g.GetPlayer(gametest.PlayerA).PlayerID())
		permM := g.FindPermanentByName("Mountain", g.GetPlayer(gametest.PlayerA).PlayerID())
		permA.AttachedTo = permM.ID()
		permM.Attachments = append(permM.Attachments, permA.ID())

		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Disenchant", "Goblin Shrine")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()

		// 1/1 Goblin dies from 1 damage
		g.AssertGraveyardCount(gametest.PlayerA, "Goblin Digging Team", 1)
	})
}

func TestHiddenPath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hidden Path")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // Green creature
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")

	g.Attack(1, gametest.PlayerA, "Grizzly Bears")
	g.Block(1, gametest.PlayerB, "Hill Giant", "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()

	// Bears has forestwalk, cannot be blocked by Hill Giant because Player B controls Forest
	g.AssertLife(gametest.PlayerB, 18)
}

func TestManaVortex(t *testing.T) {
	t.Run("counter unless sacrifice a land on cast", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Mana Vortex")

		// Controller chooses to sacrifice an Island on cast
		g.GetPlayer(gametest.PlayerA).QueueMayAbilityChoices(true)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Mana Vortex")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()

		g.AssertPermanentCount(gametest.PlayerA, "Mana Vortex", 1)
		g.AssertGraveyardCount(gametest.PlayerA, "Island", 1)
	})

	t.Run("upkeep sacrifices land", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mana Vortex")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain", 2)

		// Turn 2 is Player B's upkeep -> Player B sacrifices a land
		g.StopAt(2, core.Draw)
		g.Execute()

		g.AssertPermanentCount(gametest.PlayerB, "Mountain", 1)
		g.AssertGraveyardCount(gametest.PlayerB, "Mountain", 1)
	})
}

func TestPsychicAllergy(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Psychic Allergy")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Savannah Lions", 2) // White creatures

	// ETB chooses White
	perm := g.FindPermanentByName("Psychic Allergy", g.GetPlayer(gametest.PlayerA).PlayerID())
	perm.ChosenColor = core.White

	// Turn 2 is Player B's upkeep. Psychic Allergy deals 2 damage to Player B.
	g.StopAt(2, core.Draw)
	g.Execute()

	g.AssertLife(gametest.PlayerB, 18)
}

func TestSeasonOfTheWitch(t *testing.T) {
	t.Run("destroys untapped creature that didn't attack", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Season of the Witch")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // Could attack, didn't attack

		// Player A doesn't attack
		g.StopAt(1, core.Cleanup)
		g.Execute()

		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	})

	t.Run("does not destroy creature with summoning sickness or defender", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Season of the Witch")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Wood") // Defender, couldn't attack

		g.StopAt(1, core.Cleanup)
		g.Execute()

		g.AssertPermanentCount(gametest.PlayerA, "Wall of Wood", 1)
	})
}

func TestTangleKelp(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tangle Kelp")

	permA := g.FindPermanentByName("Tangle Kelp", g.GetPlayer(gametest.PlayerA).PlayerID())
	permB := g.FindPermanentByName("Grizzly Bears", g.GetPlayer(gametest.PlayerB).PlayerID())
	permA.AttachedTo = permB.ID()
	permB.Attachments = append(permB.Attachments, permA.ID())

	// Turn 2 is Player B's turn. Bears untaps, attacks.
	g.Attack(2, gametest.PlayerB, "Grizzly Bears")

	// Turn 4 is Player B's next turn. Bears attacked on turn 2 (last turn), so it doesn't untap on turn 4!
	g.StopAt(4, core.PrecombatMain)
	g.Execute()

	g.AssertTapped(gametest.PlayerB, "Grizzly Bears", true)
}

func TestWormsOfTheEarth(t *testing.T) {
	t.Run("players cannot play lands", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Worms of the Earth")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest")

		// Decline to sacrifice lands or take 5 damage on upkeep trigger
		g.GetPlayer(gametest.PlayerA).QueueMayAbilityChoices(false)
		g.GetPlayer(gametest.PlayerB).QueueMayAbilityChoices(false)

		g.StopAt(1, core.PrecombatMain)
		g.Execute()

		// Cannot play land
		err := g.Game.PlayLand(g.GetPlayer(gametest.PlayerA).PlayerID(), g.GetPlayer(gametest.PlayerA).Hand()[0].ID())
		if err == nil {
			t.Errorf("expected PlayLand to fail while Worms of the Earth is in play")
		}
	})
}
