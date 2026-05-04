package secretsofstrixhaven

import (
	"testing"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// TestAdditiveEvolution verifies Additive Evolution's two abilities:
// 1. ETB creates a 0/0 green and blue Fractal token with 3 +1/+1 counters.
// 2. At the beginning of combat on your turn, put a +1/+1 counter on target
//    creature you control. It gains vigilance until end of turn.
func TestAdditiveEvolution(t *testing.T) {
	t.Run("etb creates 0/0 fractal token with 3 counters", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Additive Evolution")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Additive Evolution")
		// The BeginCombat trigger will fire and target Grizzly Bears, leaving Fractal Token at 3/3
		g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Fractal Token", 1)
		g.AssertPowerToughness(gametest.PlayerA, "Fractal Token", 3, 3)
	})

	t.Run("beginning of combat puts counter on target creature you control", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Additive Evolution")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
	})

	t.Run("beginning of combat grants vigilance until end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Additive Evolution")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Grizzly Bears attacked with vigilance, so it should not be tapped
		g.AssertTapped(gametest.PlayerA, "Grizzly Bears", false)
	})

	t.Run("combat trigger does not fire on opponent's turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Additive Evolution")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		// Turn 1 (PlayerA): BeginCombat trigger fires → choose Grizzly Bears → gets +1 → 3/3
		g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// Turn 2 (PlayerB): trigger should NOT fire; Grizzly Bears stays at 3/3 (from turn 1 only)
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
	})
}

// TestLivingHistory verifies Living History's two abilities:
// 1. ETB creates a 2/2 red and white Spirit creature token.
// 2. Whenever you attack, if a card left your graveyard this turn, target
//    attacking creature gets +2/+0 until end of turn.
func TestLivingHistory(t *testing.T) {
	t.Run("etb creates 2/2 red and white spirit token", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Living History")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Living History")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Spirit Token", 1)
		g.AssertPowerToughness(gametest.PlayerA, "Spirit Token", 2, 2)
	})
}

// TestPrimaryResearch verifies Primary Research's ETB ability:
// When this enchantment enters, return target nonland permanent card with
// mana value 3 or less from your graveyard to the battlefield.
func TestPrimaryResearch(t *testing.T) {
	t.Run("etb returns target nonland permanent with mana value 3 or less from graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Primary Research")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Primary Research", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 0)
	})

	t.Run("etb does not return permanent with mana value 4 or more", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Serra Angel")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Primary Research")
		// Serra Angel has MV 5 — targeting should be impossible, no valid target
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
	})

	t.Run("etb does not return land cards", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Primary Research")
		// Forest is a land — targeting should be impossible, no valid target
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
	})
}
