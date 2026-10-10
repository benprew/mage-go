package gametest

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestNextTurnDurationBeforeFirstStep(t *testing.T) {
	tg := NewTestGame(t)
	player := tg.GetPlayer(PlayerA).PlayerID()
	p := tg.PutOnBattlefield(mage.NewCreature("Duration subject", "", 2, 4), player)
	if err := mage.ApplyEffect(tg.Game, mage.UntilNextTurn(mage.You, mage.GrantKeywordModification(mage.This, core.Flying)), p.ID(), player, nil); err != nil {
		t.Fatal(err)
	}
	tg.StopAt(3, core.Untap)
	tg.Execute()
	if tg.FindPermanent(p.ID()).HasKeyword(core.Flying) {
		t.Fatal("turn entry retained duration before first step")
	}
}
