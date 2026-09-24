package jumpstart

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestToweringWaveMysticTypedDefinition(t *testing.T) {
	card, err := mage.CreateCard("Towering-Wave Mystic")
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if len(card.CastTargets()) != 0 {
		t.Fatal("creature spell requires a target")
	}
	var event, amount, target, mill bool
	var visit func(mage.DefinitionMetadata)
	visit = func(n mage.DefinitionMetadata) {
		event = event || n.Role == "event-pattern" && n.Text == "source deals damage"
		amount = amount || n.Role == "number-binding"
		target = target || n.Role == "target-binding" && n.Text == "player"
		mill = mill || n.Text == "mill for player" && len(n.Children) == 2
		for _, child := range n.Children {
			visit(child)
		}
	}
	for _, ability := range card.Abilities() {
		visit(mage.InspectDefinition(ability))
	}
	if !event || !amount || !target || !mill {
		t.Fatalf("event=%v amount=%v target=%v mill=%v", event, amount, target, mill)
	}
}

func TestToweringWaveMysticTypedDamage(t *testing.T) {
	for _, scenario := range []string{"opponent", "controller", "creature", "partial prevention", "full prevention", "zero", "other source", "source exit", "source return", "copy", "invalid target", "two events"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
			for _, player := range []gametest.PlayerRef{gametest.PlayerA, gametest.PlayerB} {
				for _, name := range []string{"Mountain", "Forest", "Island", "Plains", "Swamp"} {
					g.AddCard(core.ZoneLibrary, player, name)
				}
			}
			id := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Towering-Wave Mystic")
			other := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
			target, choice := b.PlayerID(), "PlayerB"
			if scenario == "controller" || scenario == "creature" {
				target, choice = a.PlayerID(), "PlayerA"
			}
			g.ChooseTarget(gametest.PlayerA, choice)
			want := 2
			switch scenario {
			case "partial prevention":
				g.AddPreventionShield(b.PlayerID(), 1)
				want = 1
			case "full prevention":
				g.AddPreventionShield(b.PlayerID(), 2)
				want = 0
			case "zero", "other source":
				want = 0
			}
			switch scenario {
			case "creature":
				g.DealDamageToPermanent(g.FindPermanent(other), 2, id)
			case "controller":
				g.DealDamageToPlayer(a, 2, id)
			case "zero":
				g.DealDamageToPlayer(b, 0, id)
			case "other source":
				g.DealDamageToPlayer(b, 2, other)
			default:
				g.DealDamageToPlayer(b, 2, id)
			}
			if scenario == "two events" {
				g.ChooseTarget(gametest.PlayerA, choice)
				g.DealDamageToPlayer(b, 1, id)
				want = 3
			}
			if scenario == "source exit" || scenario == "source return" {
				card := g.FindPermanent(id).Card
				g.RemoveFromBattlefield(g.FindPermanent(id))
				if scenario == "source return" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			}
			g.PutTriggersOnStack()
			stacks := 1
			if want == 0 {
				stacks = 0
			}
			if scenario == "two events" {
				stacks = 2
			}
			if g.StackSize() != stacks {
				t.Fatalf("stack=%d, want %d", g.StackSize(), stacks)
			}
			if len(a.Graveyard()) != 0 || len(b.Graveyard()) != 0 {
				t.Fatal("milled before resolution")
			}
			if stacks > 0 {
				original := g.StackPeek()
				if len(original.Targets) != 1 || original.Targets[0] != target || original.Controller != a.PlayerID() {
					t.Fatal("wrong trigger target or controller")
				}
				if scenario == "copy" {
					g.ChooseTarget(gametest.PlayerB, "PlayerA")
					cp := g.CopyStackObjectDirect(original, b.PlayerID(), true)
					if cp == nil || cp.Targets[0] != a.PlayerID() || original.Targets[0] != b.PlayerID() {
						t.Fatal("copy targets are not independent")
					}
					g.ResolveTopOfStack()
					if len(a.Graveyard()) != 2 || len(b.Graveyard()) != 0 {
						t.Fatal("copy lost the damage amount or target")
					}
				}
				if scenario == "invalid target" {
					original.Targets[0] = other
					want = 0
				}
			}
			g.ResolveStack()
			if len(g.Game.GetPlayer(target).Graveyard()) != want {
				t.Fatalf("milled %d, want %d", len(g.Game.GetPlayer(target).Graveyard()), want)
			}
			if want > 0 {
				player := gametest.PlayerB
				if target == a.PlayerID() {
					player = gametest.PlayerA
				}
				g.AssertGraveyardCount(player, "Mountain", 1)
				if want > 1 {
					g.AssertGraveyardCount(player, "Forest", 1)
				}
			}
		})
	}
}

func TestToweringWaveMysticCombat(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		t.Run(map[bool]string{false: "player", true: "creature"}[blocked], func(t *testing.T) {
			g := gametest.NewTestGame(t)
			g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Towering-Wave Mystic")
			g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Mountain", 5)
			if blocked {
				g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
				g.Block(1, gametest.PlayerB, "Grizzly Bears", "Towering-Wave Mystic")
			}
			g.ChooseTarget(gametest.PlayerA, "PlayerB")
			g.Attack(1, gametest.PlayerA, "Towering-Wave Mystic")
			g.StopAt(1, core.EndStep)
			g.Execute()
			g.AssertGraveyardCount(gametest.PlayerB, "Mountain", 2)
			life := 18
			if blocked {
				life = 20
				g.AssertGraveyardCount(gametest.PlayerA, "Towering-Wave Mystic", 1)
			}
			g.AssertLife(gametest.PlayerB, life)
		})
	}
}
