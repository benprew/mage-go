package arabian

import (
	"testing"

	"github.com/mage/mage"
	_ "github.com/mage/mage/cards" // register base cards for test creatures
)

func TestCamel(t *testing.T) {
	t.Run("desert_damage_prevented_while_attacking", func(t *testing.T) {
		// Camel (0/1) attacks. Opponent activates Desert targeting Camel at
		// end of combat. The 1 damage from Desert should be prevented because
		// Camel is attacking and Desert is a Desert.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Camel")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Desert")
		g.Attack(1, mage.PlayerA, "Camel")
		g.ActivateAbility(1, mage.EndCombat, mage.PlayerB, "Desert", "Camel")
		g.StopAt(1, mage.PostcombatMain)
		g.Execute()
		// Camel should survive — Desert damage was prevented
		g.AssertPermanentCount(mage.PlayerA, "Camel", 1)
		g.AssertLife(mage.PlayerB, 20) // no combat damage from 0-power Camel
	})

	t.Run("non_desert_damage_not_prevented", func(t *testing.T) {
		// Camel (0/1) attacks. Opponent casts Lightning Bolt targeting Camel.
		// This damage is NOT from a Desert, so it should NOT be prevented.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Camel")
		g.AddCard(mage.ZoneHand, mage.PlayerB, "Lightning Bolt")
		g.Attack(1, mage.PlayerA, "Camel")
		g.CastSpell(1, mage.EndCombat, mage.PlayerB, "Lightning Bolt", "Camel")
		g.StopAt(1, mage.PostcombatMain)
		g.Execute()
		// Camel should die — Lightning Bolt is not a Desert
		g.AssertGraveyardCount(mage.PlayerA, "Camel", 1)
	})

	t.Run("not_attacking_no_prevention", func(t *testing.T) {
		// Camel is on the battlefield but NOT attacking. Desert deals 1 damage
		// to another attacking creature. Camel's prevention should not apply
		// to itself when not attacking.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Camel")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Desert")
		// Only Grizzly Bears attacks, not Camel
		g.Attack(1, mage.PlayerA, "Grizzly Bears")
		g.ActivateAbility(1, mage.EndCombat, mage.PlayerB, "Desert", "Grizzly Bears")
		g.StopAt(1, mage.PostcombatMain)
		g.Execute()
		// Grizzly Bears should take 1 damage (2/2 -> 2/1, survives)
		g.AssertPermanentCount(mage.PlayerA, "Grizzly Bears", 1)
	})

	t.Run("banded_creature_also_protected", func(t *testing.T) {
		// Camel (0/1, banding) and War Elephant (2/2, banding) attack as a band.
		// Desert targets the War Elephant. Damage should be prevented because
		// War Elephant is banded with the attacking Camel.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Camel")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "War Elephant")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Desert")
		g.Attack(1, mage.PlayerA, "Camel", "War Elephant")
		g.ActivateAbility(1, mage.EndCombat, mage.PlayerB, "Desert", "War Elephant")
		g.StopAt(1, mage.PostcombatMain)
		g.Execute()
		// War Elephant should survive — Desert damage was prevented via Camel's band
		g.AssertPermanentCount(mage.PlayerA, "War Elephant", 1)
		g.AssertPermanentCount(mage.PlayerA, "Camel", 1)
	})
}

func TestAbuJafar(t *testing.T) {
	t.Run("attacking_destroys_blockers_when_dies", func(t *testing.T) {
		// Abu Ja'far (0/1) attacks, blocked by a 2/2. Abu Ja'far dies to combat
		// damage, trigger destroys the blocker.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Abu Ja'far")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Grizzly Bears") // 2/2
		g.Attack(1, mage.PlayerA, "Abu Ja'far")
		g.Block(1, mage.PlayerB, "Grizzly Bears", "Abu Ja'far")
		g.StopAt(1, mage.PostcombatMain)
		g.Execute()
		g.AssertGraveyardCount(mage.PlayerA, "Abu Ja'far", 1)
		g.AssertGraveyardCount(mage.PlayerB, "Grizzly Bears", 1)
	})

	t.Run("blocking_destroys_attacker_when_dies", func(t *testing.T) {
		// Opponent attacks with a 3/3, Abu Ja'far (0/1) blocks and dies,
		// trigger destroys the attacker.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Abu Ja'far")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Centaur Courser") // 3/3
		g.Attack(2, mage.PlayerB, "Centaur Courser")
		g.Block(2, mage.PlayerA, "Abu Ja'far", "Centaur Courser")
		g.StopAt(2, mage.PostcombatMain)
		g.Execute()
		g.AssertGraveyardCount(mage.PlayerA, "Abu Ja'far", 1)
		g.AssertGraveyardCount(mage.PlayerB, "Centaur Courser", 1)
	})

	t.Run("no_combat_no_effect", func(t *testing.T) {
		// Abu Ja'far is destroyed outside combat — nothing else dies.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Abu Ja'far")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Grizzly Bears")
		g.AddCard(mage.ZoneHand, mage.PlayerB, "Lightning Bolt")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerB, "Lightning Bolt", "Abu Ja'far")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(mage.PlayerA, "Abu Ja'far", 1)
		g.AssertPermanentCount(mage.PlayerB, "Grizzly Bears", 1)
	})
}

func TestDanDan(t *testing.T) {
	t.Run("sacrifice_when_you_control_no_islands", func(t *testing.T) {
		g := mage.NewTestGame(t)
		island := g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Island")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Dandân")

		g.ExilePermanent(g.FindPermanent(island))

		g.StopAt(1, mage.EndStep)
		g.Execute()

		g.AssertPermanentCount(mage.PlayerA, "Island", 0)
		g.AssertPermanentCount(mage.PlayerA, "Dandân", 0)
	})
	t.Run("cannot_attack_if_defender_controls_no_island", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Island")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Dandân")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Mountain")

		g.Attack(1, mage.PlayerA, "Dandân")
		g.StopAt(1, mage.EndStep)
		g.Execute()

		g.AssertLife(mage.PlayerB, 20)
	})

	t.Run("can_attack_if_defender_controls_island", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Island")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Dandân")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Island")

		g.Attack(1, mage.PlayerA, "Dandân")
		g.StopAt(1, mage.EndStep)
		g.Execute()

		g.AssertLife(mage.PlayerB, 16)
	})
}
