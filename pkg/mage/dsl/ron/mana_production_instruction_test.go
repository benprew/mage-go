package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONManaProductionInstructionGameplay(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, state := range []string{"normal", "unpaid", "countered", "copy", "clone"} {
			t.Run(set+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, set, "Dark Ritual")
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if state != "unpaid" {
					a.ManaPool().Add(core.Black, 1)
				}
				err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0)
				if state == "unpaid" {
					if err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 || a.ManaPool().TotalMana() != 0 {
						t.Fatal("failed cast changed state")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				obj := g.StackPeek()
				if obj.IsAbility || g.StackSize() != 1 || a.ManaPool().TotalMana() != 0 {
					t.Fatal("mana spell bypassed stack")
				}
				instructions, ok := mage.InspectManaProductionInstructions(mage.Sequence(obj.Effects...))
				if !ok || len(instructions) != 3 {
					t.Fatal("RON lost native mana declarations")
				}
				for _, instruction := range instructions {
					if instruction.Color != core.Black || instruction.Subject != mage.You {
						t.Fatal("wrong native production")
					}
				}
				for range 2 {
					if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
						t.Fatal("complete Ritual query unknown")
					}
				}
				if a.ManaPool().TotalMana() != 0 || b.ManaPool().TotalMana() != 0 || g.StackSize() != 1 {
					t.Fatal("inspection produced mana")
				}
				switch state {
				case "countered":
					g.CounterStackObject(obj.ID)
					if a.ManaPool().TotalMana() != 0 || g.StackSize() != 0 {
						t.Fatal("countered Ritual produced mana")
					}
					return
				case "copy":
					obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
					if obj == nil {
						t.Fatal("copy failed")
					}
					if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
						t.Fatal("copy query unknown")
					}
					g.ResolveTopOfStack()
					if b.ManaPool().TotalMana() != 3 || a.ManaPool().TotalMana() != 0 {
						t.Fatal("copy used original controller")
					}
				case "clone":
					clone := g.Clone()
					clone.ResolveStack()
					if clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 3 || a.ManaPool().TotalMana() != 0 {
						t.Fatal("clone shared pool")
					}
				}
				g.ResolveStack()
				if a.ManaPool().TotalMana() != 3 || a.Life() != 20 || len(a.Graveyard()) != 1 {
					t.Fatal("wrong Ritual outcome")
				}
			})
		}
	}
}

func TestRONManaProductionInstructionComposition(t *testing.T) {
	for _, test := range []struct {
		body   string
		colors []core.Color
		amount int
	}{
		{`AddMana(And([ManaProduceW,And([ManaProduceU,ManaProduceC])]))`, []core.Color{core.White, core.Blue, core.Colorless}, 1},
		{`AddManaRepeated(ManaProduceR,Integer(0))`, []core.Color{core.Red}, 0},
		{`AddManaRepeated(ManaProduceG,Integer(3))`, []core.Color{core.Green}, 3},
	} {
		factory := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([`+test.body+`]))`)
		g, a, _ := ronGame()
		obj := castRONSpell(t, g, a, factory())
		a.ManaPool().Clear()
		if _, ok := mage.InspectManaProductionInstructions(mage.Sequence(obj.Effects...)); !ok {
			t.Fatal("native lowering lost")
		}
		if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
			t.Fatal("fixed production unknown")
		}
		g.ResolveStack()
		if a.ManaPool().TotalMana() != len(test.colors)*test.amount {
			t.Fatal("wrong production amount")
		}
		for _, color := range test.colors {
			if a.ManaPool().Count(color) != test.amount {
				t.Fatal("wrong production color")
			}
		}
	}
}

func TestRONManaProductionInstructionOrder(t *testing.T) {
	for _, prefix := range []bool{false, true} {
		actions := `DestroyPermanent(Ref_TargetPermanent),AddMana(ManaProduceB)`
		if prefix {
			actions = `AddMana(ManaProduceB),DestroyPermanent(Ref_TargetPermanent)`
		}
		factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Land))],ActionList([`+actions+`])))`)
		g, a, b := ronGame()
		land := g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
		predicate, _ := mage.InspectPermanentCollection(mage.MatchingPermanents(mage.AnyPermanent))
		watched := predicate.Select(g)
		obj := castRONSpell(t, g, a, factory(), land.ID())
		a.ManaPool().Clear()
		certain, known := g.InspectSpellDestruction(obj, watched)
		if certain != !prefix || known != !prefix {
			t.Fatalf("prefix=%v proof=%v/%v", prefix, certain, known)
		}
		g.ResolveStack()
		if a.ManaPool().Count(core.Black) != 1 || g.FindPermanent(land.ID()) != nil {
			t.Fatal("ordered execution differs")
		}
	}
}

func TestRONManaProductionInstructionInvalid(t *testing.T) {
	for _, body := range []string{`AddMana()`, `AddMana(ManaProduceB,ManaProduceB)`, `AddMana("ManaProduceB")`, `AddMana(ManaProduceB(1))`, `AddMana(ManaProducePurple)`, `AddManaRepeated(ManaProduceB,Integer(-1))`, `AddManaRepeated(ManaProduceB,Integer("3"))`, `AddManaRepeated(ManaProduceB,TheNumberOfCountersRemovedThisWay)`} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Invalid mana",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+body+`]))])`)); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}
