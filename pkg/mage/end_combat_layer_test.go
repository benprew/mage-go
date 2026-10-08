package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestEndCombatLayerPhaseBoundary(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(1)
	card := NewCreature("Observer", "", 2, 5, WithAbility(At(EachEndOfCombat, GainLifeFor(You, PowerOf(This)))))
	subject := g.PutOnBattlefield(card, a.PlayerID())
	if err := ApplyEffect(g, Until(core.EndOfCombat, ModifyPT(This, Fixed(4), Fixed(0)), GrantKeywordModification(This, core.Flying)), subject.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	if subject.CurrentPower(g) != 6 || !subject.HasKeyword(core.Flying) {
		t.Fatal("combat duration did not start")
	}
	clone := g.Clone()
	g.PhaseIn(g.PhaseOut(subject.ID()))
	g.RunStepWithPriority(core.EndCombat)
	if a.Life() != 26 {
		t.Fatal("layers expired before end-of-combat triggers resolved")
	}
	if g.FindPermanent(subject.ID()).CurrentPower(g) != 2 || g.FindPermanent(subject.ID()).HasKeyword(core.Flying) {
		t.Fatal("layers survived end of combat phase")
	}
	if clone.FindPermanent(subject.ID()).CurrentPower(clone) != 6 {
		t.Fatal("clone shared expiry")
	}
	clone.RunStepWithPriority(core.EndCombat)
	if clone.GetPlayer(a.PlayerID()).Life() != 26 || clone.FindPermanent(subject.ID()).CurrentPower(clone) != 2 {
		t.Fatal("cloned phase boundary changed")
	}
	g.RunStepWithPriority(core.EndCombat)
	if a.Life() != 28 {
		t.Fatal("later combat inherited expired bonus")
	}
}
