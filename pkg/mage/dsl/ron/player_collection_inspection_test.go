package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPlayerCollectionInspectionSyphonSoul(t *testing.T) {
	for _, state := range []string{"normal", "partial", "prevented", "copy"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Syphon Soul")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast accepted")
			}
			a.ManaPool().Add(core.Black, 3)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			prevention := 0
			if state == "partial" {
				prevention = 1
			}
			if state == "prevented" {
				prevention = 2
			}
			if prevention > 0 {
				g.AddPreventionShield(b.PlayerID(), prevention)
			}
			if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				for range 2 {
					certain, known := game.InspectSpellDestruction(game.StackPeek(), nil)
					if certain || !known {
						t.Fatalf("complete Syphon Soul query=%v/%v", certain, known)
					}
				}
				if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatal("query resolved damage")
				}
			}
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone shared life")
			}
			g.ResolveStack()
			wantA, wantB := 22-prevention, 18+prevention
			if state == "copy" {
				wantA, wantB = 20, 20
			}
			for _, game := range []*mage.Game{g, clone} {
				if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != wantB {
					t.Fatal("wrong actual damage, saved total or controller")
				}
			}
		})
	}
}

func TestRONPlayerCollectionInspectionMixed(t *testing.T) {
	for _, players := range []string{`Opponent`, `Other(You)`, `Other(TheActivePlayer)`} {
		source := `Card(Name:"Mixed opponents",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([SpellDealsDamage(ThisSpell,Integer(2),MultipleRecipients([EachPlayer(` + players + `),EachPermanent(IsCardtype(Creature))]))]))])`
		factory, err := Compile(nameScopeNode(t, source))
		if err != nil {
			t.Fatal(err)
		}
		g, a, b := ronGame()
		p := g.PutOnBattlefield(mage.NewCreature("member", "", 1, 5), a.PlayerID())
		card := factory()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		wantA, wantB := 20, 18
		if players == `Other(TheActivePlayer)` {
			g.SetActivePlayerIndex(1)
			wantA, wantB = 18, 20
		}
		certain, known := g.InspectSpellDestruction(g.StackPeek(), nil)
		if certain || !known {
			t.Fatal("mixed opponent query unknown")
		}
		g.ResolveStack()
		if a.Life() != wantA || b.Life() != wantB || g.FindPermanent(p.ID()).Damage != 2 {
			t.Fatal("mixed recipient execution")
		}
	}
}

func TestRONPlayerCollectionInspectionInvalid(t *testing.T) {
	for _, players := range []string{`Opponent()`, `Other()`, `Other(You,You)`, `Other(Ref_TargetPlayer)`, `AnyPlayer()`} {
		source := `Card(Name:"Invalid players",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([SpellDealsDamage(ThisSpell,Integer(2),EachPlayer(` + players + `)),GainLife(TheAmountOfDamageDealtThisWay)]))])`
		if _, err := Compile(nameScopeNode(t, source)); err == nil {
			t.Fatal("invalid collection accepted", players)
		}
	}
}

func TestRONPlayerCollectionInspectionTarget(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		recipients := `EachPlayer(Other(Ref_TargetPlayer))`
		if mixed {
			recipients = `MultipleRecipients([Player(Ref_TargetPlayer),EachPermanent(IsCardtype(Creature))])`
		}
		continuation := ""
		if !mixed {
			continuation = `,GainLife(TheAmountOfDamageDealtThisWay)`
		}
		factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([SpellDealsDamage(ThisSpell,Integer(2),`+recipients+`)`+continuation+`])))`)
		for range 2 {
			g, a, b := ronGame()
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
				t.Fatal(err)
			}
			if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			for _, game := range []*mage.Game{g, g.Clone()} {
				certain, known := game.InspectSpellDestruction(game.StackPeek(), nil)
				if certain || !known {
					t.Fatal("bound player collection unknown")
				}
				game.ResolveStack()
				wantA, wantB := 18, 22
				if mixed {
					wantA, wantB = 20, 16
				}
				if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != wantB {
					t.Fatal("target declaration changed with copy controller or factory")
				}
			}
		}
	}
}
