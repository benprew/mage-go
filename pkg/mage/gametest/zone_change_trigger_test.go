package gametest

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// Sanity: a legacy EntersBattlefieldTrigger draws a card from this exact
// scenario. If this passes and TestOnEnterZone_FiresOnETB fails, the
// EvtZoneChange dispatch is the broken piece (not the test setup).
func TestOnEnterZone_LegacySanity(t *testing.T) {
	const cardName = "OEZ Legacy Sanity"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			return mage.NewCreature(cardName, "{2}", 1, 1,
				mage.WithAbility(mage.EntersBattlefieldTrigger(
					mage.DrawCards(mage.Fixed(1)),
					false,
				)),
			)
		})
	}
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, cardName)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Mountain", 2)
	tg.AddCard(core.ZoneLibrary, PlayerA, "Plains", 5)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, cardName)
	tg.StopAt(1, core.EndStep)
	tg.Execute()
	tg.AssertPermanentCount(PlayerA, "Plains", 1)
}

// OnEnterZone(ZoneBattlefield) fires for a self ETB via the new EvtZoneChange
// dispatch path (rather than the legacy EvtEntersBattlefield). Smoke test for
// Phase 1 of the zone-change refactor.
func TestOnEnterZone_FiresOnETB(t *testing.T) {
	const cardName = "OEZ ETB Draw"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			return mage.NewCreature(cardName, "{2}", 1, 1,
				mage.WithAbility(mage.OnEnterZone(
					core.ZoneBattlefield,
					mage.DrawCards(mage.Fixed(1)),
					false,
				)),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, cardName)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Mountain", 2)
	tg.AddCard(core.ZoneLibrary, PlayerA, "Plains", 5)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, cardName)
	tg.StopAt(1, core.EndStep)
	tg.Execute()
	// ETB drew a Plains.
	tg.AssertPermanentCount(PlayerA, "Plains", 1)
}

// OnLeaveZone(ZoneBattlefield, ZoneGraveyard) fires when the source is put
// into a graveyard from the battlefield via any path — sacrifice, destroy,
// or SBA. Verifies the SelfGraveyard capture works through EvtZoneChange.
func TestOnLeaveZone_FiresOnSacrifice(t *testing.T) {
	const cardName = "OLZ Terrarion-like"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			return mage.NewArtifact(cardName, "{1}",
				mage.WithActivatedAbility(
					mage.FuncEffect("noop", mage.EffectProperties{},
						func(_ *mage.Game, _, _ uuid.UUID, _ []uuid.UUID) error { return nil }),
					mage.ManaCostOf("{0}"),
					mage.WithCost(mage.SacrificeSourceCost()),
				),
				mage.WithAbility(mage.OnLeaveZone(
					core.ZoneBattlefield,
					core.ZoneGraveyard,
					mage.DrawCards(mage.Fixed(1)),
					false,
				)),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, cardName)
	tg.AddCard(core.ZoneLibrary, PlayerA, "Forest", 5)
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, cardName)
	tg.StopAt(1, core.EndStep)
	tg.Execute()
	tg.AssertGraveyardCount(PlayerA, cardName, 1)
	tg.AssertPermanentCount(PlayerA, "Forest", 1)
}
