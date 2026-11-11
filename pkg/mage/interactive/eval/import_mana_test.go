package eval

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestImportManaEvaluationParity(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	for _, color := range []core.Color{core.Green, core.AnyColor} {
		var effect = mage.AddMana(color, 1)
		if color == core.AnyColor {
			effect = mage.AddAnyMana(1, core.Green)
		}
		old := g.PutOnBattlefield(mage.NewCreature("Legacy", "{G}", 1, 1, mage.WithAbility(mage.NewManaAbility(color))), a.PlayerID())
		compiled := g.PutOnBattlefield(mage.NewCreature("Compiled", "{G}", 1, 1, mage.WithAbility(mage.NewActivated(mage.Tap(), effect))), a.PlayerID())
		if !hasManaAbility(compiled) || hasUtilityActivatedAbility(compiled) || ClassifyPermanent(compiled) != ClassifyPermanent(old) {
			t.Error("compiled mana creature has a different role")
		}
		if abilityBonus(compiled) != abilityBonus(old) || abilityBonusInGame(compiled, g) != abilityBonusInGame(old, g) {
			t.Error("compiled mana creature has a different ability score")
		}
		if evalNonCreaturePermanent(compiled, g) != evalNonCreaturePermanent(old, g) {
			t.Error("compiled mana source has a different permanent score")
		}
	}
}
