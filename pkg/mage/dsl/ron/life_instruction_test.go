package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLifeInstructionRealCards(t *testing.T) {
	for _, name := range []string{"Healing Salve", "Stream of Life", "Alabaster Potion"} {
		sets := []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"}
		if name == "Alabaster Potion" {
			sets = []string{"LEG"}
		}
		for _, set := range sets {
			for _, state := range []string{"normal", "copy", "clone", "zero", "unpaid"} {
				t.Run(name+"/"+set+"/"+state, func(t *testing.T) {
					g, a, b := ronGame()
					card := repositoryCard(t, set, name)
					card.SetOwner(a.PlayerID())
					a.AddToHand(card)
					x, want := 4, 4
					if name == "Healing Salve" {
						x, want = 0, 3
					}
					if state == "zero" && name != "Healing Salve" {
						x, want = 0, 0
					}
					if state == "unpaid" {
						if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, x); err == nil {
							t.Fatal("unpaid spell accepted")
						}
						if g.StackSize() != 0 || len(a.Hand()) != 1 || b.Life() != 20 {
							t.Fatal("failed casting changed state")
						}
						return
					}
					for _, color := range []core.Color{core.White, core.Green, core.Colorless} {
						a.ManaPool().Add(color, 20)
					}
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, x); err != nil {
						t.Fatal(err)
					}
					obj := g.StackPeek()
					instructions, ok := mage.InspectLifeInstructions(mage.Sequence(obj.Effects...))
					if !ok || len(instructions) != 1 || !instructions[0].Gain {
						t.Fatal("RON lost native life instruction")
					}
					if state == "copy" {
						obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
						if obj == nil {
							t.Fatal("copy failed")
						}
					}
					for range 2 {
						if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
							t.Fatal("life tree has no pure result")
						}
					}
					if b.Life() != 20 || a.Life() != 20 {
						t.Fatal("inspection changed life")
					}
					if state == "clone" {
						clone := g.Clone()
						clone.ResolveTopOfStack()
						if clone.GetPlayer(b.PlayerID()).Life() != 20+want || b.Life() != 20 {
							t.Fatal("clone life state crossed")
						}
					}
					g.ResolveTopOfStack()
					if b.Life() != 20+want || a.Life() != 20 {
						t.Fatal("wrong life recipient or amount")
					}
				})
			}
		}
	}
}

func TestRONLifeInstructionScopes(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,LoseLife(Integer(2))),PlayerAction(You,GainLife(Integer(1)))])))`)
	g, a, b := ronGame()
	first := castRONSpell(t, g, a, factory(), b.PlayerID())
	second := castRONSpell(t, g, a, factory(), a.PlayerID())
	for _, obj := range []*mage.StackObject{first, second} {
		if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
			t.Fatal("life scopes unknown")
		}
	}
	g.AddPreventionShield(b.PlayerID(), 10)
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 18 {
		t.Fatal("instance scopes or life loss used damage prevention")
	}
}

func TestRONLifeInstructionInvalid(t *testing.T) {
	for _, body := range []string{`GainLife()`, `GainLife(Integer(1),Integer(2))`, `LoseLife(Integer())`, `PlayerAction(Ref_TargetPlayer,GainLife(Integer(1)))`, `PlayerAction(ThisPermanent,LoseLife(Integer(1)))`, `GainLife(LifeTotalOfPlayer(LoseLife_ThatPlayer))`} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Invalid life",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+body+`]))])`)); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}
