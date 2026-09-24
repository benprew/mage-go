package dsl_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTypedLifeGainOutcomeSequence(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	source := dsl.NewCreature("Modifier", "", 1, 1)
	g.PutOnBattlefield(source, a.PlayerID())
	g.AddLifeGainModifier(source.ID(), func(_ *mage.Game, _ uuid.UUID, amount int) int { return amount * 2 })
	body := dsl.Sequence(dsl.GainLife(dsl.You, dsl.Fixed(3)), dsl.GainLife(dsl.You, dsl.Fixed(2)))
	card := dsl.NewSorcery("Life", "", dsl.WithAbility(dsl.Spell(body)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := mage.ApplyEffect(g, body, card.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 30 || b.Life() != 20 || g.PlayerLifeGainedThisTurn(a.PlayerID()) != 10 {
		t.Fatal("sequence did not report each actual gain")
	}
}
