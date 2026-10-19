package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronDrawnReturnPlayer struct {
	*mage.BasePlayer
	accept    bool
	choices   int
	decisions []bool
}

func (p *ronDrawnReturnPlayer) ChooseMayAbility(string) bool {
	p.choices++
	if len(p.decisions) >= p.choices {
		return p.decisions[p.choices-1]
	}
	return p.accept
}

func TestRONDrawnCardReturnLibrary(t *testing.T) {
	for _, accept := range []bool{false, true} {
		a := &ronDrawnReturnPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: accept}
		b := mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		g.SetActivePlayerIndex(0)
		g.SetStep(core.PrecombatMain)
		g.SetTurn(2)
		c := repositoryCard(t, "LEG", "Sylvan Library")
		addRONHandCard(a.BasePlayer, c)
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil {
			t.Fatal("unpaid cast")
		}
		a.ManaPool().Add(core.Colorless, 1)
		a.ManaPool().Add(core.Green, 1)
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		for range 6 {
			a.AddToLibrary(mage.NewLand("Draw"))
		}
		g.PlayerDrawCard(a)
		g.FireEvent(core.GameEvent{Type: core.EvtDrawStep, PlayerID: b.PlayerID()})
		g.PutTriggersOnStack()
		if g.StackSize() != 0 {
			t.Fatal("opponent draw step")
		}
		g.FireEvent(core.GameEvent{Type: core.EvtDrawStep, PlayerID: a.PlayerID()})
		g.PutTriggersOnStack()
		if g.StackSize() != 1 || a.choices != 0 {
			t.Fatal("trigger timing")
		}
		g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
		g.ResolveStack()
		wantHand, wantLife := 1, 20
		if accept {
			wantHand, wantLife = 3, 12
		}
		if len(a.Hand()) != wantHand || a.Life() != wantLife {
			t.Fatal("draw choice or payment", len(a.Hand()), a.Life())
		}
	}
}

func TestRONDrawnCardReturnDeclarations(t *testing.T) {
	for _, action := range []string{`PayLifeOrPutCardsDrawnThisTurnOnTopOfLibrary`, `PayLifeOrPutCardsDrawnThisTurnOnTopOfLibrary(Integer(2))`, `PayLifeOrPutCardsDrawnThisTurnOnTopOfLibrary(Integer(-1),Integer(4))`, `PayLifeOrPutCardsDrawnThisTurnOnTopOfLibrary(X,Integer(4))`, `PayLifeOrPutCardsDrawnThisTurnOnTopOfLibrary(Integer(2),Integer(-1))`} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+action+`]))])`)); err == nil {
			t.Fatal("accepted", action)
		}
	}
	for _, rule := range []string{
		`Activated(PayLifeOrPutCardsDrawnThisTurnOnTopOfLibrary(Integer(2),Integer(4)),ActionList([]))`,
		`AsPermanentEnters(ThisPermanent,[PayLifeOrPutCardsDrawnThisTurnOnTopOfLibrary(Integer(2),Integer(4))])`,
		`PermanentLayerEffect(ThisPermanent,[PayLifeOrPutCardsDrawnThisTurnOnTopOfLibrary(Integer(2),Integer(4))])`,
		`ReplaceWouldDraw(APlayerWouldDrawExceptFirstDrawStepDraw(AnyPlayer),[PayLifeOrPutCardsDrawnThisTurnOnTopOfLibrary(Integer(2),Integer(4))])`,
		`SpellActions(ActionList([MayCost(DrawNumberCards(Integer(2))),If(CostWasPaid(),[GainLife(Integer(1))])]))`,
		`SpellActions(ActionList([MayCost(DrawNumberCards(Integer(-1))),If(CostWasPaid,[GainLife(Integer(1))])]))`,
		`SpellActions(ActionList([MayCost(DrawNumberCards(ValueX)),If(CostWasPaid,[GainLife(Integer(1))])]))`,
		`SpellActions(ActionList([MustCost(DrawNumberCards(Integer(2))),If(CostWasPaid,[GainLife(Integer(1))])]))`,
		`SpellActions(ActionList([MayCost(DrawNumberCards(Integer(2))),IfElse(CostWasPaid,[GainLife(Integer(1))],[GainLife(Integer(2))])]))`,
		`SpellActions(ActionList([MayCost(DrawNumberCards(Integer(2))),If(CostWasPaid,[GainLife(Integer(1))]),If(CostWasPaid,[GainLife(Integer(2))])]))`,
	} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`)); err == nil {
			t.Fatal("accepted", rule)
		}
	}
}
