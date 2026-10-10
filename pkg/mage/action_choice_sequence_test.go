package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestResolutionChoiceSequenceBindings(t *testing.T) {
	for _, mode := range []int{0, 1} {
		a := &resolutionChooser{BasePlayer: NewBasePlayer("A"), choice: mode}
		b := NewBasePlayer("B")
		g := NewGame(a, b)
		outer, local := BindNumber("outer"), BindNumber("local")
		effect := Sequence(
			SnapshotNumber(Fixed(2), outer),
			ChooseAtResolution(You,
				ResolutionOption{Label: "save", Body: Sequence(
					SnapshotNumber(AddValues(FromNumber(outer), Fixed(3)), local),
					AtNext(EachEndStep, GainLifeFor(You, FromNumber(local))),
					SnapshotNumber(Fixed(99), local))},
				ResolutionOption{Label: "gain", Body: GainLifeFor(You, Fixed(7))}),
			GainLifeFor(You, FromNumber(outer)))
		card := NewInstant("Choice frame", "", WithAbility(Spell(effect)))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		want := 22
		if mode == 1 {
			want = 29
		}
		if a.Life() != want || a.prompts != 1 {
			t.Fatal("choice changed an inherited input or ran both bodies")
		}
		clone := g.Clone()
		clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		clone.PutTriggersOnStack()
		clone.ResolveStack()
		if mode == 0 {
			want = 27
		}
		if clone.GetPlayer(a.PlayerID()).Life() != want || a.Life() != 22+7*mode {
			t.Fatal("delay lost its branch value or shared clone storage")
		}
	}
}
