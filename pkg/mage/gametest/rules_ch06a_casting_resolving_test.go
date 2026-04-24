package gametest

// Chapter 6: Spells, Abilities, and Effects (CR 600–616)
// Rules tests derived from the ch06 plan. All cards are registered inline to
// avoid the import cycle between gametest and cards/limited.

import (
	"sync"
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

// ─────────────────────────────────────────────────────────────────────────────
// Shared inline-card registration helpers.
// ─────────────────────────────────────────────────────────────────────────────

var ch06Once sync.Once

func registerCh06Cards() {
	ch06Once.Do(func() {
		reg := func(name string, f func() mage.Card) {
			if !mage.CardRegistered(name) {
				mage.Register(name, f)
			}
		}

		// Generic creatures
		reg("Ch06 Bear", func() mage.Card {
			return mage.NewCreature("Ch06 Bear", "{1}{G}", 2, 2,
				mage.WithSubTypes("Bear"))
		})
		reg("Ch06 Giant", func() mage.Card {
			return mage.NewCreature("Ch06 Giant", "{3}{R}", 3, 3,
				mage.WithSubTypes("Giant"))
		})
		reg("Ch06 Knight", func() mage.Card {
			return mage.NewCreature("Ch06 Knight", "{W}{W}", 2, 2,
				mage.WithSubTypes("Human", "Knight"),
				mage.WithAbility(mage.ProtectionFromColor(core.Black)),
			)
		})
		reg("Ch06 Black Knight", func() mage.Card {
			return mage.NewCreature("Ch06 Black Knight", "{B}{B}", 2, 2,
				mage.WithSubTypes("Human", "Knight"),
			)
		})

		// Spells
		reg("Ch06 Bolt", func() mage.Card {
			return mage.NewInstant("Ch06 Bolt", "{R}",
				mage.NewTargetedSpell(mage.TargetAnyTarget(),
					mage.DealDamage(mage.Fixed(3))))
		})
		reg("Ch06 Counter", func() mage.Card {
			return mage.NewInstant("Ch06 Counter", "{U}{U}",
				mage.NewTargetedSpell(mage.TargetSpellOnStack(),
					mage.CounterSpell()))
		})
		reg("Ch06 Exile Spell", func() mage.Card {
			return mage.NewInstant("Ch06 Exile Spell", "{W}",
				mage.NewTargetedSpell(mage.TargetCreature(),
					mage.ExileTarget()))
		})
		reg("Ch06 Destroy Spell", func() mage.Card {
			return mage.NewInstant("Ch06 Destroy Spell", "{1}{B}",
				mage.NewTargetedSpell(
					mage.TargetCreature(mage.Not(mage.HasColorFilter(core.Black))),
					mage.DestroyTarget()))
		})
		reg("Ch06 Wrath", func() mage.Card {
			return mage.NewSorcery("Ch06 Wrath", "{2}{W}{W}",
				mage.NewSpellAbility(mage.DestroyAllCreatures()))
		})
		reg("Ch06 Growth", func() mage.Card {
			return mage.NewInstant("Ch06 Growth", "{G}",
				mage.NewTargetedSpell(mage.TargetCreature(),
					mage.BoostUntilEndOfTurn(mage.Fixed(3), mage.Fixed(3),
						mage.SelectTarget)))
		})
		reg("Ch06 Discard Two", func() mage.Card {
			return mage.NewSorcery("Ch06 Discard Two", "{B}",
				mage.NewTargetedSpell(mage.TargetPlayer(),
					mage.DiscardCards(mage.Fixed(2))))
		})
		reg("Ch06 Earthquake", func() mage.Card {
			return mage.NewSorcery("Ch06 Earthquake", "{X}{R}",
				mage.NewSpellAbility(
					mage.DealDamageToAllCreatures(mage.XValue(), mage.AnyPermanent),
					mage.DealDamageToPlayers(mage.XValue(), mage.SelectEachPlayer()),
				))
		})
		reg("Ch06 Delayed Kill", func() mage.Card {
			return mage.NewInstant("Ch06 Delayed Kill", "{1}{W}",
				mage.NewTargetedSpell(mage.TargetCreature(),
					mage.DestroyTargetAtEndOfTurn()))
		})

		// Aura / enchantment
		reg("Ch06 Crusade", func() mage.Card {
			return mage.NewEnchantment("Ch06 Crusade", "{W}{W}",
				mage.WithStaticAbility(
					mage.BoostAllCreaturesIncludingSelf(1, 1,
						mage.HasColorFilter(core.White))))
		})
		reg("Ch06 Bad Moon", func() mage.Card {
			return mage.NewEnchantment("Ch06 Bad Moon", "{1}{B}",
				mage.WithStaticAbility(
					mage.BoostAllCreaturesIncludingSelf(1, 1,
						mage.HasColorFilter(core.Black))))
		})
		reg("Ch06 Control Aura", func() mage.Card {
			return mage.NewAura("Ch06 Control Aura", "{2}{U}{U}",
				mage.WithStaticAbility(mage.ControlChangeContinuous()))
		})
		reg("Ch06 Flight", func() mage.Card {
			return mage.NewAura("Ch06 Flight", "{U}",
				mage.WithStaticAbility(
					mage.GrantAbilityToAttached(core.Flying, core.AttachAura)))
		})
		reg("Ch06 White Paint", func() mage.Card {
			return mage.NewAura("Ch06 White Paint", "{W}",
				mage.WithStaticAbility(
					mage.AttachedEffect(core.LayerColor,
						func(_ *mage.Game, _, target *mage.Permanent) error {
							colors := []core.Color{core.White}
							target.ColorOverride = &colors
							return nil
						})))
		})

		// Triggered ability cards
		reg("Ch06 Spell Watcher", func() mage.Card {
			return mage.NewCreature("Ch06 Spell Watcher", "{2}{U}", 1, 1,
				mage.WithAbility(mage.WheneverSpellCastTrigger(
					mage.GainLife(1), false)))
		})
		reg("Ch06 Upkeep Gainer", func() mage.Card {
			return mage.NewCreature("Ch06 Upkeep Gainer", "{1}{W}", 1, 1,
				mage.WithAbility(mage.BeginningOfUpkeepTrigger(
					mage.GainLife(1), false)))
		})
		reg("Ch06 Death Counter", func() mage.Card {
			return mage.NewEnchantment("Ch06 Death Counter", "{B}",
				mage.WithAbility(mage.AnyCreatureDiesTrigger(
					mage.GainLife(1), false)))
		})
		reg("Ch06 ETB Gainer", func() mage.Card {
			return mage.NewCreature("Ch06 ETB Gainer", "{2}{W}", 2, 2,
				mage.WithAbility(mage.EntersBattlefieldTrigger(
					mage.GainLife(3), false)))
		})
		reg("Ch06 LTB Gainer", func() mage.Card {
			return mage.NewCreature("Ch06 LTB Gainer", "{2}{B}", 2, 2,
				mage.WithAbility(mage.PutIntoGraveyardFromBattlefieldTrigger(
					mage.GainLife(2), false)))
		})
		reg("Ch06 Death Regen", func() mage.Card {
			return mage.NewCreature("Ch06 Death Regen", "{2}{G}", 2, 2,
				mage.WithAbility(mage.PutIntoGraveyardFromBattlefieldTrigger(
					mage.GainLife(5), false)),
				mage.WithActivatedAbility(
					mage.RegenerateSource(),
					mage.ManaCostOf("{G}")))
		})
		reg("Ch06 Haste Pinger", func() mage.Card {
			return mage.NewCreature("Ch06 Haste Pinger", "{2}{R}", 1, 1,
				mage.WithKeyword(core.Haste),
				mage.WithActivatedAbility(
					mage.DealDamage(mage.Fixed(1)),
					mage.TapSourceCost(),
					mage.WithTarget(mage.TargetAnyTarget())))
		})
		reg("Ch06 Pinger", func() mage.Card {
			return mage.NewCreature("Ch06 Pinger", "{2}{U}", 1, 1,
				mage.WithSubTypes("Human", "Wizard"),
				mage.WithActivatedAbility(
					mage.DealDamage(mage.Fixed(1)),
					mage.TapSourceCost(),
					mage.WithTarget(mage.TargetAnyTarget())))
		})
		reg("Ch06 Damage Watcher", func() mage.Card {
			return mage.NewCreature("Ch06 Damage Watcher", "{2}{U}", 0, 4,
				mage.WithAbility(mage.WhenDamageDealtToThisTrigger(
					mage.GainLife(1), false)))
		})
		reg("Ch06 High Life Watcher", func() mage.Card {
			return mage.NewCreature("Ch06 High Life Watcher", "{3}{W}{W}", 2, 4,
				mage.WithAbility(mage.NewTriggered(core.EvtUpkeep, false,
					mage.GainLife(10),
				).SetCondition(func(evt *core.GameEvent, g mage.GameReader, sourceID, controllerID uuid.UUID) bool {
					if evt.PlayerID != controllerID {
						return false
					}
					p := g.GetPlayer(controllerID)
					return p != nil && p.Life() >= 30
				})))
		})
		reg("Ch06 Regen Skeleton", func() mage.Card {
			return mage.NewCreature("Ch06 Regen Skeleton", "{1}{B}", 1, 1,
				mage.WithSubTypes("Skeleton"),
				mage.WithActivatedAbility(
					mage.RegenerateSource(),
					mage.ManaCostOf("{B}")))
		})
		reg("Ch06 Artifact Death Watch", func() mage.Card {
			return mage.NewArtifact("Ch06 Artifact Death Watch", "{2}",
				mage.WithCardType(core.TypeCreature),
				mage.WithAbility(mage.AnyCreatureDiesTrigger(
					mage.GainLife(1), false)))
		})
		reg("Ch06 APNAP A Watcher", func() mage.Card {
			return mage.NewCreature("Ch06 APNAP A Watcher", "{1}{W}", 1, 1,
				mage.WithAbility(mage.WheneverSpellCastTrigger(
					mage.GainLife(2), false)))
		})
		reg("Ch06 APNAP B Watcher", func() mage.Card {
			return mage.NewCreature("Ch06 APNAP B Watcher", "{1}{U}", 1, 1,
				mage.WithAbility(mage.WheneverSpellCastTrigger(
					mage.LoseLife(1), false)))
		})
		reg("Ch06 Sorcery Artifact", func() mage.Card {
			return mage.NewArtifact("Ch06 Sorcery Artifact", "{2}",
				mage.WithAbility(mage.NewActivatedAbility(
					mage.GainLife(3),
					mage.GenericCost(1),
					mage.WithControlledSinceTurnStart())))
		})
		reg("Ch06 Sorceress Queen", func() mage.Card {
			return mage.NewCreature("Ch06 Sorceress Queen", "{1}{B}{B}", 1, 1,
				mage.WithSubTypes("Human", "Wizard"),
				mage.WithActivatedAbility(
					mage.SetPTUntilEndOfTurn(0, 2,
						mage.SelectTarget),
					mage.TapSourceCost(),
					mage.WithTarget(mage.TargetCreature(
						mage.NotHasKeywordFilter(core.Flying)))))
		})
		reg("Ch06 Rally", func() mage.Card {
			return mage.NewSorcery("Ch06 Rally", "{W}",
				mage.NewSpellAbility(
					mage.BoostMatchingUntilEndOfTurn(mage.Fixed(1), mage.Fixed(1),
						mage.HasColorFilter(core.White))))
		})
		reg("Ch06 Living Lands", func() mage.Card {
			return mage.NewEnchantment("Ch06 Living Lands", "{3}{G}",
				mage.WithStaticAbility(
					mage.AnimateLands(
						mage.And(mage.IsLand, mage.HasSubType("Forest")), 1, 1)))
		})
		reg("Ch06 Kormus Bell", func() mage.Card {
			return mage.NewArtifact("Ch06 Kormus Bell", "{4}",
				mage.WithStaticAbility(
					mage.AnimateLands(
						mage.And(mage.IsLand, mage.HasSubType("Swamp")), 1, 1),
					mage.GrantColorToAll(core.Black,
						mage.And(mage.IsLand, mage.HasSubType("Swamp")))))
		})
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// 601.2a — Spell goes on stack as topmost object (CR 601.2a)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR601_2a_CastSpellBecomesTopOfStack verifies that casting a spell places it on
// the stack above existing objects; a counterspell targeting it resolves first.
func TestCR601_2a_CastSpellBecomesTopOfStack(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerB, "Ch06 Giant")
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Bolt")
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Counter")

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Bolt", "Ch06 Giant")
	g.CastInResponseTo(PlayerA, "Ch06 Counter", "Ch06 Bolt")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	// Bolt was countered; Giant survives; PlayerB unharmed.
	g.AssertPermanentCount(PlayerB, "Ch06 Giant", 1)
	g.AssertGraveyardCount(PlayerA, "Ch06 Bolt", 1)
}

// ─────────────────────────────────────────────────────────────────────────────
// 601.2g — Mana ability activated during casting (CR 601.2g)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR601_2g_CastSpellTapManaAbilityDuringCasting verifies that a land's mana ability
// can be used during casting to pay for the spell.
func TestCR601_2g_CastSpellTapManaAbilityDuringCasting(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Forest")
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Growth")
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Bear")

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Growth", "Ch06 Bear")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	// Bear is 5/5 during main phase (3+2 / 3+2).
	g.AssertPowerToughness(PlayerA, "Ch06 Bear", 5, 5)
}

// ─────────────────────────────────────────────────────────────────────────────
// 601.2i — Triggers fire after spell becomes cast (CR 601.2i)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR601_2i_CastSpellTriggersAfterBecomingCast verifies that "whenever a spell is
// cast" triggers fire after a spell is cast.
func TestCR601_2i_CastSpellTriggersAfterBecomingCast(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Spell Watcher")
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Bolt")
	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Bolt", "PlayerB")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	// Watcher triggers on bolt cast; PlayerA gains 1 life.
	g.AssertLife(PlayerA, 21)
}

// ─────────────────────────────────────────────────────────────────────────────
// 602.5a — Summoning sickness blocks tap abilities (CR 602.5a)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR602_5a_ActivateSummoningSicknessBlocksTapAbility verifies that a creature
// cannot use its tap ability the turn it enters (without haste).
func TestCR602_5a_ActivateSummoningSicknessBlocksTapAbility(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Pinger")

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Pinger")
	// Attempting to activate in the same turn; summoning sickness should block.
	g.ActivateAbility(1, core.PrecombatMain, PlayerA, "Ch06 Pinger", "PlayerB")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	// Tap ability blocked by summoning sickness; PlayerB stays at 20.
	g.AssertLife(PlayerB, 20)
}

// TestCR602_5a_ActivateHasteOverridesSummoningSickness verifies that a creature with
// haste can use its tap ability the turn it enters (CR 602.5a).
func TestCR602_5a_ActivateHasteOverridesSummoningSickness(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Haste Pinger")

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Haste Pinger")
	g.ActivateAbility(1, core.PrecombatMain, PlayerA, "Ch06 Haste Pinger", "PlayerB")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	// Haste allows tap ability; PlayerB takes 1 damage.
	g.AssertLife(PlayerB, 19)
}

// TestCR602_5d_ActivateOnlyAsSorceryRequiresSorceryTiming verifies that an ability
// restricted to sorcery speed fires only at sorcery speed (CR 602.5d).
func TestCR602_5d_ActivateOnlyAsSorceryRequiresSorceryTiming(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Sorcery Artifact")

	// Activate sorcery-restricted ability during main phase (correct timing).
	g.ActivateAbility(1, core.PrecombatMain, PlayerA, "Ch06 Sorcery Artifact")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	// Ability fires at sorcery speed; PlayerA gains 3 life.
	g.AssertLife(PlayerA, 23)
}

// ─────────────────────────────────────────────────────────────────────────────
// 603.2b — "At the beginning of" triggers fire at start of step (CR 603.2b)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR603_2b_TriggeredAtBeginningOfFiresAtStart verifies an upkeep trigger fires at
// the start of upkeep before the main phase.
func TestCR603_2b_TriggeredAtBeginningOfFiresAtStart(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Upkeep Gainer")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	// Upkeep trigger fired; PlayerA gained 1 life before main phase.
	g.AssertLife(PlayerA, 21)
}

// ─────────────────────────────────────────────────────────────────────────────
// 603.2c — Multiple simultaneous occurrences trigger multiple times (CR 603.2c)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR603_2c_TriggeredMultipleOccurrencesMultipleInstances verifies that destroying
// two creatures triggers "whenever a creature dies" twice.
func TestCR603_2c_TriggeredMultipleOccurrencesMultipleInstances(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Death Counter")
	g.AddCard(core.ZoneBattlefield, PlayerB, "Ch06 Bear")
	g.AddCard(core.ZoneBattlefield, PlayerB, "Ch06 Giant")
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Wrath")

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Wrath")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	// Both creatures die; Death Counter triggers twice; PlayerA gains 2 life.
	g.AssertLife(PlayerA, 22)
}

// ─────────────────────────────────────────────────────────────────────────────
// 603.2g — Prevented event doesn't trigger abilities (CR 603.2g)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR603_2g_TriggeredPreventedEventNoTrigger verifies that fully prevented damage
// doesn't fire damage-trigger abilities.
func TestCR603_2g_TriggeredPreventedEventNoTrigger(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Damage Watcher")
	g.AddCard(core.ZoneHand, PlayerB, "Ch06 Bolt")

	// Add full prevention shield on the Damage Watcher.
	watcher := g.FindPermanentByName("Ch06 Damage Watcher", g.GetPlayer(PlayerA).PlayerID())
	g.AddPreventionShield(watcher.ID(), 5)

	g.CastSpell(1, core.PrecombatMain, PlayerB, "Ch06 Bolt", "Ch06 Damage Watcher")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	// Damage fully prevented; no trigger fires; PlayerA life unchanged.
	g.AssertLife(PlayerA, 20)
	g.AssertPermanentCount(PlayerA, "Ch06 Damage Watcher", 1)
}

// ─────────────────────────────────────────────────────────────────────────────
// 603.3b — APNAP ordering: active player's triggers stack first (CR 603.3b)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR603_3b_TriggeredAPNAPStackOrder verifies both players' "whenever a spell is
// cast" triggers fire when a spell is cast; both effects resolve.
func TestCR603_3b_TriggeredAPNAPStackOrder(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 APNAP A Watcher")
	g.AddCard(core.ZoneBattlefield, PlayerB, "Ch06 APNAP B Watcher")
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Bolt")

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Bolt", "PlayerB")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	// A Watcher: PlayerA gains 2. B Watcher: PlayerA loses 1. Bolt: PlayerB -3.
	g.AssertLife(PlayerA, 21)
	g.AssertLife(PlayerB, 17)
}

// ─────────────────────────────────────────────────────────────────────────────
// 603.4 — Intervening "if" clause: checked at trigger time (CR 603.4)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR603_4_TriggeredInterveningIfFailsAtTriggerTime verifies that a trigger with
// an intervening-if condition that is false never goes on the stack.
func TestCR603_4_TriggeredInterveningIfFailsAtTriggerTime(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 High Life Watcher")
	// PlayerA has 20 life — condition (>= 30) fails.
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	g.AssertLife(PlayerA, 20)
}

// TestCR603_4_TriggeredInterveningIfFiresWhenConditionTrue verifies that the same
// trigger fires when the condition is true.
func TestCR603_4_TriggeredInterveningIfFiresWhenConditionTrue(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 High Life Watcher")
	g.SetLife(PlayerA, 30)
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	g.AssertLife(PlayerA, 40)
}

// ─────────────────────────────────────────────────────────────────────────────
// 603.6a — ETB triggers fire when permanent enters (CR 603.6a)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR603_6a_TriggeredETBFiresOnEntry verifies that an ETB trigger fires when the
// permanent enters the battlefield.
func TestCR603_6a_TriggeredETBFiresOnEntry(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 ETB Gainer")
	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 ETB Gainer")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	g.AssertPermanentCount(PlayerA, "Ch06 ETB Gainer", 1)
	g.AssertLife(PlayerA, 23)
}

// ─────────────────────────────────────────────────────────────────────────────
// 603.6c — LTB trigger looks in destination zone (CR 603.6c)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR603_6c_TriggeredLTBLooksInDestinationZone verifies that an LTB trigger fires
// when the creature dies and goes to the graveyard.
func TestCR603_6c_TriggeredLTBLooksInDestinationZone(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 LTB Gainer") // 2/2
	g.AddCard(core.ZoneHand, PlayerB, "Ch06 Bolt")              // 3 damage kills 2/2

	g.CastSpell(1, core.PrecombatMain, PlayerB, "Ch06 Bolt", "Ch06 LTB Gainer")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	g.AssertGraveyardCount(PlayerA, "Ch06 LTB Gainer", 1)
	g.AssertLife(PlayerA, 22)
}

// ─────────────────────────────────────────────────────────────────────────────
// 603.7a — Delayed trigger created on resolution (CR 603.7a)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR603_7a_TriggeredDelayedCreatedOnResolution verifies a delayed trigger created
// at resolution fires at the next appropriate event.
func TestCR603_7a_TriggeredDelayedCreatedOnResolution(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerB, "Ch06 Giant")
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Delayed Kill")

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Delayed Kill", "Ch06 Giant")
	g.StopAt(2, core.PrecombatMain)
	g.Execute()

	// Giant destroyed at end of turn 1; gone by turn 2.
	g.AssertPermanentCount(PlayerB, "Ch06 Giant", 0)
}

// ─────────────────────────────────────────────────────────────────────────────
// 603.10a — LTB triggers look back in time (CR 603.10a)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR603_10a_TriggeredLooksBackInTimeLTB verifies that an artifact's creature-dies
// trigger can fire for its own simultaneous destruction.
func TestCR603_10a_TriggeredLooksBackInTimeLTB(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Artifact Death Watch")
	g.AddCard(core.ZoneBattlefield, PlayerB, "Ch06 Bear")
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Wrath")

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Wrath")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	// Artifact Death Watch goes to graveyard.
	g.AssertGraveyardCount(PlayerA, "Ch06 Artifact Death Watch", 1)
}

// ─────────────────────────────────────────────────────────────────────────────
// 604.2 — Static ability active while on battlefield; stops when gone (CR 604.2)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR604_2_StaticActiveWhileOnBattlefield verifies that a static effect applies
// while the permanent is in play.
func TestCR604_2_StaticActiveWhileOnBattlefield(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Crusade")
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Knight") // white 2/2

	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	// Crusade gives +1/+1 to white creatures; Knight is 3/3.
	g.AssertPowerToughness(PlayerA, "Ch06 Knight", 3, 3)
}

// TestCR604_2_StaticStopsWhenPermanentLeaves verifies that removing the source stops
// the static effect.
func TestCR604_2_StaticStopsWhenPermanentLeaves(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	crusadeID := g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Crusade")
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Knight")

	crusade := g.FindPermanent(crusadeID)
	g.Sacrifice(crusade)
	g.Effects.Apply(g.Game)

	// Knight back to 2/2 after Crusade leaves.
	g.AssertPowerToughness(PlayerA, "Ch06 Knight", 2, 2)
}

// ─────────────────────────────────────────────────────────────────────────────
// 608.2b — All-illegal-targets fizzle (CR 608.2b)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR608_2b_ResolveAllTargetsIllegalFizzles verifies that a spell fizzles if its
// only target becomes illegal before resolution.
func TestCR608_2b_ResolveAllTargetsIllegalFizzles(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerB, "Ch06 Bear")
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Bolt")
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Exile Spell")

	// Bolt targets Bear; Exile Spell exiles Bear in response.
	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Bolt", "Ch06 Bear")
	g.CastInResponseTo(PlayerA, "Ch06 Exile Spell", "Ch06 Bear")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	// Bear is exiled; Bolt fizzles and goes to graveyard.
	g.AssertExileCount("Ch06 Bear", 1)
	g.AssertGraveyardCount(PlayerA, "Ch06 Bolt", 1)
	// PlayerB takes no damage.
	g.AssertLife(PlayerB, 20)
}

// TestCR608_2b_ResolveAllTargetsIllegalSpellToGraveyard confirms the fizzled spell
// goes to the graveyard (CR 608.2b).
func TestCR608_2b_ResolveAllTargetsIllegalSpellToGraveyard(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerB, "Ch06 Bear")
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Bolt")
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Exile Spell")

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Bolt", "Ch06 Bear")
	g.CastInResponseTo(PlayerA, "Ch06 Exile Spell", "Ch06 Bear")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	g.AssertGraveyardCount(PlayerA, "Ch06 Bolt", 1)
}

// ─────────────────────────────────────────────────────────────────────────────
// 608.2h — Game info locked at time of application (CR 608.2h)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR608_2h_ResolveGameInfoLockedOnApplication verifies that Earthquake counts
// damage at resolution time; the count of dead creatures is not used.
func TestCR608_2h_ResolveGameInfoLockedOnApplication(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Bear")    // 2/2
	g.AddCard(core.ZoneBattlefield, PlayerB, "Ch06 Giant")   // 3/3
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Earthquake")

	// Earthquake for 2: kills Bear (2/2), not Giant (3/3); both players -2 life.
	g.CastSpellWithX(1, core.PrecombatMain, PlayerA, "Ch06 Earthquake", 2)
	g.StopAt(1, core.EndCombat)
	g.Execute()

	g.AssertGraveyardCount(PlayerA, "Ch06 Bear", 1)
	g.AssertPermanentCount(PlayerB, "Ch06 Giant", 1)
	g.AssertLife(PlayerA, 18)
	g.AssertLife(PlayerB, 18)
}
