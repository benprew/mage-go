package gametest

import (
	"testing"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestFindActivatableAbilityUsesRelativeTargetContext(t *testing.T) {
	for _, tc := range []struct {
		name        string
		filter      mage.ObjectFilter
		self, enemy bool
	}{
		{"controller", mage.ObjectControlledBy(mage.SelectController()), true, false},
		{"other", mage.OtherThan(mage.SourceObject()), false, true},
		{"combined", mage.ObjectAnd(mage.ObjectControlledBy(mage.SelectController()), mage.OtherThan(mage.SourceObject())), false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tg := NewTestGame(t)
			owner := tg.playerA.PlayerID()
			source := tg.PutOnBattlefield(mage.NewArtifact("Source", "{0}", mage.WithAbility(mage.NewActivated(
				mage.LifePayCost(1), mage.WithTarget(mage.TargetPermanent(tc.filter)), mage.GainLife(2),
			))), owner)
			other := tg.PutOnBattlefield(mage.NewCreature("Other", "{G}", 1, 1), owner)
			enemy := tg.PutOnBattlefield(mage.NewCreature("Enemy", "{G}", 1, 1), tg.playerB.PlayerID())
			for _, candidate := range []struct {
				permanent *mage.Permanent
				legal     bool
			}{
				{other, true}, {source, tc.self}, {enemy, tc.enemy},
			} {
				want := candidate.legal

				_, idx, err := tg.findActivatableAbilityByName(owner, "Source", []uuid.UUID{candidate.permanent.ID()})
				if (err == nil) != want {
					t.Errorf("candidate %s: got err=%v, want legal=%v", candidate.permanent.Name(), err, want)
				}
				if err == nil && idx != 0 {
					t.Errorf("got ability index %d", idx)
				}
			}
		})
	}
}

func TestActivateAbilityWithRelativePermanentTarget(t *testing.T) {
	const name = "Relative Target Source"
	mage.Register(name, func() *mage.Card {
		return mage.NewArtifact(name, "{0}", mage.WithAbility(mage.NewActivated(
			mage.LifePayCost(1),
			mage.WithTarget(mage.TargetPermanent(mage.ObjectAnd(
				mage.IsCardType(core.TypeCreature), mage.ObjectControlledBy(mage.SelectController()), mage.OtherThan(mage.SourceObject()),
			))),
			mage.GainLife(2),
		)))
	})
	const creature = "Relative Target Creature"
	mage.Register(creature, func() *mage.Card { return mage.NewCreature(creature, "{G}", 1, 1) })
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, name)
	tg.AddCard(core.ZoneBattlefield, PlayerA, creature)
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, name, creature)
	tg.StopAt(1, core.EndStep)
	tg.Execute()
	tg.AssertLife(PlayerA, 21)
}
