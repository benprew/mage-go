package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestMayAttachPublicDeclaration(t *testing.T) {
	entered := dsl.BindPermanent("entered enchantment")
	created := dsl.BindPermanent("created token")
	cat := dsl.DefineToken("Cat", dsl.TokenTypes(core.TypeCreature), dsl.TokenSubtypes("Cat"), dsl.TokenColors(core.White), dsl.TokenPT(2, 2))
	body := dsl.Sequence(dsl.CreateOneToken(dsl.You, cat, created),
		dsl.IfPermanentHasEnchantmentSubtype(dsl.From(entered), "Aura",
			dsl.MayAttach(dsl.From(entered), dsl.From(created))))
	card := dsl.NewCreature("Watcher", "", 3, 3, dsl.WithAbility(dsl.WhenEvent(
		dsl.EnchantmentYouControlEnters().WithPermanent(entered), body)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
