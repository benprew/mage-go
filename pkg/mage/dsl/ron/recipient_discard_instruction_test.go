package ron

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONRecipientDiscardInstructionMindBomb(t *testing.T) {
	for _, mode := range []string{"normal", "empty", "prevention", "copy", "clone"} {
		t.Run(mode, func(t *testing.T) {
			a := &ronOptionalDiscardPlayer{BasePlayer: mage.NewBasePlayer("A"), count: 1}
			b := &ronOptionalDiscardPlayer{BasePlayer: mage.NewBasePlayer("B"), count: 2}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			if mode != "empty" {
				for range 3 {
					a.AddToHand(mage.NewInstant("A", ""))
					b.AddToHand(mage.NewInstant("B", ""))
				}
			}
			obj := castRONSpell(t, g, a.BasePlayer, repositoryCard(t, "DRK", "Mind Bomb"))
			if mode == "copy" {
				obj = g.CopySpellOnStack(obj.Card.ID(), b.PlayerID(), false)
			}
			if mode == "prevention" {
				g.AddPreventionShield(b.PlayerID(), 1)
			}
			g.PutOnBattlefield(mage.NewLand("Watched"), a.PlayerID())
			for range 2 {
				certain, known := g.InspectSpellDestruction(obj, nil)
				if certain || !known {
					t.Fatal("Mind Bomb lacks native inspection")
				}
			}
			if a.requests != 0 || b.requests != 0 || a.Life() != 20 || b.Life() != 20 {
				t.Fatal("query executed choices")
			}
			if mode == "clone" {
				clone := g.Clone()
				if certain, known := clone.InspectSpellDestruction(clone.StackPeek(), nil); certain || !known {
					t.Fatal("clone lost tree")
				}
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != 17 || a.Life() != 20 {
					t.Fatal("clone isolation")
				}
			}
			g.ResolveTopOfStack()
			wantA, wantB := 18, 19
			if mode == "empty" {
				wantA, wantB = 17, 17
			}
			if mode == "prevention" {
				wantB = 20
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("life %d/%d want %d/%d", a.Life(), b.Life(), wantA, wantB)
			}
		})
	}
}

func TestRONRecipientDiscardInstructionBindings(t *testing.T) {
	factory, err := Compile(repositoryRecord(t, "DRK", "Mind Bomb", "bomb.ron"))
	if err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronGame()
	first := castRONSpell(t, g, a, factory())
	if certain, known := g.InspectSpellDestruction(first, nil); certain || !known {
		t.Fatal("first factory binding lost")
	}
	g.ResolveStack()
	second := castRONSpell(t, g, a, factory())
	for _, obj := range []*mage.StackObject{second} {
		if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
			t.Fatal("factory binding lost")
		}
	}
	g.ResolveStack()
	if a.Life() != 14 {
		t.Fatal("independent execution lost")
	}
}

func TestRONRecipientDiscardInstructionInvalid(t *testing.T) {
	producer := `EachPlayerAction(AnyPlayer,DiscardUptoNumberCards(Integer(3)))`
	damage := `SpellDealsDamage(ThisSpell,Minus(Integer(3),TheNumberOfCardsDiscardedByPlayerThisWay(DealsDamage_ThatPlayer)),EachPlayer(AnyPlayer))`
	for _, body := range []string{damage, damage + `,` + producer, producer + `,SpellDealsDamage(ThisSpell,Minus(Integer(3),TheNumberOfCardsDiscardedByPlayerThisWay()),EachPlayer(AnyPlayer))`, producer + `,SpellDealsDamage(ThisSpell,Minus(Integer(3),TheNumberOfCardsDiscardedByPlayerThisWay(Ref_TargetPermanent)),EachPlayer(AnyPlayer))`, producer + `,GainLife(TheNumberOfCardsDiscardedByPlayerThisWay(DealsDamage_ThatPlayer))`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad result",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([`+body+`]))])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		_, err = Compile(nodes[0])
		if err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}
