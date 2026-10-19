package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONActivationManaReceipt(t *testing.T) {
	for _, cost := range []string{"{U}{U}", "{1}{U}", "{2}", "{W/U}{W/U}", "{2}{U}", "{G}"} {
		t.Run(cost, func(t *testing.T) {
			g, a, _ := ronGame()
			chooser := &ronHandChoicePlayer{BasePlayer: a, accept: true}
			g.SetPlayerAt(0, chooser)
			rule := `Activated(PayManaX([ManaCostX],ValueX),ActionList([MayCost(ChooseACardInHand(And([IsCardtype(Creature),ManaCostIsSubsetOfManaPaidForThisAbility]))),If(CostWasPaid,[GainLife(Integer(1))])]))`
			source := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", rule)(), a.PlayerID())
			a.AddToHand(mage.NewCreature("Candidate", cost, 1, 1))
			a.ManaPool().Add(core.Blue, 2)
			g.SetXValue(2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			a.ManaPool().Add(core.Green, 9)
			g.ResolveStack()
			want := 21
			if cost == "{2}{U}" || cost == "{G}" {
				want = 20
			}
			if a.Life() != want {
				t.Fatalf("life=%d want=%d", a.Life(), want)
			}
		})
	}
}

func TestRONActivationManaReceiptInvalid(t *testing.T) {
	for _, filter := range []string{
		`ManaCostIsSubsetOfManaPaidForThisAbility()`,
		`ManaCostIsSubsetOfManaPaidForThisAbility(Integer(1))`,
		`Or([ManaCostIsSubsetOfManaPaidForThisAbility,IsCardtype(Creature)])`,
	} {
		rule := `Activated(PayManaX([ManaCostX],ValueX),ActionList([MayCost(ChooseACardInHand(` + filter + `)),If(CostWasPaid,[GainLife(Integer(1))])]))`
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`)); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
	body := `MayCost(ChooseACardInHand(ManaCostIsSubsetOfManaPaidForThisAbility)),If(CostWasPaid,[GainLife(Integer(1))])`
	for _, rule := range []string{
		`SpellActions(ActionList([` + body + `]))`,
		`AsPermanentEnters(ThisPermanent,[` + body + `])`,
		`Activated(PayManaX([ManaCostX],ValueX),ActionList([CreateDelayedTriggeredAbility(AtTheBeginningOfTheNextEndStep,[` + body + `])]))`,
	} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`)); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
	for _, ctx := range []scope{{resolving: true}, {activation: true, x: true, resolving: true, activationMana: &activationManaPlan{}, delayedStep: true}} {
		if _, _, err := compileActivationManaHandFilter(nameScopeNode(t, `ManaCostIsSubsetOfManaPaidForThisAbility`), ctx); err == nil {
			t.Fatal("accepted wrong scope")
		}
	}
}

func TestRONActivationManaReceiptMaskPrerequisite(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			record := repositoryRecord(t, set, "Illusionary Mask", set+".ron")
			if _, err := Compile(record); err != nil {
				t.Fatalf("complete Mask record: %v", err)
			}
			rule := record.Fields["Rules"].Args[0]
			rule.Args[1].Args[0].Args[1].Args[1].Args = []Node{nameScopeNode(t, `GainLife(Integer(3))`)}
			record.Fields["Rules"] = Node{Kind: List, Args: []Node{rule}}
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			for _, cost := range []string{"{U}{U}", "{G}"} {
				g, a, _ := ronGame()
				g.SetPlayerAt(0, &ronHandChoicePlayer{BasePlayer: a, accept: true})
				a.AddToHand(mage.NewCreature("Candidate", cost, 1, 1))
				source := g.PutOnBattlefield(factory(), a.PlayerID())
				g.SetStep(core.PrecombatMain)
				a.ManaPool().Add(core.Blue, 2)
				g.SetXValue(2)
				g.SetXValue(2)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				want := 20
				if cost == "{U}{U}" {
					want = 23
				}
				if a.Life() != want {
					t.Fatalf("life=%d want=%d", a.Life(), want)
				}
			}
		})
	}
}

func TestRONActivationManaReceiptPlayerAndSource(t *testing.T) {
	for _, scenario := range []string{"left", "returned", "control", "phased"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			chooser := &ronHandChoicePlayer{BasePlayer: b, accept: true}
			g.SetPlayerAt(1, chooser)
			b.AddToHand(mage.NewCreature("Other hand", "{U}", 1, 1))
			b.ManaPool().Add(core.Green, 1)
			rule := `Activated(PayManaX([ManaCostX],ValueX),Targeted([TargetPlayer(Opponent)],ActionList([PlayerMayCost(Ref_TargetPlayer,ChooseACardInHand(ManaCostIsSubsetOfManaPaidForThisAbility)),If(CostWasPaid,[GainLife(Integer(1))])])))`
			card := permanentScopeFactory(t, "Artifact", rule)()
			source := g.PutOnBattlefield(card, a.PlayerID())
			a.ManaPool().Add(core.Blue, 1)
			g.SetXValue(1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "left":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			case "returned":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phased":
				g.PhaseOut(source.ID())
			}
			g.ResolveStack()
			if a.Life() != 21 || b.Life() != 20 || chooser.choices != 1 || b.ManaPool().TotalMana() != 1 {
				t.Fatal("receipt followed the source or the chooser's pool")
			}
		})
	}
}
