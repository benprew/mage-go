package dsl_test

import (
	"fmt"
	"testing"

	_ "github.com/benprew/mage-go/cards"
	_ "github.com/benprew/mage-go/cards/antiquities"
	_ "github.com/benprew/mage-go/cards/arabian"
	_ "github.com/benprew/mage-go/cards/astral"
	_ "github.com/benprew/mage-go/cards/fallen_empires"
	_ "github.com/benprew/mage-go/cards/fourthedition"
	_ "github.com/benprew/mage-go/cards/jumpstart"
	_ "github.com/benprew/mage-go/cards/legends"
	_ "github.com/benprew/mage-go/cards/limited"
	_ "github.com/benprew/mage-go/cards/promo"
	_ "github.com/benprew/mage-go/cards/secretsofstrixhaven"
	_ "github.com/benprew/mage-go/cards/thedark"
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestStateSacrificeMigration(t *testing.T) {
	for _, name := range []string{"Goblins of the Flarg", "Vodalian Knights", "Giant Shark", "Pirate Ship", "Sea Serpent", "Dandân", "Island Fish Jasconius", "Merchant Ship", "Serendib Djinn", "Drop of Honey", "Mana Vortex", "Jihad"} {
		t.Run(name, func(t *testing.T) {
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			card, err := mage.CreateCard(name)
			if err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(card, a.PlayerID())
			p := g.MutablePermanent(card.ID())
			if name == "Goblins of the Flarg" {
				g.PutOnBattlefield(mage.NewCreature("Dwarf", "", 1, 1, mage.WithSubTypes("Dwarf")), a.PlayerID())
			}
			if name == "Jihad" {
				p.ChosenColor = core.Green
				p.ChosenPlayer = b.PlayerID()
			}
			g.CheckStateBasedActions()
			if g.FindPermanent(card.ID()) == nil {
				t.Fatal("sacrificed before trigger resolution")
			}
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			var state *mage.StackObject
			for _, obj := range g.Stack().Objects() {
				if obj.SourceID == card.ID() {
					state = obj
				}
			}
			if state == nil {
				t.Fatal("no sacrifice trigger")
			}
			// A response can remove the condition without canceling the instruction.
			g.PutOnBattlefield(mage.NewLand("Island", mage.WithSubTypes("Island")), a.PlayerID())
			g.PutOnBattlefield(mage.NewCreature("Green", "{G}", 1, 1), b.PlayerID())
			if name == "Goblins of the Flarg" {
				for _, other := range g.FilterBattlefield(mage.HasSubType("Dwarf")) {
					g.DoSacrifice(other)
				}
			}
			g.Stack().RemoveByID(state.ID)
			g.ResolveStackObject(state)
			if g.FindPermanent(card.ID()) != nil {
				t.Fatal("trigger did not sacrifice source after response")
			}
		})
	}
}

func TestRegisteredDefinitions(t *testing.T) {
	if err := mage.DefaultRegistry.ValidateDefinitions(); err != nil {
		t.Fatal(err)
	}
}

func TestJihadCurrentColorAndTokenCondition(t *testing.T) {
	for _, token := range []bool{false, true} {
		t.Run(fmt.Sprint(token), func(t *testing.T) {
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			green := mage.NewCreature("Green", "{G}", 1, 1)
			g.PutOnBattlefield(green, b.PlayerID())
			g.MutablePermanent(green.ID()).IsToken = token
			card, err := mage.CreateCard("Jihad")
			if err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(card, a.PlayerID())
			g.PutTriggersOnStack()
			for g.Stack().Size() > 0 {
				g.Stack().Pop()
			}
			source := g.MutablePermanent(card.ID())
			source.ChosenColor = core.Green
			source.ChosenPlayer = b.PlayerID()
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if !token {
				if g.Stack().Size() != 0 {
					t.Fatal("nontoken green permanent did not prevent sacrifice")
				}
				g.ChangeObjectColor(green.ID(), core.Blue)
				g.PutTriggersOnStack()
			}
			if g.Stack().Size() != 1 {
				t.Fatal("missing sacrifice trigger")
			}
			g.ResolveStackObject(g.Stack().Pop())
			if g.FindPermanent(card.ID()) != nil {
				t.Fatal("Jihad was not sacrificed")
			}
		})
	}
}
