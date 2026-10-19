package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONFuturePreventionInstructionRealCards(t *testing.T) {
	for _, name := range []string{"Healing Salve", "Alabaster Potion"} {
		sets := []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"}
		if name == "Alabaster Potion" {
			sets = []string{"LEG"}
		}
		for _, set := range sets {
			for _, state := range []string{"player", "permanent", "copy", "clone", "zero", "illegal", "unpaid"} {
				t.Run(name+"/"+set+"/"+state, func(t *testing.T) {
					g, a, b := ronModalGame(1)
					victim := g.PutOnBattlefield(mage.NewCreature("Recipient", "", 1, 20), b.PlayerID())
					id := b.PlayerID()
					if state == "permanent" || state == "illegal" {
						id = victim.ID()
					}
					card := repositoryCard(t, set, name)
					card.SetOwner(a.PlayerID())
					a.AddToHand(card)
					x, budget, mana := 4, 4, 6
					if name == "Healing Salve" {
						x, budget, mana = 0, 3, 1
					}
					if state == "zero" && name == "Alabaster Potion" {
						x, budget, mana = 0, 0, 2
					}
					if state == "unpaid" {
						if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{id}, x); err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 {
							t.Fatal("failed casting changed state")
						}
						return
					}
					a.ManaPool().Add(core.White, mana)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{id}, x); err != nil {
						t.Fatal(err)
					}
					obj := g.StackPeek()
					instructions, ok := mage.InspectFuturePreventionInstructions(mage.Sequence(obj.Effects...))
					if !ok || len(instructions) != 1 {
						t.Fatal("RON lost native prevention instruction")
					}
					if state == "copy" {
						obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
						if obj == nil {
							t.Fatal("copy failed")
						}
					}
					if state == "illegal" {
						g.FindPermanent(id).GrantBaseAttr(core.Shroud)
					}
					for _, game := range []*mage.Game{g, g.Clone()} {
						for range 2 {
							if certain, known := game.InspectSpellDestruction(game.StackPeek(), nil); certain || !known {
								t.Fatal("prevention tree has no pure result")
							}
						}
					}
					g.DealDamageToPlayer(b, 1, uuid.New())
					if b.Life() != 19 {
						t.Fatal("inspection installed shield")
					}
					if state == "clone" {
						clone := g.Clone()
						clone.ResolveTopOfStack()
						clone.DealDamageToPlayer(clone.GetPlayer(b.PlayerID()), budget+1, uuid.New())
						if clone.GetPlayer(b.PlayerID()).Life() != 18 || b.Life() != 19 {
							t.Fatal("clone shared shield state")
						}
					}
					g.ResolveTopOfStack()
					if id == victim.ID() {
						g.DealDamageToPermanent(g.FindPermanent(id), budget+1, uuid.New())
						want := 1
						if state == "illegal" {
							want = budget + 1
						}
						if g.FindPermanent(id).Damage != want {
							t.Fatal("wrong permanent shield or target legality")
						}
					} else {
						g.DealDamageToPlayer(b, budget+1, uuid.New())
						if b.Life() != 18 || a.Life() != 20 {
							t.Fatal("wrong player shield or amount")
						}
					}
				})
			}
		}
	}
}

func TestRONFuturePreventionInstructionInvalid(t *testing.T) {
	for _, body := range []string{
		`CreateFuturePreventDamage()`,
		`CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Integer(-1),Player(You)),[PreventThatDamage])`,
		`CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(ValueX,Player(You)),[PreventThatDamage])`,
		`CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Integer(1),Ref_AnyTarget),[PreventThatDamage])`,
		`CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Integer(1),Player(You)),[PreventThatDamage()])`,
		`CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Integer(1),Player(You)),[PreventThatDamage,PreventThatDamage])`,
	} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Invalid shield",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+body+`]))])`)); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONFuturePreventionInstructionScopes(t *testing.T) {
	g, a, b := ronModalGame(1)
	first := castRONSpell(t, g, a.BasePlayer, repositoryCard(t, "LEA", "Healing Salve"), a.PlayerID())
	second := castRONSpell(t, g, a.BasePlayer, repositoryCard(t, "LEA", "Healing Salve"), b.PlayerID())
	one, ok := mage.InspectFuturePreventionInstructions(mage.Sequence(first.Effects...))
	if !ok || len(one) != 1 {
		t.Fatal("first declaration missing")
	}
	two, ok := mage.InspectFuturePreventionInstructions(mage.Sequence(second.Effects...))
	if !ok || len(two) != 1 || one[0].Recipient == two[0].Recipient {
		t.Fatal("factories share target declaration")
	}
	g.AddPreventionShield(a.PlayerID(), 2)
	for range 2 {
		for _, obj := range []*mage.StackObject{first, second} {
			if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
				t.Fatal("independent target scope is unknown")
			}
		}
	}
	g.DealDamageToPlayer(a, 3, uuid.New())
	if a.Life() != 19 {
		t.Fatal("inspection changed an existing shield")
	}
	g.ResolveStack()
	g.DealDamageToPlayer(a, 4, uuid.New())
	g.DealDamageToPlayer(b, 4, uuid.New())
	if a.Life() != 18 || b.Life() != 19 {
		t.Fatal("target scopes crossed")
	}
}
