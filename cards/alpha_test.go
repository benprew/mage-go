package cards

import (
	"testing"

	"github.com/mage/mage"
)

// Ensure all Alpha card packages are imported.
var _ = registerAlphaCreatures
var _ = registerAlphaSpells
var _ = registerAlphaEnchantments
var _ = registerAlphaArtifacts
var _ = registerAlphaLands

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
			g := mage.NewTestGame(t)
			g.AddCard(mage.ZoneBattlefield, mage.PlayerA, tt.name)
			g.StopAt(1, mage.PrecombatMain)
			g.Execute()
			g.AssertPowerToughness(mage.PlayerA, tt.name, tt.power, tt.tough)
		})
	}
}

// ===== WALLS (DEFENDER) =====

func TestAlphaWalls(t *testing.T) {
	t.Run("Wall of Stone cannot attack", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Wall of Stone")
		g.Attack(1, mage.PlayerA, "Wall of Stone")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		// Wall of Stone has Defender so it shouldn't have dealt damage
		g.AssertLife(mage.PlayerB, 20)
	})

	t.Run("Wall of Swords can block flying", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Wall of Swords")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Air Elemental")
		g.Attack(2, mage.PlayerB, "Air Elemental")
		g.Block(2, mage.PlayerA, "Wall of Swords", "Air Elemental")
		g.StopAt(2, mage.EndCombat)
		g.Execute()
		// Wall of Swords (3/5 flying defender) blocks Air Elemental (4/4 flying)
		// Wall takes 4 damage (survives with 5 toughness), Air Elemental takes 3
		g.AssertLife(mage.PlayerA, 20)
		g.AssertPermanentCount(mage.PlayerA, "Wall of Swords", 1)
		g.AssertPermanentCount(mage.PlayerB, "Air Elemental", 1)
	})

	t.Run("Wall of Wood P/T correct", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Wall of Wood")
		g.StopAt(1, mage.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(mage.PlayerA, "Wall of Wood", 0, 3)
		g.AssertHasAbility(mage.PlayerA, "Wall of Wood", mage.Defender, true)
	})
}

// ===== LANDWALK =====

func TestAlphaLandwalk(t *testing.T) {
	t.Run("Bog Wraith unblockable when opponent has Swamp", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Bog Wraith")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Swamp")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Hill Giant") // would-be blocker
		g.Attack(1, mage.PlayerA, "Bog Wraith")
		g.Block(1, mage.PlayerB, "Hill Giant", "Bog Wraith")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		// Bog Wraith has swampwalk, opponent has Swamp, so can't be blocked
		g.AssertLife(mage.PlayerB, 17) // 3 damage from Bog Wraith
	})

	t.Run("Bog Wraith can be blocked when no Swamp", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Bog Wraith")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Forest") // no Swamp
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Hill Giant")
		g.Attack(1, mage.PlayerA, "Bog Wraith")
		g.Block(1, mage.PlayerB, "Hill Giant", "Bog Wraith")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		// Hill Giant blocks Bog Wraith, both trade
		g.AssertLife(mage.PlayerB, 20)
	})

	t.Run("Shanodin Dryads has forestwalk", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Shanodin Dryads")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Forest")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Grizzly Bears")
		g.Attack(1, mage.PlayerA, "Shanodin Dryads")
		g.Block(1, mage.PlayerB, "Grizzly Bears", "Shanodin Dryads")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		g.AssertLife(mage.PlayerB, 19) // 1 damage through, unblockable
	})
}

// ===== FEAR =====

func TestAlphaFear(t *testing.T) {
	t.Run("Fear aura makes creature blocked only by black/artifact", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Fear")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Hill Giant") // red, not black
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Fear", "Grizzly Bears")
		g.Attack(1, mage.PlayerA, "Grizzly Bears")
		g.Block(1, mage.PlayerB, "Hill Giant", "Grizzly Bears")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		// Hill Giant can't block a creature with Fear (not black, not artifact)
		g.AssertLife(mage.PlayerB, 18) // 2 damage from unblocked Grizzly Bears
	})

	t.Run("Fear creature can be blocked by black creature", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Fear")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Scathe Zombies") // black 2/2
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Fear", "Grizzly Bears")
		g.Attack(1, mage.PlayerA, "Grizzly Bears")
		g.Block(1, mage.PlayerB, "Scathe Zombies", "Grizzly Bears")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		// Scathe Zombies is black, can block Fear creature. Both 2/2, both die.
		g.AssertLife(mage.PlayerB, 20)
		g.AssertPermanentCount(mage.PlayerA, "Grizzly Bears", 0)
		g.AssertPermanentCount(mage.PlayerB, "Scathe Zombies", 0)
	})
}

// ===== FIREBREATHING =====

func TestAlphaFirebreathing(t *testing.T) {
	t.Run("Shivan Dragon firebreathing", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Shivan Dragon")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Shivan Dragon")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Shivan Dragon")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Shivan Dragon is 5/5, +1/+0 twice = 7/5
		g.AssertPowerToughness(mage.PlayerA, "Shivan Dragon", 7, 5)
	})

	t.Run("Frozen Shade pump", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Frozen Shade")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Frozen Shade")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Frozen Shade is 0/1, +1/+1 = 1/2
		g.AssertPowerToughness(mage.PlayerA, "Frozen Shade", 1, 2)
	})
}

// ===== LORD EFFECTS =====

func TestAlphaLords(t *testing.T) {
	t.Run("Goblin King boosts other Goblins", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Goblin King")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Mons's Goblin Raiders")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Hill Giant") // not a Goblin
		g.StopAt(1, mage.PrecombatMain)
		g.Execute()
		// Goblin King itself should NOT get the bonus (other Goblin creatures)
		g.AssertPowerToughness(mage.PlayerA, "Goblin King", 2, 2)
		// Mons's Goblin Raiders (1/1 Goblin) gets +1/+1 = 2/2
		g.AssertPowerToughness(mage.PlayerA, "Mons's Goblin Raiders", 2, 2)
		// Hill Giant is not a Goblin, stays 3/3
		g.AssertPowerToughness(mage.PlayerA, "Hill Giant", 3, 3)
		// Mons's Goblin Raiders should have mountainwalk
		g.AssertHasAbility(mage.PlayerA, "Mons's Goblin Raiders", mage.Mountainwalk, true)
	})

	t.Run("Lord of Atlantis boosts Merfolk", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Lord of Atlantis")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Merfolk of the Pearl Trident")
		g.StopAt(1, mage.PrecombatMain)
		g.Execute()
		// Lord of Atlantis itself should NOT get the bonus
		g.AssertPowerToughness(mage.PlayerA, "Lord of Atlantis", 2, 2)
		// Merfolk of the Pearl Trident (1/1) gets +1/+1 = 2/2
		g.AssertPowerToughness(mage.PlayerA, "Merfolk of the Pearl Trident", 2, 2)
		g.AssertHasAbility(mage.PlayerA, "Merfolk of the Pearl Trident", mage.Islandwalk, true)
	})

	t.Run("Crusade boosts white creatures", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Crusade")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Savannah Lions")   // white
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears")    // green
		g.StopAt(1, mage.PrecombatMain)
		g.Execute()
		// Savannah Lions (2/1 white) gets +1/+1 = 3/2
		g.AssertPowerToughness(mage.PlayerA, "Savannah Lions", 3, 2)
		// Grizzly Bears is green, no bonus
		g.AssertPowerToughness(mage.PlayerA, "Grizzly Bears", 2, 2)
	})

	t.Run("Bad Moon boosts black creatures", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Bad Moon")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Scathe Zombies") // black
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears")  // green
		g.StopAt(1, mage.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(mage.PlayerA, "Scathe Zombies", 3, 3)
		g.AssertPowerToughness(mage.PlayerA, "Grizzly Bears", 2, 2)
	})
}

// ===== X SPELLS =====

func TestAlphaXSpells(t *testing.T) {
	t.Run("Fireball deals X damage", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Fireball")
		g.CastSpellWithX(1, mage.PrecombatMain, mage.PlayerA, "Fireball", 5, "PlayerB")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertLife(mage.PlayerB, 15)
	})

	t.Run("Drain Life deals X and gains X", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.SetLife(mage.PlayerA, 15)
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Drain Life")
		g.CastSpellWithX(1, mage.PrecombatMain, mage.PlayerA, "Drain Life", 3, "PlayerB")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertLife(mage.PlayerB, 17) // 20 - 3
		g.AssertLife(mage.PlayerA, 18) // 15 + 3
	})

	t.Run("Earthquake damages creatures without flying and players", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears")  // 2/2, no flying
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Air Elemental")  // 4/4, flying
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Earthquake")
		g.CastSpellWithX(1, mage.PrecombatMain, mage.PlayerA, "Earthquake", 3)
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Grizzly Bears: 2/2, takes 3 damage -> dies
		g.AssertPermanentCount(mage.PlayerA, "Grizzly Bears", 0)
		// Air Elemental: 4/4 with flying, immune to Earthquake
		g.AssertPermanentCount(mage.PlayerA, "Air Elemental", 1)
		// Both players take 3 damage
		g.AssertLife(mage.PlayerA, 17)
		g.AssertLife(mage.PlayerB, 17)
	})

	t.Run("Mind Twist discards X cards", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Mind Twist")
		g.AddCard(mage.ZoneHand, mage.PlayerB, "Grizzly Bears")
		g.AddCard(mage.ZoneHand, mage.PlayerB, "Hill Giant")
		g.AddCard(mage.ZoneHand, mage.PlayerB, "Lightning Bolt")
		g.CastSpellWithX(1, mage.PrecombatMain, mage.PlayerA, "Mind Twist", 2, "PlayerB")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// PlayerB had 3 cards, discarded 2
		// Count remaining hand
		player := g.Players[1]
		if len(player.Hand()) != 1 {
			t.Errorf("expected 1 card in hand, got %d", len(player.Hand()))
		}
	})

	t.Run("Braingeyser draws X cards", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Braingeyser")
		g.AddCard(mage.ZoneLibrary, mage.PlayerA, "Grizzly Bears")
		g.AddCard(mage.ZoneLibrary, mage.PlayerA, "Hill Giant")
		g.AddCard(mage.ZoneLibrary, mage.PlayerA, "Lightning Bolt")
		g.CastSpellWithX(1, mage.PrecombatMain, mage.PlayerA, "Braingeyser", 2, "PlayerA")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// PlayerA cast Braingeyser (removed from hand), drew 2 cards
		player := g.Players[0]
		if len(player.Hand()) != 2 {
			t.Errorf("expected 2 cards in hand, got %d", len(player.Hand()))
		}
	})
}

// ===== BOUNCE =====

func TestAlphaBounce(t *testing.T) {
	t.Run("Unsummon returns creature to hand", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Grizzly Bears")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Unsummon")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Unsummon", "Grizzly Bears")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(mage.PlayerB, "Grizzly Bears", 0)
		g.AssertHandCount(mage.PlayerB, "Grizzly Bears", 1)
	})
}

// ===== RAISE DEAD =====

func TestAlphaRaiseDead(t *testing.T) {
	t.Run("Raise Dead returns creature from graveyard to hand", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneGraveyard, mage.PlayerA, "Grizzly Bears")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Raise Dead")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Raise Dead", "Grizzly Bears")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(mage.PlayerA, "Grizzly Bears", 0)
		g.AssertHandCount(mage.PlayerA, "Grizzly Bears", 1)
	})
}

// ===== DISCARD =====

func TestAlphaDiscard(t *testing.T) {
	t.Run("Hypnotic Specter forces discard on damage", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Hypnotic Specter")
		g.AddCard(mage.ZoneHand, mage.PlayerB, "Grizzly Bears")
		g.AddCard(mage.ZoneHand, mage.PlayerB, "Hill Giant")
		g.Attack(1, mage.PlayerA, "Hypnotic Specter")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		// Hypnotic Specter deals 2 damage and forces discard
		g.AssertLife(mage.PlayerB, 18)
		player := g.Players[1]
		if len(player.Hand()) != 1 {
			t.Errorf("expected 1 card in hand after discard, got %d", len(player.Hand()))
		}
	})
}

// ===== SWORDS TO PLOWSHARES =====

func TestAlphaSwordsToPlowshares(t *testing.T) {
	t.Run("Exile creature and controller gains life equal to power", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Hill Giant") // 3/3
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Swords to Plowshares")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Swords to Plowshares", "Hill Giant")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Hill Giant is exiled
		g.AssertPermanentCount(mage.PlayerB, "Hill Giant", 0)
		g.AssertGraveyardCount(mage.PlayerB, "Hill Giant", 0) // not in graveyard, exiled
		// PlayerB (controller of Hill Giant) gains 3 life
		g.AssertLife(mage.PlayerB, 23)
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
				g := mage.NewTestGame(t)
				g.AddCard(mage.ZoneBattlefield, mage.PlayerA, d.name)
				g.StopAt(1, mage.PrecombatMain)
				g.Execute()
				perm := g.FindPermanentByName(d.name, g.Players[0].PlayerID())
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
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Mox Ruby")
		g.StopAt(1, mage.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(mage.PlayerA, "Mox Ruby", 1)
	})

	t.Run("Sol Ring produces 2 colorless", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Sol Ring")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Sol Ring")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertTapped(mage.PlayerA, "Sol Ring", true)
	})
}

// ===== UPKEEP TRIGGERS =====

func TestAlphaUpkeepTriggers(t *testing.T) {
	t.Run("Lord of the Pit deals damage on upkeep", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Lord of the Pit")
		g.StopAt(1, mage.PrecombatMain)
		g.Execute()
		// Lord of the Pit deals 7 to controller on upkeep
		g.AssertLife(mage.PlayerA, 13) // 20 - 7
	})

	t.Run("Mana Vault deals damage on upkeep", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Mana Vault")
		g.StopAt(1, mage.PrecombatMain)
		g.Execute()
		g.AssertLife(mage.PlayerA, 19) // 20 - 1
	})
}

// ===== PESTILENCE =====

func TestAlphaPestilence(t *testing.T) {
	t.Run("Pestilence deals 1 to all creatures and players", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Pestilence")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Hill Giant")     // 3/3
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Grizzly Bears")  // 2/2
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Pestilence")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Both players lose 1 life
		g.AssertLife(mage.PlayerA, 19)
		g.AssertLife(mage.PlayerB, 19)
		// Creatures take 1 damage but survive
		g.AssertPermanentCount(mage.PlayerA, "Hill Giant", 1)
		g.AssertPermanentCount(mage.PlayerB, "Grizzly Bears", 1)
	})
}

// ===== DEMONIC TUTOR =====

func TestAlphaDemonicTutor(t *testing.T) {
	t.Run("Search library and put card in hand", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Demonic Tutor")
		g.AddCard(mage.ZoneLibrary, mage.PlayerA, "Lightning Bolt")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Demonic Tutor")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertHandCount(mage.PlayerA, "Lightning Bolt", 1)
	})
}

// ===== TIME WALK =====

func TestAlphaTimeWalk(t *testing.T) {
	t.Run("Time Walk grants extra turn", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Time Walk")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Time Walk")
		// After Time Walk, PlayerA should get an extra turn
		// StopAt turn 2 which should still be PlayerA's turn
		g.StopAt(2, mage.PrecombatMain)
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
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Craw Wurm") // 6/4
		g.AddCard(mage.ZoneHand, mage.PlayerB, "Fog")
		g.Attack(1, mage.PlayerA, "Craw Wurm")
		g.CastSpell(1, mage.DeclareBlockers, mage.PlayerB, "Fog")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		// Fog prevents all combat damage
		g.AssertLife(mage.PlayerB, 20)
	})
}

// ===== DESTROY EFFECTS =====

func TestAlphaDestroyEffects(t *testing.T) {
	t.Run("Armageddon destroys all lands", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Plains")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Forest")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Island")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Swamp")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Armageddon")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Armageddon")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(mage.PlayerA, "Plains", 0)
		g.AssertPermanentCount(mage.PlayerA, "Forest", 0)
		g.AssertPermanentCount(mage.PlayerB, "Island", 0)
		g.AssertPermanentCount(mage.PlayerB, "Swamp", 0)
	})

	t.Run("Sinkhole destroys target land", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Forest")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Island")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Sinkhole")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Sinkhole", "Forest")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(mage.PlayerB, "Forest", 0)
		g.AssertPermanentCount(mage.PlayerB, "Island", 1) // untouched
	})

	t.Run("Shatter destroys target artifact", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Sol Ring")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Shatter")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Shatter", "Sol Ring")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(mage.PlayerB, "Sol Ring", 0)
	})

	t.Run("Disenchant destroys enchantment", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Bad Moon")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Disenchant")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Disenchant", "Bad Moon")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(mage.PlayerB, "Bad Moon", 0)
	})

	t.Run("Tranquility destroys all enchantments", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Crusade")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Bad Moon")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Hill Giant") // creature, not enchantment
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Tranquility")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Tranquility")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(mage.PlayerA, "Crusade", 0)
		g.AssertPermanentCount(mage.PlayerB, "Bad Moon", 0)
		g.AssertPermanentCount(mage.PlayerA, "Hill Giant", 1)
	})

	t.Run("Terror destroys nonblack nonartifact creature", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Hill Giant") // red, not black
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Terror")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Terror", "Hill Giant")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(mage.PlayerB, "Hill Giant", 0)
	})
}

// ===== AURAS =====

func TestAlphaAuras(t *testing.T) {
	t.Run("Unholy Strength gives +2/+1", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Unholy Strength")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Unholy Strength", "Grizzly Bears")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(mage.PlayerA, "Grizzly Bears", 4, 3)
	})

	t.Run("Weakness gives -2/-1", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Hill Giant") // 3/3
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Weakness")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Weakness", "Hill Giant")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(mage.PlayerB, "Hill Giant", 1, 2)
	})

	t.Run("Flight grants flying", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Flight")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Flight", "Grizzly Bears")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertHasAbility(mage.PlayerA, "Grizzly Bears", mage.Flying, true)
	})

	t.Run("Lance grants first strike", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Lance")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Lance", "Grizzly Bears")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertHasAbility(mage.PlayerA, "Grizzly Bears", mage.FirstStrike, true)
	})
}

// ===== DARK RITUAL =====

func TestAlphaDarkRitual(t *testing.T) {
	t.Run("Dark Ritual adds 3 black mana", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Dark Ritual")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Dark Ritual")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		pool := g.Players[0].ManaPool()
		if pool.Count(mage.Black) < 3 {
			t.Errorf("expected at least 3 black mana, got %d", pool.Count(mage.Black))
		}
	})
}

// ===== ANCESTRAL RECALL =====

func TestAlphaAncestralRecall(t *testing.T) {
	t.Run("Ancestral Recall draws 3 cards", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Ancestral Recall")
		g.AddCard(mage.ZoneLibrary, mage.PlayerA, "Grizzly Bears")
		g.AddCard(mage.ZoneLibrary, mage.PlayerA, "Hill Giant")
		g.AddCard(mage.ZoneLibrary, mage.PlayerA, "Lightning Bolt")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Ancestral Recall", "PlayerA")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		player := g.Players[0]
		if len(player.Hand()) != 3 {
			t.Errorf("expected 3 cards in hand, got %d", len(player.Hand()))
		}
	})
}

// ===== COMBAT KEYWORDS =====

func TestAlphaCombatKeywords(t *testing.T) {
	t.Run("Trample deals excess damage to player", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Craw Wurm") // 6/4 Trample is on War Mammoth
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "War Mammoth") // 3/3 Trample
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Grizzly Bears") // 2/2
		g.Attack(1, mage.PlayerA, "War Mammoth")
		g.Block(1, mage.PlayerB, "Grizzly Bears", "War Mammoth")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		// War Mammoth (3/3 trample) blocked by Grizzly Bears (2/2)
		// War Mammoth assigns 2 to Grizzly Bears (lethal), 1 tramples to player
		g.AssertLife(mage.PlayerB, 19)
		g.AssertPermanentCount(mage.PlayerB, "Grizzly Bears", 0)
	})

	t.Run("First Strike White Knight vs Black Knight", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "White Knight")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Black Knight")
		g.Attack(1, mage.PlayerA, "White Knight")
		g.Block(1, mage.PlayerB, "Black Knight", "White Knight")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		// White Knight has protection from black so Black Knight can't block
		// White Knight deals 2 damage unblocked
		g.AssertLife(mage.PlayerB, 18)
	})
}

// ===== REGROWTH =====

func TestAlphaRegrowth(t *testing.T) {
	t.Run("Regrowth returns any card from graveyard to hand", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneGraveyard, mage.PlayerA, "Lightning Bolt")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Regrowth")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Regrowth", "Lightning Bolt")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(mage.PlayerA, "Lightning Bolt", 0)
		g.AssertHandCount(mage.PlayerA, "Lightning Bolt", 1)
	})
}

// ===== GIANT GROWTH =====

func TestAlphaGiantGrowth(t *testing.T) {
	t.Run("Giant Growth gives +3/+3 until end of turn", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Giant Growth")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Giant Growth", "Grizzly Bears")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(mage.PlayerA, "Grizzly Bears", 5, 5) // 2+3/2+3
	})
}

// ===== HEALING SALVE =====

func TestAlphaHealingSalve(t *testing.T) {
	t.Run("Healing Salve gains 3 life", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.SetLife(mage.PlayerA, 15)
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Healing Salve")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Healing Salve", "PlayerA")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertLife(mage.PlayerA, 18)
	})
}

// ===== RESURRECTION =====

func TestAlphaResurrection(t *testing.T) {
	t.Run("Resurrection returns creature from graveyard to battlefield", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneGraveyard, mage.PlayerA, "Serra Angel")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Resurrection")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Resurrection", "Serra Angel")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(mage.PlayerA, "Serra Angel", 0)
		g.AssertPermanentCount(mage.PlayerA, "Serra Angel", 1)
	})
}

// ===== ICY MANIPULATOR =====

func TestAlphaIcyManipulator(t *testing.T) {
	t.Run("Icy Manipulator taps target permanent", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Icy Manipulator")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Hill Giant")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Icy Manipulator", "Hill Giant")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertTapped(mage.PlayerB, "Hill Giant", true)
		g.AssertTapped(mage.PlayerA, "Icy Manipulator", true)
	})
}

// ===== PRODIGAL SORCERER =====

func TestAlphaProdigalSorcerer(t *testing.T) {
	t.Run("Tim deals 1 damage", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Prodigal Sorcerer")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Prodigal Sorcerer", "PlayerB")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertLife(mage.PlayerB, 19)
	})
}

// ===== ROYAL ASSASSIN =====

func TestAlphaRoyalAssassin(t *testing.T) {
	t.Run("Royal Assassin destroys tapped creature", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Royal Assassin")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Hill Giant")
		// First tap the Hill Giant (using Icy Manipulator would be ideal, but just set tapped)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Icy Manipulator")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Icy Manipulator", "Hill Giant")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Royal Assassin", "Hill Giant")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(mage.PlayerB, "Hill Giant", 0)
	})
}

// ===== FLASHFIRES / TSUNAMI =====

func TestAlphaFlashfires(t *testing.T) {
	t.Run("Flashfires destroys all Plains", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Plains", 2)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Mountain")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Plains")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Flashfires")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Flashfires")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(mage.PlayerA, "Plains", 0)
		g.AssertPermanentCount(mage.PlayerB, "Plains", 0)
		g.AssertPermanentCount(mage.PlayerA, "Mountain", 1) // not a Plains
	})
}

// ===== FUNGUSAUR =====

func TestAlphaFungusaur(t *testing.T) {
	t.Run("Fungusaur gets counter when damaged", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Fungusaur")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Lightning Bolt")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Lightning Bolt", "Fungusaur")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Fungusaur is 2/2, takes 3 damage -> dies (toughness 2)
		// But if it survived, it would get a counter. Since 3 >= 2, it dies.
		g.AssertPermanentCount(mage.PlayerA, "Fungusaur", 0)
	})

	t.Run("Fungusaur survives 1 damage gets counter", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Fungusaur") // 2/2
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Prodigal Sorcerer")
		g.ActivateAbility(2, mage.PrecombatMain, mage.PlayerB, "Prodigal Sorcerer", "Fungusaur")
		g.StopAt(2, mage.BeginCombat)
		g.Execute()
		// Fungusaur takes 1 damage, gets a +1/+1 counter = becomes 3/3
		g.AssertPermanentCount(mage.PlayerA, "Fungusaur", 1)
		g.AssertCounterCount(mage.PlayerA, "Fungusaur", mage.P1P1, 1)
	})
}

// ===== WANDERLUST (Aura upkeep damage to attached controller) =====

func TestWanderlustUpkeepDamage(t *testing.T) {
	t.Run("Wanderlust deals 1 damage to enchanted creature's controller on upkeep", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Grizzly Bears")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Wanderlust")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Wanderlust", "Grizzly Bears")
		// Turn 2 is PlayerB's turn; upkeep fires trigger on enchanted creature's controller
		g.StopAt(2, mage.PrecombatMain)
		g.Execute()
		// PlayerB controls Grizzly Bears which has Wanderlust attached
		// At PlayerB's upkeep, Wanderlust deals 1 damage to PlayerB
		g.AssertLife(mage.PlayerB, 19)
		g.AssertLife(mage.PlayerA, 20) // PlayerA unaffected
	})
}

// ===== CURSED LAND (Aura upkeep damage to attached controller) =====

func TestCursedLandUpkeepDamage(t *testing.T) {
	t.Run("Cursed Land deals 1 damage to enchanted land's controller on upkeep", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Forest")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Cursed Land")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Cursed Land", "Forest")
		// Turn 2 is PlayerB's turn
		g.StopAt(2, mage.PrecombatMain)
		g.Execute()
		g.AssertLife(mage.PlayerB, 19)
		g.AssertLife(mage.PlayerA, 20)
	})
}

// ===== FEEDBACK (Aura upkeep damage to attached controller) =====

func TestFeedbackUpkeepDamage(t *testing.T) {
	t.Run("Feedback deals 1 damage to enchanted enchantment's controller on upkeep", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Bad Moon")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Feedback")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Feedback", "Bad Moon")
		// Turn 2 is PlayerB's turn
		g.StopAt(2, mage.PrecombatMain)
		g.Execute()
		g.AssertLife(mage.PlayerB, 19)
		g.AssertLife(mage.PlayerA, 20)
	})
}

// ===== WARP ARTIFACT (Aura upkeep damage to attached controller) =====

func TestWarpArtifactUpkeepDamage(t *testing.T) {
	t.Run("Warp Artifact deals 1 damage to enchanted artifact's controller on upkeep", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Sol Ring")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Warp Artifact")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Warp Artifact", "Sol Ring")
		// Turn 2 is PlayerB's turn
		g.StopAt(2, mage.PrecombatMain)
		g.Execute()
		g.AssertLife(mage.PlayerB, 19)
		g.AssertLife(mage.PlayerA, 20)
	})
}

// ===== JUMP (Grant flying until end of turn) =====

func TestJumpGrantsFlying(t *testing.T) {
	t.Run("Jump grants flying until end of turn", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Jump")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Jump", "Grizzly Bears")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Grizzly Bears should now have flying
		g.AssertHasAbility(mage.PlayerA, "Grizzly Bears", mage.Flying, true)
	})

	t.Run("Jump flying wears off at end of turn", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Jump")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Jump", "Grizzly Bears")
		// Stop at turn 2 (after cleanup clears EOT effects)
		g.StopAt(2, mage.Upkeep)
		g.Execute()
		// Flying should be gone now
		g.AssertHasAbility(mage.PlayerA, "Grizzly Bears", mage.Flying, false)
	})
}

// ===== ANKH OF MISHRA (Damage when land enters) =====

func TestAnkhOfMishraDamageOnLandEntry(t *testing.T) {
	t.Run("Ankh of Mishra deals 2 damage when any land enters battlefield", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Ankh of Mishra")
		// Directly put a land onto the battlefield for PlayerB
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Forest")
		// Ankh should have triggered dealing 2 damage to Forest's controller (PlayerB)
		g.StopAt(1, mage.PrecombatMain)
		g.Execute()
		g.AssertLife(mage.PlayerB, 18) // 20 - 2
		g.AssertLife(mage.PlayerA, 20)
	})

	t.Run("Ankh of Mishra triggers for each land", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Ankh of Mishra")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Forest")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Island")
		g.StopAt(1, mage.PrecombatMain)
		g.Execute()
		g.AssertLife(mage.PlayerB, 16) // 20 - 2 - 2
	})
}

// ===== ORCISH ORIFLAMME (Attacking creatures you control get +1/+0) =====

func TestOrcishOriflammeBoostsAttackers(t *testing.T) {
	t.Run("Orcish Oriflamme boosts attacking creatures", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Orcish Oriflamme")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears") // 2/2
		g.Attack(1, mage.PlayerA, "Grizzly Bears")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		// Grizzly Bears attacks with +1/+0 from Oriflamme = 3 damage
		g.AssertLife(mage.PlayerB, 17)
	})

	t.Run("Orcish Oriflamme does not boost non-attacking creatures", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Orcish Oriflamme")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears") // 2/2
		g.StopAt(1, mage.PrecombatMain)
		g.Execute()
		// Grizzly Bears not attacking, should be base 2/2
		g.AssertPowerToughness(mage.PlayerA, "Grizzly Bears", 2, 2)
	})
}

// ===== CASTLE (Untapped creatures you control get +0/+2) =====

func TestCastleBoostsUntapped(t *testing.T) {
	t.Run("Castle boosts untapped creatures", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Castle")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears") // 2/2
		g.StopAt(1, mage.PrecombatMain)
		g.Execute()
		// Grizzly Bears is untapped, gets +0/+2 = 2/4
		g.AssertPowerToughness(mage.PlayerA, "Grizzly Bears", 2, 4)
	})

	t.Run("Castle does not boost tapped creatures", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Castle")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Icy Manipulator")
		// Tap the Grizzly Bears
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Icy Manipulator", "Grizzly Bears")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Grizzly Bears is tapped, no +0/+2 from Castle
		g.AssertPowerToughness(mage.PlayerA, "Grizzly Bears", 2, 2)
	})
}
