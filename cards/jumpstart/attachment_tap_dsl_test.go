package jumpstart

import (
	"testing"

	_ "github.com/benprew/mage-go/cards/limited"
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAuraEntryTapLifetime(t *testing.T) {
	for _, name := range []string{"Paralyze", "Waterknot"} {
		for _, change := range []string{"none", "move", "leave before placement", "leave", "return", "host returns", "phase host", "phase cycle", "control"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				card, err := mage.CreateCard(name)
				if err != nil {
					t.Fatal(err)
				}
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
				first, second := mage.NewCreature("First", "", 2, 2), mage.NewCreature("Second", "", 2, 2)
				for _, host := range []mage.Card{first, second} {
					host.SetOwner(b.PlayerID())
					g.PutOnBattlefield(host, b.PlayerID())
				}
				card.SetOwner(a.PlayerID())
				g.PutOnBattlefield(card, a.PlayerID())
				g.Attach(card.ID(), first.ID())
				if change == "leave before placement" {
					g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
				}
				g.PutTriggersOnStack()
				if g.StackSize() != 1 || len(g.StackPeek().TargetSpecs) != 0 {
					t.Fatal("entry must create one nontargeted trigger")
				}
				wantFirst, wantSecond := true, false
				switch change {
				case "move":
					g.Attach(card.ID(), second.ID())
					wantFirst, wantSecond = false, true
				case "leave", "return":
					g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
					if change == "return" {
						g.PutOnBattlefield(card, a.PlayerID())
						g.Attach(card.ID(), second.ID())
					}
				case "host returns":
					g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
					g.PutOnBattlefield(first, b.PlayerID())
					wantFirst = false
				case "phase host":
					g.PhaseOut(first.ID())
					wantFirst = false
				case "phase cycle":
					g.PhaseIn(g.PhaseOut(first.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: second.ID(), TargetID: first.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				}
				clone := g.Clone()
				clone.ResolveTopOfStack()
				if clone.FindPermanentIncludingPhased(first.ID()).Tapped != wantFirst || clone.FindPermanent(second.ID()).Tapped != wantSecond {
					t.Fatal("entry tap used the wrong attachment incarnation")
				}
				if g.FindPermanentIncludingPhased(first.ID()).Tapped || g.FindPermanent(second.ID()).Tapped {
					t.Fatal("clone changed the original game")
				}
			})
		}
	}
}

func TestAuraEntryTapTypedDefinitions(t *testing.T) {
	for _, name := range []string{"Paralyze", "Waterknot"} {
		t.Run(name, func(t *testing.T) {
			card, err := mage.CreateCard(name)
			if err != nil {
				t.Fatal(err)
			}
			var event, tap, attached bool
			var visit func(mage.DefinitionMetadata)
			visit = func(n mage.DefinitionMetadata) {
				event = event || n.Role == "event-pattern"
				tap = tap || n.Role == "resolving-effect" && n.Properties.Taps && n.Properties.Outcome == mage.OutcomeDetriment
				attached = attached || n.Role == "permanent-ref" && n.Text == "attached permanent"
				for _, child := range n.Children {
					visit(child)
				}
			}
			visit(mage.InspectDefinition(card.Abilities()[0]))
			if !event || !tap || !attached {
				t.Fatalf("typed event=%v tap=%v attachment=%v", event, tap, attached)
			}
		})
	}
}
