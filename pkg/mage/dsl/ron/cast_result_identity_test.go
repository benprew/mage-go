package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const castResultAction = `MayAction(CastCardInHandWithoutPayingAsAFacedownCreatureSpell(TheCardInHandChosenThisWay,PT(2,2)))`
const castResultDelay = `CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([TapPermanent(ThePermanentSpellResolvedThisWay)]))`

func TestRONCastResultIdentity(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		for _, scenario := range []string{"resolve", "decline", "counter", "return", "phase", "phase-cycle", "control", "source-left", "copy", "clone"} {
			t.Run(set+"/"+scenario, func(t *testing.T) {
				record := repositoryRecord(t, set, "Illusionary Mask", set+".ron")
				rule := record.Fields["Rules"].Args[0]
				branch := &rule.Args[1].Args[0].Args[1].Args[1]
				// The delayed tap tests identity without the pending face-up replacement.
				delay := nameScopeNode(t, castResultDelay)
				branch.Args = []Node{branch.Args[0], delay}
				record.Fields["Rules"] = Node{Kind: List, Args: []Node{rule}}
				factory, err := Compile(record)
				if err != nil {
					t.Fatal(err)
				}
				g, a, b := ronGame()
				answers := []bool{true, scenario != "decline"}
				chooser := &ronOptionalCastPlayer{BasePlayer: a, answers: answers}
				g.SetPlayerAt(0, chooser)
				source := g.PutOnBattlefield(factory(), a.PlayerID())
				g.SetStep(core.PrecombatMain)
				card := mage.NewCreature("Private", "{U}", 5, 5)
				a.AddToHand(card)
				a.ManaPool().Add(core.Blue, 1)
				g.SetXValue(1)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if scenario == "copy" {
					g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
					chooser.answers = []bool{true, true, true, false}
				}
				g.ResolveTopOfStack()
				if p := g.FindPermanent(card.ID()); p != nil {
					t.Fatal("cast resolved during ability")
				}
				if scenario == "counter" {
					g.CounterStackObject(g.StackPeek().ID)
				}
				var clone *mage.Game
				if scenario == "clone" {
					clone = g.Clone()
					clone.CounterStackObject(clone.StackPeek().ID)
				}
				g.ResolveStack()
				p := g.FindPermanent(card.ID())
				if scenario == "return" {
					g.BouncePermanentToHand(p)
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				}
				if scenario == "phase" || scenario == "phase-cycle" {
					ids := g.PhaseOut(p.ID())
					if scenario == "phase-cycle" {
						g.PhaseIn(ids)
					}
				}
				if scenario == "control" {
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				if scenario == "source-left" {
					g.BouncePermanentToHand(source)
				}
				for _, game := range []*mage.Game{g, clone} {
					if game == nil {
						continue
					}
					game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					game.PutTriggersOnStack()
					game.ResolveStack()
					want := game == g && scenario != "decline" && scenario != "counter" && scenario != "return" && scenario != "phase"
					p = game.FindPermanentIncludingPhased(card.ID())
					if p != nil && p.Tapped != want {
						t.Fatalf("tap=%v want=%v", p.Tapped, want)
					}
					if want && p == nil {
						t.Fatal("missing resolved permanent")
					}
				}
			})
		}
	}
}

func TestRONCastResultIdentityInvalid(t *testing.T) {
	for _, body := range []string{
		castResultDelay,
		castResultDelay + `,` + castResultAction,
		`MayCost(ChooseACardInHand(AnyCard)),If(CostWasPaid,[` + castResultAction + `]),` + castResultDelay,
		`MayCost(ChooseACardInHand(AnyCard)),IfElse(CostWasPaid,[` + castResultAction + `],[` + castResultDelay + `])`,
		`MayCost(ChooseACardInHand(AnyCard)),If(CostWasPaid,[ChooseAnAction([[` + castResultAction + `],[GainLife(Integer(1))]]),` + castResultDelay + `])`,
		`MayCost(ChooseACardInHand(AnyCard)),If(CostWasPaid,[` + castResultAction + `,TapPermanent(ThePermanentSpellResolvedThisWay())])`,
	} {
		record := nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`)
		if _, err := Compile(record); err == nil {
			t.Fatal("accepted invalid cast result scope", body)
		}
	}
}

func TestRONCastResultIdentityDirectAndFactories(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([MayCost(ChooseACardInHand(AnyCard)),If(CostWasPaid,[CastCardInHandWithoutPayingAsAFacedownCreatureSpell(TheCardInHandChosenThisWay,PT(2,2)),`+castResultDelay+`])]))`)
	g, a, _ := ronGame()
	chooser := &ronOptionalCastPlayer{BasePlayer: a, answers: []bool{true, true}}
	g.SetPlayerAt(0, chooser)
	cards := []*mage.BaseCard{mage.NewCreature("One", "", 3, 3), mage.NewCreature("Two", "", 4, 4)}
	for _, card := range cards {
		a.AddToHand(card)
		source := g.PutOnBattlefield(factory(), a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	if chooser.offers != 2 {
		t.Fatal("direct cast offered an extra choice")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	for _, card := range cards {
		if p := g.FindPermanent(card.ID()); p == nil || !p.Tapped {
			t.Fatal("factory results shared state")
		}
	}
}
