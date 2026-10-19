package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestRONNativeWholeHandDiscardLowering(t *testing.T) {
	for _, tc := range []struct{ set, name string }{
		{"LEA", "Wheel of Fortune"}, {"LEB", "Wheel of Fortune"}, {"2ED", "Wheel of Fortune"}, {"3ED", "Wheel of Fortune"}, {"CED", "Wheel of Fortune"}, {"CEI", "Wheel of Fortune"},
		{"LEA", "Contract from Below"}, {"LEB", "Contract from Below"}, {"2ED", "Contract from Below"}, {"3ED", "Contract from Below"}, {"CED", "Contract from Below"}, {"CEI", "Contract from Below"},
		{"LEG", "Knowledge Vault"}, {"LEG", "Nicol Bolas"},
	} {
		t.Run(tc.set+"/"+tc.name, func(t *testing.T) {
			record := repositoryRecord(t, tc.set, tc.name, "whole-hand.ron")
			action := contextualQueryNode(t, record, "DiscardHand")
			builds, err := compileHandPlayerAction(action, scope{resolving: true})
			if tc.name == "Wheel of Fortune" {
				builds, err = compileEachPlayerAction(contextualQueryNode(t, record, "EachPlayerAction"), scope{resolving: true})
			}
			if err != nil {
				t.Fatal(err)
			}
			declaration, ok := mage.InspectWholeHandDiscardInstruction(builds[0](instanceBindings{}))
			if !ok {
				t.Fatal("lost native declaration")
			}
			if tc.name == "Wheel of Fortune" && declaration.Subjects != mage.EachPlayer {
				t.Fatal("lost collection")
			}
			if tc.name != "Wheel of Fortune" && declaration.Subject == nil {
				t.Fatal("lost singular player")
			}
			g, a, b := ronGame()
			randomRONHand(t, a, 2)
			randomRONHand(t, b, 3)
			if err := mage.ApplyEffect(g, declaration.Effect(), a.PlayerID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			wantB := 3
			if tc.name == "Wheel of Fortune" {
				wantB = 0
			}
			if len(a.Hand()) != 0 || len(b.Hand()) != wantB {
				t.Fatal("wrong discarded hand")
			}
		})
	}
}

func TestRONNativeWholeHandDiscardQuery(t *testing.T) {
	for _, action := range []string{"DiscardHand", "EachPlayerAction(AnyPlayer,DiscardHand)", "PlayerAction(You,DiscardHand)", "PlayerAction(TheActivePlayer,DiscardHand)"} {
		for _, state := range []string{"only", "prefix", "suffix", "clone", "copy"} {
			t.Run(action+"/"+state, func(t *testing.T) {
				body := action
				if state == "prefix" {
					body += ",DestroyEachPermanent(IsCardtype(Land))"
				}
				if state == "suffix" {
					body = "DestroyEachPermanent(IsCardtype(Land))," + body
				}
				card := permanentScopeFactory(t, "Sorcery", "SpellActions(ActionList(["+body+"]))")()
				g, a, b := ronGame()
				obj := castRONSpell(t, g, a, card)
				randomRONHand(t, a, 2)
				randomRONHand(t, b, 3)
				if state == "clone" {
					g = g.Clone()
					obj = g.StackPeek()
				}
				if state == "copy" {
					obj = g.CopyStackObjectDirect(obj, a.PlayerID(), false)
				}
				for range 2 {
					certain, known := g.InspectSpellDestruction(obj, nil)
					if certain || known != (state != "prefix") {
						t.Fatalf("query=%v/%v", certain, known)
					}
				}
				if len(g.GetPlayer(a.PlayerID()).Hand()) != 2 || len(g.GetPlayer(b.PlayerID()).Hand()) != 3 {
					t.Fatal("query discarded cards")
				}
				g.ResolveTopOfStack()
				wantB := 3
				if strings.HasPrefix(action, "EachPlayerAction") {
					wantB = 0
				}
				if len(g.GetPlayer(a.PlayerID()).Hand()) != 0 || len(g.GetPlayer(b.PlayerID()).Hand()) != wantB {
					t.Fatal("wrong hand discarded at resolution")
				}
			})
		}
	}
}

func TestRONNativeWholeHandDiscardInvalid(t *testing.T) {
	for _, action := range []string{
		"DiscardHand()", "DiscardHand(You)", "PlayerAction(You,DiscardHand())",
		"EachPlayerAction(AnyPlayer,DiscardHand())", "PlayerAction(ActionPlayer,DiscardHand)",
		"PlayerAction(Trigger_ThatPlayer,DiscardHand)", "PlayerAction(Ref_TargetPlayer,DiscardHand)",
		"EachPlayerAction(AnyPlayer,DiscardHand),PlayerAction(ActionPlayer,DiscardHand)",
	} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([`+action+`]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}

func TestRONNativeWholeHandDiscardTargets(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,DiscardHand)])))`)
	first, second := factory(), factory()
	if first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("factory shared ability state")
	}
	g, a, b := ronGame()
	for i, card := range []mage.Card{first, second} {
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		target := b.PlayerID()
		if i == 1 {
			target = a.PlayerID()
		}
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
			t.Fatal(err)
		}
	}
	randomRONHand(t, a, 2)
	randomRONHand(t, b, 3)
	obj := g.StackPeek()
	if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
		t.Fatal("exact target unknown")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if len(a.Hand()) != 2 || len(b.Hand()) != 3 {
		t.Fatal("clone changed original hands")
	}
	g.ResolveTopOfStack()
	if len(a.Hand()) != 0 || len(b.Hand()) != 3 {
		t.Fatal("target binding leaked between instances")
	}
	if certain, known := g.InspectSpellDestruction(g.StackPeek(), nil); certain || !known {
		t.Fatal("second instance lost its target")
	}
	g.ResolveStack()
	if len(a.Hand()) != 0 || len(b.Hand()) != 0 || len(a.Graveyard()) != 4 || len(b.Graveyard()) != 3 {
		t.Fatal("discard lost its target binding")
	}
}
