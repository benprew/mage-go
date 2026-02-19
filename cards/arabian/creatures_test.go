package arabian

import (
	"os"
	"testing"

	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
	"github.com/mage/mage/pkg/mage/gametest"
	_ "github.com/mage/mage/cards/limited" // register base cards for test creatures
)

func TestMain(m *testing.M) {
	mage.Register("Centaur Courser", func() mage.Card {
		c := mage.NewCreature("Centaur Courser", "{2}{G}", 3, 3, "Centaur", "Warrior")
		return c
	})
	os.Exit(m.Run())
}

func TestCamel(t *testing.T) {
	t.Run("desert_damage_prevented_while_attacking", func(t *testing.T) {
		// Camel (0/1) attacks. Opponent activates Desert targeting Camel at
		// end of combat. The 1 damage from Desert should be prevented because
		// Camel is attacking and Desert is a Desert.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Camel")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Desert")
		g.Attack(1, gametest.PlayerA, "Camel")
		g.ActivateAbility(1, core.EndCombat, gametest.PlayerB, "Desert", "Camel")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// Camel should survive — Desert damage was prevented
		g.AssertPermanentCount(gametest.PlayerA, "Camel", 1)
		g.AssertLife(gametest.PlayerB, 20) // no combat damage from 0-power Camel
	})

	t.Run("non_desert_damage_not_prevented", func(t *testing.T) {
		// Camel (0/1) attacks. Opponent casts Lightning Bolt targeting Camel.
		// This damage is NOT from a Desert, so it should NOT be prevented.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Camel")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.Attack(1, gametest.PlayerA, "Camel")
		g.CastSpell(1, core.EndCombat, gametest.PlayerB, "Lightning Bolt", "Camel")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// Camel should die — Lightning Bolt is not a Desert
		g.AssertGraveyardCount(gametest.PlayerA, "Camel", 1)
	})

	t.Run("not_attacking_no_prevention", func(t *testing.T) {
		// Camel is on the battlefield but NOT attacking. Desert deals 1 damage
		// to another attacking creature. Camel's prevention should not apply
		// to itself when not attacking.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Camel")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Desert")
		// Only Grizzly Bears attacks, not Camel
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.ActivateAbility(1, core.EndCombat, gametest.PlayerB, "Desert", "Grizzly Bears")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// Grizzly Bears should take 1 damage (2/2 -> 2/1, survives)
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	})

	t.Run("banded_creature_also_protected", func(t *testing.T) {
		// Camel (0/1, banding) and War Elephant (2/2, banding) attack as a band.
		// Desert targets the War Elephant. Damage should be prevented because
		// War Elephant is banded with the attacking Camel.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Camel")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "War Elephant")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Desert")
		g.Attack(1, gametest.PlayerA, "Camel", "War Elephant")
		g.ActivateAbility(1, core.EndCombat, gametest.PlayerB, "Desert", "War Elephant")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// War Elephant should survive — Desert damage was prevented via Camel's band
		g.AssertPermanentCount(gametest.PlayerA, "War Elephant", 1)
		g.AssertPermanentCount(gametest.PlayerA, "Camel", 1)
	})
}

func TestAbuJafar(t *testing.T) {
	t.Run("attacking_destroys_blockers_when_dies", func(t *testing.T) {
		// Abu Ja'far (0/1) attacks, blocked by a 2/2. Abu Ja'far dies to combat
		// damage, trigger destroys the blocker.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Abu Ja'far")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2
		g.Attack(1, gametest.PlayerA, "Abu Ja'far")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Abu Ja'far")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Abu Ja'far", 1)
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	})

	t.Run("blocking_destroys_attacker_when_dies", func(t *testing.T) {
		// Opponent attacks with a 3/3, Abu Ja'far (0/1) blocks and dies,
		// trigger destroys the attacker.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Abu Ja'far")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Centaur Courser") // 3/3
		g.Attack(2, gametest.PlayerB, "Centaur Courser")
		g.Block(2, gametest.PlayerA, "Abu Ja'far", "Centaur Courser")
		g.StopAt(2, core.PostcombatMain)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Abu Ja'far", 1)
		g.AssertGraveyardCount(gametest.PlayerB, "Centaur Courser", 1)
	})

	t.Run("no_combat_no_effect", func(t *testing.T) {
		// Abu Ja'far is destroyed outside combat — nothing else dies.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Abu Ja'far")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Abu Ja'far")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Abu Ja'far", 1)
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestGiantTortoise(t *testing.T) {
	t.Run("Giant Tortoise gets +0/+3 when untapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Giant Tortoise")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Giant Tortoise", 1, 4)
	})

	t.Run("Giant Tortoise doesn't get it when tapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Giant Tortoise")
		g.Attack(1, gametest.PlayerA, "Giant Tortoise")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Giant Tortoise", 1, 1)
	})
}

func TestDanDan(t *testing.T) {
	t.Run("sacrifice_when_you_control_no_islands", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		island := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dandân")

		g.ExilePermanent(g.FindPermanent(island))

		g.StopAt(1, core.EndStep)
		g.Execute()

		g.AssertPermanentCount(gametest.PlayerA, "Island", 0)
		g.AssertPermanentCount(gametest.PlayerA, "Dandân", 0)
	})
	t.Run("cannot_attack_if_defender_controls_no_island", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dandân")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")

		g.Attack(1, gametest.PlayerA, "Dandân")
		g.StopAt(1, core.EndStep)
		g.Execute()

		g.AssertLife(gametest.PlayerB, 20)
	})

	t.Run("can_attack_if_defender_controls_island", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dandân")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")

		g.Attack(1, gametest.PlayerA, "Dandân")
		g.StopAt(1, core.EndStep)
		g.Execute()

		g.AssertLife(gametest.PlayerB, 16)
	})
}
