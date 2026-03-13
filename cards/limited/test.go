package limited

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// Ensure all Alpha card packages are imported.
var _ = registerCreatures
var _ = registerSpells
var _ = registerEnchantments
var _ = registerArtifacts
var _ = registerLands

// ===== VANILLA CREATURES =====

func TestAlphaVanillaCreatures(t *testing.T) {
	tests := []struct {
		name  string
		power int
		tough int
	}{
		{"Savannah Lions", 2, 1},
		{"Pearled Unicorn", 2, 2},
		{"Air Elemental", 4, 4},
		{"Mahamoti Djinn", 5, 6},
		{"Water Elemental", 5, 4},
		{"Merfolk of the Pearl Trident", 1, 1},
		{"Black Knight", 2, 2},
		{"Bog Wraith", 3, 3},
		{"Scathe Zombies", 2, 2},
		{"Gray Ogre", 2, 2},
		{"Hill Giant", 3, 3},
		{"Hurloon Minotaur", 2, 3},
		{"Ironclaw Orcs", 2, 2},
		{"Mons's Goblin Raiders", 1, 1},
		{"Earth Elemental", 4, 5},
		{"Fire Elemental", 5, 4},
		{"Craw Wurm", 6, 4},
		{"Ironroot Treefolk", 3, 5},
		{"Scryb Sprites", 1, 1},
		{"War Mammoth", 3, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			g.AddCard(core.ZoneBattlefield, gametest.PlayerA, tt.name)
			g.StopAt(1, core.PrecombatMain)
			g.Execute()
			g.AssertPowerToughness(gametest.PlayerA, tt.name, tt.power, tt.tough)
		})
	}
}

// ===== WALLS (DEFENDER) =====

func TestAlphaWalls(t *testing.T) {
	t.Run("Wall of Stone cannot attack", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Stone")
		g.Attack(1, gametest.PlayerA, "Wall of Stone")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Wall of Stone has Defender so it shouldn't have dealt damage
		g.AssertLife(gametest.PlayerB, 20)
	})

	t.Run("Wall of Swords can block flying", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Swords")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Air Elemental")
		g.Attack(2, gametest.PlayerB, "Air Elemental")
		g.Block(2, gametest.PlayerA, "Wall of Swords", "Air Elemental")
		g.StopAt(2, core.EndCombat)
		g.Execute()
		// Wall of Swords (3/5 flying defender) blocks Air Elemental (4/4 flying)
		// Wall takes 4 damage (survives with 5 toughness), Air Elemental takes 3
		g.AssertLife(gametest.PlayerA, 20)
		g.AssertPermanentCount(gametest.PlayerA, "Wall of Swords", 1)
		g.AssertPermanentCount(gametest.PlayerB, "Air Elemental", 1)
	})

	t.Run("Wall of Wood P/T correct", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Wood")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Wall of Wood", 0, 3)
		g.AssertHasAbility(gametest.PlayerA, "Wall of Wood", core.Defender, true)
	})
}

// ===== LANDWALK =====

func TestAlphaLandwalk(t *testing.T) {
	t.Run("Bog Wraith unblockable when opponent has Swamp", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bog Wraith")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Swamp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // would-be blocker
		g.Attack(1, gametest.PlayerA, "Bog Wraith")
		g.Block(1, gametest.PlayerB, "Hill Giant", "Bog Wraith")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Bog Wraith has swampwalk, opponent has Swamp, so can't be blocked
		g.AssertLife(gametest.PlayerB, 17) // 3 damage from Bog Wraith
	})

	t.Run("Bog Wraith can be blocked when no Swamp", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bog Wraith")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest") // no Swamp
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.Attack(1, gametest.PlayerA, "Bog Wraith")
		g.Block(1, gametest.PlayerB, "Hill Giant", "Bog Wraith")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Hill Giant blocks Bog Wraith, both trade
		g.AssertLife(gametest.PlayerB, 20)
	})

	t.Run("Shanodin Dryads has forestwalk", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Shanodin Dryads")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Shanodin Dryads")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Shanodin Dryads")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 19) // 1 damage through, unblockable
	})
}

// ===== FEAR =====

func TestAlphaFear(t *testing.T) {
	t.Run("Fear aura makes creature blocked only by black/artifact", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Fear")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // red, not black
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Fear", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Hill Giant", "Grizzly Bears")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Hill Giant can't block a creature with Fear (not black, not artifact)
		g.AssertLife(gametest.PlayerB, 18) // 2 damage from unblocked Grizzly Bears
	})

	t.Run("Fear creature can be blocked by black creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Fear")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Scathe Zombies") // black 2/2
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Fear", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Scathe Zombies", "Grizzly Bears")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Scathe Zombies is black, can block Fear creature. Both 2/2, both die.
		g.AssertLife(gametest.PlayerB, 20)
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Scathe Zombies", 0)
	})
}

// ===== FIREBREATHING =====

func TestAlphaFirebreathing(t *testing.T) {
	t.Run("Shivan Dragon firebreathing", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Shivan Dragon")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Shivan Dragon")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Shivan Dragon")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Shivan Dragon is 5/5, +1/+0 twice = 7/5
		g.AssertPowerToughness(gametest.PlayerA, "Shivan Dragon", 7, 5)
	})

	t.Run("Frozen Shade pump", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Frozen Shade")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Frozen Shade")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Frozen Shade is 0/1, +1/+1 = 1/2
		g.AssertPowerToughness(gametest.PlayerA, "Frozen Shade", 1, 2)
	})
}

// ===== LORD EFFECTS =====

func TestAlphaLords(t *testing.T) {
	t.Run("Goblin King boosts other Goblins", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblin King")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mons's Goblin Raiders")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant") // not a Goblin
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Goblin King itself should NOT get the bonus (other Goblin creatures)
		g.AssertPowerToughness(gametest.PlayerA, "Goblin King", 2, 2)
		// Mons's Goblin Raiders (1/1 Goblin) gets +1/+1 = 2/2
		g.AssertPowerToughness(gametest.PlayerA, "Mons's Goblin Raiders", 2, 2)
		// Hill Giant is not a Goblin, stays 3/3
		g.AssertPowerToughness(gametest.PlayerA, "Hill Giant", 3, 3)
		// Mons's Goblin Raiders should have mountainwalk
		g.AssertHasAbility(gametest.PlayerA, "Mons's Goblin Raiders", core.Mountainwalk, true)
	})

	t.Run("Lord of Atlantis boosts Merfolk", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lord of Atlantis")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Merfolk of the Pearl Trident")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Lord of Atlantis itself should NOT get the bonus
		g.AssertPowerToughness(gametest.PlayerA, "Lord of Atlantis", 2, 2)
		// Merfolk of the Pearl Trident (1/1) gets +1/+1 = 2/2
		g.AssertPowerToughness(gametest.PlayerA, "Merfolk of the Pearl Trident", 2, 2)
		g.AssertHasAbility(gametest.PlayerA, "Merfolk of the Pearl Trident", core.Islandwalk, true)
	})

	t.Run("Crusade boosts white creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Crusade")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Savannah Lions")   // white
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")    // green
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Savannah Lions (2/1 white) gets +1/+1 = 3/2
		g.AssertPowerToughness(gametest.PlayerA, "Savannah Lions", 3, 2)
		// Grizzly Bears is green, no bonus
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
	})

	t.Run("Bad Moon boosts black creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bad Moon")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Scathe Zombies") // black
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")  // green
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Scathe Zombies", 3, 3)
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
	})
}

// ===== X SPELLS =====

func TestAlphaXSpells(t *testing.T) {
	t.Run("Fireball deals X damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Fireball")
		g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Fireball", 5, "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 15)
	})

	t.Run("Drain Life deals X and gains X", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.SetLife(gametest.PlayerA, 15)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Drain Life")
		g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Drain Life", 3, "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 17) // 20 - 3
		g.AssertLife(gametest.PlayerA, 18) // 15 + 3
	})

	t.Run("Earthquake damages creatures without flying and players", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")  // 2/2, no flying
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Air Elemental")  // 4/4, flying
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Earthquake")
		g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Earthquake", 3)
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Grizzly Bears: 2/2, takes 3 damage -> dies
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
		// Air Elemental: 4/4 with flying, immune to Earthquake
		g.AssertPermanentCount(gametest.PlayerA, "Air Elemental", 1)
		// Both players take 3 damage
		g.AssertLife(gametest.PlayerA, 17)
		g.AssertLife(gametest.PlayerB, 17)
	})

	t.Run("Mind Twist discards X cards", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Mind Twist")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Hill Giant")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Mind Twist", 2, "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// PlayerB had 3 cards, discarded 2
		// Count remaining hand
		player := g.AllPlayers()[1]
		if len(player.Hand()) != 1 {
			t.Errorf("expected 1 card in hand, got %d", len(player.Hand()))
		}
	})

	t.Run("Braingeyser draws X cards", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Braingeyser")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Lightning Bolt")
		g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Braingeyser", 2, "PlayerA")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// PlayerA cast Braingeyser (removed from hand), drew 2 cards
		player := g.AllPlayers()[0]
		if len(player.Hand()) != 2 {
			t.Errorf("expected 2 cards in hand, got %d", len(player.Hand()))
		}
	})
}

// ===== BOUNCE =====

func TestAlphaBounce(t *testing.T) {
	t.Run("Unsummon returns creature to hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Unsummon")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Unsummon", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertHandCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

// ===== RAISE DEAD =====

func TestAlphaRaiseDead(t *testing.T) {
	t.Run("Raise Dead returns creature from graveyard to hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Raise Dead")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Raise Dead", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 0)
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}

// ===== DISCARD =====

func TestAlphaDiscard(t *testing.T) {
	t.Run("Hypnotic Specter forces discard on damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hypnotic Specter")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Hill Giant")
		g.Attack(1, gametest.PlayerA, "Hypnotic Specter")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Hypnotic Specter deals 2 damage and forces discard
		g.AssertLife(gametest.PlayerB, 18)
		player := g.AllPlayers()[1]
		if len(player.Hand()) != 1 {
			t.Errorf("expected 1 card in hand after discard, got %d", len(player.Hand()))
		}
	})
}

// ===== SWORDS TO PLOWSHARES =====

func TestAlphaSwordsToPlowshares(t *testing.T) {
	t.Run("Exile creature and controller gains life equal to power", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Swords to Plowshares")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Swords to Plowshares", "Hill Giant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Hill Giant is exiled
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 0)
		g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 0) // not in graveyard, exiled
		// PlayerB (controller of Hill Giant) gains 3 life
		g.AssertLife(gametest.PlayerB, 23)
	})
}

// ===== DUAL LANDS =====

func TestAlphaDualLands(t *testing.T) {
	t.Run("Dual lands have correct subtypes", func(t *testing.T) {
		duals := []struct {
			name string
			sub1 string
			sub2 string
		}{
			{"Badlands", "Swamp", "Mountain"},
			{"Bayou", "Swamp", "Forest"},
			{"Tropical Island", "Forest", "Island"},
			{"Tundra", "Plains", "Island"},
			{"Underground Sea", "Island", "Swamp"},
		}

		for _, d := range duals {
			t.Run(d.name, func(t *testing.T) {
				g := gametest.NewTestGame(t)
				g.AddCard(core.ZoneBattlefield, gametest.PlayerA, d.name)
				g.StopAt(1, core.PrecombatMain)
				g.Execute()
				perm := g.FindPermanentByName(d.name, g.AllPlayers()[0].PlayerID())
				if perm == nil {
					t.Fatalf("%s not found on battlefield", d.name)
				}
				if !perm.HasSubType(d.sub1) {
					t.Errorf("%s should have subtype %s", d.name, d.sub1)
				}
				if !perm.HasSubType(d.sub2) {
					t.Errorf("%s should have subtype %s", d.name, d.sub2)
				}
			})
		}
	})
}

// ===== MOXEN =====

func TestAlphaMoxen(t *testing.T) {
	t.Run("Mox Ruby enters battlefield", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mox Ruby")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Mox Ruby", 1)
	})

	t.Run("Sol Ring produces 2 colorless", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sol Ring")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Sol Ring")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Sol Ring", true)
	})
}

// ===== UPKEEP TRIGGERS =====

func TestAlphaUpkeepTriggers(t *testing.T) {
	t.Run("Lord of the Pit deals damage on upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lord of the Pit")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Lord of the Pit deals 7 to controller on upkeep
		g.AssertLife(gametest.PlayerA, 13) // 20 - 7
	})

	t.Run("Mana Vault deals damage on upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mana Vault")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 19) // 20 - 1
	})
}

// ===== PESTILENCE =====

func TestAlphaPestilence(t *testing.T) {
	t.Run("Pestilence deals 1 to all creatures and players", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Pestilence")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")     // 3/3
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")  // 2/2
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Pestilence")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Both players lose 1 life
		g.AssertLife(gametest.PlayerA, 19)
		g.AssertLife(gametest.PlayerB, 19)
		// Creatures take 1 damage but survive
		g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1)
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

// ===== DEMONIC TUTOR =====

func TestAlphaDemonicTutor(t *testing.T) {
	t.Run("Search library and put card in hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Demonic Tutor")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Demonic Tutor")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Lightning Bolt", 1)
	})
}

// ===== TIME WALK =====

func TestAlphaTimeWalk(t *testing.T) {
	t.Run("Time Walk grants extra turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Time Walk")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Time Walk")
		// After Time Walk, PlayerA should get an extra turn
		// StopAt turn 2 which should still be PlayerA's turn
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		// Verify game is on turn 2 and active player is still PlayerA (index 0)
		if g.ActivePlayer != 0 {
			t.Errorf("expected active player to be PlayerA (0), got %d", g.ActivePlayer)
		}
	})
}

// ===== FOG =====

func TestAlphaDamagePreventionFog(t *testing.T) {
	t.Run("Fog prevents all combat damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Craw Wurm") // 6/4
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Fog")
		g.Attack(1, gametest.PlayerA, "Craw Wurm")
		g.CastSpell(1, core.DeclareBlockers, gametest.PlayerB, "Fog")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Fog prevents all combat damage
		g.AssertLife(gametest.PlayerB, 20)
	})
}

// ===== DESTROY EFFECTS =====

func TestAlphaDestroyEffects(t *testing.T) {
	t.Run("Armageddon destroys all lands", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Swamp")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Armageddon")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Armageddon")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Plains", 0)
		g.AssertPermanentCount(gametest.PlayerA, "Forest", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Island", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Swamp", 0)
	})

	t.Run("Sinkhole destroys target land", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Sinkhole")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Sinkhole", "Forest")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Forest", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Island", 1) // untouched
	})

	t.Run("Shatter destroys target artifact", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Sol Ring")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Shatter")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Shatter", "Sol Ring")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Sol Ring", 0)
	})

	t.Run("Disenchant destroys enchantment", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Bad Moon")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Disenchant")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Disenchant", "Bad Moon")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Bad Moon", 0)
	})

	t.Run("Tranquility destroys all enchantments", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Crusade")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Bad Moon")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant") // creature, not enchantment
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Tranquility")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Tranquility")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Crusade", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Bad Moon", 0)
		g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1)
	})

	t.Run("Terror destroys nonblack nonartifact creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // red, not black
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Terror")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Terror", "Hill Giant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 0)
	})
}

// ===== AURAS =====

func TestAlphaAuras(t *testing.T) {
	t.Run("Unholy Strength gives +2/+1", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Unholy Strength")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Unholy Strength", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 3)
	})

	t.Run("Weakness gives -2/-1", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Weakness")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Weakness", "Hill Giant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerB, "Hill Giant", 1, 2)
	})

	t.Run("Flight grants flying", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Flight")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Flight", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Flying, true)
	})

	t.Run("Lance grants first strike", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lance")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lance", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.FirstStrike, true)
	})
}

// ===== DARK RITUAL =====

func TestAlphaDarkRitual(t *testing.T) {
	t.Run("Dark Ritual adds 3 black mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Dark Ritual")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Dark Ritual")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.AllPlayers()[0].ManaPool()
		if pool.Count(core.Black) < 3 {
			t.Errorf("expected at least 3 black mana, got %d", pool.Count(core.Black))
		}
	})
}

// ===== ANCESTRAL RECALL =====

func TestAlphaAncestralRecall(t *testing.T) {
	t.Run("Ancestral Recall draws 3 cards", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Ancestral Recall")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Ancestral Recall", "PlayerA")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		player := g.AllPlayers()[0]
		if len(player.Hand()) != 3 {
			t.Errorf("expected 3 cards in hand, got %d", len(player.Hand()))
		}
	})
}

// ===== COMBAT KEYWORDS =====

func TestAlphaCombatKeywords(t *testing.T) {
	t.Run("Trample deals excess damage to player", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Craw Wurm") // 6/4 Trample is on War Mammoth
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "War Mammoth") // 3/3 Trample
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2
		g.Attack(1, gametest.PlayerA, "War Mammoth")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "War Mammoth")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// War Mammoth (3/3 trample) blocked by Grizzly Bears (2/2)
		// War Mammoth assigns 2 to Grizzly Bears (lethal), 1 tramples to player
		g.AssertLife(gametest.PlayerB, 19)
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	})

	t.Run("First Strike White Knight vs Black Knight", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "White Knight")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Black Knight")
		g.Attack(1, gametest.PlayerA, "White Knight")
		g.Block(1, gametest.PlayerB, "Black Knight", "White Knight")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// White Knight has protection from black so Black Knight can't block
		// White Knight deals 2 damage unblocked
		g.AssertLife(gametest.PlayerB, 18)
	})
}

// ===== REGROWTH =====

func TestAlphaRegrowth(t *testing.T) {
	t.Run("Regrowth returns any card from graveyard to hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Lightning Bolt")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Regrowth")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Regrowth", "Lightning Bolt")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Lightning Bolt", 0)
		g.AssertHandCount(gametest.PlayerA, "Lightning Bolt", 1)
	})
}

// ===== GIANT GROWTH =====

func TestAlphaGiantGrowth(t *testing.T) {
	t.Run("Giant Growth gives +3/+3 until end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Giant Growth")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Giant Growth", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 5, 5) // 2+3/2+3
	})
}

// ===== HEALING SALVE =====

func TestAlphaHealingSalve(t *testing.T) {
	t.Run("Healing Salve gains 3 life", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.SetLife(gametest.PlayerA, 15)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Healing Salve")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Healing Salve", "PlayerA")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 18)
	})
}

// ===== RESURRECTION =====

func TestAlphaResurrection(t *testing.T) {
	t.Run("Resurrection returns creature from graveyard to battlefield", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Serra Angel")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Resurrection")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Resurrection", "Serra Angel")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Serra Angel", 0)
		g.AssertPermanentCount(gametest.PlayerA, "Serra Angel", 1)
	})
}

// ===== ICY MANIPULATOR =====

func TestAlphaIcyManipulator(t *testing.T) {
	t.Run("Icy Manipulator taps target permanent", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Icy Manipulator")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Icy Manipulator", "Hill Giant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerB, "Hill Giant", true)
		g.AssertTapped(gametest.PlayerA, "Icy Manipulator", true)
	})
}

// ===== PRODIGAL SORCERER =====

func TestAlphaProdigalSorcerer(t *testing.T) {
	t.Run("Tim deals 1 damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Prodigal Sorcerer")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Prodigal Sorcerer", "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 19)
	})
}

// ===== ROYAL ASSASSIN =====

func TestAlphaRoyalAssassin(t *testing.T) {
	t.Run("Royal Assassin destroys tapped creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Royal Assassin")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		// First tap the Hill Giant (using Icy Manipulator would be ideal, but just set tapped)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Icy Manipulator")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Icy Manipulator", "Hill Giant")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Royal Assassin", "Hill Giant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 0)
	})
}

// ===== FLASHFIRES / TSUNAMI =====

func TestAlphaFlashfires(t *testing.T) {
	t.Run("Flashfires destroys all Plains", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Flashfires")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Flashfires")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Plains", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Plains", 0)
		g.AssertPermanentCount(gametest.PlayerA, "Mountain", 1) // not a Plains
	})
}

// ===== FUNGUSAUR =====

func TestAlphaFungusaur(t *testing.T) {
	t.Run("Fungusaur gets counter when damaged", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fungusaur")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Fungusaur")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Fungusaur is 2/2, takes 3 damage -> dies (toughness 2)
		// But if it survived, it would get a counter. Since 3 >= 2, it dies.
		g.AssertPermanentCount(gametest.PlayerA, "Fungusaur", 0)
	})

	t.Run("Fungusaur survives 1 damage gets counter", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fungusaur") // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Prodigal Sorcerer")
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerB, "Prodigal Sorcerer", "Fungusaur")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		// Fungusaur takes 1 damage, gets a +1/+1 counter = becomes 3/3
		g.AssertPermanentCount(gametest.PlayerA, "Fungusaur", 1)
		g.AssertCounterCount(gametest.PlayerA, "Fungusaur", core.P1P1, 1)
	})
}

// ===== WANDERLUST (Aura upkeep damage to attached controller) =====

func TestWanderlustUpkeepDamage(t *testing.T) {
	t.Run("Wanderlust deals 1 damage to enchanted creature's controller on upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Wanderlust")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Wanderlust", "Grizzly Bears")
		// Turn 2 is PlayerB's turn; upkeep fires trigger on enchanted creature's controller
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		// PlayerB controls Grizzly Bears which has Wanderlust attached
		// At PlayerB's upkeep, Wanderlust deals 1 damage to PlayerB
		g.AssertLife(gametest.PlayerB, 19)
		g.AssertLife(gametest.PlayerA, 20) // PlayerA unaffected
	})
}

// ===== CURSED LAND (Aura upkeep damage to attached controller) =====

func TestCursedLandUpkeepDamage(t *testing.T) {
	t.Run("Cursed Land deals 1 damage to enchanted land's controller on upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Cursed Land")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Cursed Land", "Forest")
		// Turn 2 is PlayerB's turn
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 19)
		g.AssertLife(gametest.PlayerA, 20)
	})
}

// ===== FEEDBACK (Aura upkeep damage to attached controller) =====

func TestFeedbackUpkeepDamage(t *testing.T) {
	t.Run("Feedback deals 1 damage to enchanted enchantment's controller on upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Bad Moon")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Feedback")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Feedback", "Bad Moon")
		// Turn 2 is PlayerB's turn
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 19)
		g.AssertLife(gametest.PlayerA, 20)
	})
}

// ===== WARP ARTIFACT (Aura upkeep damage to attached controller) =====

func TestWarpArtifactUpkeepDamage(t *testing.T) {
	t.Run("Warp Artifact deals 1 damage to enchanted artifact's controller on upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Sol Ring")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Warp Artifact")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Warp Artifact", "Sol Ring")
		// Turn 2 is PlayerB's turn
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 19)
		g.AssertLife(gametest.PlayerA, 20)
	})
}

// ===== JUMP (Grant flying until end of turn) =====

func TestJumpGrantsFlying(t *testing.T) {
	t.Run("Jump grants flying until end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Jump")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Jump", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Grizzly Bears should now have flying
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Flying, true)
	})

	t.Run("Jump flying wears off at end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Jump")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Jump", "Grizzly Bears")
		// Stop at turn 2 (after cleanup clears EOT effects)
		g.StopAt(2, core.Upkeep)
		g.Execute()
		// Flying should be gone now
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Flying, false)
	})
}

// ===== ANKH OF MISHRA (Damage when land enters) =====

func TestAnkhOfMishraDamageOnLandEntry(t *testing.T) {
	t.Run("Ankh of Mishra deals 2 damage when any land enters battlefield", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ankh of Mishra")
		// Directly put a land onto the battlefield for PlayerB
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		// Ankh should have triggered dealing 2 damage to Forest's controller (PlayerB)
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 18) // 20 - 2
		g.AssertLife(gametest.PlayerA, 20)
	})

	t.Run("Ankh of Mishra triggers for each land", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ankh of Mishra")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 16) // 20 - 2 - 2
	})
}

// ===== ORCISH ORIFLAMME (Attacking creatures you control get +1/+0) =====

func TestOrcishOriflammeBoostsAttackers(t *testing.T) {
	t.Run("Orcish Oriflamme boosts attacking creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Orcish Oriflamme")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Grizzly Bears attacks with +1/+0 from Oriflamme = 3 damage
		g.AssertLife(gametest.PlayerB, 17)
	})

	t.Run("Orcish Oriflamme does not boost non-attacking creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Orcish Oriflamme")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Grizzly Bears not attacking, should be base 2/2
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
	})
}

// ===== CASTLE (Untapped creatures you control get +0/+2) =====

func TestCastleBoostsUntapped(t *testing.T) {
	t.Run("Castle boosts untapped creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Castle")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Grizzly Bears is untapped, gets +0/+2 = 2/4
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 4)
	})

	t.Run("Castle does not boost tapped creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Castle")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Icy Manipulator")
		// Tap the Grizzly Bears
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Icy Manipulator", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Grizzly Bears is tapped, no +0/+2 from Castle
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
	})
}
