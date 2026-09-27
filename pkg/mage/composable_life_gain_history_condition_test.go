package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestControllerGainedLifeThisTurnActualOutcomeAndIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	condition := ControllerGainedLifeThisTurn()
	if condition.checkState(g, StateContext{ControllerID: a.PlayerID()}) {
		t.Fatal("matched before life gain")
	}
	source := NewInstant("Gain", "")
	g.effects.AddReplacement(&lifeOutcomeReplacement{consume: true})
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(2)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if condition.checkState(g, StateContext{ControllerID: a.PlayerID()}) {
		t.Fatal("matched a replaced life gain")
	}
	g.effects.replacements = nil
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(0)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if condition.checkState(g, StateContext{ControllerID: a.PlayerID()}) {
		t.Fatal("matched zero life gain")
	}
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(2)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !condition.checkState(g, StateContext{SourceID: uuid.New(), ControllerID: a.PlayerID()}) {
		t.Fatal("absent source hid controller history")
	}
	if condition.checkState(g, StateContext{ControllerID: b.PlayerID()}) || condition.checkState(g, StateContext{ControllerID: uuid.Nil}) {
		t.Fatal("condition matched another or missing controller")
	}
	clone := g.Clone()
	clone.trackers.Turn.ResetForNewTurn(b.PlayerID())
	if condition.checkState(clone, StateContext{ControllerID: a.PlayerID()}) || !condition.checkState(g, StateContext{ControllerID: a.PlayerID()}) {
		t.Fatal("clone shared life-gain history")
	}
	g.trackers.Turn.ResetForNewTurn(b.PlayerID())
	if condition.checkState(g, StateContext{ControllerID: a.PlayerID()}) {
		t.Fatal("history survived turn reset")
	}
}

func TestControllerGainedLifeThisTurnResolutionAndStatic(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	condition := ControllerGainedLifeThisTurn()
	effect := IfState(condition, GainLifeFor(You, Fixed(1)))
	spell := NewInstant("Conditional", "", Spell(effect))
	static := NewCreature("Conditional creature", "", 1, 1, WithAbility(Static(While(condition, GrantKeywordModification(This, Flying)))))
	g.PutOnBattlefield(static, a.PlayerID())
	for _, card := range []Card{spell, static} {
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	if err := ApplyEffect(g, effect, spell.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || g.FindPermanent(static.ID()).HasKeyword(Flying) {
		t.Fatal("condition was active before gain")
	}
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(2)), spell.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	if err := ApplyEffect(g, effect, spell.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 20 || !g.FindPermanent(static.ID()).HasKeyword(Flying) {
		t.Fatal("condition used the wrong controller or did not update static rule")
	}
	if err := ApplyEffect(g, effect, spell.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 23 {
		t.Fatal("resolution branch did not use controller history")
	}
	g.trackers.Turn.ResetForNewTurn(b.PlayerID())
	g.ApplyContinuousEffects()
	if g.FindPermanent(static.ID()).HasKeyword(Flying) {
		t.Fatal("static condition survived turn reset")
	}
	meta := InspectDefinition(condition)
	if meta.Role != "state-condition" || meta.Text != "controller gained life this turn" {
		t.Fatalf("inspection = %+v", meta)
	}
}
