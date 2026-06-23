package ai

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAbilityActivationIsRedundant_DoesNotSuppressAdditionalEffects(t *testing.T) {
	controller := mage.NewBasePlayer("Alice")
	opponent := mage.NewBasePlayer("Bob")
	g := mage.NewGame(controller, opponent)
	card := mage.NewCreature("Rogue", "{1}{U}", 2, 2, mage.WithKeyword(core.Flying))
	card.SetOwner(controller.PlayerID())
	target := g.PutOnBattlefield(card, controller.PlayerID())
	effects := []mage.Effect{
		mage.GrantKeyword(core.Flying),
		mage.Boost(mage.Fixed(1), mage.Fixed(0)),
	}

	if AbilityActivationIsRedundant(g, controller.PlayerID(), target.ID(), effects, []uuid.UUID{target.ID()}) {
		t.Fatal("keyword already present must not suppress an activation with an additional boost")
	}
}
