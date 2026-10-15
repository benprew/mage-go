package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLostManaTransfer(t *testing.T) {
	for _, body := range []string{
		`PlayerAction(Ref_TargetPlayer,LoseUnspentMana),AddMana(TheManaLostThisWay)`,
		`PlayerAction(Ref_TargetPlayer,LoseUnspentMana),PlayerAction(You,AddMana(TheManaLostThisWay))`,
		`PlayerActions(You,[PlayerAction(Ref_TargetPlayer,LoseUnspentMana),AddMana(TheManaLostThisWay)])`,
	} {
		g, a, b := ronGame()
		p := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(AnyPlayer)],ActionList([`+body+`])))`)(), a.PlayerID())
		b.ManaPool().Add(core.Red, 2)
		b.ManaPool().AddRestricted(core.Colorless, 3, core.ArtifactSpellsOnly{})
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		clone := g.Clone()
		g.ResolveStack()
		if a.ManaPool().Count(core.Red) != 2 || a.ManaPool().Count(core.Colorless) != 3 || b.ManaPool().TotalMana() != 0 || a.ManaPool().TotalProducedThisTurn() != 0 {
			t.Fatal("wrong saved mana transfer")
		}
		if a.ManaPool().CanPay(core.ManaCost{Generic: 5}, nil) {
			t.Fatal("restriction lost")
		}
		if clone.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != 5 {
			t.Fatal("pending clone changed")
		}
		clone.ResolveStack()
		if clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 5 {
			t.Fatal("pending clone lost bindings")
		}
	}
}

func TestRONLostManaInvalid(t *testing.T) {
	for _, body := range []string{
		`AddMana(TheManaLostThisWay)`,
		`AddMana(TheManaLostThisWay),LoseUnspentMana`,
		`LoseUnspentMana(),AddMana(TheManaLostThisWay)`,
		`LoseUnspentMana,AddMana(TheManaLostThisWay())`,
		`LoseUnspentMana,AddMana(TheManaLostThisWay,Integer(1))`,
		`MayAction(LoseUnspentMana),AddMana(TheManaLostThisWay)`,
		`PlayerActions(You,[LoseUnspentMana]),AddMana(TheManaLostThisWay)`,
		`LoseUnspentMana,AddManaRepeated(TheManaLostThisWay,Integer(2))`,
	} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),ManaCost:[],Rules:[SpellActions(ActionList([`+body+`]))])`)); err == nil {
			t.Fatal("accepted", body)
		}
	}
}

func TestRONLostManaExistingRecords(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, name := range []string{"Mana Short", "Power Sink"} {
			if c := repositoryCard(t, set, name); c == nil {
				t.Fatal(name)
			}
		}
	}
}

func TestRONLostManaSpellAndShadowing(t *testing.T) {
	for _, body := range []string{
		`PlayerAction(Ref_TargetPlayer,LoseUnspentMana),LoseUnspentMana,AddMana(TheManaLostThisWay)`,
		`PlayerAction(Ref_TargetPlayer,LoseUnspentMana),PlayerAction(Ref_TargetPlayer,LoseUnspentMana),AddMana(TheManaLostThisWay)`,
	} {
		g, a, b := ronGame()
		node := nameScopeNode(t, `Card(Name:"Transfer",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),ManaCost:[],Rules:[SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([`+body+`])))])`)
		factory, err := Compile(node)
		if err != nil {
			t.Fatal(err)
		}
		c := factory()
		c.SetOwner(a.PlayerID())
		a.AddToHand(c)
		b.ManaPool().Add(core.Blue, 3)
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if a.ManaPool().TotalMana() != 0 || b.ManaPool().TotalMana() != 0 {
			t.Fatal("later producer did not shadow earlier loss")
		}
	}
}

func TestRONLostManaCopyAndSource(t *testing.T) {
	g, a, b := ronGame()
	c := permanentScopeFactory(t, "Artifact", `Activated(SacrificePermanent(ThisPermanent),ActionList([LoseUnspentMana,AddMana(TheManaLostThisWay)]))`)()
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	a.ManaPool().Add(core.Green, 2)
	b.ManaPool().Add(core.Red, 3)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(p.ID()) != nil {
		t.Fatal("source did not leave")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.ManaPool().Count(core.Green) != 2 || b.ManaPool().Count(core.Red) != 3 || a.ManaPool().TotalProducedThisTurn() != 2 || b.ManaPool().TotalProducedThisTurn() != 3 {
		t.Fatal("copy or LKI mixed players or produced mana")
	}
}

func TestRONLostManaContexts(t *testing.T) {
	for _, rule := range []string{
		`AsPermanentEnters(ThisPermanent,[LoseUnspentMana,AddMana(TheManaLostThisWay)])`,
		`ReplaceWouldDraw(APlayerWouldDrawExceptFirstDrawStepDraw(AnyPlayer),[LoseUnspentMana,AddMana(TheManaLostThisWay)])`,
		`Activated(AddMana(TheManaLostThisWay),ActionList([LoseUnspentMana]))`,
		`EachPermanentLayerEffect(IsCardtype(Creature),[AddMana(TheManaLostThisWay)])`,
	} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Bad context",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),ManaCost:[],Rules:[`+rule+`])`)); err == nil {
			t.Fatal("accepted", rule)
		}
	}
	g, a, b := ronGame()
	p := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([LoseUnspentMana,AddMana(TheManaLostThisWay)]))`)(), a.PlayerID())
	a.ManaPool().Add(core.Green, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(p.ID())
	g.ResolveStack()
	if a.ManaPool().Count(core.Green) != 2 || b.ManaPool().TotalMana() != 0 {
		t.Fatal("phased source changed saved player")
	}
}
