package thedark

import (
	"testing"

	_ "github.com/benprew/mage-go/cards/legends"
	_ "github.com/benprew/mage-go/cards/limited"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestVanillaCreatures(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblin Hero")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Scarwood Goblins")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Squire")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	g.AssertPowerToughness(gametest.PlayerA, "Goblin Hero", 2, 2)
	g.AssertPowerToughness(gametest.PlayerA, "Scarwood Goblins", 2, 2)
	g.AssertPowerToughness(gametest.PlayerA, "Squire", 1, 2)
}

func TestBogRats(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bog Rats")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Wood")

	g.Attack(1, gametest.PlayerA, "Bog Rats")
	g.Block(1, gametest.PlayerB, "Wall of Wood", "Bog Rats")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()

	// Wall cannot block Bog Rats, so Player B takes 1 damage
	g.AssertLife(gametest.PlayerB, 19)
}

func TestMarshGoblins(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Marsh Goblins")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Swamp")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")

	g.Attack(1, gametest.PlayerA, "Marsh Goblins")
	g.Block(1, gametest.PlayerB, "Grizzly Bears", "Marsh Goblins")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()

	// Swampwalk makes it unblockable because Player B controls a Swamp
	g.AssertLife(gametest.PlayerB, 19)
}

func TestWaterWurm(t *testing.T) {
	t.Run("without island", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Water Wurm")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()

		g.AssertPowerToughness(gametest.PlayerA, "Water Wurm", 1, 1)
	})

	t.Run("with opponent island", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Water Wurm")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()

		g.AssertPowerToughness(gametest.PlayerA, "Water Wurm", 1, 2)
	})
}

func TestKnightsOfThorn(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Knights of Thorn")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Goblin Hero")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	k := g.FindPermanentByName("Knights of Thorn", g.GetPlayer(gametest.PlayerA).PlayerID())
	if k == nil {
		t.Fatal("Knights of Thorn not found")
	}
	if !k.HasKeyword(core.Banding) {
		t.Error("Knights of Thorn missing Banding")
	}

	// In combat: red Goblin Hero (2/2) attacks on turn 2, Knights of Thorn (2/2 pro-red) blocks.
	// Goblin Hero takes 2 damage and dies; Knights of Thorn takes 0 damage due to protection.
	g2 := gametest.NewTestGame(t)
	g2.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Knights of Thorn")
	g2.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Goblin Hero")
	g2.Attack(2, gametest.PlayerB, "Goblin Hero")
	g2.Block(2, gametest.PlayerA, "Knights of Thorn", "Goblin Hero")
	g2.StopAt(2, core.PostcombatMain)
	g2.Execute()

	g2.AssertPermanentCount(gametest.PlayerA, "Knights of Thorn", 1)
	g2.AssertPermanentCount(gametest.PlayerB, "Goblin Hero", 0)
}

func TestDrowned(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Drowned")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")

	// Player A regenerates Drowned before damage
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Drowned")
	g.CastSpell(1, core.BeginCombat, gametest.PlayerB, "Lightning Bolt", "Drowned")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	g.AssertPermanentCount(gametest.PlayerA, "Drowned", 1)
	g.AssertTapped(gametest.PlayerA, "Drowned", true)
}

func TestCoalGolem(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Coal Golem")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 3)

	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Coal Golem")
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertPermanentCount(gametest.PlayerA, "Coal Golem", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Coal Golem", 1)
	pool := g.GetPlayer(gametest.PlayerA).ManaPool()
	if got := pool.CountProducedThisTurn(core.Red); got < 3 {
		t.Fatalf("want at least 3 red mana produced, got %d", got)
	}
}

func TestNiallSilvain(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Niall Silvain")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")

	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Niall Silvain", "Grizzly Bears")
	g.CastSpell(1, core.BeginCombat, gametest.PlayerB, "Lightning Bolt", "Grizzly Bears")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertTapped(gametest.PlayerA, "Grizzly Bears", true)
}

func TestExorcist(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Exorcist")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Bog Rats")

	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Exorcist", "Bog Rats")
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertPermanentCount(gametest.PlayerB, "Bog Rats", 0)
	g.AssertGraveyardCount(gametest.PlayerB, "Bog Rats", 1)
}

func TestFireDrake(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fire Drake")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)

	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Fire Drake")
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertPowerToughness(gametest.PlayerA, "Fire Drake", 2, 2)
}

func TestGoblinDiggingTeam(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblin Digging Team")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Wood")

	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Goblin Digging Team", "Wall of Wood")
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertPermanentCount(gametest.PlayerA, "Goblin Digging Team", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Goblin Digging Team", 1)
	g.AssertPermanentCount(gametest.PlayerB, "Wall of Wood", 0)
	g.AssertGraveyardCount(gametest.PlayerB, "Wall of Wood", 1)
}

func TestGraveRobbers(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grave Robbers")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerB, "Black Vise")

	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Grave Robbers", "Black Vise")
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertGraveyardCount(gametest.PlayerB, "Black Vise", 0)
	g.AssertExileCount("Black Vise", 1)
	g.AssertLife(gametest.PlayerA, 22)
}

func TestMerfolkAssassin(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Merfolk Assassin")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Devouring Deep")

	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Merfolk Assassin", "Devouring Deep")
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertPermanentCount(gametest.PlayerB, "Devouring Deep", 0)
	g.AssertGraveyardCount(gametest.PlayerB, "Devouring Deep", 1)
}

func TestMiracleWorker(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Miracle Worker")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Holy Armor")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")

	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Holy Armor", "Grizzly Bears")
	g.ActivateAbility(1, core.PostcombatMain, gametest.PlayerA, "Miracle Worker", "Holy Armor")
	g.StopAt(1, core.EndStep)
	g.Execute()

	g.AssertPermanentCount(gametest.PlayerA, "Holy Armor", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Holy Armor", 1)
}

func TestSavaenElves(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Savaen Elves")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Wild Growth")

	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Wild Growth", "Forest")
	g.ActivateAbility(1, core.PostcombatMain, gametest.PlayerA, "Savaen Elves", "Wild Growth")
	g.StopAt(1, core.EndStep)
	g.Execute()

	g.AssertPermanentCount(gametest.PlayerA, "Wild Growth", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Wild Growth", 1)
}

func TestScavengerFolk(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Scavenger Folk")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Black Vise")

	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Scavenger Folk", "Black Vise")
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertPermanentCount(gametest.PlayerA, "Scavenger Folk", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Scavenger Folk", 1)
	g.AssertPermanentCount(gametest.PlayerB, "Black Vise", 0)
	g.AssertGraveyardCount(gametest.PlayerB, "Black Vise", 1)
}

func TestScarwoodHag(t *testing.T) {
	t.Run("grant forestwalk", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Scarwood Hag")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 4)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")

		g.ActivateAbilityIndex(1, core.PrecombatMain, gametest.PlayerA, "Scarwood Hag", 0, "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()

		bears := g.FindPermanentByName("Grizzly Bears", g.GetPlayer(gametest.PlayerA).PlayerID())
		if bears == nil || !bears.HasKeyword(core.Forestwalk) {
			t.Errorf("Grizzly Bears expected to gain Forestwalk")
		}
	})

	t.Run("lose forestwalk", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Scarwood Hag")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Shanodin Dryads")

		g.ActivateAbilityIndex(1, core.PrecombatMain, gametest.PlayerA, "Scarwood Hag", 1, "Shanodin Dryads")
		g.StopAt(1, core.BeginCombat)
		g.Execute()

		dryads := g.FindPermanentByName("Shanodin Dryads", g.GetPlayer(gametest.PlayerB).PlayerID())
		if dryads == nil || dryads.HasKeyword(core.Forestwalk) {
			t.Errorf("Shanodin Dryads expected to lose Forestwalk")
		}
	})
}

func TestWitchHunter(t *testing.T) {
	t.Run("ping player", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Witch Hunter")

		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Witch Hunter", "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()

		g.AssertLife(gametest.PlayerB, 19)
	})

	t.Run("bounce creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Witch Hunter")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")

		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Witch Hunter", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()

		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertHandCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestWormwoodTreefolk(t *testing.T) {
	t.Run("forestwalk ability", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wormwood Treefolk")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)

		g.ActivateAbilityIndex(1, core.PrecombatMain, gametest.PlayerA, "Wormwood Treefolk", 0)
		g.StopAt(1, core.BeginCombat)
		g.Execute()

		tree := g.FindPermanentByName("Wormwood Treefolk", g.GetPlayer(gametest.PlayerA).PlayerID())
		if tree == nil || !tree.HasKeyword(core.Forestwalk) {
			t.Errorf("Wormwood Treefolk expected to gain Forestwalk")
		}
		g.AssertLife(gametest.PlayerA, 18)
	})

	t.Run("swampwalk ability", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wormwood Treefolk")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)

		g.ActivateAbilityIndex(1, core.PrecombatMain, gametest.PlayerA, "Wormwood Treefolk", 1)
		g.StopAt(1, core.BeginCombat)
		g.Execute()

		tree := g.FindPermanentByName("Wormwood Treefolk", g.GetPlayer(gametest.PlayerA).PlayerID())
		if tree == nil || !tree.HasKeyword(core.Swampwalk) {
			t.Errorf("Wormwood Treefolk expected to gain Swampwalk")
		}
		g.AssertLife(gametest.PlayerA, 18)
	})
}

func TestElectricEel(t *testing.T) {
	t.Run("etb damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Electric Eel")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")

		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Electric Eel")
		g.StopAt(1, core.BeginCombat)
		g.Execute()

		g.AssertPermanentCount(gametest.PlayerA, "Electric Eel", 1)
		g.AssertLife(gametest.PlayerA, 19)
	})

	t.Run("activated pump and damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Electric Eel")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)

		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Electric Eel")
		g.StopAt(1, core.BeginCombat)
		g.Execute()

		g.AssertPowerToughness(gametest.PlayerA, "Electric Eel", 3, 1)
		g.AssertLife(gametest.PlayerA, 18)
	})
}

func TestElvesOfDeepShadow(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Elves of Deep Shadow")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Elves of Deep Shadow")
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertTapped(gametest.PlayerA, "Elves of Deep Shadow", true)
	g.AssertLife(gametest.PlayerA, 19)
}

func TestGoblinsOfTheFlarg(t *testing.T) {
	t.Run("survives without dwarf and has mountainwalk", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblins of the Flarg")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")

		g.Attack(1, gametest.PlayerA, "Goblins of the Flarg")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Goblins of the Flarg")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()

		g.AssertPermanentCount(gametest.PlayerA, "Goblins of the Flarg", 1)
		g.AssertLife(gametest.PlayerB, 19)
	})

	t.Run("sacrificed when controlling a dwarf", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblins of the Flarg")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dwarven Warriors")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()

		g.AssertPermanentCount(gametest.PlayerA, "Goblins of the Flarg", 0)
		g.AssertPermanentCount(gametest.PlayerA, "Dwarven Warriors", 1)
	})
}

func TestPeopleOfTheWoods(t *testing.T) {
	t.Run("toughness equals number of forests", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "People of the Woods")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
		g.StopAt(1, core.PrecombatMain)
		g.Execute()

		g.AssertPowerToughness(gametest.PlayerA, "People of the Woods", 1, 3)
	})

	t.Run("dies with zero forests", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "People of the Woods")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()

		g.AssertPermanentCount(gametest.PlayerA, "People of the Woods", 0)
	})
}

func TestOrcGeneral(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Orc General")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ironclaw Orcs")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblin Hero")

	g.ChoosePermanent(gametest.PlayerA, "Goblin Hero")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Orc General")
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertPermanentCount(gametest.PlayerA, "Goblin Hero", 0)
	g.AssertPowerToughness(gametest.PlayerA, "Ironclaw Orcs", 3, 3)
	g.AssertPowerToughness(gametest.PlayerA, "Orc General", 2, 2)
}

func TestScarecrow(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Scarecrow")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 6)   // 6 mana
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Fire Drake")    // 1/2 Flying
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2 non-flying

	// Turn 2: Player B attacks with both Fire Drake and Grizzly Bears
	g.ActivateAbility(2, core.BeginCombat, gametest.PlayerA, "Scarecrow")
	g.Attack(2, gametest.PlayerB, "Fire Drake", "Grizzly Bears")
	g.StopAt(2, core.PostcombatMain)
	g.Execute()

	// Only Grizzly Bears deals damage; Fire Drake damage is prevented
	g.AssertLife(gametest.PlayerA, 18)
}

func TestTracker(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tracker")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Squire") // 1/2

	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Tracker", "Squire")
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertPermanentCount(gametest.PlayerB, "Squire", 0)
	tracker := g.FindPermanentByName("Tracker", g.GetPlayer(gametest.PlayerA).PlayerID())
	if tracker == nil {
		t.Fatal("Tracker not found")
	}
	if tracker.Damage != 1 {
		t.Errorf("Tracker damage = %d, want 1", tracker.Damage)
	}
}

func TestEaterOfTheDead(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Eater of the Dead")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerB, "Grizzly Bears")

	// Tap Eater of the Dead first by attacking on turn 1
	g.Attack(1, gametest.PlayerA, "Eater of the Dead")
	// On postcombat main, activate ability to untap
	g.ActivateAbility(1, core.PostcombatMain, gametest.PlayerA, "Eater of the Dead", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()

	g.AssertExileCount("Grizzly Bears", 1)
	g.AssertTapped(gametest.PlayerA, "Eater of the Dead", false)
}

func TestBanshee(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Banshee")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 4)

	g.ActivateAbilityWithX(1, core.PrecombatMain, gametest.PlayerA, "Banshee", 3, "PlayerB")
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	// Banshee deals floor(3/2) = 1 damage to PlayerB and ceil(3/2) = 2 damage to PlayerA
	g.AssertLife(gametest.PlayerB, 19)
	g.AssertLife(gametest.PlayerA, 18)
}

func TestGiantShark(t *testing.T) {
	t.Run("cant attack without defending island", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Giant Shark")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.Attack(1, gametest.PlayerA, "Giant Shark")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()

		g.AssertLife(gametest.PlayerB, 20)
	})

	t.Run("sacrificed without islands", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Giant Shark")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()

		g.AssertPermanentCount(gametest.PlayerA, "Giant Shark", 0)
	})

	t.Run("pumps when blocking damaged creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Giant Shark")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Prodigal Sorcerer")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3

		// Turn 2: Ping Hill Giant with Prodigal Sorcerer, then Hill Giant attacks and Shark blocks
		g.ActivateAbility(2, core.BeginCombat, gametest.PlayerA, "Prodigal Sorcerer", "Hill Giant")
		g.Attack(2, gametest.PlayerB, "Hill Giant")
		g.Block(2, gametest.PlayerA, "Giant Shark", "Hill Giant")
		g.StopAt(2, core.PostcombatMain)
		g.Execute()

		// Giant Shark triggers and gets +2/+0 (6/4) and trample
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 0)
		g.AssertPowerToughness(gametest.PlayerA, "Giant Shark", 6, 4)
	})
}

func TestGoblinWizard(t *testing.T) {
	t.Run("put goblin onto battlefield", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblin Wizard")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Goblin Hero")

		g.ActivateAbilityIndex(1, core.PrecombatMain, gametest.PlayerA, "Goblin Wizard", 0)
		g.StopAt(1, core.BeginCombat)
		g.Execute()

		g.AssertPermanentCount(gametest.PlayerA, "Goblin Hero", 1)
		g.AssertHandCount(gametest.PlayerA, "Goblin Hero", 0)
	})

	t.Run("grant protection from white", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblin Wizard")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblin Hero")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "White Knight")

		g.ActivateAbilityIndex(1, core.PrecombatMain, gametest.PlayerA, "Goblin Wizard", 1, "Goblin Hero")
		g.Attack(1, gametest.PlayerA, "Goblin Hero")
		g.Block(1, gametest.PlayerB, "White Knight", "Goblin Hero")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()

		// Protection from white prevents block/damage from White Knight
		g.AssertLife(gametest.PlayerB, 18)
	})
}

func TestPreacher(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Preacher")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")

	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Preacher", "Grizzly Bears")
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
}

func TestScarwoodBandits(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Scarwood Bandits")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Black Vise")

	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Scarwood Bandits", "Black Vise")
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertPermanentCount(gametest.PlayerA, "Black Vise", 1)
	g.AssertPermanentCount(gametest.PlayerB, "Black Vise", 0)
}

func TestNecropolis(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Necropolis")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears") // MV 2

	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Necropolis")
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertExileCount("Grizzly Bears", 1)
	g.AssertPowerToughness(gametest.PlayerA, "Necropolis", 0, 3)
}

func TestLurker(t *testing.T) {
	t.Run("cant be target of spells before attacking or blocking", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lurker")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")

		// Player B tries to bolt Lurker on turn 1 before it attacks
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Lurker")
		g.StopAt(1, core.BeginCombat)
		g.Execute()

		// Lurker was an illegal target; remains on battlefield
		g.AssertPermanentCount(gametest.PlayerA, "Lurker", 1)
	})

	t.Run("can be target of spells after attacking", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lurker")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")

		g.Attack(1, gametest.PlayerA, "Lurker")
		g.CastSpell(1, core.PostcombatMain, gametest.PlayerB, "Lightning Bolt", "Lurker")
		g.StopAt(1, core.EndStep)
		g.Execute()

		g.AssertPermanentCount(gametest.PlayerA, "Lurker", 0)
	})
}

func TestTheFallen(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "The Fallen")

	// Turn 1: Attack and deal combat damage to Player B
	g.Attack(1, gametest.PlayerA, "The Fallen")
	// Turn 3: At Player A's next upkeep, The Fallen deals 1 damage to Player B
	g.StopAt(3, core.Draw)
	g.Execute()

	// Combat damage (2) + Upkeep damage (1) = 3 total damage
	g.AssertLife(gametest.PlayerB, 17)
}

func TestSpittingSlug(t *testing.T) {
	t.Run("pays mana for first strike", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Spitting Slug")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2

		g.Attack(2, gametest.PlayerB, "Grizzly Bears")
		g.Block(2, gametest.PlayerA, "Spitting Slug", "Grizzly Bears")
		g.StopAt(2, core.PostcombatMain)
		g.Execute()

		// Spitting Slug gets first strike and kills Grizzly Bears before taking damage
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		slug := g.FindPermanentByName("Spitting Slug", g.GetPlayer(gametest.PlayerA).PlayerID())
		if slug == nil {
			t.Fatal("Spitting Slug not found")
		}
		if slug.Damage != 0 {
			t.Errorf("Spitting Slug damage = %d, want 0", slug.Damage)
		}
	})

	t.Run("does not pay, blocker gets first strike", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Spitting Slug")     // 2/4, no mana
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wormwood Treefolk") // 4/4

		g.Attack(1, gametest.PlayerA, "Spitting Slug")
		g.Block(1, gametest.PlayerB, "Wormwood Treefolk", "Spitting Slug")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()

		// Wormwood Treefolk gained first strike, dealt 4 damage, killing Spitting Slug
		g.AssertPermanentCount(gametest.PlayerA, "Spitting Slug", 0)
		treefolk := g.FindPermanentByName("Wormwood Treefolk", g.GetPlayer(gametest.PlayerB).PlayerID())
		if treefolk == nil {
			t.Fatal("Wormwood Treefolk not found")
		}
		if treefolk.Damage != 0 {
			t.Errorf("Wormwood Treefolk damage = %d, want 0", treefolk.Damage)
		}
	})
}

func TestWhippoorwill(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Whippoorwill")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Drudge Skeletons")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Swamp")

	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Whippoorwill", "Drudge Skeletons")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Drudge Skeletons")
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertPermanentCount(gametest.PlayerB, "Drudge Skeletons", 0)
	g.AssertExileCount("Drudge Skeletons", 1)
}

func TestFrankensteinsMonster(t *testing.T) {
	t.Run("dies if cannot exile X creature cards", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Frankenstein's Monster")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears") // only 1 card

		// Cast with X=2, but only 1 creature in graveyard
		g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Frankenstein's Monster", 2)
		g.StopAt(1, core.BeginCombat)
		g.Execute()

		g.AssertPermanentCount(gametest.PlayerA, "Frankenstein's Monster", 0)
		g.AssertGraveyardCount(gametest.PlayerA, "Frankenstein's Monster", 1)
	})

	t.Run("exiles X creature cards and enters with counters", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Frankenstein's Monster")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Goblin Hero")

		// Cast with X=2
		g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Frankenstein's Monster", 2)
		g.StopAt(1, core.BeginCombat)
		g.Execute()

		g.AssertPermanentCount(gametest.PlayerA, "Frankenstein's Monster", 1)
		g.AssertExileCount("Grizzly Bears", 1)
		g.AssertExileCount("Goblin Hero", 1)
		// Base 0/1 + 2 counters (+1/+1 default or chosen) = 2/3
		g.AssertPowerToughness(gametest.PlayerA, "Frankenstein's Monster", 2, 3)
	})
}

func TestNamelessRace(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Nameless Race")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "White Knight")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mesa Pegasus")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerB, "Benalish Hero")

	// Total white cards/nontoken permanents = 3
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Nameless Race", 3)
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertLife(gametest.PlayerA, 17)
	g.AssertPowerToughness(gametest.PlayerA, "Nameless Race", 3, 3)
}
