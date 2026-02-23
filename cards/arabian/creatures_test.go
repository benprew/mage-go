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
		return mage.NewCreature("Centaur Courser", "{2}{G}", 3, 3,
			mage.WithSubTypes("Centaur", "Warrior"),
		)
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

func TestStoneThrowingDevils(t *testing.T) {
	t.Run("first_strike_kills_blocker_before_normal_damage", func(t *testing.T) {
		// Stone-Throwing Devils (1/1 first strike) blocks a 2/1.
		// Devils deals 1 first strike damage, killing the 2/1 before it strikes back.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Stone-Throwing Devils")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Savannah Lions") // 2/1
		g.Attack(2, gametest.PlayerB, "Savannah Lions")
		g.Block(2, gametest.PlayerA, "Stone-Throwing Devils", "Savannah Lions")
		g.StopAt(2, core.PostcombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Stone-Throwing Devils", 1)
		g.AssertGraveyardCount(gametest.PlayerB, "Savannah Lions", 1)
	})
}

func TestBirdMaiden(t *testing.T) {
	t.Run("cannot_be_blocked_by_ground_creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bird Maiden")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Bird Maiden")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Bird Maiden")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// Grizzly Bears can't block a flyer — Bird Maiden deals 1 to player
		g.AssertLife(gametest.PlayerB, 19)
	})
}

func TestDancingScimitar(t *testing.T) {
	t.Run("has_flying", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dancing Scimitar")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Dancing Scimitar")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Dancing Scimitar")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// Can't block flyer — deals 1 to player
		g.AssertLife(gametest.PlayerB, 19)
	})
}

func TestJuzamDjinn(t *testing.T) {
	t.Run("upkeep_deals_1_to_controller", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Juzám Djinn")
		g.StopAt(3, core.PrecombatMain) // PlayerA's turn 3 = 2nd upkeep
		g.Execute()
		// Turn 1 upkeep: 20→19, Turn 3 upkeep: 19→18
		g.AssertLife(gametest.PlayerA, 18)
	})

	t.Run("opponent_unaffected", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Juzám Djinn")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 20)
	})
}

func TestSerendibEfreet(t *testing.T) {
	t.Run("upkeep_deals_1_to_controller", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Serendib Efreet")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 19)
	})

	t.Run("has_flying", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Serendib Efreet")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Serendib Efreet")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Serendib Efreet")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// Can't block flyer
		g.AssertLife(gametest.PlayerB, 17)
	})
}

func TestJununEfreet(t *testing.T) {
	t.Run("survives_if_can_pay_BB", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Junún Efreet")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Junún Efreet", 1)
	})

	t.Run("sacrificed_if_no_black_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Junún Efreet")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Junún Efreet", 0)
	})
}

func TestErgRaiders(t *testing.T) {
	t.Run("didnt_attack_takes_2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Erg Raiders")
		// Don't attack — end step should deal 2 to controller
		g.StopAt(2, core.Upkeep) // stop at opponent's upkeep to see end step resolved
		g.Execute()
		g.AssertLife(gametest.PlayerA, 18)
	})

	t.Run("attacked_no_damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Erg Raiders")
		g.Attack(1, gametest.PlayerA, "Erg Raiders")
		g.StopAt(2, core.Upkeep)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 20)
	})
}

func TestKingSuleiman(t *testing.T) {
	t.Run("destroys_djinn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "King Suleiman")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Juzám Djinn")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "King Suleiman", "Juzám Djinn")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerB, "Juzám Djinn", 1)
	})

	t.Run("destroys_efreet", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "King Suleiman")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Serendib Efreet")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "King Suleiman", "Serendib Efreet")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerB, "Serendib Efreet", 1)
	})

	t.Run("cannot_target_non_djinn_efreet", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "King Suleiman")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "King Suleiman", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Grizzly Bears should survive — not a valid target
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestWyluliWolf(t *testing.T) {
	t.Run("boosts_target_until_eot", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wyluli Wolf")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Wyluli Wolf", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
	})

	t.Run("boost_expires_at_eot", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wyluli Wolf")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Wyluli Wolf", "Grizzly Bears")
		g.StopAt(2, core.Upkeep) // next turn
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
	})
}

func TestAliBaba(t *testing.T) {
	t.Run("taps_a_wall", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ali Baba")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Stone")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Ali Baba", "Wall of Stone")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerB, "Wall of Stone", true)
	})

	t.Run("cannot_target_non_wall", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ali Baba")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Ali Baba", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerB, "Grizzly Bears", false)
	})
}

func TestBrassMan(t *testing.T) {
	t.Run("doesnt_untap_normally", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Brass Man")
		g.Attack(1, gametest.PlayerA, "Brass Man")
		// Turn 3 is PlayerA's next turn — Brass Man should still be tapped if no mana
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Brass Man", true)
	})

	t.Run("pays_1_to_untap_at_upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Brass Man")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.Attack(1, gametest.PlayerA, "Brass Man")
		// Turn 3 upkeep: pays {1} from Mountain → untaps
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Brass Man", false)
	})
}

func TestIslandFishJasconius(t *testing.T) {
	t.Run("sacrificed_when_no_islands", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		island := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island Fish Jasconius")
		g.ExilePermanent(g.FindPermanent(island))
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Island Fish Jasconius", 0)
	})

	t.Run("cant_attack_if_defender_has_no_island", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island Fish Jasconius")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")
		g.Attack(1, gametest.PlayerA, "Island Fish Jasconius")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 20)
	})
}

func TestElHajjaj(t *testing.T) {
	t.Run("gains_life_when_deals_combat_damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "El-Hajjâj")
		g.SetLife(gametest.PlayerA, 15)
		g.Attack(1, gametest.PlayerA, "El-Hajjâj")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// Deals 1 damage to PlayerB, gains 1 life (lifelink)
		g.AssertLife(gametest.PlayerB, 19)
		g.AssertLife(gametest.PlayerA, 16)
	})
}

func TestRukhEgg(t *testing.T) {
	t.Run("creates_token_when_dies", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rukh Egg")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Rukh Egg")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Rukh Egg", 1)
		g.AssertPermanentCount(gametest.PlayerA, "Bird", 1)
	})
}

func TestHasranOgress(t *testing.T) {
	t.Run("attacks_pays_2_no_damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hasran Ogress")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.Attack(1, gametest.PlayerA, "Hasran Ogress")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 20) // paid {2}, no self-damage
	})

	t.Run("attacks_cant_pay_takes_3", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hasran Ogress")
		g.Attack(1, gametest.PlayerA, "Hasran Ogress")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 17) // took 3 damage
	})
}

func TestDesertNomads(t *testing.T) {
	t.Run("unblockable_if_defender_controls_desert", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Desert Nomads")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Desert")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Desert Nomads")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Desert Nomads")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// Desertwalk — can't be blocked
		g.AssertLife(gametest.PlayerB, 18)
	})

	t.Run("blockable_if_no_desert", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Desert Nomads")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Desert Nomads")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Desert Nomads")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// Blocked normally — both trade (2/2 vs 2/2)
		g.AssertLife(gametest.PlayerB, 20)
	})

	t.Run("desert_damage_prevented", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Desert Nomads")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Desert")
		g.Attack(1, gametest.PlayerA, "Desert Nomads")
		g.ActivateAbility(1, core.EndCombat, gametest.PlayerB, "Desert", "Desert Nomads")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// Desert damage should be prevented
		g.AssertPermanentCount(gametest.PlayerA, "Desert Nomads", 1)
	})
}

func TestKirdApe(t *testing.T) {
	t.Run("with_forest_is_2_3", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kird Ape")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Kird Ape", 2, 3)
	})

	t.Run("without_forest_is_1_1", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kird Ape")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Kird Ape", 1, 1)
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

func TestMijaeDjinn(t *testing.T) {
	t.Run("win_flip_stays_in_combat", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mijae Djinn") // 6/3
		g.CoinFlipResults = []bool{true}                                  // win
		g.Attack(1, gametest.PlayerA, "Mijae Djinn")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Won flip — stays in combat, deals 6 damage
		g.AssertLife(gametest.PlayerB, 14)
	})

	t.Run("lose_flip_removed_and_tapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mijae Djinn") // 6/3
		g.CoinFlipResults = []bool{false}                                 // lose
		g.Attack(1, gametest.PlayerA, "Mijae Djinn")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Lost flip — removed from combat, no damage dealt
		g.AssertLife(gametest.PlayerB, 20)
		g.AssertTapped(gametest.PlayerA, "Mijae Djinn", true)
	})
}

func TestYdwenEfreet(t *testing.T) {
	t.Run("win_flip_blocks_normally", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Ydwen Efreet")  // 3/6
		g.CoinFlipResults = []bool{true}                                    // win
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Ydwen Efreet", "Grizzly Bears")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Won flip — blocks normally, kills Bears
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
		g.AssertLife(gametest.PlayerB, 20)
	})

	t.Run("lose_flip_removed_attacker_hits_through", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Ydwen Efreet")  // 3/6
		g.CoinFlipResults = []bool{false}                                   // lose
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Ydwen Efreet", "Grizzly Bears")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Lost flip — removed from combat, attacker hits through
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
		g.AssertLife(gametest.PlayerB, 18)
	})
}

func TestSorceressQueen(t *testing.T) {
	t.Run("sets_target_to_0_2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sorceress Queen")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Sorceress Queen", "Hill Giant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerB, "Hill Giant", 0, 2)
	})

	t.Run("reverts_after_end_of_turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sorceress Queen")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Sorceress Queen", "Hill Giant")
		g.StopAt(2, core.PrecombatMain) // next turn
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerB, "Hill Giant", 3, 3)
	})
}

func TestSingingTree(t *testing.T) {
	t.Run("sets_attacker_power_to_0", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Singing Tree")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Juzám Djinn") // 5/5
		g.Attack(1, gametest.PlayerA, "Juzám Djinn")
		// Activate after attackers declared
		g.ActivateAbility(1, core.DeclareBlockers, gametest.PlayerB, "Singing Tree", "Juzám Djinn")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Power set to 0, toughness unchanged (5)
		g.AssertPowerToughness(gametest.PlayerA, "Juzám Djinn", 0, 5)
		// 0 power = no combat damage
		g.AssertLife(gametest.PlayerB, 20)
	})

	t.Run("reverts_after_end_of_turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Singing Tree")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant") // 3/3
		g.Attack(1, gametest.PlayerA, "Hill Giant")
		g.ActivateAbility(1, core.DeclareBlockers, gametest.PlayerB, "Singing Tree", "Hill Giant")
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Hill Giant", 3, 3)
	})
}
