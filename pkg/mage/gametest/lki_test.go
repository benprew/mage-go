package gametest

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// LKI snapshot lets death-trigger conditions read the dying creature's
// controller after the permanent has moved to the graveyard.
// EventSourceControlledByOpponent must succeed when an opponent's creature
// dies — without LKI, FindPermanent returns nil and the condition would
// always fail.
func TestLKI_DeathTriggerReadsOpponentController(t *testing.T) {
	const cardName = "LKI Death Sentinel"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			return mage.NewEnchantment(cardName, "{0}",
				mage.WithAbility(mage.NewTriggered(core.EvtCreatureDied, false,
					mage.GainLife(1),
				).SetConditionData(mage.EventSourceControlledByOpponent{})),
			)
		})
	}

	// Sentinel under PlayerA. PlayerB's creature dies. Sentinel triggers.
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, cardName)
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Grizzly Bears")
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Mountain", 1)
	tg.AddCard(core.ZoneHand, PlayerA, "Lightning Bolt", 2)
	// Bolt twice for 6 damage (more than enough to kill 2/2). One bolt resolves
	// per cast; we cast both back-to-back targeting Bears.
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Lightning Bolt", "Grizzly Bears")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// Bears died. LKI snapshot recorded controller = PlayerB.
	// Sentinel's EventSourceControlledByOpponent reads LKI -> matches.
	tg.AssertLife(PlayerA, 21)
}

// Direct accessor: Game.LKI(id) should return a snapshot keyed by the
// permanent's ID after RemoveFromBattlefield.
func TestLKI_DirectAccessor(t *testing.T) {
	const cardName = "LKI Probe"
	captured := false
	var capturedController uuid.UUID
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			return mage.NewEnchantment(cardName, "{0}",
				mage.WithAbility(mage.NewTriggered(core.EvtCreatureDied, false,
					mage.FuncEffect("probe", mage.EffectProperties{},
						func(g *mage.Game, _, _ uuid.UUID, _ []uuid.UUID) error {
							return nil
						}),
				).SetCondition(func(evt *core.GameEvent, gr mage.GameReader, _, _ uuid.UUID) bool {
					if game, ok := gr.(*mage.Game); ok {
						if view := game.LookupObject(evt.SourceID); view != nil {
							captured = true
							capturedController = view.ViewController()
						}
					}
					return false
				})),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, cardName)
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Grizzly Bears")
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Mountain", 1)
	tg.AddCard(core.ZoneHand, PlayerA, "Lightning Bolt")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Lightning Bolt", "Grizzly Bears")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	if !captured {
		t.Fatalf("expected LKI to be populated for died creature")
	}
	if capturedController == uuid.Nil {
		t.Fatalf("expected LKI controller to be set")
	}
}
