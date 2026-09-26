package arabian

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRukhEggDelayedTokenSurvivesSourceReturn(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	egg, err := mage.CreateCard("Rukh Egg")
	if err != nil {
		t.Fatal(err)
	}
	egg.SetOwner(a.PlayerID())
	g.PutOnBattlefield(egg, a.PlayerID())
	g.DoSacrifice(g.FindPermanent(egg.ID()))
	g.PutTriggersOnStack()
	g.ResolveStack()
	g.PutOnBattlefield(egg, b.PlayerID())
	clone := g.Clone()
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	birds := clone.FilterBattlefield(mage.And(mage.IsToken, mage.HasSubType("Bird")))
	if len(birds) != 1 || birds[0].ControllerID() != a.PlayerID() || birds[0].CurrentPower(clone) != 4 || birds[0].CurrentToughness(clone) != 4 || !mage.HasColorFilter(core.Red).Match(birds[0], clone) || !birds[0].HasKeyword(core.Flying) {
		t.Fatalf("incorrect delayed token: %+v", birds)
	}
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if len(clone.FilterBattlefield(mage.IsToken)) != 1 || len(g.FilterBattlefield(mage.IsToken)) != 0 {
		t.Fatal("delayed trigger repeated or changed the original game")
	}
}

func TestArabianComposableDefinitions(t *testing.T) {
	for name, role := range map[string]string{"Rukh Egg": "event-trigger", "Kird Ape": "continuous-rule", "Fishliver Oil": "continuous-rule", "Unstable Mutation": "event-trigger"} {
		t.Run(name, func(t *testing.T) {
			card, err := mage.CreateCard(name)
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition(name, card); err != nil {
				t.Fatal(err)
			}
			found := false
			var check func(mage.DefinitionMetadata)
			check = func(node mage.DefinitionMetadata) {
				found = found || node.Role == role
				for _, child := range node.Children {
					check(child)
				}
			}
			for _, ability := range card.Abilities() {
				check(mage.InspectDefinition(ability))
			}
			if !found {
				t.Fatalf("missing %s", role)
			}
		})
	}
}
