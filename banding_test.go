package mage

import "testing"

func init() {
	registerBandingTestCards()
}

// registerBandingTestCards registers the cards used in banding tests.
// The root package cannot import cards/, so we register them here.
func registerBandingTestCards() {
	if !CardRegistered("Benalish Hero") {
		Register("Benalish Hero", func() Card {
			c := NewCreature("Benalish Hero", "{W}", 1, 1, "Human", "Soldier")
			c.AddAbility(NewKeywordAbility(Banding))
			return c
		})
	}
	if !CardRegistered("Mesa Pegasus") {
		Register("Mesa Pegasus", func() Card {
			c := NewCreature("Mesa Pegasus", "{1}{W}", 1, 1, "Pegasus")
			c.AddAbility(NewKeywordAbility(Flying))
			c.AddAbility(NewKeywordAbility(Banding))
			return c
		})
	}
	if !CardRegistered("Timber Wolves") {
		Register("Timber Wolves", func() Card {
			c := NewCreature("Timber Wolves", "{G}", 1, 1, "Wolf")
			c.AddAbility(NewKeywordAbility(Banding))
			return c
		})
	}
	if !CardRegistered("Grizzly Bears") {
		Register("Grizzly Bears", func() Card {
			c := NewCreature("Grizzly Bears", "{1}{G}", 2, 2, "Bear")
			return c
		})
	}
	if !CardRegistered("Hill Giant") {
		Register("Hill Giant", func() Card {
			c := NewCreature("Hill Giant", "{3}{R}", 3, 3, "Giant")
			return c
		})
	}
	if !CardRegistered("Gray Ogre") {
		Register("Gray Ogre", func() Card {
			c := NewCreature("Gray Ogre", "{2}{R}", 2, 2, "Ogre")
			return c
		})
	}
	if !CardRegistered("Craw Wurm") {
		Register("Craw Wurm", func() Card {
			c := NewCreature("Craw Wurm", "{4}{G}{G}", 6, 4, "Wurm")
			return c
		})
	}
	if !CardRegistered("War Mammoth") {
		Register("War Mammoth", func() Card {
			c := NewCreature("War Mammoth", "{3}{G}", 3, 3, "Elephant")
			c.AddAbility(NewKeywordAbility(Trample))
			return c
		})
	}
	if !CardRegistered("Black Knight") {
		Register("Black Knight", func() Card {
			c := NewCreature("Black Knight", "{B}{B}", 2, 2, "Human", "Knight")
			c.AddAbility(NewKeywordAbility(FirstStrike))
			c.AddAbility(ProtectionFromColor(White))
			return c
		})
	}
	if !CardRegistered("Helm of Chatzuk") {
		Register("Helm of Chatzuk", func() Card {
			c := NewArtifact("Helm of Chatzuk", "{1}")
			// {1}, {T}: Target creature gains banding until end of turn.
			ab := NewActivatedAbility(
				GrantKeywordUntilEndOfTurn(Banding, SelectTarget),
				GenericCost(1),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreature()),
			)
			c.AddAbility(ab)
			return c
		})
	}
	if !CardRegistered("Camel") {
		Register("Camel", func() Card {
			c := NewCreature("Camel", "{W}", 0, 1, "Camel")
			c.AddAbility(NewKeywordAbility(Banding))
			return c
		})
	}
}

// TestBandFormation verifies that banding creatures correctly form bands when attacking.
func TestBandFormation(t *testing.T) {
	t.Run("two_banding_creatures_form_band", func(t *testing.T) {
		// Two creatures with banding attack together and form a band.
		g := NewTestGame(t)
		g.AddCard(ZoneBattlefield, PlayerA, "Benalish Hero")
		g.AddCard(ZoneBattlefield, PlayerA, "Timber Wolves")
		g.FormBand(1, PlayerA, "Benalish Hero", "Timber Wolves")
		g.Attack(1, PlayerA, "Benalish Hero", "Timber Wolves")
		g.StopAt(1, DeclareBlockers)
		g.Execute()
		g.AssertBanded(PlayerA, "Benalish Hero", PlayerA, "Timber Wolves", true)
	})

	t.Run("three_banding_creatures_form_band", func(t *testing.T) {
		// Three banding creatures can all band together.
		g := NewTestGame(t)
		g.AddCard(ZoneBattlefield, PlayerA, "Benalish Hero")
		g.AddCard(ZoneBattlefield, PlayerA, "Mesa Pegasus")
		g.AddCard(ZoneBattlefield, PlayerA, "Timber Wolves")
		g.FormBand(1, PlayerA, "Benalish Hero", "Mesa Pegasus", "Timber Wolves")
		g.Attack(1, PlayerA, "Benalish Hero", "Mesa Pegasus", "Timber Wolves")
		g.StopAt(1, DeclareBlockers)
		g.Execute()
		g.AssertBanded(PlayerA, "Benalish Hero", PlayerA, "Mesa Pegasus", true)
		g.AssertBanded(PlayerA, "Benalish Hero", PlayerA, "Timber Wolves", true)
		g.AssertBanded(PlayerA, "Mesa Pegasus", PlayerA, "Timber Wolves", true)
	})

	t.Run("banding_plus_one_non_banding", func(t *testing.T) {
		// A banding creature can bring one non-banding creature into its band.
		g := NewTestGame(t)
		g.AddCard(ZoneBattlefield, PlayerA, "Benalish Hero")
		g.AddCard(ZoneBattlefield, PlayerA, "Grizzly Bears")
		g.FormBand(1, PlayerA, "Benalish Hero", "Grizzly Bears")
		g.Attack(1, PlayerA, "Benalish Hero", "Grizzly Bears")
		g.StopAt(1, DeclareBlockers)
		g.Execute()
		g.AssertBanded(PlayerA, "Benalish Hero", PlayerA, "Grizzly Bears", true)
	})

	t.Run("non_banding_creatures_dont_band", func(t *testing.T) {
		// Two creatures without banding cannot form a band.
		g := NewTestGame(t)
		g.AddCard(ZoneBattlefield, PlayerA, "Grizzly Bears")
		g.AddCard(ZoneBattlefield, PlayerA, "Hill Giant")
		g.Attack(1, PlayerA, "Grizzly Bears", "Hill Giant")
		g.StopAt(1, DeclareBlockers)
		g.Execute()
		g.AssertBanded(PlayerA, "Grizzly Bears", PlayerA, "Hill Giant", false)
	})

	t.Run("single_banding_creature_no_band", func(t *testing.T) {
		// A banding creature attacking alone is not in a band.
		g := NewTestGame(t)
		g.AddCard(ZoneBattlefield, PlayerA, "Benalish Hero")
		g.Attack(1, PlayerA, "Benalish Hero")
		g.StopAt(1, DeclareBlockers)
		g.Execute()
		// A lone attacker is never banded with itself.
		// No AssertBanded needed; confirm it simply attacks without error.
		g.AssertPermanentCount(PlayerA, "Benalish Hero", 1)
	})

	t.Run("at_most_one_without_banding", func(t *testing.T) {
		// A band may contain at most one creature without banding.
		// Benalish Hero bands with Grizzly Bears. Hill Giant attacks solo.
		g := NewTestGame(t)
		g.AddCard(ZoneBattlefield, PlayerA, "Benalish Hero")
		g.AddCard(ZoneBattlefield, PlayerA, "Grizzly Bears")
		g.AddCard(ZoneBattlefield, PlayerA, "Hill Giant")
		g.FormBand(1, PlayerA, "Benalish Hero", "Grizzly Bears")
		g.Attack(1, PlayerA, "Benalish Hero", "Grizzly Bears", "Hill Giant")
		g.StopAt(1, DeclareBlockers)
		g.Execute()
		// Hero + Bears form a band; Hill Giant is solo.
		g.AssertBanded(PlayerA, "Benalish Hero", PlayerA, "Grizzly Bears", true)
		g.AssertBanded(PlayerA, "Benalish Hero", PlayerA, "Hill Giant", false)
		g.AssertBanded(PlayerA, "Grizzly Bears", PlayerA, "Hill Giant", false)
	})
}

// TestBandingBlocking verifies that blocking one band member blocks the entire band.
func TestBandingBlocking(t *testing.T) {
	t.Run("blocking_one_member_blocks_entire_band", func(t *testing.T) {
		// When one member of an attacking band is blocked, the whole band is blocked.
		// Gray Ogre blocks Timber Wolves; Benalish Hero should also be blocked.
		g := NewTestGame(t)
		g.AddCard(ZoneBattlefield, PlayerA, "Benalish Hero")
		g.AddCard(ZoneBattlefield, PlayerA, "Timber Wolves")
		g.AddCard(ZoneBattlefield, PlayerB, "Gray Ogre")
		g.FormBand(1, PlayerA, "Benalish Hero", "Timber Wolves")
		g.Attack(1, PlayerA, "Benalish Hero", "Timber Wolves")
		g.Block(1, PlayerB, "Gray Ogre", "Timber Wolves") // blocks only Wolves by name
		// Attacking player distributes Gray Ogre's 2 damage across the band.
		g.ChooseBandingDistribution(PlayerA, map[string]int{"Benalish Hero": 1, "Timber Wolves": 1})
		g.StopAt(1, PostcombatMain)
		g.Execute()
		// The entire band was blocked, so PlayerB takes 0 combat damage.
		g.AssertLife(PlayerB, 20)
	})

	t.Run("blocking_band_with_flyer", func(t *testing.T) {
		// A non-flying creature may block a flying+banding creature if it blocks
		// another member of the same band. Blocking Hero blocks the entire band
		// including Mesa Pegasus.
		g := NewTestGame(t)
		g.AddCard(ZoneBattlefield, PlayerA, "Mesa Pegasus") // 1/1 flying+banding
		g.AddCard(ZoneBattlefield, PlayerA, "Benalish Hero")
		g.AddCard(ZoneBattlefield, PlayerB, "Grizzly Bears")
		g.FormBand(1, PlayerA, "Mesa Pegasus", "Benalish Hero")
		g.Attack(1, PlayerA, "Mesa Pegasus", "Benalish Hero")
		// Grizzly Bears can't normally block Mesa Pegasus, but blocks Hero instead.
		g.Block(1, PlayerB, "Grizzly Bears", "Benalish Hero")
		// Attacking player distributes Bears' 2 damage; put it all on Hero.
		g.ChooseBandingDistribution(PlayerA, map[string]int{"Mesa Pegasus": 0, "Benalish Hero": 2})
		g.StopAt(1, PostcombatMain)
		g.Execute()
		// Entire band was blocked; PlayerB takes 0 damage.
		g.AssertLife(PlayerB, 20)
	})
}

// TestBandingAttackDamage verifies that the attacking player distributes
// a blocker's damage across band members.
func TestBandingAttackDamage(t *testing.T) {
	t.Run("controller_distributes_blocker_damage_saves_creature", func(t *testing.T) {
		// Benalish Hero (1/1) + Grizzly Bears (2/2) band vs Hill Giant (3/3) blocker.
		// Attacking player puts all 3 incoming damage on Hero (dies); Bears survives.
		// Band deals 1+2=3 to Hill Giant (dies).
		g := NewTestGame(t)
		g.AddCard(ZoneBattlefield, PlayerA, "Benalish Hero")
		g.AddCard(ZoneBattlefield, PlayerA, "Grizzly Bears")
		g.AddCard(ZoneBattlefield, PlayerB, "Hill Giant")
		g.FormBand(1, PlayerA, "Benalish Hero", "Grizzly Bears")
		g.Attack(1, PlayerA, "Benalish Hero", "Grizzly Bears")
		g.Block(1, PlayerB, "Hill Giant", "Benalish Hero")
		g.ChooseBandingDistribution(PlayerA, map[string]int{"Benalish Hero": 3, "Grizzly Bears": 0})
		g.StopAt(1, PostcombatMain)
		g.Execute()
		g.AssertGraveyardCount(PlayerA, "Benalish Hero", 1)  // 3 damage, 1 toughness → dead
		g.AssertPermanentCount(PlayerA, "Grizzly Bears", 1)  // 0 damage → survives
		g.AssertGraveyardCount(PlayerB, "Hill Giant", 1)     // 3 total damage (1+2) → dead
	})

	t.Run("smaller_blocker_damage_distributed", func(t *testing.T) {
		// Benalish Hero (1/1) + Grizzly Bears (2/2) band vs Gray Ogre (2/2) blocker.
		// Put both incoming damage on Hero (dies); Bears takes 0 (survives).
		// Band deals 3 to Ogre (dies).
		g := NewTestGame(t)
		g.AddCard(ZoneBattlefield, PlayerA, "Benalish Hero")
		g.AddCard(ZoneBattlefield, PlayerA, "Grizzly Bears")
		g.AddCard(ZoneBattlefield, PlayerB, "Gray Ogre")
		g.FormBand(1, PlayerA, "Benalish Hero", "Grizzly Bears")
		g.Attack(1, PlayerA, "Benalish Hero", "Grizzly Bears")
		g.Block(1, PlayerB, "Gray Ogre", "Benalish Hero")
		g.ChooseBandingDistribution(PlayerA, map[string]int{"Benalish Hero": 2, "Grizzly Bears": 0})
		g.StopAt(1, PostcombatMain)
		g.Execute()
		g.AssertGraveyardCount(PlayerA, "Benalish Hero", 1) // 2 damage, 1 toughness → dead
		g.AssertPermanentCount(PlayerA, "Grizzly Bears", 1) // 0 damage → survives
		g.AssertGraveyardCount(PlayerB, "Gray Ogre", 1)     // 3 total damage (1+2) → dead
	})

	t.Run("multiple_blockers_on_band", func(t *testing.T) {
		// Benalish Hero (1/1) + Grizzly Bears (2/2) band.
		// Gray Ogre (2/2) + Hill Giant (3/3) both block.
		// 5 total incoming damage. Both attackers have only 3 combined toughness → both die.
		g := NewTestGame(t)
		g.AddCard(ZoneBattlefield, PlayerA, "Benalish Hero")
		g.AddCard(ZoneBattlefield, PlayerA, "Grizzly Bears")
		g.AddCard(ZoneBattlefield, PlayerB, "Gray Ogre")
		g.AddCard(ZoneBattlefield, PlayerB, "Hill Giant")
		g.FormBand(1, PlayerA, "Benalish Hero", "Grizzly Bears")
		g.Attack(1, PlayerA, "Benalish Hero", "Grizzly Bears")
		g.Block(1, PlayerB, "Gray Ogre", "Benalish Hero")
		g.Block(1, PlayerB, "Hill Giant", "Benalish Hero")
		// 5 total damage: put 1 on Hero (dies), 4 on Bears (dies). Or 3+2, etc.
		g.ChooseBandingDistribution(PlayerA, map[string]int{"Benalish Hero": 1, "Grizzly Bears": 4})
		g.StopAt(1, PostcombatMain)
		g.Execute()
		g.AssertGraveyardCount(PlayerA, "Benalish Hero", 1)
		g.AssertGraveyardCount(PlayerA, "Grizzly Bears", 1)
	})

	t.Run("without_banding_normal_damage", func(t *testing.T) {
		// Control: Grizzly Bears (2/2) alone, no banding. Blocked by Hill Giant (3/3).
		// Normal rules: Bears takes 3 (dies), Hill Giant takes 2 (survives).
		g := NewTestGame(t)
		g.AddCard(ZoneBattlefield, PlayerA, "Grizzly Bears")
		g.AddCard(ZoneBattlefield, PlayerB, "Hill Giant")
		g.Attack(1, PlayerA, "Grizzly Bears")
		g.Block(1, PlayerB, "Hill Giant", "Grizzly Bears")
		g.StopAt(1, PostcombatMain)
		g.Execute()
		g.AssertGraveyardCount(PlayerA, "Grizzly Bears", 1) // 3 damage, 2 toughness → dead
		g.AssertPermanentCount(PlayerB, "Hill Giant", 1)    // 2 damage, 3 toughness → survives
	})
}

// TestBandingBlockDamage verifies that a blocking band gives the defending player
// control over how the attacker's damage is distributed.
func TestBandingBlockDamage(t *testing.T) {
	t.Run("blocking_band_distributes_attacker_damage", func(t *testing.T) {
		// Hill Giant (3/3) attacks. Benalish Hero (1/1) + Timber Wolves (1/1) block as band.
		// Blocking player puts all 3 damage on Hero (dies); Wolves survives.
		// Blockers deal 1+1=2 to Hill Giant (survives with 1 remaining).
		g := NewTestGame(t)
		g.AddCard(ZoneBattlefield, PlayerA, "Hill Giant")
		g.AddCard(ZoneBattlefield, PlayerB, "Benalish Hero")
		g.AddCard(ZoneBattlefield, PlayerB, "Timber Wolves")
		g.Attack(1, PlayerA, "Hill Giant")
		g.Block(1, PlayerB, "Benalish Hero", "Hill Giant")
		g.Block(1, PlayerB, "Timber Wolves", "Hill Giant")
		// Blocking player distributes Hill Giant's 3 damage across the blocking band.
		g.ChooseBandingDistribution(PlayerB, map[string]int{"Benalish Hero": 3, "Timber Wolves": 0})
		g.StopAt(1, PostcombatMain)
		g.Execute()
		g.AssertGraveyardCount(PlayerB, "Benalish Hero", 1) // 3 damage, 1 toughness → dead
		g.AssertPermanentCount(PlayerB, "Timber Wolves", 1) // 0 damage → survives
		g.AssertPermanentCount(PlayerA, "Hill Giant", 1)    // 2 damage, 3 toughness → survives
	})

	t.Run("blocking_band_saves_creature", func(t *testing.T) {
		// Grizzly Bears (2/2) attacks. Hero (1/1) + Wolves (1/1) block as band.
		// Put 2 damage on Hero (dies); Wolves survives. Blockers deal 2 to Bears (dies).
		g := NewTestGame(t)
		g.AddCard(ZoneBattlefield, PlayerA, "Grizzly Bears")
		g.AddCard(ZoneBattlefield, PlayerB, "Benalish Hero")
		g.AddCard(ZoneBattlefield, PlayerB, "Timber Wolves")
		g.Attack(1, PlayerA, "Grizzly Bears")
		g.Block(1, PlayerB, "Benalish Hero", "Grizzly Bears")
		g.Block(1, PlayerB, "Timber Wolves", "Grizzly Bears")
		g.ChooseBandingDistribution(PlayerB, map[string]int{"Benalish Hero": 2, "Timber Wolves": 0})
		g.StopAt(1, PostcombatMain)
		g.Execute()
		g.AssertGraveyardCount(PlayerB, "Benalish Hero", 1) // 2 damage, 1 toughness → dead
		g.AssertPermanentCount(PlayerB, "Timber Wolves", 1) // 0 damage → survives
		g.AssertGraveyardCount(PlayerA, "Grizzly Bears", 1) // 1+1=2 total damage → dead
	})
}

// TestBandingTrample verifies trample interactions with banding.
func TestBandingTrample(t *testing.T) {
	t.Run("trample_vs_blocking_band", func(t *testing.T) {
		// War Mammoth (3/3 trample) attacks.
		// Benalish Hero (1/1) + Timber Wolves (1/1) block as band (total toughness 2).
		// Defending player distributes 2 damage to blockers (1 each, both die); 1 tramples.
		g := NewTestGame(t)
		g.AddCard(ZoneBattlefield, PlayerA, "War Mammoth")
		g.AddCard(ZoneBattlefield, PlayerB, "Benalish Hero")
		g.AddCard(ZoneBattlefield, PlayerB, "Timber Wolves")
		g.Attack(1, PlayerA, "War Mammoth")
		g.Block(1, PlayerB, "Benalish Hero", "War Mammoth")
		g.Block(1, PlayerB, "Timber Wolves", "War Mammoth")
		// Defending player distributes lethal damage; 1 tramples through.
		g.ChooseBandingDistribution(PlayerB, map[string]int{"Benalish Hero": 1, "Timber Wolves": 1})
		g.StopAt(1, PostcombatMain)
		g.Execute()
		g.AssertGraveyardCount(PlayerB, "Benalish Hero", 1) // 1 damage, 1 toughness → dead
		g.AssertGraveyardCount(PlayerB, "Timber Wolves", 1) // 1 damage, 1 toughness → dead
		g.AssertLife(PlayerB, 19)                           // 1 trample damage
	})

	t.Run("attacking_band_with_trample_unblocked", func(t *testing.T) {
		// War Mammoth gains banding via Helm of Chatzuk.
		// War Mammoth (3/3) + Benalish Hero (1/1) form a band and attack unblocked.
		// Defender takes 3+1=4 damage.
		g := NewTestGame(t)
		g.AddCard(ZoneBattlefield, PlayerA, "War Mammoth")
		g.AddCard(ZoneBattlefield, PlayerA, "Benalish Hero")
		g.AddCard(ZoneBattlefield, PlayerA, "Helm of Chatzuk")
		// Activate Helm to give War Mammoth banding before combat.
		g.ActivateAbility(1, PrecombatMain, PlayerA, "Helm of Chatzuk", "War Mammoth")
		g.FormBand(1, PlayerA, "War Mammoth", "Benalish Hero")
		g.Attack(1, PlayerA, "War Mammoth", "Benalish Hero")
		g.StopAt(1, PostcombatMain)
		g.Execute()
		g.AssertLife(PlayerB, 16) // 3+1=4 damage unblocked
	})
}

// TestBandingEdgeCases covers unusual banding scenarios.
func TestBandingEdgeCases(t *testing.T) {
	t.Run("zero_power_in_band", func(t *testing.T) {
		// Camel (0/1 banding) + Benalish Hero (1/1 banding) attack unblocked.
		// Defender takes 0+1=1 damage.
		g := NewTestGame(t)
		g.AddCard(ZoneBattlefield, PlayerA, "Camel")
		g.AddCard(ZoneBattlefield, PlayerA, "Benalish Hero")
		g.FormBand(1, PlayerA, "Camel", "Benalish Hero")
		g.Attack(1, PlayerA, "Camel", "Benalish Hero")
		g.StopAt(1, PostcombatMain)
		g.Execute()
		g.AssertLife(PlayerB, 19) // 0+1=1 damage
	})

	t.Run("first_strike_in_band", func(t *testing.T) {
		// Black Knight gains banding via Helm of Chatzuk.
		// Black Knight (2/2 first strike) + Grizzly Bears (2/2) band, blocked by Craw Wurm (6/4).
		// First strike step: Knight deals 2 to Craw Wurm.
		// Regular step: Bears deals 2 to Craw Wurm (total 4; Craw Wurm has 4 toughness → survives).
		// Craw Wurm deals 6 to banded attackers: attacking player distributes fatally to both.
		g := NewTestGame(t)
		g.AddCard(ZoneBattlefield, PlayerA, "Black Knight")
		g.AddCard(ZoneBattlefield, PlayerA, "Grizzly Bears")
		g.AddCard(ZoneBattlefield, PlayerA, "Helm of Chatzuk")
		g.AddCard(ZoneBattlefield, PlayerB, "Craw Wurm")
		// Give Black Knight banding before combat.
		g.ActivateAbility(1, PrecombatMain, PlayerA, "Helm of Chatzuk", "Black Knight")
		g.FormBand(1, PlayerA, "Black Knight", "Grizzly Bears")
		g.Attack(1, PlayerA, "Black Knight", "Grizzly Bears")
		g.Block(1, PlayerB, "Craw Wurm", "Black Knight")
		// Craw Wurm deals 6 total. Attacking player distributes: both die (combined toughness 4).
		// (In first-strike step, same distribution logic applies.)
		g.ChooseBandingDistribution(PlayerA, map[string]int{"Black Knight": 4, "Grizzly Bears": 2})
		g.StopAt(1, PostcombatMain)
		g.Execute()
		g.AssertGraveyardCount(PlayerA, "Black Knight", 1) // 4 damage, 2 toughness → dead
		g.AssertGraveyardCount(PlayerA, "Grizzly Bears", 1) // 2 damage, 2 toughness → dead
		// Craw Wurm: Knight (FS) deals 2, Bears deals 2 = 4 total on 4 toughness → lethal, dies.
		g.AssertGraveyardCount(PlayerB, "Craw Wurm", 1)
	})
}
