package interactive

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRulesTextKeywordsBeforeOtherAbilities(t *testing.T) {
	card := mage.NewCreature("Bird", "{U}", 1, 1, mage.WithKeyword(core.Flying), mage.WithAbility(mage.ProtectionFromColor(core.Red)), mage.WithKeyword(core.Haste))
	text := buildRulesText(card)
	if !strings.HasPrefix(text, "Flying, Haste, Protection from Red") {
		t.Fatalf("unexpected rules order: %s", text)
	}
}
