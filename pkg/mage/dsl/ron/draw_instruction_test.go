package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDrawInstructionRealCards(t *testing.T) {
	for _, name := range []string{"Ancestral Recall", "Braingeyser"} {
		sets := []string{"LEA", "LEB", "2ED", "CED", "CEI"}
		if name == "Braingeyser" {
			sets = append(sets, "3ED")
		}
		for _, set := range sets {
			for _, state := range []string{"normal", "copy", "clone", "zero", "unpaid", "replacement", "retarget", "self"} {
				t.Run(name+"/"+set+"/"+state, func(t *testing.T) {
					g, a, b := ronGame()
					seedRONDrawLibrary(a)
					seedRONDrawLibrary(b)
					card := repositoryCard(t, set, name)
					card.SetOwner(a.PlayerID())
					a.AddToHand(card)
					x, want := 3, 3
					if name == "Ancestral Recall" {
						x = 0
					} else if state == "zero" {
						x, want = 0, 0
					}
					target := b.PlayerID()
					if state == "self" {
						target = a.PlayerID()
					}
					if state == "unpaid" {
						if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, x); err == nil {
							t.Fatal("unpaid spell accepted")
						}
						if g.StackSize() != 0 || len(a.Hand()) != 1 || len(b.Hand()) != 0 {
							t.Fatal("failed casting changed state")
						}
						return
					}
					a.ManaPool().Add(core.Blue, 10)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, x); err != nil {
						t.Fatal(err)
					}
					obj := g.StackPeek()
					instructions, ok := mage.InspectDrawInstructions(mage.Sequence(obj.Effects...))
					if !ok || len(instructions) != 1 {
						t.Fatal("RON lost native draw instruction")
					}
					if state == "copy" || state == "retarget" {
						obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
						if obj == nil {
							t.Fatal("copy failed")
						}
						if state == "retarget" {
							obj.Targets[0] = a.PlayerID()
							target = a.PlayerID()
						}
					}
					if state == "replacement" {
						if err := mage.ApplyEffect(g, mage.ReplaceNextDrawThisTurn(mage.You, mage.GainLifeFor(mage.You, mage.Fixed(2))), uuid.Nil, b.PlayerID(), nil); err != nil {
							t.Fatal(err)
						}
						want--
					}
					for _, game := range []*mage.Game{g, g.Clone()} {
						for range 2 {
							if certain, known := game.InspectSpellDestruction(game.StackPeek(), nil); certain || !known {
								t.Fatal("draw tree has no pure result")
							}
						}
						if len(game.GetPlayer(target).Hand()) != 0 || game.GetPlayer(b.PlayerID()).Life() != 20 {
							t.Fatal("inspection consumed a draw or replacement")
						}
						game.ResolveTopOfStack()
						if len(game.GetPlayer(target).Hand()) != want {
							t.Fatal("wrong draw recipient or amount")
						}
						if state == "replacement" && game.GetPlayer(b.PlayerID()).Life() != 22 {
							t.Fatal("inspection consumed replacement or execution bypassed it")
						}
					}
				})
			}
		}
	}
}

func TestRONDrawInstructionScopes(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,DrawACard),PlayerAction(You,DrawNumberCards(Integer(2)))])))`)
	g, a, b := ronGame()
	seedRONDrawLibrary(a)
	seedRONDrawLibrary(b)
	first := castRONSpell(t, g, a, factory(), b.PlayerID())
	second := castRONSpell(t, g, b, factory(), a.PlayerID())
	for _, obj := range []*mage.StackObject{first, second} {
		if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
			t.Fatal("draw scopes unknown")
		}
	}
	g.ResolveStack()
	if len(a.Hand()) != 3 || len(b.Hand()) != 3 {
		t.Fatal("factory or resolving controller scopes crossed")
	}
}

func TestRONDrawInstructionInvalid(t *testing.T) {
	for _, body := range []string{`DrawACard()`, `DrawNumberCards()`, `DrawNumberCards(Integer(1),Integer(2))`, `DrawNumberCards(Integer())`, `PlayerAction(Ref_TargetPlayer,DrawACard)`, `PlayerAction(ThisPermanent,DrawACard)`, `DrawNumberCards(TheNumberOfCardsDrawnThisWay)`} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Invalid draw",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+body+`]))])`)); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONDrawInstructionSavedResultBoundary(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([DrawACard,DrawACard,DiscardTheCardDrawnThisWay]))`)
	g, a, _ := ronGame()
	seedRONDrawLibrary(a)
	obj := castRONSpell(t, g, a, factory())
	if _, ok := mage.InspectDrawInstructions(mage.Sequence(obj.Effects...)); ok {
		t.Fatal("saved draw result accepted as a plain draw tree")
	}
	if certain, known := g.InspectSpellDestruction(obj, nil); certain || known {
		t.Fatal("saved draw continuation accepted by bounded spell query")
	}
	g.ResolveStack()
	if len(a.Hand()) != 1 || len(a.Graveyard()) != 2 {
		t.Fatal("draw result scope changed")
	}
}
