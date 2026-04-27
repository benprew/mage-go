// Package gametest: rules tests for CR Chapter 1 (Game Concepts, rules 100–123).
// Each function targets a specific subrule noted in the comment above it.
package gametest

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// ── Rule 103.4 ──────────────────────────────────────────────────────────────

// CR 103.4: Each player begins the game with 20 life.
func TestCR103_4_StartingLife20(t *testing.T) {
	tg := NewTestGame(t)
	tg.StopAt(1, core.Upkeep)
	tg.Execute()
	tg.AssertLife(PlayerA, 20)
	tg.AssertLife(PlayerB, 20)
}

// ── Rule 103.8 ───────────────────────────────────────────────────────────────

// CR 103.8: The starting player (PlayerA in test harness) takes the first turn.
func TestCR103_8_FirstTurnActivePlayer(t *testing.T) {
	tg := NewTestGame(t)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()
	// Turn 1 ends after PlayerA's turn in the harness convention.
	// AssertTotalTurns(1) confirms we are still in turn 1.
	tg.AssertTotalTurns(1)
}

// ── Rule 103.3 ───────────────────────────────────────────────────────────────

// CR 103.3: Each player shuffles their library; libraries exist from game start.
func TestCR103_3_LibraryInitialized(t *testing.T) {
	tg := NewTestGame(t)
	tg.StopAt(1, core.Upkeep)
	tg.Execute()
	// padLibraries() ensures each player has a non-empty library.
	tg.AssertLibraryCount(PlayerA, "", 0) // 0 named cards in the padded library
	// Just confirm game is running normally (library exists, no deck-out).
	tg.AssertGameOver(false)
}

// ── Rule 104.2a / 104.3b ─────────────────────────────────────────────────────

// CR 104.2a: A player wins if all opponents have left the game.
// CR 104.3b (SBA): A player at 0 life loses.
func TestCR104_2a_WinLastPlayerStanding(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears") // 2/2
	tg.SetLife(PlayerB, 2)
	tg.Attack(1, PlayerA, "Grizzly Bears")
	tg.PlayToEnd()
	tg.AssertWinner(PlayerA)
	tg.AssertGameOver(true)
}

// CR 104.3b: Losing at exactly 0 life.
func TestCR104_3b_LoseAtZeroLife(t *testing.T) {
	tg := NewTestGame(t)
	tg.SetLife(PlayerB, 0)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()
	tg.AssertGameOver(true)
	tg.AssertWinner(PlayerA)
}

// CR 104.3b: Losing at negative life triggers the same SBA.
func TestCR104_3b_LoseAtNegativeLife(t *testing.T) {
	tg := NewTestGame(t)
	tg.SetLife(PlayerB, -5)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()
	tg.AssertGameOver(true)
	tg.AssertWinner(PlayerA)
}

// ── Rule 104.3d ──────────────────────────────────────────────────────────────

// CR 104.3d: A player with 10 or more poison counters loses.
func TestCR104_3d_LoseTenPoison(t *testing.T) {
	tg := NewTestGame(t)
	tg.GetPlayer(PlayerB).AddPoisonCounters(10)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()
	tg.AssertGameOver(true)
	tg.AssertWinner(PlayerA)
}

// CR 104.3d (negative): 9 poison counters is not lethal.
func TestCR104_3d_NinePoisonNoLoss(t *testing.T) {
	tg := NewTestGame(t)
	tg.GetPlayer(PlayerB).AddPoisonCounters(9)
	tg.StopAt(1, core.EndStep)
	tg.Execute()
	tg.AssertGameOver(false)
	tg.AssertPoisonCounters(PlayerB, 9)
}

// ── Rule 104.5 ───────────────────────────────────────────────────────────────

// CR 104.5: A player who loses the game is no longer in the game.
// Verified by asserting the winner is the surviving player after a loss by 0 life.
func TestCR104_5_PlayerLeavesOnLoss(t *testing.T) {
	tg := NewTestGame(t)
	tg.SetLife(PlayerB, 0)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()
	tg.AssertWinner(PlayerA)
	tg.AssertGameOver(true)
}

// ── Rule 101.1 (card text overrides rules) ───────────────────────────────────

// CR 101.1: Card text overrides the rules when they directly contradict.
// Dwarven Warriors' activated ability makes a creature with power ≤ 2 unblockable,
// overriding the default rule that allows blocking.
func TestCR101_1_CardTextOverridesRule(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Dwarven Warriors") // {2}{R} 1/1
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Savannah Lions")   // 2/1 — power ≤ 2
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Hill Giant")       // 3/3
	tg.SetLife(PlayerB, 20)
	// Make Savannah Lions unblockable via Dwarven Warriors' ability.
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, "Dwarven Warriors", "Savannah Lions")
	tg.Attack(1, PlayerA, "Savannah Lions")
	tg.Block(1, PlayerB, "Hill Giant", "Savannah Lions")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()
	// Lion is unblockable this turn → deals 2 damage directly.
	tg.AssertLife(PlayerB, 18)
}

// ── Rule 101.2 (cant overrides can for blocking) ─────────────────────────────

// CR 101.2: A "can't block" effect overrides any "can block" permission.
// Icy Manipulator taps Hill Giant before blockers are declared, preventing it
// from blocking even though it otherwise could.
func TestCR101_2_CantOverridesCanBlocking(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "White Knight") // 2/2
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Hill Giant")   // 3/3
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Icy Manipulator")
	// Tap Hill Giant (preventing it from blocking).
	tg.ActivateAbility(1, core.BeginCombat, PlayerA, "Icy Manipulator", "Hill Giant")
	tg.Attack(1, PlayerA, "White Knight")
	tg.Block(1, PlayerB, "Hill Giant", "White Knight")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()
	// Tapped Hill Giant can't block; White Knight deals 2 damage.
	tg.AssertLife(PlayerB, 18)
}

// ── Rule 101.3 ───────────────────────────────────────────────────────────────

// CR 101.3: Impossible instructions are simply ignored.
// Balance says "each player sacrifices creatures until they have as many
// creatures as the player with the fewest." When PlayerB has 0 creatures and
// PlayerA has 1, PlayerA must sac down to 0 — but PlayerB has nothing to do;
// the engine must not error or crash.
func TestCR101_3_ImpossibleSacrificeIgnored(t *testing.T) {
	tg := NewTestGame(t)
	// PlayerA controls one creature.
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	// PlayerA's hand: Balance.
	tg.AddCard(core.ZoneHand, PlayerA, "Balance")
	// PlayerB has no creatures — the "sacrifice a creature" sub-instruction
	// for PlayerB is impossible and must be silently ignored.
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Balance")
	tg.StopAt(1, core.EndStep)
	tg.Execute()
	// Game continues normally.
	tg.AssertGameOver(false)
}

// CR 101.3: Discarding when hand is empty is impossible; ignored.
// Timetwister forces each player to discard their hand then draw 7;
// PlayerB has no cards to discard — that instruction must be a no-op.
func TestCR101_3_ImpossibleDiscardIgnored(t *testing.T) {
	tg := NewTestGame(t)
	// PlayerB has no cards in hand (harness does not deal a starting hand).
	tg.AddCard(core.ZoneHand, PlayerA, "Timetwister")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Timetwister")
	tg.StopAt(1, core.EndStep)
	tg.Execute()
	// Game continues normally; empty-hand discard for PlayerB was a no-op.
	tg.AssertGameOver(false)
}

// ── Rule 105.2 ───────────────────────────────────────────────────────────────

// CR 105.2: A permanent's color is determined by its mana cost.
func TestCR105_2_ColorFromManaCostBlue(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Merfolk of the Pearl Trident") // {U} — blue
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()
	tg.AssertHasColor(PlayerA, "Merfolk of the Pearl Trident", core.Blue, true)
	tg.AssertHasColor(PlayerA, "Merfolk of the Pearl Trident", core.White, false)
}

// CR 105.2: A multicolored permanent has both colors.
func TestCR105_2_ColorFromManaCostMulticolor(t *testing.T) {
	multiName := "Two Color Test Creature"
	if !mage.CardRegistered(multiName) {
		mage.Register(multiName, func() mage.Card {
			// {W}{B} cost makes it White and Black.
			return mage.NewCreature(multiName, "{W}{B}", 2, 2, mage.WithSubTypes("Knight"))
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, multiName)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()
	tg.AssertHasColor(PlayerA, multiName, core.White, true)
	tg.AssertHasColor(PlayerA, multiName, core.Black, true)
	tg.AssertHasColor(PlayerA, multiName, core.Green, false)
}

// CR 105.2c: A colorless artifact has no colors.
func TestCR105_2c_ColorlessNoColors(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Sol Ring") // colorless artifact
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()
	tg.AssertHasColor(PlayerA, "Sol Ring", core.White, false)
	tg.AssertHasColor(PlayerA, "Sol Ring", core.Blue, false)
	tg.AssertHasColor(PlayerA, "Sol Ring", core.Black, false)
	tg.AssertHasColor(PlayerA, "Sol Ring", core.Red, false)
	tg.AssertHasColor(PlayerA, "Sol Ring", core.Green, false)
}

// ── Rule 106.3 / 106.12 ──────────────────────────────────────────────────────

// CR 106.3 + CR 106.12: Tapping a basic land activates its mana ability,
// which places mana in the player's pool.
func TestCR106_3_LandProducesMana(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Forest")
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, "Forest")
	// Stop at BeginCombat so PrecombatMain runs fully and the activation fires.
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()
	tg.AssertManaProducedAtLeast(PlayerA, core.Green, 1)
	tg.AssertTapped(PlayerA, "Forest", true)
}

// CR 106.12: Tapping a Plains for mana taps the permanent.
func TestCR106_12_TapForManaBasic(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Plains")
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, "Plains")
	// Stop at BeginCombat so PrecombatMain runs fully and the activation fires.
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()
	tg.AssertManaProducedAtLeast(PlayerA, core.White, 1)
	tg.AssertTapped(PlayerA, "Plains", true)
}

// ── Rule 107.1b ──────────────────────────────────────────────────────────────

// CR 107.1b: A creature with negative power deals 0 combat damage.
func TestCR107_1b_NegativePowerNoDamage(t *testing.T) {
	debuffName := "Neg Power Bear"
	if !mage.CardRegistered(debuffName) {
		mage.Register(debuffName, func() mage.Card {
			return mage.NewCreature(debuffName, "{1}{G}", 2, 4, mage.WithSubTypes("Bear"))
		})
	}
	debuffEffect := "Shrink Effect"
	if !mage.CardRegistered(debuffEffect) {
		mage.Register(debuffEffect, func() mage.Card {
			// "Target creature gets -5/+0 until end of turn."
			return mage.NewInstant(debuffEffect, "{G}",
				mage.NewTargetedSpell(mage.TargetCreature(),
					mage.BoostUntilEndOfTurn(mage.Fixed(-5), mage.Fixed(0), mage.SelectTarget)),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, debuffName)
	tg.AddCard(core.ZoneHand, PlayerA, debuffEffect)
	tg.SetLife(PlayerB, 20)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, debuffEffect, debuffName)
	tg.Attack(1, PlayerA, debuffName)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()
	// Creature is now -3/4; attack deals 0 combat damage (negative power floor).
	tg.AssertLife(PlayerB, 20)
	tg.AssertPowerToughness(PlayerA, debuffName, -3, 4)
}

// ── Rule 107.3a ──────────────────────────────────────────────────────────────

// CR 107.3a: When casting a spell with X in the cost, the controller chooses X.
func TestCR107_3a_XSpellControllerChoosesX(t *testing.T) {
	tg := NewTestGame(t)
	tg.SetLife(PlayerB, 20)
	tg.AddCard(core.ZoneHand, PlayerA, "Fireball")
	tg.CastSpellWithX(1, core.PrecombatMain, PlayerA, "Fireball", 3, "PlayerB")
	tg.StopAt(1, core.EndStep)
	tg.Execute()
	// Fireball with X=3 deals 3 damage to PlayerB.
	tg.AssertLife(PlayerB, 17)
}

// ── Rule 107.5 (tap symbol) ──────────────────────────────────────────────────

// CR 107.5: {T} taps the permanent to pay the cost.
func TestCR107_5_TapSymbolTapsPermanent(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Forest")
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, "Forest")
	// Stop at BeginCombat so PrecombatMain runs fully and the activation fires.
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()
	tg.AssertTapped(PlayerA, "Forest", true)
}

// CR 107.5: A creature with summoning sickness cannot activate tap abilities.
// (Already covered by TestSummoningSickness_CreatureCannotTapForManaTurnPlayed
// in attr_integration_test.go for the mana-ability case; here we use a non-mana
// tap ability — Icy Manipulator — cast this turn.)
func TestCR107_5_SummoningSicknessTapAbility(t *testing.T) {
	tg := NewTestGame(t)
	// Llanowar Elves: {G}, {T}: Add {G}. Cast on turn 1, same turn check.
	tg.AddCard(core.ZoneHand, PlayerA, "Llanowar Elves")
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Forest")
	// Cast the Elves on turn 1, then try to tap them for mana on the same turn.
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Llanowar Elves")
	// The ActivateAbility below will be attempted in PrecombatMain after the
	// creature resolved; the harness ignores failed activations gracefully.
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, "Llanowar Elves")
	tg.StopAt(1, core.EndStep)
	tg.Execute()
	// Llanowar Elves should still be untapped (couldn't activate due to sickness).
	tg.AssertTapped(PlayerA, "Llanowar Elves", false)
}

// ── Rule 109.3 ───────────────────────────────────────────────────────────────

// CR 109.3: An object's characteristics include name, power, toughness, color, card type.
func TestCR109_3_ObjectCharacteristicsCreature(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears") // 2/2 Green creature
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()
	tg.AssertPowerToughness(PlayerA, "Grizzly Bears", 2, 2)
	tg.AssertHasColor(PlayerA, "Grizzly Bears", core.Green, true)
	tg.AssertPermanentCount(PlayerA, "Grizzly Bears", 1)
}

// ── Rule 110.1 ───────────────────────────────────────────────────────────────

// CR 110.1: A card or token on the battlefield is a permanent.
func TestCR110_1_PermanentEntersBattlefield(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()
	tg.AssertPermanentCount(PlayerA, "Grizzly Bears", 1)
}

// CR 110.1: When a permanent leaves the battlefield it ceases to be a permanent.
func TestCR110_1_PermanentLeavesOnDeath(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.AddCard(core.ZoneHand, PlayerB, "Terror")
	tg.CastSpell(1, core.PrecombatMain, PlayerB, "Terror", "Grizzly Bears")
	tg.StopAt(1, core.EndStep)
	tg.Execute()
	tg.AssertPermanentCount(PlayerA, "Grizzly Bears", 0)
	tg.AssertGraveyardCount(PlayerA, "Grizzly Bears", 1)
}

// CR 110.4: Instants and sorceries cannot be permanents; after resolution
// they go to the graveyard (or are countered) and never appear on battlefield.
func TestCR110_4_InstantCannotBePermanent(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Giant Growth")
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Giant Growth", "Grizzly Bears")
	tg.StopAt(1, core.EndStep)
	tg.Execute()
	// Giant Growth never appears on the battlefield.
	tg.AssertPermanentCount(PlayerA, "Giant Growth", 0)
	tg.AssertGraveyardCount(PlayerA, "Giant Growth", 1)
}

// ── Rule 110.5 / 110.5b ──────────────────────────────────────────────────────

// CR 110.5: A permanent's default status is untapped.
func TestCR110_5_DefaultUntapped(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()
	tg.AssertTapped(PlayerA, "Grizzly Bears", false)
}

// CR 110.5b: Some permanents enter the battlefield tapped.
// We verify by inspecting the permanent directly after AddCard (which calls
// PutOnBattlefield), before Execute() runs the Untap step.
func TestCR110_5b_EntersTapped(t *testing.T) {
	ruinsName := "Test Enters Tapped Land"
	if !mage.CardRegistered(ruinsName) {
		mage.Register(ruinsName, func() mage.Card {
			return mage.NewLand(ruinsName,
				mage.WithKeyword(core.EntersTapped),
				mage.WithManaAbility(core.Red),
			)
		})
	}

	tg := NewTestGame(t)
	permID := tg.AddCard(core.ZoneBattlefield, PlayerA, ruinsName)
	// Assert tapped state directly on the permanent before any steps run.
	perm := tg.FindPermanent(permID)
	if perm == nil {
		t.Fatal("permanent not found")
	}
	if !perm.Tapped {
		t.Errorf("TestCR110_5b_EntersTapped: permanent with EntersTapped should be tapped immediately after entering, got untapped")
	}
}

// ── Rule 111.6 ───────────────────────────────────────────────────────────────

// CR 111.6: Tokens are subject to effects that affect permanents of their type.
// Lord of Atlantis gives all Merfolk (including token Merfolk) +1/+1.
// We use a simpler proxy: register a "boost all creatures" enchantment card
// and confirm the token's P/T increases.
func TestCR111_6_TokenAffectedByPermanentEffects(t *testing.T) {
	boostAll := "Boost All Creatures Enchantment"
	if !mage.CardRegistered(boostAll) {
		mage.Register(boostAll, func() mage.Card {
			return mage.NewEnchantment(boostAll, "{2}{G}",
				mage.WithStaticAbility(mage.BoostAllCreaturesIncludingSelf(1, 1, mage.IsCreature)),
			)
		})
	}

	tg := NewTestGame(t)
	// The Hive creates 1/1 flying Wasp tokens.
	tg.AddCard(core.ZoneBattlefield, PlayerA, "The Hive")
	tg.AddCard(core.ZoneBattlefield, PlayerA, boostAll)
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, "The Hive")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// Wasp token (1/1) is boosted to 2/2 by the global enchantment.
	tg.AssertPowerToughness(PlayerA, "Wasp", 2, 2)
}

// ── Rule 111.7 ───────────────────────────────────────────────────────────────

// CR 111.7: A token exists on the battlefield after being created.
// (The follow-up test verifies it ceases to exist when destroyed.)
func TestCR111_7_TokenExistsOnBattlefield(t *testing.T) {
	tg := NewTestGame(t)
	// The Hive creates 1/1 flying Wasp artifact creature tokens.
	tg.AddCard(core.ZoneBattlefield, PlayerA, "The Hive")
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, "The Hive")
	tg.StopAt(1, core.EndStep)
	tg.Execute()
	// Token exists on the battlefield.
	tg.AssertPermanentCount(PlayerA, "Wasp", 1)
}

// CR 111.7: Token destroyed → ceases to exist, graveyard count stays 0.
func TestCR111_7_TokenCeasesAfterDeath(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "The Hive")
	tg.AddCard(core.ZoneHand, PlayerB, "Wrath of God")
	// Activate Hive on turn 1 to create a Wasp.
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, "The Hive")
	// PlayerB casts Wrath of God on turn 2 to destroy it.
	tg.CastSpell(2, core.PrecombatMain, PlayerB, "Wrath of God")
	tg.StopAt(2, core.EndStep)
	tg.Execute()

	// Token ceased to exist; graveyard count for "Wasp" should be 0.
	tg.AssertGraveyardCount(PlayerA, "Wasp", 0)
	tg.AssertPermanentCount(PlayerA, "Wasp", 0)
}

// CR 111.7: Tokens never enter the graveyard or hand — they cease to exist
// when they would leave the battlefield.
func TestCR111_7_TokenNeverEntersGraveyardOrHand(t *testing.T) {
	t.Run("token does not enter graveyard when destroyed", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, "The Hive")
		tg.AddCard(core.ZoneHand, PlayerB, "Wrath of God")
		tg.ActivateAbility(1, core.PrecombatMain, PlayerA, "The Hive")
		tg.CastSpell(2, core.PrecombatMain, PlayerB, "Wrath of God")
		tg.StopAt(2, core.EndStep)
		tg.Execute()
		tg.AssertGraveyardCount(PlayerA, "Wasp", 0)
	})

	t.Run("token does not enter hand when bounced", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, "The Hive")
		tg.AddCard(core.ZoneHand, PlayerB, "Unsummon")
		tg.ActivateAbility(1, core.PrecombatMain, PlayerA, "The Hive")
		tg.CastSpell(2, core.PrecombatMain, PlayerB, "Unsummon", "Wasp")
		tg.StopAt(2, core.EndStep)
		tg.Execute()
		tg.AssertPermanentCount(PlayerA, "Wasp", 0)
		tg.AssertHandCount(PlayerA, "Wasp", 0)
	})
}
