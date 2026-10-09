package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSourceCardPayment(t *testing.T) {
	for _, cost := range []string{`SacrificePermanent(ThisPermanent)`, `Exile([Permanent(ThisPermanent)],[])`} {
		factory := permanentScopeFactory(t, "Artifact", `Activated(`+cost+`,Targeted([TargetAnteCard(AnyCard)],ActionList([ExchangeOwnershipOfTwoCards(ThisPermanent,Ref_TargetAnteCard)])))`)
		a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
		g, _ := mage.NewGameWithAnte(a, b, nil, nil)
		source := factory()
		source.SetOwner(a.PlayerID())
		p := g.PutOnBattlefield(source, a.PlayerID())
		other := mage.NewArtifact("other", "")
		other.SetOwner(b.PlayerID())
		b.AddToAnte(other)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{other.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if g.FindCardAnywhere(source.ID()).Owner() != b.PlayerID() || g.FindCardAnywhere(other.ID()).Owner() != a.PlayerID() {
			t.Fatal("source result lost")
		}
	}
}

func TestRONSourceCardTempestRequirement(t *testing.T) {
	record := staticBaseRecord(t, "LEG", "Tempest Efreet")
	if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "PutCardFromHandIntoPlayersHand") {
		t.Fatal("expected next independent blocker", err)
	}
}

func TestRONSourceCardInvalid(t *testing.T) {
	for _, rule := range []string{
		`Activated(And([SacrificePermanent(ThisPermanent),Exile([Permanent(ThisPermanent)],[])]),Targeted([TargetAnteCard(AnyCard)],ActionList([ExchangeOwnershipOfTwoCards(ThisPermanent,Ref_TargetAnteCard)])))`,
		`Activated(PayMana([]),Targeted([TargetAnteCard(AnyCard)],ActionList([ExchangeOwnershipOfTwoCards(ThisPermanent,Ref_TargetAnteCard)])))`,
		`Activated(SacrificePermanent(ThisPermanent),Targeted([TargetAnteCard(AnyCard)],ActionList([ExchangeOwnershipOfTwoCards(ThisPermanent(),Ref_TargetAnteCard)])))`,
		`Activated(SacrificePermanent(ThisPermanent),Targeted([TargetAnteCard(AnyCard)],ActionList([ExchangeOwnershipOfTwoCards(ThisCard,Ref_TargetAnteCard)])))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted unbound source", rule)
		}
	}
}

func TestRONSourceCardTempestGameplay(t *testing.T) {
	for _, state := range []string{"decline", "paid", "returned", "clone"} {
		t.Run(state, func(t *testing.T) {
			record := staticBaseRecord(t, "LEG", "Tempest Efreet")
			rules := record.Fields["Rules"]
			unpaid := &rules.Args[1].Args[1].Args[1].Args[0].Args[1].Args[1]
			unpaid.Args = unpaid.Args[:2]
			record.Fields["Rules"] = rules
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			a := mage.NewBasePlayer("A")
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: state == "paid"}
			g, err := mage.NewGameWithAnte(a, b, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			g.SetTurn(2)
			g.SetStep(core.PrecombatMain)
			source := factory()
			source.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(source, a.PlayerID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			revealed := mage.NewArtifact("revealed", "")
			revealed.SetOwner(b.PlayerID())
			b.AddToHand(revealed)
			if err = g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if state == "returned" {
				c, _ := a.RemoveFromGraveyard(source.ID())
				a.AddToGraveyard(c)
			}
			clone := g.Clone()
			g.ResolveStack()
			swapped := state == "decline" || state == "clone"
			if (g.FindCardAnywhere(source.ID()).Owner() == b.PlayerID()) != swapped || (g.FindCardAnywhere(revealed.ID()).Owner() == a.PlayerID()) != swapped {
				t.Fatal("source exchange or unpaid binding lost")
			}
			if state == "paid" && b.Life() != 10 {
				t.Fatal("payment lost")
			}
			if clone.FindCardAnywhere(revealed.ID()).Owner() != b.PlayerID() {
				t.Fatal("clone shared reveal result")
			}
		})
	}
}

func TestRONSourceCardGroupedExile(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(AnyPermanent)],ActionList([Exile([Permanent(ThisPermanent),Permanent(Ref_TargetPermanent)],[]),ExchangeOwnershipOfTwoCards(ThisPermanent,TheSecondCardExiledThisWay)])))`)
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g, _ := mage.NewGameWithAnte(a, b, nil, nil)
	source := factory()
	source.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(source, a.PlayerID())
	other := mage.NewArtifact("other", "")
	other.SetOwner(b.PlayerID())
	q := g.PutOnBattlefield(other, b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindExiledCard(source.ID()).Card.Owner() != b.PlayerID() || g.FindExiledCard(other.ID()).Card.Owner() != a.PlayerID() {
		t.Fatal("grouped source reference lost")
	}
}

func TestRONSourceCardInstancesAndDelay(t *testing.T) {
	for _, delayed := range []bool{false, true} {
		body := `ExchangeOwnershipOfTwoCards(ThisPermanent,Ref_TargetAnteCard)`
		if delayed {
			body = `CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + body + `]))`
		}
		factory := permanentScopeFactory(t, "Artifact", `Activated(SacrificePermanent(ThisPermanent),Targeted([TargetAnteCard(AnyCard)],ActionList([`+body+`])))`)
		a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
		g, _ := mage.NewGameWithAnte(a, b, nil, nil)
		one, two := factory(), factory()
		one.SetOwner(a.PlayerID())
		two.SetOwner(b.PlayerID())
		p, q := g.PutOnBattlefield(one, a.PlayerID()), g.PutOnBattlefield(two, b.PlayerID())
		x, y := mage.NewArtifact("x", ""), mage.NewArtifact("y", "")
		x.SetOwner(b.PlayerID())
		y.SetOwner(a.PlayerID())
		b.AddToAnte(x)
		a.AddToAnte(y)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{x.ID()}); err != nil {
			t.Fatal(err)
		}
		if err := g.ActivateAbilityByIndex(b.PlayerID(), q.ID(), 0, []uuid.UUID{y.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if delayed {
			for range 2 {
				g.WaitingDelayedGraveyardSources()
				if g.FindCardAnywhere(one.ID()).Owner() != a.PlayerID() {
					t.Fatal("query ran body")
				}
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
		}
		if g.FindCardAnywhere(one.ID()).Owner() != b.PlayerID() || g.FindCardAnywhere(two.ID()).Owner() != a.PlayerID() || g.FindCardAnywhere(x.ID()).Owner() != a.PlayerID() || g.FindCardAnywhere(y.ID()).Owner() != b.PlayerID() {
			t.Fatal("instances shared source results")
		}
	}
}
