package gametest

import (
	"sync"
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

var sacSnapOnce sync.Once

func registerSacrificedSnapshotCards() {
	sacSnapOnce.Do(func() {
		// Test "Fling" — As an additional cost, sacrifice a creature.
		// Deals damage equal to the sacrificed creature's power to any target.
		if !mage.CardRegistered("Test Fling") {
			mage.Register("Test Fling", func() mage.Card {
				return mage.NewInstant("Test Fling", "{1}{R}",
					mage.NewTargetedSpell(mage.TargetAnyTarget(),
						mage.FuncEffect(
							"deal damage equal to sacrificed creature's power",
							mage.EffectProperties{},
							func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
								snap := g.LastSacrificed()
								if snap == nil || len(targets) == 0 {
									return nil
								}
								if pl := g.GetPlayer(targets[0]); pl != nil {
									g.DealDamageToPlayer(pl, snap.Power, sourceID)
									return nil
								}
								if perm := g.FindPermanent(targets[0]); perm != nil {
									g.DealDamageToPermanent(perm, snap.Power, sourceID)
								}
								return nil
							},
						),
					),
					mage.WithAdditionalCost(mage.SacrificeCreatureCost()),
				)
			})
		}
	})
}

// TestSacrificedSnapshot_FlingPower verifies an effect can read the
// sacrificed creature's power via Game.LastSacrificed() during resolution.
func TestSacrificedSnapshot_FlingPower(t *testing.T) {
	registerSacrificedSnapshotCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Hill Giant")
	tg.AddCard(core.ZoneHand, PlayerA, "Test Fling")
	tg.SetLife(PlayerB, 20)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Fling", "PlayerB")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertLife(PlayerB, 17)
	tg.AssertGraveyardCount(PlayerA, "Hill Giant", 1)
}
