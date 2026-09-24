package jumpstart

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestArchonOfJusticeTypedDefinition(t *testing.T) {
	card, err := mage.CreateCard("Archon of Justice")
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	var event, exile bool
	var visit func(mage.DefinitionMetadata)
	visit = func(node mage.DefinitionMetadata) {
		if node.Role == "event-pattern" {
			event = true
		}
		if node.Role == "resolving-effect" && strings.HasPrefix(node.Text, "exile") {
			exile = true
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	for _, ability := range card.Abilities() {
		visit(mage.InspectDefinition(ability))
	}
	if !event || !exile {
		t.Fatalf("typed death/exile metadata missing: event=%v exile=%v", event, exile)
	}
}

func TestArchonOfJusticeDeathExile(t *testing.T) {
	for _, scenario := range []string{"creature", "land", "artifact", "enchantment", "own permanent", "returned target", "phased target", "phase cycle", "source returned", "source exiled", "stolen source", "control change"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
			name := "Grizzly Bears"
			switch scenario {
			case "land":
				name = "Island"
			case "artifact":
				name = "Sol Ring"
			case "enchantment":
				name = "Crusade"
			}
			owner := gametest.PlayerB
			if scenario == "own permanent" {
				owner = gametest.PlayerA
			}
			target := g.AddCard(core.ZoneBattlefield, owner, name)
			targetCard := g.FindPermanent(target).Card
			source := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Archon of Justice")
			sourceCard := g.FindPermanent(source).Card
			chooser := gametest.PlayerA
			if scenario == "stolen source" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: uuid.New(), TargetID: source, ControllerID: b.PlayerID(), Duration: core.Indefinite})
				chooser = gametest.PlayerB
			}
			g.ChooseTarget(chooser, name)
			if scenario == "source exiled" {
				g.ExilePermanent(g.FindPermanent(source))
			} else {
				g.DestroyPermanent(g.FindPermanent(source))
			}
			g.PutTriggersOnStack()
			if scenario != "source exiled" {
				g.AssertGraveyardCount(gametest.PlayerA, "Archon of Justice", 1)
				if g.StackSize() != 1 {
					t.Fatalf("stack size = %d, want 1", g.StackSize())
				}
				obj := g.StackPeek()
				expectedController := a.PlayerID()
				if scenario == "stolen source" {
					expectedController = b.PlayerID()
				}
				if obj.Controller != expectedController {
					t.Fatal("trigger lost its source controller")
				}
				if len(obj.Targets) != 1 || obj.Targets[0] != target {
					t.Fatal("trigger selected the wrong target")
				}
				if g.FindPermanent(target) == nil {
					t.Fatal("trigger resolved before priority")
				}
			}
			switch scenario {
			case "returned target":
				g.RemoveFromBattlefield(g.FindPermanent(target))
				g.PutOnBattlefield(targetCard, b.PlayerID())
			case "phased target":
				g.PhaseOut(target)
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(target))
			case "source returned":
				a.RemoveFromGraveyard(source)
				g.PutOnBattlefield(sourceCard, a.PlayerID())
			case "control change":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: uuid.New(), TargetID: target, ControllerID: a.PlayerID(), Duration: core.Indefinite})
			}
			g.ResolveStack()
			if scenario == "phased target" {
				g.PhaseIn([]uuid.UUID{target})
			}
			survives := scenario == "returned target" || scenario == "phased target" || scenario == "source exiled"
			if (g.FindPermanent(target) != nil) != survives {
				t.Fatal("wrong exile result")
			}
			want := 1
			if survives {
				want = 0
			}
			g.AssertExileCount(name, want)
			g.AssertGraveyardCount(owner, name, 0)
		})
	}
}
