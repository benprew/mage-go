// Package gametest: rules tests for CR 703–732 (Additional Rules).
// Focus: Rule 704 State-Based Actions.
package gametest

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// ============================================================
// 704.5a — Player at 0 or less life loses the game.
// NOTE: 104.3b/SBA duplicate tests removed; canonical copies live in
// rules_ch01_game_concepts_test.go (TestCR104_3b_LoseAtZeroLife, etc.).
// ============================================================

// ============================================================
// 704.5c — Player with 10+ poison counters loses.
// NOTE: 104.3d duplicate tests removed; canonical copies live in
// rules_ch01_game_concepts_test.go (TestCR104_3d_LoseTenPoison, etc.).
// ============================================================

// ============================================================
// 704.5d — Token in zone other than battlefield ceases to exist.
// (Distinct from CR 111.7 which covers token existence while on battlefield.)
// ============================================================

// TestCR704_5d_TokenCeasesInGraveyard verifies that a destroyed token does not remain in the
// graveyard (CR 704.5d).
func TestCR704_5d_TokenCeasesInGraveyard(t *testing.T) {
	const killName = "SBA Token Kill Spell"
	const tokenCreatorName = "SBA Token Creator"
	if !mage.CardRegistered(tokenCreatorName) {
		mage.Register(tokenCreatorName, func() mage.Card {
			return mage.NewSorcery(tokenCreatorName, "{1}{G}",
				mage.NewSpellAbility(mage.CreateToken(
					"SBA Test Saproling", 1, 1,
					[]core.CardType{core.TypeCreature},
					[]string{"Saproling"},
				)),
			)
		})
	}
	if !mage.CardRegistered(killName) {
		mage.Register(killName, func() mage.Card {
			return mage.NewInstant(killName, "{1}{B}",
				mage.NewTargetedSpell(mage.TargetCreature(), mage.DestroyTarget()),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, tokenCreatorName)
	tg.AddCard(core.ZoneHand, PlayerA, killName)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, tokenCreatorName)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, killName, "SBA Test Saproling")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	// Token should not appear in PlayerA's graveyard (ceases to exist per 704.5d)
	tg.AssertGraveyardCount(PlayerA, "SBA Test Saproling", 0)
}

// ============================================================
// 704.5f — Creature with toughness 0 or less is put into graveyard; regeneration can't replace.
// ============================================================

// TestCR704_5f_NegativeToughnessDies verifies that a creature with negative toughness from -1/-1
// counters is put into its owner's graveyard (CR 704.5f).
// Note: TestZeroToughnessDies covers the 0-toughness case (already in mechanics_test.go).
func TestCR704_5f_NegativeToughnessDies(t *testing.T) {
	const crName = "SBA Minus Two Toughness Creature"
	if !mage.CardRegistered(crName) {
		mage.Register(crName, func() mage.Card {
			return mage.NewCreature(crName, "{1}{G}", 3, 2, mage.WithSubTypes("Beast"))
		})
	}

	tg := NewTestGame(t)
	id := tg.AddCard(core.ZoneBattlefield, PlayerA, crName)
	perm := tg.FindPermanent(id)
	perm.AddCounter(core.M1M1, 3) // 3/2 with three -1/-1 = 0/-1 toughness
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, crName, 0)
	tg.AssertGraveyardCount(PlayerA, crName, 1)
}

// TestCR704_5f_ZeroToughnessNoRegeneration verifies that regeneration cannot save a creature with 0
// toughness; it is put into the graveyard regardless (CR 704.5f).
func TestCR704_5f_ZeroToughnessNoRegeneration(t *testing.T) {
	const regenName = "SBA Regen Creature Zero Tough"
	if !mage.CardRegistered(regenName) {
		mage.Register(regenName, func() mage.Card {
			return mage.NewCreature(regenName, "{2}{G}", 3, 1,
				mage.WithSubTypes("Troll"),
				mage.WithActivatedAbility(
					mage.RegenerateSource(),
					mage.ManaCostOf("{G}"),
				),
			)
		})
	}

	tg := NewTestGame(t)
	id := tg.AddCard(core.ZoneBattlefield, PlayerA, regenName)
	perm := tg.FindPermanent(id)
	perm.AddCounter(core.M1M1, 1) // 3/1 with -1/-1 = 2/0 toughness
	// Activate regeneration before SBA fires
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, regenName)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	// Regeneration cannot prevent the SBA for 0 toughness (creature bypasses destruction)
	tg.AssertPermanentCount(PlayerA, regenName, 0)
	tg.AssertGraveyardCount(PlayerA, regenName, 1)
}

// ============================================================
// 704.5g — Creature with lethal damage is destroyed; regeneration can replace.
// ============================================================

// TestCR704_5g_LethalDamageDestroysCreature verifies that a creature with lethal damage is destroyed
// during the combat SBA check (CR 704.5g).
func TestCR704_5g_LethalDamageDestroysCreature(t *testing.T) {
	const atkName = "SBA Lethal Attacker"
	const defName = "SBA Lethal Defender"
	if !mage.CardRegistered(atkName) {
		mage.Register(atkName, func() mage.Card {
			return mage.NewCreature(atkName, "{3}{R}", 5, 5, mage.WithSubTypes("Dragon"))
		})
	}
	if !mage.CardRegistered(defName) {
		mage.Register(defName, func() mage.Card {
			return mage.NewCreature(defName, "{1}{W}", 2, 2, mage.WithSubTypes("Knight"))
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, atkName)
	tg.AddCard(core.ZoneBattlefield, PlayerB, defName)
	tg.Attack(1, PlayerA, atkName)
	tg.Block(1, PlayerB, defName, atkName)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	// 2/2 blocker takes 5 damage — lethal — destroyed; attacker survives (2 < 5)
	tg.AssertPermanentCount(PlayerB, defName, 0)
	tg.AssertGraveyardCount(PlayerB, defName, 1)
	tg.AssertPermanentCount(PlayerA, atkName, 1)
}

// TestCR704_5g_LethalDamageExact verifies that a creature with damage exactly equal to its toughness
// is destroyed (CR 704.5g).
func TestCR704_5g_LethalDamageExact(t *testing.T) {
	const atkName = "SBA Exact Lethal Attacker"
	const defName = "SBA Exact Lethal Defender"
	if !mage.CardRegistered(atkName) {
		mage.Register(atkName, func() mage.Card {
			return mage.NewCreature(atkName, "{1}{R}", 2, 2, mage.WithSubTypes("Goblin"))
		})
	}
	if !mage.CardRegistered(defName) {
		mage.Register(defName, func() mage.Card {
			return mage.NewCreature(defName, "{1}{W}", 2, 2, mage.WithSubTypes("Soldier"))
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, atkName)
	tg.AddCard(core.ZoneBattlefield, PlayerB, defName)
	tg.Attack(1, PlayerA, atkName)
	tg.Block(1, PlayerB, defName, atkName)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	// Both 2/2s deal exactly 2 damage — each has lethal — both die simultaneously
	tg.AssertPermanentCount(PlayerA, atkName, 0)
	tg.AssertGraveyardCount(PlayerA, atkName, 1)
	tg.AssertPermanentCount(PlayerB, defName, 0)
	tg.AssertGraveyardCount(PlayerB, defName, 1)
}

// TestCR704_5g_RegenerationSavesFromLethalDamage verifies that regeneration replaces destruction
// when a creature would be destroyed (CR 704.5g). Uses a destroy spell so the
// shield is active before resolution; combat damage timing is tested separately.
func TestCR704_5g_RegenerationSavesFromLethalDamage(t *testing.T) {
	const defName = "SBA Regen Troll Lethal"
	const killSpell = "SBA Regen Lethal Kill"
	if !mage.CardRegistered(defName) {
		mage.Register(defName, func() mage.Card {
			return mage.NewCreature(defName, "{2}{G}", 3, 3,
				mage.WithSubTypes("Troll"),
				mage.WithActivatedAbility(
					mage.RegenerateSource(),
					mage.ManaCostOf("{G}"),
				),
			)
		})
	}
	if !mage.CardRegistered(killSpell) {
		mage.Register(killSpell, func() mage.Card {
			return mage.NewInstant(killSpell, "{1}{B}",
				mage.NewTargetedSpell(mage.TargetCreature(), mage.DestroyTarget()),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, defName)
	tg.AddCard(core.ZoneHand, PlayerB, killSpell)
	// Activate regeneration to set up a shield, then cast destroy spell in same step
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, defName)
	tg.CastSpell(1, core.PrecombatMain, PlayerB, killSpell, defName)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	// Regeneration shield replaces destroy — troll survives
	tg.AssertPermanentCount(PlayerA, defName, 1)
}

// ============================================================
// 704.5h — Creature dealt any damage by a deathtouch source is destroyed; regeneration can replace.
// ============================================================

// TestCR704_5h_DeathtouchDestroys verifies that a creature dealt any damage by a deathtouch creature
// is destroyed (CR 704.5h).
func TestCR704_5h_DeathtouchDestroys(t *testing.T) {
	const atkName = "SBA Deathtouch Attacker"
	const defName = "SBA Deathtouch Victim"
	if !mage.CardRegistered(atkName) {
		mage.Register(atkName, func() mage.Card {
			return mage.NewCreature(atkName, "{1}{B}", 1, 1,
				mage.WithSubTypes("Snake"),
				mage.WithKeyword(core.Deathtouch),
			)
		})
	}
	if !mage.CardRegistered(defName) {
		mage.Register(defName, func() mage.Card {
			return mage.NewCreature(defName, "{3}{G}", 6, 6, mage.WithSubTypes("Wurm"))
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, atkName)
	tg.AddCard(core.ZoneBattlefield, PlayerB, defName)
	tg.Attack(1, PlayerA, atkName)
	tg.Block(1, PlayerB, defName, atkName)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	// 6/6 is dealt 1 damage by deathtouch 1/1 — destroyed by SBA 704.5h
	tg.AssertPermanentCount(PlayerB, defName, 0)
	tg.AssertGraveyardCount(PlayerB, defName, 1)
}

// TestCR704_5h_RegenerationSavesFromDeathtouch verifies that regeneration replaces deathtouch
// destruction (CR 704.5h). The deathtouch SBA fires as a destroy effect; the regen
// shield is pre-activated before blockers are declared so it is in place when damage
// is dealt and the SBA fires.
func TestCR704_5h_RegenerationSavesFromDeathtouch(t *testing.T) {
	const atkName = "SBA DT Regen Attacker"
	const defName = "SBA DT Regen Defender"
	if !mage.CardRegistered(atkName) {
		mage.Register(atkName, func() mage.Card {
			return mage.NewCreature(atkName, "{1}{B}", 1, 1,
				mage.WithSubTypes("Snake"),
				mage.WithKeyword(core.Deathtouch),
			)
		})
	}
	if !mage.CardRegistered(defName) {
		mage.Register(defName, func() mage.Card {
			return mage.NewCreature(defName, "{2}{G}", 3, 3,
				mage.WithSubTypes("Troll"),
				mage.WithActivatedAbility(
					mage.RegenerateSource(),
					mage.ManaCostOf("{G}"),
				),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, atkName)
	tg.AddCard(core.ZoneBattlefield, PlayerB, defName)
	// Activate regeneration during precombat main so the shield is in place before combat damage
	tg.ActivateAbility(1, core.PrecombatMain, PlayerB, defName)
	tg.Attack(1, PlayerA, atkName)
	tg.Block(1, PlayerB, defName, atkName)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	// Deathtouch would destroy the troll, but regeneration shield replaces that destruction
	tg.AssertPermanentCount(PlayerB, defName, 1)
}

// ============================================================
// 704.5j — Two legendary permanents with same name: controller chooses one to keep.
// ============================================================

// TestCR704_5j_LegendRulePlayerChooses verifies that when a player has two legendary creatures with the
// same name, the player chooses which one to keep (CR 704.5j).
func TestCR704_5j_LegendRulePlayerChooses(t *testing.T) {
	const legName = "SBA Legend Rule Chooser"
	if !mage.CardRegistered(legName) {
		mage.Register(legName, func() mage.Card {
			return mage.NewCreature(legName, "{2}{W}", 3, 3,
				mage.WithSuperTypes(core.SuperLegendary),
				mage.WithSubTypes("Human", "Knight"),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, legName)
	tg.AddCard(core.ZoneBattlefield, PlayerA, legName)
	// Player A has two copies; default ChoosePermanent picks the first candidate
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	// Only one copy remains — the legend rule fired and player chose
	tg.AssertPermanentCount(PlayerA, legName, 1)
	tg.AssertGraveyardCount(PlayerA, legName, 1)
}

// TestCR704_5j_LegendRuleDifferentControllers verifies that two players each controlling their own copy
// of a legendary permanent is legal — no legend rule fires (CR 704.5j).
func TestCR704_5j_LegendRuleDifferentControllers(t *testing.T) {
	const legName = "SBA Legend Rule Each Player"
	if !mage.CardRegistered(legName) {
		mage.Register(legName, func() mage.Card {
			return mage.NewCreature(legName, "{2}{W}", 3, 3,
				mage.WithSuperTypes(core.SuperLegendary),
				mage.WithSubTypes("Human", "Knight"),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, legName)
	tg.AddCard(core.ZoneBattlefield, PlayerB, legName)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	// Each player controls their own copy — legend rule checks per controller, not globally
	tg.AssertPermanentCount(PlayerA, legName, 1)
	tg.AssertPermanentCount(PlayerB, legName, 1)
}

// ============================================================
// 704.5m — Aura attached to illegal object goes to owner's graveyard.
// ============================================================

// TestCR704_5m_AuraFallsOffDestroyedCreature verifies that an Aura goes to its owner's graveyard when
// the enchanted creature dies (CR 704.5m).
// Note: TestAttachmentsOnSacrifice/TestAttachmentsOnExile already cover most attachment SBA;
// this confirms the basic combat-death path.
func TestCR704_5m_AuraFallsOffDestroyedCreature(t *testing.T) {
	const auraName = "SBA SBA Test Aura Boost"
	const crName = "SBA Aura Host Creature"
	const killName = "SBA Aura Host Killer"
	if !mage.CardRegistered(auraName) {
		mage.Register(auraName, func() mage.Card {
			return mage.NewBoostAura(auraName, "{1}{G}", 2, 2)
		})
	}
	if !mage.CardRegistered(crName) {
		mage.Register(crName, func() mage.Card {
			return mage.NewCreature(crName, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
		})
	}
	if !mage.CardRegistered(killName) {
		mage.Register(killName, func() mage.Card {
			return mage.NewInstant(killName, "{1}{B}",
				mage.NewTargetedSpell(mage.TargetCreature(), mage.DestroyTarget()),
			)
		})
	}

	tg := NewTestGame(t)
	crID := tg.AddCard(core.ZoneBattlefield, PlayerA, crName)
	auraID := tg.AddCard(core.ZoneBattlefield, PlayerA, auraName)
	tg.Attach(auraID, crID)
	tg.AddCard(core.ZoneHand, PlayerB, killName)
	tg.CastSpell(2, core.PrecombatMain, PlayerB, killName, crName)
	tg.StopAt(2, core.EndCombat)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, crName, 0)
	tg.AssertPermanentCount(PlayerA, auraName, 0)
	tg.AssertGraveyardCount(PlayerA, auraName, 1)
}

// TestCR704_5n_EquipmentDetachesOnCreatureDeath verifies that Equipment becomes unattached when its
// host creature dies, but the Equipment itself remains on the battlefield (CR 704.5n).
func TestCR704_5n_EquipmentDetachesOnCreatureDeath(t *testing.T) {
	const equipName = "SBA Equipment Detach Sword"
	const crName = "SBA Equipment Host Creature"
	const killName = "SBA Equipment Host Killer"
	if !mage.CardRegistered(equipName) {
		mage.Register(equipName, func() mage.Card {
			return mage.NewEquipment(equipName, "{2}",
				mage.WithAbility(mage.StaticAbility(mage.BoostAttached(2, 2, core.AttachEquipment))),
			)
		})
	}
	if !mage.CardRegistered(crName) {
		mage.Register(crName, func() mage.Card {
			return mage.NewCreature(crName, "{1}{G}", 2, 2, mage.WithSubTypes("Soldier"))
		})
	}
	if !mage.CardRegistered(killName) {
		mage.Register(killName, func() mage.Card {
			return mage.NewInstant(killName, "{1}{B}",
				mage.NewTargetedSpell(mage.TargetCreature(), mage.DestroyTarget()),
			)
		})
	}

	tg := NewTestGame(t)
	crID := tg.AddCard(core.ZoneBattlefield, PlayerA, crName)
	equipID := tg.AddCard(core.ZoneBattlefield, PlayerA, equipName)
	tg.Attach(equipID, crID)
	tg.AddCard(core.ZoneHand, PlayerB, killName)
	tg.CastSpell(2, core.PrecombatMain, PlayerB, killName, crName)
	tg.StopAt(2, core.EndCombat)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, crName, 0)
	tg.AssertPermanentCount(PlayerA, equipName, 1) // Equipment stays on battlefield
	equip := tg.FindPermanentByName(equipName, tg.GetPlayer(PlayerA).PlayerID())
	if equip.IsAttached() {
		t.Error("Equipment should be unattached after host creature died")
	}
}

// ============================================================
// 704.5q — +1/+1 and -1/-1 counters cancel each other N for N.
// ============================================================

// TestCR704_5q_CountersCancelOut verifies that equal numbers of +1/+1 and -1/-1 counters cancel each
// other out completely (CR 704.5q).
// Note: TestCounterAnnihilation in mechanics_test.go covers the 3 vs 2 partial case.
func TestCR704_5q_CountersCancelOut(t *testing.T) {
	const crName = "SBA Counter Cancel Creature"
	if !mage.CardRegistered(crName) {
		mage.Register(crName, func() mage.Card {
			return mage.NewCreature(crName, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
		})
	}

	tg := NewTestGame(t)
	id := tg.AddCard(core.ZoneBattlefield, PlayerA, crName)
	perm := tg.FindPermanent(id)
	perm.AddCounter(core.P1P1, 2)
	perm.AddCounter(core.M1M1, 2)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	tg.AssertCounterCount(PlayerA, crName, core.P1P1, 0)
	tg.AssertCounterCount(PlayerA, crName, core.M1M1, 0)
	tg.AssertPowerToughness(PlayerA, crName, 2, 2)
}

// TestCR704_5q_CountersCancelOutPartial verifies that when a permanent has more +1/+1 counters than
// -1/-1 counters, the -1/-1 counters are removed and one +1/+1 counter remains (CR 704.5q).
func TestCR704_5q_CountersCancelOutPartial(t *testing.T) {
	const crName = "SBA Counter Partial Cancel"
	if !mage.CardRegistered(crName) {
		mage.Register(crName, func() mage.Card {
			return mage.NewCreature(crName, "{1}{G}", 2, 2, mage.WithSubTypes("Wolf"))
		})
	}

	tg := NewTestGame(t)
	id := tg.AddCard(core.ZoneBattlefield, PlayerA, crName)
	perm := tg.FindPermanent(id)
	perm.AddCounter(core.P1P1, 3)
	perm.AddCounter(core.M1M1, 2)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	tg.AssertCounterCount(PlayerA, crName, core.P1P1, 1)
	tg.AssertCounterCount(PlayerA, crName, core.M1M1, 0)
	tg.AssertPowerToughness(PlayerA, crName, 3, 3) // 2/2 base + 1 net +1/+1
}

// ============================================================
// 704.3 — SBAs checked repeatedly until none apply.
// ============================================================

// TestCR704_3_SBARepeated verifies that multiple SBAs fire in the same SBA check pass — here two
// 0-toughness creatures both die simultaneously (CR 704.3).
func TestCR704_3_SBARepeated(t *testing.T) {
	const crA = "SBA Repeat Zero Tough A"
	const crB = "SBA Repeat Zero Tough B"
	for _, n := range []string{crA, crB} {
		n := n
		if !mage.CardRegistered(n) {
			mage.Register(n, func() mage.Card {
				return mage.NewCreature(n, "{G}", 1, 1, mage.WithSubTypes("Insect"))
			})
		}
	}

	tg := NewTestGame(t)
	idA := tg.AddCard(core.ZoneBattlefield, PlayerA, crA)
	idB := tg.AddCard(core.ZoneBattlefield, PlayerB, crB)
	permA := tg.FindPermanent(idA)
	permB := tg.FindPermanent(idB)
	permA.AddCounter(core.M1M1, 1)
	permB.AddCounter(core.M1M1, 1)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	// Both 1/1s had a -1/-1 counter — both become 0/0 and both die in the same SBA pass
	tg.AssertPermanentCount(PlayerA, crA, 0)
	tg.AssertGraveyardCount(PlayerA, crA, 1)
	tg.AssertPermanentCount(PlayerB, crB, 0)
	tg.AssertGraveyardCount(PlayerB, crB, 1)
}
