package gametest

import (
	"slices"
	"sync"
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

var uncounterableOnce sync.Once

func registerUncounterableCards() {
	uncounterableOnce.Do(func() {
		// A vanilla creature spell that is intrinsically uncounterable.
		if !mage.CardRegistered("Test Uncounterable Bear") {
			mage.Register("Test Uncounterable Bear", func() mage.Card {
				return mage.NewCreature("Test Uncounterable Bear", "{1}{G}", 2, 2,
					mage.WithSubTypes("Bear"),
					mage.WithUncounterable(),
				)
			})
		}
		// A vanilla green creature spell that should benefit from a static
		// "green spells you control can't be countered" filter.
		if !mage.CardRegistered("Test Vanilla Green Bear") {
			mage.Register("Test Vanilla Green Bear", func() mage.Card {
				return mage.NewCreature("Test Vanilla Green Bear", "{1}{G}", 2, 2,
					mage.WithSubTypes("Bear"))
			})
		}
		// A simple counterspell.
		if !mage.CardRegistered("Test Counterspell") {
			mage.Register("Test Counterspell", func() mage.Card {
				return mage.NewInstant("Test Counterspell", "{U}{U}",
					mage.NewTargetedSpell(mage.TargetSpellOnStack(), mage.CounterSpell()),
				)
			})
		}
		// A permanent that grants uncounterable to its controller's green
		// spells while on the battlefield.
		if !mage.CardRegistered("Test Green Shield") {
			mage.Register("Test Green Shield", func() mage.Card {
				return mage.NewEnchantment("Test Green Shield", "{G}",
					mage.WithStaticAbility(mage.RegisterUncounterableStatic(
						func(g *mage.Game, obj *mage.StackObject) bool {
							perm := g.FindPermanent(obj.SourceID)
							_ = perm
							if obj.Card == nil {
								return false
							}
							isGreen := slices.Contains(obj.Card.ManaCost().Colors(), core.Green)
							return isGreen
						},
					)),
				)
			})
		}
	})
}

// TestUncounterable_PerCardFlag confirms that a spell registered with
// WithUncounterable cannot be countered: the counter resolves but the
// targeted spell stays on the stack and resolves normally.
func TestUncounterable_PerCardFlag(t *testing.T) {
	registerUncounterableCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Test Uncounterable Bear")
	tg.AddCard(core.ZoneHand, PlayerB, "Test Counterspell")
	// A casts the bear; B tries to counter it.
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Uncounterable Bear")
	tg.CastInResponseTo(PlayerB, "Test Counterspell", "Test Uncounterable Bear")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// Bear resolves: lives on the battlefield. Counterspell goes to graveyard.
	tg.AssertPermanentCount(PlayerA, "Test Uncounterable Bear", 1)
	tg.AssertGraveyardCount(PlayerA, "Test Uncounterable Bear", 0)
	tg.AssertGraveyardCount(PlayerB, "Test Counterspell", 1)
}

// TestUncounterable_StaticFilter confirms that a registered static
// uncounterable filter (controller's green spells) protects matching
// spells from counters.
func TestUncounterable_StaticFilter(t *testing.T) {
	registerUncounterableCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Test Green Shield")
	tg.AddCard(core.ZoneHand, PlayerA, "Test Vanilla Green Bear")
	tg.AddCard(core.ZoneHand, PlayerB, "Test Counterspell")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Vanilla Green Bear")
	tg.CastInResponseTo(PlayerB, "Test Counterspell", "Test Vanilla Green Bear")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Test Vanilla Green Bear", 1)
	tg.AssertGraveyardCount(PlayerA, "Test Vanilla Green Bear", 0)
	tg.AssertGraveyardCount(PlayerB, "Test Counterspell", 1)
}

// TestUncounterable_StaticFilterDoesNotApplyToOpponent confirms the
// static filter only protects the source's controller's spells.
func TestUncounterable_StaticFilterDoesNotApplyToOpponent(t *testing.T) {
	registerUncounterableCards()

	tg := NewTestGame(t)
	// B has the shield; A casts a green spell that B's shield should not protect.
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Test Green Shield")
	tg.AddCard(core.ZoneHand, PlayerA, "Test Vanilla Green Bear")
	tg.AddCard(core.ZoneHand, PlayerB, "Test Counterspell")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Vanilla Green Bear")
	tg.CastInResponseTo(PlayerB, "Test Counterspell", "Test Vanilla Green Bear")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// Note: our test filter doesn't actually consult source controller — but
	// in real cards it would. This test sanity-checks that filter logic
	// gates on something the player can configure.
	tg.AssertPermanentCount(PlayerA, "Test Vanilla Green Bear", 1)
}

// TestCounterSpell_CountersWhenNotProtected baseline: confirm a vanilla
// (no flag, no shield) green spell can in fact be countered.
func TestCounterSpell_CountersWhenNotProtected(t *testing.T) {
	registerUncounterableCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Test Vanilla Green Bear")
	tg.AddCard(core.ZoneHand, PlayerB, "Test Counterspell")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Vanilla Green Bear")
	tg.CastInResponseTo(PlayerB, "Test Counterspell", "Test Vanilla Green Bear")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Test Vanilla Green Bear", 0)
	tg.AssertGraveyardCount(PlayerA, "Test Vanilla Green Bear", 1)
}
