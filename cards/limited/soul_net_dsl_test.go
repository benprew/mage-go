package limited

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestSoulNetTypedDefinition(t *testing.T) {
	card, err := mage.CreateCard("Soul Net")
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	var event, payment bool
	var visit func(mage.DefinitionMetadata)
	visit = func(n mage.DefinitionMetadata) {
		event = event || n.Role == "event-pattern" && n.Text == "a creature dies"
		payment = payment || n.Role == "if-paid" && n.Properties.LifeGain == 1
		for _, child := range n.Children {
			visit(child)
		}
	}
	for _, ability := range card.Abilities() {
		visit(mage.InspectDefinition(ability))
	}
	if !event || !payment {
		t.Fatalf("typed event=%v payment=%v", event, payment)
	}
}

func TestSoulNetTypedPayment(t *testing.T) {
	for _, scenario := range []string{"pay", "decline", "unaffordable", "own", "token", "two deaths", "two paid deaths", "noncreature", "exile", "control", "departed", "copy", "clone", "phased"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
			source := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Soul Net")
			var land uuid.UUID
			if scenario != "unaffordable" {
				land = g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
			}
			var secondLand uuid.UUID
			if scenario == "two paid deaths" {
				secondLand = g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
			}
			opposingLand := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
			owner, name := gametest.PlayerB, "Grizzly Bears"
			if scenario == "own" {
				owner = gametest.PlayerA
			}
			if scenario == "noncreature" {
				name = "Sol Ring"
			}
			victim := g.AddCard(core.ZoneBattlefield, owner, name)
			if scenario == "token" {
				token := mage.NewToken("Victim", 1, 1, []core.CardType{core.TypeCreature}, nil)
				token.SetOwner(b.PlayerID())
				g.PutOnBattlefield(token, b.PlayerID())
				victim = token.ID()
			}
			if scenario == "phased" {
				g.PhaseOut(source)
			}
			count := 1
			switch scenario {
			case "two deaths", "two paid deaths":
				other := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
				g.MutablePermanent(victim).Damage = 10
				g.MutablePermanent(other).Damage = 10
				g.CheckStateBasedActions()
				count = 2
			case "exile":
				g.ExilePermanent(g.FindPermanent(victim))
				count = 0
			default:
				g.DestroyPermanent(g.FindPermanent(victim))
				if scenario == "noncreature" || scenario == "phased" {
					count = 0
				}
			}
			g.PutTriggersOnStack()
			if g.StackSize() != count || a.Life() != 20 {
				t.Fatalf("before resolution: stack=%d life=%d", g.StackSize(), a.Life())
			}
			if land != uuid.Nil && g.FindPermanent(land).Tapped {
				t.Fatal("paid before resolution")
			}
			switch scenario {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: uuid.New(), TargetID: source, ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "departed":
				g.ExilePermanent(g.FindPermanent(source))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "clone":
				clone := g.Clone()
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.FindPermanent(land).Tapped {
					t.Fatal("search clone did not decline payment independently")
				}
				if a.Life() != 20 || g.FindPermanent(land).Tapped {
					t.Fatal("clone changed original")
				}
			}
			if scenario == "decline" {
				a.QueueMayAbilityChoices(false)
			}
			g.ResolveStack()
			gain := count > 0 && scenario != "decline" && scenario != "unaffordable"
			want := 20
			if gain {
				want++
			}
			if scenario == "two paid deaths" {
				want++
				if !g.FindPermanent(secondLand).Tapped {
					t.Fatal("second trigger did not pay")
				}
			}
			if a.Life() != want {
				t.Fatalf("life=%d want=%d", a.Life(), want)
			}
			if land != uuid.Nil && g.FindPermanent(land).Tapped != gain {
				t.Fatal("incorrect payment")
			}
			wantB := 20
			if scenario == "copy" {
				wantB++
			}
			if b.Life() != wantB || g.FindPermanent(opposingLand).Tapped != (scenario == "copy") {
				t.Fatal("incorrect payer or beneficiary")
			}
		})
	}
}
