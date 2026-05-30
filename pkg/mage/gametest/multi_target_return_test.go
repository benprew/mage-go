package gametest

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// TestMultiTargetReturnFromGraveyard verifies that
// ReturnFromGraveyardToHandTarget correctly handles a NewMultiTargetSpell
// with multiple Target slots — Soul Salvage / Faithless Salvaging "Return
// two target creature cards from your graveyard to your hand."
func TestMultiTargetReturnFromGraveyard(t *testing.T) {
	const spell = "Test Two Target Return"
	if !mage.CardRegistered(spell) {
		mage.Register(spell, func() mage.Card {
			return mage.NewSorcery(spell, "{2}{B}",
				mage.NewMultiTargetSpell(
					[]mage.Target{mage.TargetCardInYourGraveyard(mage.IsCreatureCard), mage.TargetCardInYourGraveyard(mage.IsCreatureCard)},
					mage.ReturnFromGraveyardToHandTarget(),
				),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneGraveyard, PlayerA, "Grizzly Bears")
	tg.AddCard(core.ZoneGraveyard, PlayerA, "Hill Giant")
	tg.AddCard(core.ZoneHand, PlayerA, spell)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, spell, "Grizzly Bears", "Hill Giant")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertHandCount(PlayerA, "Grizzly Bears", 1)
	tg.AssertHandCount(PlayerA, "Hill Giant", 1)
	tg.AssertGraveyardCount(PlayerA, "Grizzly Bears", 0)
	tg.AssertGraveyardCount(PlayerA, "Hill Giant", 0)
}
