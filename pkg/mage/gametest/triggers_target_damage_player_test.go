package gametest

// Engine tests for "becomes the target of a spell or ability" triggers
// (CR 603.6c, 119.5) and "deals combat damage / damage to a player" triggers.

import (
	"sync"
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

var triggerTargetDamagePlayerOnce sync.Once

func registerTriggerTargetDamagePlayerCards() {
	triggerTargetDamagePlayerOnce.Do(func() {
		reg := func(name string, f func() mage.Card) {
			if !mage.CardRegistered(name) {
				mage.Register(name, f)
			}
		}

		// "When this creature becomes the target of a spell or ability,
		// sacrifice it." (Departed Deckhand pattern.)
		reg("Trig Target Sac", func() mage.Card {
			trig := mage.WheneverBecomesTargetTrigger(mage.SacrificeSource(), false)
			return mage.NewCreature("Trig Target Sac", "{1}{U}", 2, 1,
				mage.WithSubTypes("Spirit"),
				mage.WithAbility(trig))
		})

		// "When this creature becomes the target of a spell or ability for
		// the first time each turn, you gain 1 life." (Kira-style, simplified
		// effect for testability.)
		reg("Trig Target First", func() mage.Card {
			trig := mage.WheneverBecomesTargetFirstTimeEachTurnTrigger(
				mage.GainLife(1), false)
			return mage.NewCreature("Trig Target First", "{1}{U}", 2, 2,
				mage.WithAbility(trig))
		})

		// Source spell: tap target creature. Exists purely to fire
		// becomes-target on its target.
		reg("Trig Tap Target", func() mage.Card {
			return mage.NewSorcery("Trig Tap Target", "{1}",
				mage.NewTargetedSpell(mage.TargetCreature(),
					mage.Tap()))
		})

		// Two-creature damage spell (multi-target). For becomes-target firing
		// once-per-target verification: deal 1 damage to each of two target
		// creatures. We use two single-target casts in tests rather than a
		// true multi-target spell to keep the harness simple, since the
		// engine fires one EvtBecomesTarget per target on multi-target
		// stack objects (already covered by event-firing logic).
		reg("Trig Pin", func() mage.Card {
			return mage.NewSorcery("Trig Pin", "{R}",
				mage.NewTargetedSpell(mage.TargetCreature(),
					mage.DealDamage(mage.Fixed(0))))
		})

		// "Whenever a creature you control deals combat damage to a player,
		// you gain 1 life." (Coastal Piracy pattern, simplified to GainLife.)
		reg("Trig Combat Watcher", func() mage.Card {
			return mage.NewEnchantment("Trig Combat Watcher", "{1}{U}",
				mage.WithAbility(mage.WheneverPermanentDealsCombatDamageToPlayerTrigger(
					mage.GainLife(1), false, mage.IsCreature)))
		})

		// "Whenever an artifact creature you control deals combat damage to
		// a player, you gain 2 life." (Sharding Sphinx pattern.)
		reg("Trig Artifact Combat Watcher", func() mage.Card {
			filter := mage.And(mage.IsArtifact, mage.IsCreature)
			return mage.NewEnchantment("Trig Artifact Combat Watcher", "{2}{U}",
				mage.WithAbility(mage.WheneverPermanentDealsCombatDamageToPlayerTrigger(
					mage.GainLife(2), false, filter)))
		})

		// "When enchanted creature deals damage to a player, you gain 1 life."
		// (Curiosity pattern, with GainLife instead of DrawCards for assertion
		// simplicity.)
		reg("Trig Curiosity-Like", func() mage.Card {
			return mage.NewAura("Trig Curiosity-Like", "{U}",
				mage.WithAbility(mage.WheneverEnchantedPermanentDealsDamageToPlayerTrigger(
					mage.GainLife(1), false)))
		})

		// Vanilla non-artifact attacker.
		reg("Trig Vanilla 2/2", func() mage.Card {
			return mage.NewCreature("Trig Vanilla 2/2", "{1}{G}", 2, 2)
		})

		// Vanilla artifact creature attacker.
		reg("Trig Artifact 2/2", func() mage.Card {
			return mage.NewCreature("Trig Artifact 2/2", "{3}", 2, 2,
				mage.WithCardType(core.TypeArtifact))
		})
	})
}

// TestBecomesTarget_FiresOnSpellTarget: spell that targets a creature with a
// becomes-target sacrifice trigger causes the creature to be sacrificed.
func TestBecomesTarget_FiresOnSpellTarget(t *testing.T) {
	registerTriggerTargetDamagePlayerCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Trig Target Sac")
	g.AddCard(core.ZoneHand, PlayerB, "Trig Tap Target")
	g.AddCard(core.ZoneLibrary, PlayerB, "Forest", 5)

	// PlayerB casts on turn 2 (their first main).
	g.CastSpell(2, core.PrecombatMain, PlayerB, "Trig Tap Target", "Trig Target Sac")
	g.StopAt(2, core.PostcombatMain)
	g.Execute()

	g.AssertPermanentCount(PlayerA, "Trig Target Sac", 0)
	g.AssertGraveyardCount(PlayerA, "Trig Target Sac", 1)
}

// TestBecomesTarget_FirstTimeEachTurn: "first time each turn" trigger fires
// once per turn, even if the creature is targeted multiple times.
func TestBecomesTarget_FirstTimeEachTurn(t *testing.T) {
	registerTriggerTargetDamagePlayerCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Trig Target First")
	g.AddCard(core.ZoneHand, PlayerB, "Trig Tap Target", 2)
	g.AddCard(core.ZoneLibrary, PlayerB, "Forest", 10)

	g.CastSpell(2, core.PrecombatMain, PlayerB, "Trig Tap Target", "Trig Target First")
	g.CastSpell(2, core.PrecombatMain, PlayerB, "Trig Tap Target", "Trig Target First")
	g.StopAt(2, core.PostcombatMain)
	g.Execute()

	// Only the first targeting should fire the trigger: 20 + 1 = 21.
	g.AssertLife(PlayerA, 21)
}

// TestBecomesTarget_FirstTimeResetsBetweenTurns: "first time each turn" trigger
// fires again the next turn.
func TestBecomesTarget_FirstTimeResetsBetweenTurns(t *testing.T) {
	registerTriggerTargetDamagePlayerCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Trig Target First")
	g.AddCard(core.ZoneHand, PlayerB, "Trig Tap Target", 2)
	g.AddCard(core.ZoneLibrary, PlayerB, "Forest", 10)

	g.CastSpell(2, core.PrecombatMain, PlayerB, "Trig Tap Target", "Trig Target First")
	g.CastSpell(4, core.PrecombatMain, PlayerB, "Trig Tap Target", "Trig Target First")
	g.StopAt(4, core.PostcombatMain)
	g.Execute()

	// Two distinct turns, each with a first targeting: 20 + 1 + 1 = 22.
	g.AssertLife(PlayerA, 22)
}

// TestBecomesTarget_DoesNotFireWithoutTarget: a non-targeted spell should not
// trigger becomes-target.
func TestBecomesTarget_DoesNotFireWithoutTarget(t *testing.T) {
	registerTriggerTargetDamagePlayerCards()

	registerTriggerLifeDiscardSacrificeCards() // for Trig Healing Salve
	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Trig Target Sac")
	g.AddCard(core.ZoneHand, PlayerB, "Trig Healing Salve")
	g.AddCard(core.ZoneLibrary, PlayerB, "Plains", 5)

	g.CastSpell(2, core.PrecombatMain, PlayerB, "Trig Healing Salve")
	g.StopAt(2, core.PostcombatMain)
	g.Execute()

	g.AssertPermanentCount(PlayerA, "Trig Target Sac", 1)
}

// TestWheneverPermanentDealsCombatDamageToPlayer_FiresOnUnblocked: vanilla
// creature attacking unblocked triggers the watcher enchantment.
func TestWheneverPermanentDealsCombatDamageToPlayer_FiresOnUnblocked(t *testing.T) {
	registerTriggerTargetDamagePlayerCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Trig Combat Watcher")
	g.AddCard(core.ZoneBattlefield, PlayerA, "Trig Vanilla 2/2")

	g.Attack(3, PlayerA, "Trig Vanilla 2/2")
	g.StopAt(3, core.PostcombatMain)
	g.Execute()

	g.AssertLife(PlayerB, 18) // 20 - 2
	g.AssertLife(PlayerA, 21) // 20 + 1 from trigger
}

// TestWheneverArtifactCreatureDealsCombatDamage_FiresOnlyForArtifact: a vanilla
// non-artifact creature should NOT fire the artifact-only filter trigger.
func TestWheneverArtifactCreatureDealsCombatDamage_FiresOnlyForArtifact(t *testing.T) {
	registerTriggerTargetDamagePlayerCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Trig Artifact Combat Watcher")
	g.AddCard(core.ZoneBattlefield, PlayerA, "Trig Vanilla 2/2")
	g.AddCard(core.ZoneBattlefield, PlayerA, "Trig Artifact 2/2")

	g.Attack(3, PlayerA, "Trig Vanilla 2/2", "Trig Artifact 2/2")
	g.StopAt(3, core.PostcombatMain)
	g.Execute()

	g.AssertLife(PlayerB, 16) // 20 - 4
	// Only the artifact creature fires the trigger: 20 + 2 = 22.
	g.AssertLife(PlayerA, 22)
}

// TestWheneverEnchantedPermanentDealsDamageToPlayer: aura-on-creature trigger
// fires when the enchanted creature deals combat damage to a player.
func TestWheneverEnchantedPermanentDealsDamageToPlayer(t *testing.T) {
	registerTriggerTargetDamagePlayerCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Trig Vanilla 2/2")
	g.AddCard(core.ZoneHand, PlayerA, "Trig Curiosity-Like")
	g.AddCard(core.ZoneLibrary, PlayerA, "Island", 5)

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Trig Curiosity-Like", "Trig Vanilla 2/2")
	g.Attack(3, PlayerA, "Trig Vanilla 2/2")
	g.StopAt(3, core.PostcombatMain)
	g.Execute()

	g.AssertLife(PlayerB, 18) // 20 - 2
	g.AssertLife(PlayerA, 21) // 20 + 1 from aura trigger
}
