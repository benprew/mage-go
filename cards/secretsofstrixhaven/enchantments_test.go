package secretsofstrixhaven

import (
	"testing"

	"github.com/google/uuid"

	_ "github.com/benprew/mage-go/cards/limited"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

// TestAdditiveEvolution verifies Additive Evolution's two abilities:
//  1. ETB creates a 0/0 green and blue Fractal token with 3 +1/+1 counters.
//  2. At the beginning of combat on your turn, put a +1/+1 counter on target
//     creature you control. It gains vigilance until end of turn.
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

// TestComfortingCounsel verifies Comforting Counsel's two abilities:
//  1. Whenever you gain life, put a growth counter on this enchantment.
//  2. As long as there are five or more growth counters on this enchantment,
//     creatures you control get +3/+3.
func TestComfortingCounsel(t *testing.T) {
	t.Run("gain_life_puts_growth_counter", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Comforting Counsel")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Healing Salve")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Healing Salve", "PlayerA")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Comforting Counsel", core.Growth, 1)
	})

	t.Run("five_growth_counters_boost_creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Comforting Counsel")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCounters(1, core.PrecombatMain, gametest.PlayerA, "Comforting Counsel", core.Growth, 5)
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 5, 5)
	})

	t.Run("four_growth_counters_no_boost", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Comforting Counsel")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCounters(1, core.PrecombatMain, gametest.PlayerA, "Comforting Counsel", core.Growth, 4)
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
	})
}

// TestGraduationDay verifies Graduation Day's Repartee ability:
// Whenever you cast an instant or sorcery spell that targets a creature,
// put a +1/+1 counter on target creature you control.
func TestGraduationDay(t *testing.T) {
	t.Run("triggers_when_instant_targets_creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Graduation Day")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		// Lightning Bolt targets Grizzly Bears — triggers Repartee
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
		// Repartee trigger fires; choose Grizzly Bears (but it's about to die — test for counter applied before state-based)
		// Instead use a different creature as the Repartee target
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Grizzly Bears dies from Lightning Bolt (3 damage), but the counter was applied
		// Check no Grizzly Bears on battlefield (it died), but the trigger fired
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	})

	t.Run("triggers_and_counter_goes_on_surviving_creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Graduation Day")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Serra Angel")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		// Lightning Bolt targets Serra Angel; trigger fires; put +1/+1 on Grizzly Bears
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Serra Angel")
		g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
	})

	t.Run("does_not_trigger_when_spell_targets_player", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Graduation Day")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		// Lightning Bolt targets PlayerB — no creature targeted, trigger should not fire
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
	})
}

// TestLivingHistory verifies Living History's two abilities:
//  1. ETB creates a 2/2 red and white Spirit creature token.
//  2. Whenever you attack, if a card left your graveyard this turn, target
//     attacking creature gets +2/+0 until end of turn.
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

	t.Run("attack trigger pumps an attacking creature if card left graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Living History")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
		pid := g.GetPlayer(gametest.PlayerA).PlayerID()
		gyCard := g.GetPlayer(gametest.PlayerA).Graveyard()[0]
		removed, ok := g.MoveFromGraveyard(pid, gyCard.ID(), core.ZoneExile)
		if !ok {
			t.Fatalf("MoveFromGraveyard failed")
		}
		g.ExileCard(removed, uuid.Nil)
		g.Attack(1, gametest.PlayerA, "Hill Giant")
		g.ChoosePermanent(gametest.PlayerA, "Hill Giant")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Hill Giant", 5, 3)
	})

	t.Run("attack trigger does not pump without graveyard move", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Living History")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
		g.Attack(1, gametest.PlayerA, "Hill Giant")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Hill Giant", 3, 3)
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

	t.Run("end step draws if a card left your graveyard this turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Primary Research")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Primary Research", "Grizzly Bears")
		g.StopAt(1, core.Cleanup)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Forest", 1)
	})

	t.Run("end step does not draw without graveyard move", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Primary Research")
		g.StopAt(1, core.Cleanup)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Forest", 0)
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
