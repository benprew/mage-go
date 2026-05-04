package gametest

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// TestMultiTargetUpToN exercises NewMultiTargetSpell + TargetUpToNCreatures +
// ToAllTargets() — a single Target with min=0 max=N whose effect must apply
// to each chosen target. Mirrors Dauntless Onslaught's "Up to two target
// creatures each get +2/+2 until end of turn."
func TestMultiTargetUpToN(t *testing.T) {
	const spell = "Test Up To Two Pump"
	if !mage.CardRegistered(spell) {
		mage.Register(spell, func() mage.Card {
			return mage.NewSorcery(spell, "{1}{W}",
				mage.NewMultiTargetSpell(
					[]mage.Target{mage.TargetUpToNCreatures(2)},
					mage.Boost(mage.Fixed(2), mage.Fixed(2)).Targeting(mage.ToAllTargets()),
				),
			)
		})
	}

	t.Run("two targets both get +2/+2", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Hill Giant")
		tg.AddCard(core.ZoneHand, PlayerA, spell)
		tg.CastSpell(1, core.PrecombatMain, PlayerA, spell, "Grizzly Bears", "Hill Giant")
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		tg.AssertPowerToughness(PlayerA, "Grizzly Bears", 4, 4)
		tg.AssertPowerToughness(PlayerA, "Hill Giant", 5, 5)
	})

	t.Run("single chosen target gets +2/+2", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Hill Giant")
		tg.AddCard(core.ZoneHand, PlayerA, spell)
		tg.CastSpell(1, core.PrecombatMain, PlayerA, spell, "Grizzly Bears")
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		tg.AssertPowerToughness(PlayerA, "Grizzly Bears", 4, 4)
		tg.AssertPowerToughness(PlayerA, "Hill Giant", 3, 3)
	})

	t.Run("zero targets resolves with no effect", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
		tg.AddCard(core.ZoneHand, PlayerA, spell)
		tg.CastSpell(1, core.PrecombatMain, PlayerA, spell)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		tg.AssertPowerToughness(PlayerA, "Grizzly Bears", 2, 2)
		tg.AssertGraveyardCount(PlayerA, spell, 1)
	})
}

// TestMultiTargetDistinctControllers exercises a two-Target spell whose two
// targets must have disjoint controllers (Peel from Reality: "Return target
// creature you control and target creature you don't control to their owners'
// hands"). Verifies that legality is enforced per-Target on cast, and that
// both targets are bounced at resolution.
func TestMultiTargetDistinctControllers(t *testing.T) {
	const spell = "Test Mutual Bounce"
	if !mage.CardRegistered(spell) {
		mage.Register(spell, func() mage.Card {
			return mage.NewInstant(spell, "{1}{U}",
				mage.NewMultiTargetSpell(
					[]mage.Target{
						mage.TargetCreatureYouControl(),
						mage.TargetCreatureOpponentControls(),
					},
					mage.FuncEffect("return both creatures to their owners' hands",
						mage.EffectProperties{Outcome: mage.OutcomeUnknown, IsBounce: true},
						func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							for _, tid := range targets {
								perm := g.FindPermanent(tid)
								if perm == nil {
									continue
								}
								card := perm.Card
								owner := card.Owner()
								if owner == uuid.Nil {
									owner = perm.Controller
								}
								g.RemoveFromBattlefield(perm)
								if pl := g.GetPlayer(owner); pl != nil {
									pl.AddToHand(card)
								}
							}
							return nil
						}),
				),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Hill Giant")
	tg.AddCard(core.ZoneHand, PlayerA, spell)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, spell, "Grizzly Bears", "Hill Giant")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Grizzly Bears", 0)
	tg.AssertPermanentCount(PlayerB, "Hill Giant", 0)
	tg.AssertHandCount(PlayerA, "Grizzly Bears", 1)
	tg.AssertHandCount(PlayerB, "Hill Giant", 1)
}

// TestDividedDamage exercises DealDividedDamage + TargetUpToNCreaturesOrPlayers.
// The controller chooses the distribution at cast time (CR 601.2d). Mirrors
// Flames of the Firebrand's "Flames of the Firebrand deals 3 damage divided
// as you choose among one, two, or three targets."
func TestDividedDamage(t *testing.T) {
	const spell = "Test Divided 3"
	if !mage.CardRegistered(spell) {
		mage.Register(spell, func() mage.Card {
			return mage.NewSorcery(spell, "{2}{R}",
				mage.NewMultiTargetSpell(
					[]mage.Target{mage.TargetUpToNCreaturesOrPlayers(3)},
					mage.DealDividedDamage(mage.Fixed(3)),
				),
			)
		})
	}

	t.Run("split 1/2 between two creatures", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerB, "Grizzly Bears")
		tg.AddCard(core.ZoneBattlefield, PlayerB, "Hill Giant")
		tg.AddCard(core.ZoneHand, PlayerA, spell)
		tg.ChooseDamageDistribution(PlayerA, map[string]int{
			"Grizzly Bears": 1,
			"Hill Giant":    2,
		})
		tg.CastSpell(1, core.PrecombatMain, PlayerA, spell, "Grizzly Bears", "Hill Giant")
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		tg.AssertPermanentCount(PlayerB, "Grizzly Bears", 1)
		tg.AssertPermanentCount(PlayerB, "Hill Giant", 1)
		tg.AssertGraveyardCount(PlayerA, spell, 1)
	})

	t.Run("3 to one creature kills it", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerB, "Grizzly Bears")
		tg.AddCard(core.ZoneHand, PlayerA, spell)
		tg.ChooseDamageDistribution(PlayerA, map[string]int{"Grizzly Bears": 3})
		tg.CastSpell(1, core.PrecombatMain, PlayerA, spell, "Grizzly Bears")
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		tg.AssertPermanentCount(PlayerB, "Grizzly Bears", 0)
	})

	t.Run("split among player and creature", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerB, "Grizzly Bears")
		tg.AddCard(core.ZoneHand, PlayerA, spell)
		tg.ChooseDamageDistribution(PlayerA, map[string]int{
			"PlayerB":       2,
			"Grizzly Bears": 1,
		})
		tg.CastSpell(1, core.PrecombatMain, PlayerA, spell, "PlayerB", "Grizzly Bears")
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		tg.AssertLife(PlayerB, 18)
		tg.AssertPermanentCount(PlayerB, "Grizzly Bears", 1)
	})
}
