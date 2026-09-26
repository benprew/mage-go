package gametest

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// EvtAttackersDeclared fires once per combat after all attackers have been
// declared, regardless of how many. WheneverOneOrMoreCreaturesYouControlAttack
// restricts the trigger to the controller's own combats.
func TestWheneverOneOrMoreCreaturesAttack_FiresOncePerCombat(t *testing.T) {
	const cardName = "AD One-Or-More Sentinel"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			return mage.NewEnchantment(cardName, "{0}",
				mage.WithAbility(mage.WheneverOneOrMoreCreaturesYouControlAttackTrigger(
					mage.GainLifeFor(mage.You, mage.Fixed(5)), false,
				)),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, cardName)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears", 3)
	tg.Attack(1, PlayerA, "Grizzly Bears", "Grizzly Bears", "Grizzly Bears")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// Three attackers, but the once-per-combat trigger fires exactly once: +5.
	tg.AssertLife(PlayerA, 25)
}

func TestTypedAttackGroupRealDeclaration(t *testing.T) {
	const cardName = "Typed Attack Group Watcher"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			count := mage.BindNumber("declared attackers")
			return mage.NewEnchantment(cardName, "",
				mage.WithAbility(mage.WhenEvent(
					mage.OneOrMoreCreaturesYouControlAttack().WithCount(count),
					mage.GainLifeFor(mage.You, mage.FromNumber(count)))))
		})
	}
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, cardName)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Hill Giant")
	tg.Attack(1, PlayerA, "Grizzly Bears", "Hill Giant")
	tg.StopAt(1, core.EndStep)
	tg.Execute()
	tg.AssertLife(PlayerA, 22)
}

// Trigger does NOT fire if no creature attacks.
func TestWheneverOneOrMoreCreaturesAttack_NoFireIfNoneAttack(t *testing.T) {
	const cardName = "AD No-Attack Sentinel"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			return mage.NewEnchantment(cardName, "{0}",
				mage.WithAbility(mage.WheneverOneOrMoreCreaturesYouControlAttackTrigger(
					mage.GainLifeFor(mage.You, mage.Fixed(5)), false,
				)),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, cardName)
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertLife(PlayerA, 20)
}
