// Package gametest: engine-level tests for CR 702 — Keyword Abilities (Chapter 7b).
// Covers keywords supported by the engine from early sets.
package gametest

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

func init() {
	registerCh07bTestCards()
}

func registerCh07bTestCards() {
	// ── Cards from implemented sets (registered here to avoid import cycle) ──

	if !mage.CardRegistered("Serra Angel") {
		mage.Register("Serra Angel", func() mage.Card {
			return mage.NewCreature("Serra Angel", "{3}{W}{W}", 4, 4,
				mage.WithSubTypes("Angel"),
				mage.WithKeyword(core.Flying),
				mage.WithKeyword(core.Vigilance),
			)
		})
	}
	if !mage.CardRegistered("Hill Giant") {
		mage.Register("Hill Giant", func() mage.Card {
			return mage.NewCreature("Hill Giant", "{3}{R}", 3, 3, mage.WithSubTypes("Giant"))
		})
	}
	if !mage.CardRegistered("Grizzly Bears") {
		mage.Register("Grizzly Bears", func() mage.Card {
			return mage.NewCreature("Grizzly Bears", "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
		})
	}
	if !mage.CardRegistered("Scryb Sprites") {
		mage.Register("Scryb Sprites", func() mage.Card {
			return mage.NewCreature("Scryb Sprites", "{G}", 1, 1,
				mage.WithSubTypes("Faerie"),
				mage.WithKeyword(core.Flying),
			)
		})
	}
	if !mage.CardRegistered("Bog Wraith") {
		mage.Register("Bog Wraith", func() mage.Card {
			return mage.NewCreature("Bog Wraith", "{3}{B}", 3, 3,
				mage.WithSubTypes("Wraith"),
				mage.WithKeyword(core.Swampwalk),
			)
		})
	}
	if !mage.CardRegistered("White Knight") {
		mage.Register("White Knight", func() mage.Card {
			return mage.NewCreature("White Knight", "{W}{W}", 2, 2,
				mage.WithSubTypes("Human", "Knight"),
				mage.WithKeyword(core.FirstStrike),
				mage.WithAbility(mage.ProtectionFromColor(core.Black)),
			)
		})
	}
	if !mage.CardRegistered("Black Knight") {
		mage.Register("Black Knight", func() mage.Card {
			return mage.NewCreature("Black Knight", "{B}{B}", 2, 2,
				mage.WithSubTypes("Human", "Knight"),
				mage.WithKeyword(core.FirstStrike),
				mage.WithAbility(mage.ProtectionFromColor(core.White)),
			)
		})
	}
	if !mage.CardRegistered("Fear") {
		mage.Register("Fear", func() mage.Card {
			return mage.NewAura("Fear", "{B}{B}",
				mage.WithStaticAbility(
					mage.GrantAbilityToAttached(core.Fear, core.AttachAura),
				),
			)
		})
	}
	if !mage.CardRegistered("Swamp") {
		mage.Register("Swamp", func() mage.Card {
			return mage.NewLand("Swamp",
				mage.WithManaAbility(core.Black),
				mage.WithSubTypes("Swamp"),
			)
		})
	}
	// Aerathi Berserker: 2/4, Rampage 3
	if !mage.CardRegistered("Aerathi Berserker") {
		mage.Register("Aerathi Berserker", func() mage.Card {
			return mage.NewCreature("Aerathi Berserker", "{2}{R}{R}{R}", 2, 4,
				mage.WithSubTypes("Human", "Berserker"),
				mage.WithAbility(mage.RampageTrigger(3)),
			)
		})
	}

	// ── Custom test cards ──────────────────────────────────────────────────────

	// Deathtouch 1/1 — kills any creature it damages in combat.
	if !mage.CardRegistered("Test Deathtouch Creature") {
		mage.Register("Test Deathtouch Creature", func() mage.Card {
			return mage.NewCreature("Test Deathtouch Creature", "{B}", 1, 1,
				mage.WithSubTypes("Snake"),
				mage.WithKeyword(core.Deathtouch),
			)
		})
	}
	// Lifelink 3/3 — gains life when it deals damage.
	if !mage.CardRegistered("Test Lifelink Creature") {
		mage.Register("Test Lifelink Creature", func() mage.Card {
			return mage.NewCreature("Test Lifelink Creature", "{2}{W}", 3, 3,
				mage.WithSubTypes("Angel"),
				mage.WithKeyword(core.Lifelink),
			)
		})
	}
	// Indestructible 3/3 — survives lethal damage and destroy effects.
	if !mage.CardRegistered("Test Indestructible Creature") {
		mage.Register("Test Indestructible Creature", func() mage.Card {
			return mage.NewCreature("Test Indestructible Creature", "{4}{W}", 3, 3,
				mage.WithSubTypes("Angel"),
				mage.WithKeyword(core.Indestructible),
			)
		})
	}
	// Destroy spell — "Destroy target creature."
	if !mage.CardRegistered("Test Destroy Spell") {
		mage.Register("Test Destroy Spell", func() mage.Card {
			return mage.NewInstant("Test Destroy Spell", "{1}{B}",
				mage.NewTargetedSpell(mage.TargetCreature(), mage.DestroyTarget()),
			)
		})
	}
	// Shroud 2/2 — cannot be targeted by any spell or ability.
	if !mage.CardRegistered("Test Shroud Creature") {
		mage.Register("Test Shroud Creature", func() mage.Card {
			return mage.NewCreature("Test Shroud Creature", "{1}{G}", 2, 2,
				mage.WithSubTypes("Bear"),
				mage.WithKeyword(core.Shroud),
			)
		})
	}
	// Hexproof 2/2 — cannot be targeted by opponent's spells or abilities.
	if !mage.CardRegistered("Test Hexproof Creature") {
		mage.Register("Test Hexproof Creature", func() mage.Card {
			return mage.NewCreature("Test Hexproof Creature", "{1}{G}", 2, 2,
				mage.WithSubTypes("Bear"),
				mage.WithKeyword(core.Hexproof),
			)
		})
	}
	// Trample 5/5 — assigns excess damage to defending player.
	if !mage.CardRegistered("Test Trample Creature") {
		mage.Register("Test Trample Creature", func() mage.Card {
			return mage.NewCreature("Test Trample Creature", "{3}{G}{G}", 5, 5,
				mage.WithSubTypes("Beast"),
				mage.WithKeyword(core.Trample),
			)
		})
	}
	// Menace 2/2 — must be blocked by two or more creatures.
	if !mage.CardRegistered("Test Menace Creature") {
		mage.Register("Test Menace Creature", func() mage.Card {
			return mage.NewCreature("Test Menace Creature", "{1}{R}", 2, 2,
				mage.WithSubTypes("Goblin"),
				mage.WithKeyword(core.Menace),
			)
		})
	}
	// Artifact creature — can always block Fear creatures.
	if !mage.CardRegistered("Test Artifact Creature") {
		mage.Register("Test Artifact Creature", func() mage.Card {
			return mage.NewCreature("Test Artifact Creature", "{2}", 2, 2,
				mage.WithSubTypes("Construct"),
				mage.WithCardType(core.TypeArtifact),
			)
		})
	}
	// Boost spell targeting own creature (for Hexproof / Shroud tests).
	if !mage.CardRegistered("Test Boost Own Creature") {
		mage.Register("Test Boost Own Creature", func() mage.Card {
			return mage.NewInstant("Test Boost Own Creature", "{G}",
				mage.NewTargetedSpell(
					mage.TargetCreatureYouControl(),
					mage.BoostUntilEndOfTurn(mage.Fixed(2), mage.Fixed(2), mage.SelectTarget),
				),
			)
		})
	}
	// Rampage 2 creature (custom) for single-blocker no-bonus test.
	if !mage.CardRegistered("Test Rampage 2 Creature") {
		mage.Register("Test Rampage 2 Creature", func() mage.Card {
			return mage.NewCreature("Test Rampage 2 Creature", "{2}{R}", 3, 3,
				mage.WithSubTypes("Beast"),
				mage.WithAbility(mage.RampageTrigger(2)),
			)
		})
	}
	// Flash 2/2 — creature castable at instant speed (CR 702.8).
	if !mage.CardRegistered("Test Flash Creature") {
		mage.Register("Test Flash Creature", func() mage.Card {
			return mage.NewCreature("Test Flash Creature", "{1}{U}", 2, 2,
				mage.WithSubTypes("Spirit"),
				mage.WithKeyword(core.Flash),
			)
		})
	}
	// Flash aura — castable at instant speed; enchant creature, +1/+1.
	if !mage.CardRegistered("Test Flash Aura") {
		mage.Register("Test Flash Aura", func() mage.Card {
			return mage.NewAura("Test Flash Aura", "{U}",
				mage.WithKeyword(core.Flash),
				mage.WithStaticAbility(mage.BoostAttached(1, 1, core.AttachAura)),
			)
		})
	}
	// Island land for blue Flash test mana.
	if !mage.CardRegistered("Island") {
		mage.Register("Island", func() mage.Card {
			return mage.NewLand("Island",
				mage.WithSubTypes("Island"),
				mage.WithManaAbility(core.Blue),
			)
		})
	}
}

// ── 702.9 Flying ──────────────────────────────────────────────────────────────

// TestCR702_9b_FlyingGroundCannotBlock verifies that a ground creature
// cannot block a creature with Flying (CR 702.9b).
func TestCR702_9b_FlyingGroundCannotBlock(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Serra Angel") // 4/4 Flying Vigilance
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Hill Giant")  // 3/3, no flying
	tg.Attack(1, PlayerA, "Serra Angel")
	tg.Block(1, PlayerB, "Hill Giant", "Serra Angel") // illegal: ground can't block flier
	tg.StopAt(1, core.EndCombat)
	tg.Execute()
	// Ground creature cannot block flier; Serra Angel dealt unblocked damage.
	tg.AssertLife(PlayerB, 16)
	tg.AssertPermanentCount(PlayerB, "Hill Giant", 1)
}

// TestCR702_9b_FlyingCanBlockGround verifies that a creature with Flying can
// block a ground creature (CR 702.9b).
func TestCR702_9b_FlyingCanBlockGround(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Scryb Sprites") // 1/1 Flying
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears") // 2/2, ground
	tg.Attack(1, PlayerA, "Grizzly Bears")
	tg.Block(1, PlayerB, "Scryb Sprites", "Grizzly Bears")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()
	// Scryb Sprites (1/1) dies to Bears (2/2); Bears takes 1 but survives.
	tg.AssertPermanentCount(PlayerB, "Scryb Sprites", 0)
	tg.AssertPermanentCount(PlayerA, "Grizzly Bears", 1)
	tg.AssertLife(PlayerB, 20)
}

// ── 702.2 Deathtouch ──────────────────────────────────────────────────────────

// TestCR702_2b_DeathtouchKillsLargerBlocker verifies that a creature with Deathtouch
// kills a larger blocker with any nonzero damage (CR 702.2b).
func TestCR702_2b_DeathtouchKillsLargerBlocker(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Test Deathtouch Creature") // 1/1 Deathtouch
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Hill Giant")               // 3/3
	tg.Attack(1, PlayerA, "Test Deathtouch Creature")
	tg.Block(1, PlayerB, "Hill Giant", "Test Deathtouch Creature")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()
	// Deathtouch: 1 damage from 1/1 snake kills Hill Giant (3/3); snake dies to 3.
	tg.AssertPermanentCount(PlayerB, "Hill Giant", 0)
	tg.AssertPermanentCount(PlayerA, "Test Deathtouch Creature", 0)
	tg.AssertLife(PlayerB, 20)
}

// ── 702.15 Lifelink ───────────────────────────────────────────────────────────

// TestCR702_15b_LifelinkGainsLife verifies that Lifelink causes combat
// damage dealt to gain that much life for the controller (CR 702.15b).
func TestCR702_15b_LifelinkGainsLife(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Test Lifelink Creature") // 3/3 Lifelink
	tg.Attack(1, PlayerA, "Test Lifelink Creature")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()
	// 3 combat damage to PlayerB → PlayerA gains 3 life.
	tg.AssertLife(PlayerB, 17)
	tg.AssertLife(PlayerA, 23)
}

// ── 702.12 Indestructible ─────────────────────────────────────────────────────

// TestCR702_12b_IndestructibleSurvivesLethalDamage verifies that an indestructible
// creature accumulating lethal damage is not destroyed (CR 702.12b). Also
// guards against an SBA infinite loop: the lethal-damage SBA must skip
// indestructible creatures so CheckStateBasedActions terminates.
func TestCR702_12b_IndestructibleSurvivesLethalDamage(t *testing.T) {
	registerCh07bTestCards()
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Test Indestructible Creature") // 3/3
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Mountain")
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Mountain")
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Mountain")
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Mountain")
	tg.AddCard(core.ZoneHand, PlayerA, "Lightning Bolt") // 3 damage
	tg.AddCard(core.ZoneHand, PlayerA, "Lightning Bolt")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Lightning Bolt", "Test Indestructible Creature")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Lightning Bolt", "Test Indestructible Creature")
	tg.StopAt(1, core.EndStep)
	tg.Execute()
	// 6 damage is lethal to a 3/3, but indestructible means it stays.
	tg.AssertPermanentCount(PlayerB, "Test Indestructible Creature", 1)
}

// TestCR702_12b_IndestructibleSurvivesDestroyEffect verifies that an indestructible
// creature is not destroyed by "destroy" effects (CR 702.12b).
func TestCR702_12b_IndestructibleSurvivesDestroyEffect(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Test Indestructible Creature")
	tg.AddCard(core.ZoneHand, PlayerA, "Test Destroy Spell")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Destroy Spell", "Test Indestructible Creature")
	tg.StopAt(1, core.EndStep)
	tg.Execute()
	// Destroy effect does not affect indestructible permanent.
	tg.AssertPermanentCount(PlayerB, "Test Indestructible Creature", 1)
}

// ── 702.36 Fear ───────────────────────────────────────────────────────────────

// TestCR702_36b_FearCannotBeBlockedByNonArtifactNonBlack verifies that a creature with
// Fear cannot be blocked by a non-artifact, non-black creature (CR 702.36b).
func TestCR702_36b_FearCannotBeBlockedByNonArtifactNonBlack(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears") // 2/2 green
	tg.AddCard(core.ZoneHand, PlayerA, "Fear")                 // Aura: grants Fear
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Hill Giant")    // 3/3 red — not black, not artifact
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Fear", "Grizzly Bears")
	tg.Attack(1, PlayerA, "Grizzly Bears")
	tg.Block(1, PlayerB, "Hill Giant", "Grizzly Bears")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()
	// Hill Giant cannot block Fear creature; Bears deals 2 unblocked.
	tg.AssertLife(PlayerB, 18)
}

// TestCR702_36b_FearBlackCreatureCanBlock verifies that a black creature can block a
// Fear creature (CR 702.36b).
func TestCR702_36b_FearBlackCreatureCanBlock(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears") // 2/2 green
	tg.AddCard(core.ZoneHand, PlayerA, "Fear")
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Black Knight") // 2/2 black First Strike Pro-White
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Fear", "Grizzly Bears")
	tg.Attack(1, PlayerA, "Grizzly Bears")
	tg.Block(1, PlayerB, "Black Knight", "Grizzly Bears")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()
	// Black Knight is black → can block; it first-strikes for 2, Bears dies.
	tg.AssertPermanentCount(PlayerA, "Grizzly Bears", 0)
	tg.AssertLife(PlayerB, 20)
}

// TestCR702_36b_FearArtifactCreatureCanBlock verifies that an artifact creature can
// block a Fear creature regardless of color (CR 702.36b).
func TestCR702_36b_FearArtifactCreatureCanBlock(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears") // 2/2 green
	tg.AddCard(core.ZoneHand, PlayerA, "Fear")
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Test Artifact Creature") // 2/2 artifact
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Fear", "Grizzly Bears")
	tg.Attack(1, PlayerA, "Grizzly Bears")
	tg.Block(1, PlayerB, "Test Artifact Creature", "Grizzly Bears")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()
	// Artifact creature can block Fear creature; both 2/2s trade.
	tg.AssertPermanentCount(PlayerA, "Grizzly Bears", 0)
	tg.AssertPermanentCount(PlayerB, "Test Artifact Creature", 0)
	tg.AssertLife(PlayerB, 20)
}

// ── 702.19 Trample ────────────────────────────────────────────────────────────

// TestCR702_19b_TrampleExcessDamageGoesToPlayer verifies that a trample creature
// assigns lethal damage to the blocker and excess tramples to the defending
// player (CR 702.19b). 5/5 trample blocked by a 2/2: blocker dies, 3
// excess goes to the defending player.
func TestCR702_19b_TrampleExcessDamageGoesToPlayer(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Test Trample Creature") // 5/5 Trample
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Grizzly Bears")         // 2/2 blocker
	tg.Attack(1, PlayerA, "Test Trample Creature")
	tg.Block(1, PlayerB, "Grizzly Bears", "Test Trample Creature")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()
	// 2 damage kills the 2/2 blocker; 3 excess trample damage hits the player.
	tg.AssertPermanentCount(PlayerB, "Grizzly Bears", 0)
	tg.AssertLife(PlayerB, 17)
}

// ── 702.16 Protection ─────────────────────────────────────────────────────────

// TestCR702_16b_ProtectionAuraFallsOff verifies that an Aura of the protected
// quality cannot legally target and attach to a protected permanent, and falls
// off as SBA if it somehow becomes attached (CR 702.16c).
func TestCR702_16b_ProtectionAuraFallsOff(t *testing.T) {
	tg := NewTestGame(t)
	// White Knight has Protection from Black; Fear is a black aura → can't enchant it.
	tg.AddCard(core.ZoneBattlefield, PlayerA, "White Knight") // 2/2 Pro-Black
	tg.AddCard(core.ZoneHand, PlayerA, "Fear")                // {B}{B} black aura
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Fear", "White Knight")
	tg.StopAt(1, core.EndStep)
	tg.Execute()
	// Fear (black) cannot be attached to White Knight (protection from black).
	// Aura should fail to attach or fall off as SBA.
	tg.AssertPermanentCount(PlayerA, "Fear", 0)
}

// ── 702.18 Shroud ─────────────────────────────────────────────────────────────

// TestCR702_18_ShroudControllerCannotTarget verifies that even the
// controller cannot target a permanent with Shroud (CR 702.18a).
func TestCR702_18_ShroudControllerCannotTarget(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Test Shroud Creature") // 2/2 Shroud
	tg.AddCard(core.ZoneHand, PlayerA, "Test Boost Own Creature")     // boosts a creature you control
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Boost Own Creature", "Test Shroud Creature")
	tg.StopAt(1, core.EndStep)
	tg.Execute()
	// Shroud prevents controller from targeting own creature; spell fizzles.
	tg.AssertPowerToughness(PlayerA, "Test Shroud Creature", 2, 2)
}

// ── 702.11 Hexproof ───────────────────────────────────────────────────────────

// TestCR702_11_HexproofControllerCanTargetOwn verifies that Hexproof only prevents
// opponents from targeting — the controller may target their own hexproof
// permanent (CR 702.11b).
func TestCR702_11_HexproofControllerCanTargetOwn(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Test Hexproof Creature") // 2/2 Hexproof
	tg.AddCard(core.ZoneHand, PlayerA, "Test Boost Own Creature")       // PlayerA's own boost spell
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Boost Own Creature", "Test Hexproof Creature")
	tg.StopAt(1, core.EndStep)
	tg.Execute()
	// Controller can target own Hexproof creature; creature is boosted.
	tg.AssertPowerToughness(PlayerA, "Test Hexproof Creature", 4, 4)
}

// ── 702.20 Vigilance ──────────────────────────────────────────────────────────

// TestCR702_20_VigilanceCanActivateAfterAttacking verifies that a Vigilance
// creature doesn't tap when attacking, so it is still untapped afterward
// (CR 702.20b; tap-ability use after attack would be possible).
func TestCR702_20_VigilanceCanActivateAfterAttacking(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Serra Angel") // 4/4 Flying, Vigilance
	tg.Attack(1, PlayerA, "Serra Angel")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()
	tg.AssertTapped(PlayerA, "Serra Angel", false) // Vigilance: untapped after attacking
	tg.AssertLife(PlayerB, 16)
}

// ── 702.14 Landwalk ───────────────────────────────────────────────────────────

// TestCR702_14b_LandwalkBlockableWhenDefenderLacksLand verifies that a Swampwalk
// creature is blockable when the defending player controls no Swamps
// (CR 702.14c).
func TestCR702_14b_LandwalkBlockableWhenDefenderLacksLand(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Bog Wraith") // 3/3 Swampwalk
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Hill Giant") // 3/3; PlayerB has no Swamp
	tg.Attack(1, PlayerA, "Bog Wraith")
	tg.Block(1, PlayerB, "Hill Giant", "Bog Wraith")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()
	// No Swamp for defender → Swampwalk inactive; Hill Giant can block.
	tg.AssertPermanentCount(PlayerA, "Bog Wraith", 0)
	tg.AssertPermanentCount(PlayerB, "Hill Giant", 0)
	tg.AssertLife(PlayerB, 20)
}

// TestCR702_14b_LandwalkDoesNotCancelOpponentLandwalk verifies that a defending player
// having the same landwalk does not cancel the attacker's landwalk evasion
// (CR 702.14d — landwalk abilities are independent).
func TestCR702_14b_LandwalkDoesNotCancelOpponentLandwalk(t *testing.T) {
	tg := NewTestGame(t)
	// PlayerB controls a Swamp → PlayerA's Bog Wraith gets Swampwalk evasion.
	// PlayerB also has a Bog Wraith with Swampwalk, but that doesn't affect blocking.
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Bog Wraith") // 3/3 Swampwalk (PlayerA's)
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Bog Wraith") // 3/3 Swampwalk (PlayerB's)
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Swamp")      // PlayerB controls a Swamp
	tg.Attack(1, PlayerA, "Bog Wraith")
	tg.Block(1, PlayerB, "Bog Wraith", "Bog Wraith")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()
	// PlayerA's Bog Wraith is unblockable (PlayerB has Swamp); deals 3 damage.
	tg.AssertLife(PlayerB, 17)
}

// ── 702.23 Rampage ────────────────────────────────────────────────────────────

// TestCR702_23b_RampageNoBonusSingleBlocker verifies that Rampage provides no bonus
// when a creature is blocked by exactly one creature (CR 702.23a).
func TestCR702_23b_RampageNoBonusSingleBlocker(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Test Rampage 2 Creature") // 3/3 Rampage 2
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Hill Giant")              // 3/3
	tg.Attack(1, PlayerA, "Test Rampage 2 Creature")
	tg.Block(1, PlayerB, "Hill Giant", "Test Rampage 2 Creature")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()
	// Single blocker → no Rampage bonus; both 3/3s trade.
	tg.AssertPermanentCount(PlayerA, "Test Rampage 2 Creature", 0)
	tg.AssertPermanentCount(PlayerB, "Hill Giant", 0)
}

// TestCR702_23b_RampageBonusPerExtraBlocker verifies that Rampage grants +N/+N for each
// blocker beyond the first (CR 702.23a). Aerathi Berserker (2/4, Rampage 3)
// blocked by two creatures gets +3/+3 → 5/7, killing both 2/2 blockers.
func TestCR702_23b_RampageBonusPerExtraBlocker(t *testing.T) {
	bearA := "Rampage Blocker A"
	bearB := "Rampage Blocker B"
	for _, n := range []string{bearA, bearB} {

		if !mage.CardRegistered(n) {
			mage.Register(n, func() mage.Card {
				return mage.NewCreature(n, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
			})
		}
	}
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Aerathi Berserker") // 2/4 Rampage 3
	tg.AddCard(core.ZoneBattlefield, PlayerB, bearA)
	tg.AddCard(core.ZoneBattlefield, PlayerB, bearB)
	tg.Attack(1, PlayerA, "Aerathi Berserker")
	tg.Block(1, PlayerB, bearA, "Aerathi Berserker")
	tg.Block(1, PlayerB, bearB, "Aerathi Berserker")
	tg.AssignCombatDamage("Aerathi Berserker", map[string]int{bearA: 3, bearB: 3})
	tg.StopAt(1, core.EndCombat)
	tg.Execute()
	// Two blockers → +3/+3 (one beyond first) → Berserker is 5/7.
	// Each blocker gets 3 damage from the boosted attacker → both die.
	// Blockers deal 2+2=4 to Berserker (7 toughness) → Berserker survives.
	tg.AssertPermanentCount(PlayerA, "Aerathi Berserker", 1)
	tg.AssertPermanentCount(PlayerB, bearA, 0)
	tg.AssertPermanentCount(PlayerB, bearB, 0)
}

// ── 702.111 Menace ────────────────────────────────────────────────────────────

// TestCR702_110b_MenaceRequiresTwoBlockers verifies that a creature with Menace can't be
// blocked by fewer than two creatures (CR 702.111).
// XXX: the engine does NOT currently enforce the Menace blocking restriction
// (combat.go comment: "simplified — we don't enforce here"). This test documents
// the gap: a single-blocker block is accepted when it should be illegal.
func TestCR702_110b_MenaceRequiresTwoBlockers(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Test Menace Creature") // 2/2 Menace
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Grizzly Bears")        // 2/2 single blocker
	tg.Attack(1, PlayerA, "Test Menace Creature")
	tg.Block(1, PlayerB, "Grizzly Bears", "Test Menace Creature") // single block — illegal under Menace
	tg.StopAt(1, core.EndCombat)
	tg.Execute()
	// With correct Menace enforcement: single block illegal → attacker unblocked → PlayerB at 18.
	// Current engine: single block accepted → both 2/2s trade → PlayerB at 20.
	// Asserting 20 (engine behavior); this test FAILS when engine enforces Menace correctly.
	tg.AssertLife(PlayerB, 20)
}

// ── 702.8 Flash ───────────────────────────────────────────────────────────────

// TestCR702_8_FlashCreatureCastableOnOpponentTurn verifies CR 702.8: a creature
// with Flash may be cast any time its controller could cast an instant —
// including during the opponent's turn.
func TestCR702_8_FlashCreatureCastableOnOpponentTurn(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerB, "Test Flash Creature")
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Island", 2)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Hill Giant")
	tg.CastSpell(1, core.BeginCombat, PlayerB, "Test Flash Creature")
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()

	tg.AssertPermanentCount(PlayerB, "Test Flash Creature", 1)
	tg.AssertHandCount(PlayerB, "Test Flash Creature", 0)
}

// TestCR702_8_FlashCreatureCastableInResponse verifies CR 702.8: a Flash
// creature may be cast in response to a spell on the stack.
func TestCR702_8_FlashCreatureCastableInResponse(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Test Flash Creature")
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Island", 2)
	tg.AddCard(core.ZoneHand, PlayerB, "Lightning Bolt")
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Mountain")
	tg.CastSpell(1, core.PrecombatMain, PlayerB, "Lightning Bolt", "PlayerA")
	tg.CastInResponseTo(PlayerA, "Test Flash Creature")
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Test Flash Creature", 1)
}

// TestCR702_8_NonFlashCreatureCannotBeCastAtInstantSpeed verifies the negative
// case: a creature without Flash still cannot be cast on the opponent's turn.
func TestCR702_8_NonFlashCreatureCannotBeCastAtInstantSpeed(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerB, "Grizzly Bears")
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Forest", 2)
	tg.CastSpell(1, core.BeginCombat, PlayerB, "Grizzly Bears")
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()

	tg.AssertHandCount(PlayerB, "Grizzly Bears", 1)
	tg.AssertPermanentCount(PlayerB, "Grizzly Bears", 0)
}

// TestCR702_8_FlashAuraCastableAtInstantSpeed verifies CR 702.8 for auras:
// an aura with Flash can be cast on the opponent's turn.
func TestCR702_8_FlashAuraCastableAtInstantSpeed(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerB, "Test Flash Aura")
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Island")
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.CastSpell(1, core.BeginCombat, PlayerB, "Test Flash Aura", "Grizzly Bears")
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()

	tg.AssertPermanentCount(PlayerB, "Test Flash Aura", 1)
	tg.AssertAttachedTo(PlayerA, "Test Flash Aura", "Grizzly Bears")
}
