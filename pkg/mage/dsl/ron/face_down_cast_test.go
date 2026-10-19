package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONFaceDownCast(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			record := repositoryRecord(t, set, "Illusionary Mask", set+".ron")
			if _, err := Compile(record); err != nil {
				t.Fatalf("complete Mask record: %v", err)
			}
			rule := record.Fields["Rules"].Args[0]
			branch := &rule.Args[1].Args[0].Args[1].Args[1]
			branch.Args = branch.Args[:1]
			record.Fields["Rules"] = Node{Kind: List, Args: []Node{rule}}
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			g.SetPlayerAt(0, &ronHandChoicePlayer{BasePlayer: a, accept: true})
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			g.SetStep(core.PrecombatMain)
			creature := mage.NewCreature("Hidden", "{U}", 5, 5, mage.WithKeyword(core.Flying))
			a.AddToHand(creature)
			a.ManaPool().Add(core.Blue, 1)
			g.SetXValue(1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.RemoveFromBattlefield(source)
			g.ResolveTopOfStack()
			if g.StackSize() != 1 || len(a.Hand()) != 0 || g.FindPermanent(creature.ID()) != nil {
				t.Fatal("did not cast during resolution")
			}
			obj := g.StackObjects()[0]
			if !obj.IsFaceDown() || obj.Card.Name() != "" || g.InspectFaceDownObject(b.PlayerID(), obj.ID) != nil {
				t.Fatal("spell leaked identity")
			}
			g.ResolveStack()
			p := g.FindPermanent(creature.ID())
			if p == nil || !p.FaceDown || p.CurrentPower(g) != 2 || p.HasAttr(core.Flying) {
				t.Fatal("wrong entry")
			}
		})
	}
}

func TestRONFaceDownCastInvalid(t *testing.T) {
	for _, action := range []string{
		`CastCardInHandWithoutPayingAsAFacedownCreatureSpell(TheCardInHandChosenThisWay)`,
		`CastCardInHandWithoutPayingAsAFacedownCreatureSpell(TheCardInHandChosenThisWay,PT(3,3))`,
		`CastCardInHandWithoutPayingAsAFacedownCreatureSpell(TheCardInHandChosenThisWay,PT(Integer(2),Integer(2)))`,
		`CastCardInHandWithoutPayingAsAFacedownCreatureSpell(Ref_TargetPermanent,PT(2,2))`,
	} {
		rule := `Activated(PayMana([]),ActionList([MayCost(ChooseACardInHand(AnyCard)),If(CostWasPaid,[` + action + `])]))`
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`)); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
	action := `CastCardInHandWithoutPayingAsAFacedownCreatureSpell(TheCardInHandChosenThisWay,PT(2,2))`
	for _, rule := range []string{`Activated(PayMana([]),ActionList([` + action + `]))`, `SpellActions(ActionList([` + action + `]))`} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`)); err == nil {
			t.Fatal("accepted unbound card")
		}
	}
}
