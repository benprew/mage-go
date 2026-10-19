package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONChosenManaInspectionMetamorphosis(t *testing.T) {
	for _, color := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green} {
		for _, state := range []string{"normal", "zero value", "exile", "return", "copy", "clone", "countered"} {
			t.Run(color.String()+"/"+state, func(t *testing.T) {
				a := &ronSacrificeManaChooser{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, color: color}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				card := repositoryCard(t, "ARN", "Metamorphosis")
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				cost, want := "{X}{3}{U}", 5
				if state == "zero value" {
					cost, want = "", 1
				}
				payment := mage.NewCreature("Paid", cost, 1, 1, mage.WithKeyword(core.Indestructible))
				payment.SetOwner(b.PlayerID())
				paid := g.PutOnBattlefield(payment, a.PlayerID())
				a.choice = paid.ID()
				if state == "exile" {
					g.AddExileIfWouldGoToGraveyardThisTurn(paid.ID(), paid.ID())
				}
				a.ManaPool().Add(core.Green, 1)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				if state == "return" {
					b.RemoveFromGraveyard(payment.ID())
					p := g.PutOnBattlefield(payment, b.PlayerID())
					g.PhaseOut(p.ID())
				}
				obj := g.StackPeek()
				if state == "copy" {
					obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
					if obj == nil {
						t.Fatal("copy failed")
					}
				}
				for range 2 {
					if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
						t.Fatal("paid chosen-color production unknown")
					}
				}
				if a.calls != 0 || a.ManaPool().TotalMana() != 0 || b.ManaPool().TotalMana() != 0 {
					t.Fatal("inspection made a choice or produced mana")
				}
				if state == "countered" {
					g.CounterStackObject(obj.ID)
					if a.calls != 0 || a.ManaPool().TotalMana() != 0 {
						t.Fatal("countered spell produced mana")
					}
					return
				}
				if state == "clone" {
					clone := g.Clone()
					clone.ResolveStack()
					if a.ManaPool().TotalMana() != 0 || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != want {
						t.Fatal("clone shares pool")
					}
				}
				g.ResolveStack()
				pool := a.ManaPool()
				if pool.Count(color) != want || a.calls == 0 || pool.CanPay(core.ManaCost{Generic: want}, nil) || pool.CanPay(core.ManaCost{Generic: want}, &core.SpellPaymentContext{IsArtifact: true}) || !pool.CanPay(core.ManaCost{Generic: want}, &core.SpellPaymentContext{IsCreature: true}) {
					t.Fatal("color, amount or restriction lost")
				}
				if state == "copy" && b.ManaPool().TotalMana() != want {
					t.Fatal("copy lost paid value or controller")
				}
			})
		}
	}
}

func TestRONChosenManaInspectionWorkshop(t *testing.T) {
	g, a, _ := ronGame()
	land := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Mishra's Workshop"), a.PlayerID())
	ability := mage.UnwrapAbility(land.Card.Abilities()[0]).(mage.ActivatedAbility)
	instructions, ok := mage.InspectManaProductionInstructions(mage.Sequence(ability.Effects()...))
	if !ok || len(instructions) != 3 {
		t.Fatal("restricted fixed native batch unknown")
	}
	for _, instruction := range instructions {
		if instruction.Color != core.Colorless || instruction.Subject != mage.You {
			t.Fatal("wrong production")
		}
	}
	if land.Tapped || a.ManaPool().TotalMana() != 0 {
		t.Fatal("inspection activated source")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if !land.Tapped || g.StackSize() != 0 || a.ManaPool().TotalMana() != 3 || a.ManaPool().CanPay(core.ManaCost{Generic: 3}, nil) {
		t.Fatal("execution changed")
	}
}

func TestRONChosenManaInspectionInvalid(t *testing.T) {
	for _, action := range []string{
		`AddManaRepeatedWithModifiers(AnyManaColor,Integer(2),CanOnlySpendOnSpells(IsCardtype(Land)))`,
		`AddManaRepeatedWithModifiers(AnyManaColor,Integer(2),CanOnlySpendOnSpells(IsCardtype(Creature)),Integer(1))`,
		`AddManaRepeatedWithModifiers(AnyManaColor,Plus(Integer(1),ManaValueOfTheSacrificedPermanent),CanOnlySpendOnSpells(IsCardtype(Creature)))`,
		`AddManaRepeatedWithModifiers(AnyManaColor,Integer(2),CanOnlySpendOnSpells(IsCardtype(Creature())))`,
		`AddManaWithModifiers(And([]),CanOnlySpendOnSpells(IsCardtype(Artifact)))`,
	} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+action+`]))])`)); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}
