package ron

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedCounterPaymentProtection(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		color     core.Color
	}{
		{"LEB", "Circle of Protection: Black", core.Black},
		{"LEA", "Circle of Protection: Blue", core.Blue},
		{"LEA", "Circle of Protection: Green", core.Green},
		{"LEA", "Circle of Protection: Red", core.Red},
		{"LEA", "Circle of Protection: White", core.White},
		{"ATQ", "Circle of Protection: Artifacts", core.Colorless},
		{"LEG", "Greater Realm of Preservation", core.Black},
	} {
		for _, branch := range []string{"fixed", "chosen", "mandatory", "unpaid"} {
			for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "copy", "clone"} {
				t.Run(tc.name+"/"+branch+"/"+state, func(t *testing.T) {
					record := staticBaseRecord(t, tc.set, tc.name)
					rule := record.Fields["Rules"].Args[0]
					actions := rule.Args[1].Args[0].Args
					plan, err := compileChosenDamageSourceChoice(actions[0])
					if err != nil {
						t.Fatal(err)
					}
					shield, err := compileExactSourcePrevention(actions[1], scope{chosenDamageSource: plan})
					if err != nil {
						t.Fatal(err)
					}
					costs, _, err := compileCosts(rule.Args[0])
					if err != nil {
						t.Fatal(err)
					}
					_, a, b := ronGame()
					chooser := &ronDelayedPaymentPlayer{ronSourceChoicePlayer: &ronSourceChoicePlayer{BasePlayer: a}, accept: branch != "unpaid"}
					g := mage.NewGame(chooser, b)
					instruction := `ACounterOfTypeFromPermanent(ChargeCounter,ThisPermanent)`
					prefix, condition := "MayCost", "CostWasPaid"
					if branch == "chosen" {
						instruction = `AnyNumberOfCountersOfTypeFromPermanent(ChargeCounter,ThisPermanent)`
					}
					if branch == "mandatory" {
						prefix = "MustCost"
					}
					if branch == "unpaid" {
						condition = "CostWasntPaid"
					}
					body := prefix + `(RemoveCounters([` + instruction + `])),If(` + condition + `,[GainLife(Integer(1))])`
					factory := waitingCombatFactory(t, `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))]))`)
					sourceCard := factory()
					sourceCard.SetOwner(a.PlayerID())
					source := g.PutOnBattlefield(sourceCard, a.PlayerID())
					g.ChangeObjectColor(source.ID(), tc.color)
					g.MutablePermanent(source.ID()).AddCounter(core.Charge, 3)
					original := g.CaptureDamageSource(source.ID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
					if len(g.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("delay exists before resolution")
					}
					g.ResolveStack()
					switch state {
					case "departed", "returned":
						g.BouncePermanentToHand(source)
						if state == "returned" {
							a.RemoveFromHand(sourceCard.ID())
							g.PutOnBattlefield(sourceCard, b.PlayerID())
						}
					case "phased":
						g.PhaseOut(source.ID())
					case "cycle":
						g.PhaseIn(g.PhaseOut(source.ID()))
					}
					binding := mage.BindDamageSource("waiting source")
					bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
					payment := make([]mage.Cost, len(costs))
					for i, cost := range costs {
						payment[i] = cost()
					}
					consumer := mage.NewArtifact("RON consumer", "", mage.WithAbility(mage.Activate(mage.Costs(payment...), mage.Sequence(plan.waitingDelayedPermanentChoice(bindings), shield(bindings)))))
					if err := mage.ValidateCardDefinition(consumer.Name(), consumer); err != nil {
						t.Fatal(err)
					}
					p := g.PutOnBattlefield(consumer, a.PlayerID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil || g.StackSize() != 0 || chooser.calls != 0 {
						t.Fatal("unpaid choice changed state")
					}
					a.ManaPool().Add(core.White, 2)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if chooser.calls != 0 {
						t.Fatal("choice ran at announcement")
					}
					a.ManaPool().Add(core.White, 2)
					if state == "clone" {
						clone := g.Clone()
						clone.ResolveStack()
						clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
						clone.PutTriggersOnStack()
						clone.ResolveStack()
						cloneLife := 21
						if branch == "fixed" || branch == "chosen" {
							cloneLife = 20
						}
						if clone.GetPlayer(a.PlayerID()).Life() != cloneLife || chooser.calls != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
							t.Fatal("clone lost source or shared lifetime")
						}
					}
					g.ResolveStack()
					if chooser.offers != 0 {
						t.Fatal("query paid a waiting cost")
					}
					if chooser.calls != 1 || !chooser.source.SameObject(original) {
						t.Fatal("conditional delayed body lost source choice")
					}
					g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					if len(g.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("fired ability remained waiting")
					}
					g.PutTriggersOnStack()
					g.ResolveStack()
					want := 20
					live := state != "departed" && state != "returned" && state != "phased"
					if (branch == "unpaid") || live {
						want = 21
						if state == "copy" {
							want = 22
						}
					}
					if a.Life() != want || b.Life() != 20 {
						t.Fatal("counter payment chose the wrong branch")
					}

					if state != "departed" {
						wantCounters := 3
						if live && branch != "unpaid" {
							removed := 1
							if branch == "chosen" {
								removed = 3
							}
							if state == "copy" && branch != "chosen" {
								removed *= 2
							}
							wantCounters -= removed
						}
						if state == "returned" {
							wantCounters = 0
						}
						if int(g.FindPermanentIncludingPhased(sourceCard.ID()).Counters[core.Charge]) != wantCounters {
							t.Fatal("payment removed counters from the wrong object or count")
						}
					}

					g.DealDamageToPlayerFromSource(chooser, 1, original)
					if a.Life() != want {
						t.Fatal("shield did not prevent damage")
					}
					g.DealDamageToPlayerFromSource(chooser, 1, original)
					if a.Life() != want-1 {
						t.Fatal("shield was not consumed")
					}
					if _, err := Compile(record); err != nil {
						t.Fatal("complete source-choice record rejected", err)
					}
				})
			}
		}
	}
}

func TestRONWaitingDelayedCounterPaymentInvalid(t *testing.T) {
	for _, body := range []string{
		`MayCost(RemoveCounters([])),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(RemoveCounters([ACounterOfTypeFromPermanent(ChargeCounter,HostPermanent)])),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(RemoveCounters([ACounterOfTypeFromPermanent(UnknownCounter,ThisPermanent)])),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(RemoveCounters([NumberCountersOfTypeFromPermanent(Integer(-1),ChargeCounter,ThisPermanent)])),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(` + anyChargeCost + `),If(CostWasntPaid,[GainLife(TheNumberOfCountersRemovedThisWay)])`,
		`MayCost(` + anyChargeCost + `),Unless(CostWasPaid,[GainLife(TheNumberOfCountersRemovedThisWay)])`,
		`MayCost(` + anyChargeCost + `),If(CostWasPaid,[GainLife(Integer(1))]),GainLife(TheNumberOfCountersRemovedThisWay)`,
		`GainLife(TheNumberOfCountersRemovedThisWay),MayCost(` + anyChargeCost + `),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MustCost(` + anyChargeCost + `),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(` + anyChargeCost + `),If(CostWasPaid,[])`,
		`MayCost(` + anyChargeCost + `),If(CostWasPaid,[GainLife(TheNumberOfCountersRemovedThisWay())])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", body)
		}
	}
}

func TestRONWaitingDelayedCounterPaymentInstances(t *testing.T) {
	a := &ronResolutionCounterPayer{ronCounterPayer: &ronCounterPayer{BasePlayer: mage.NewBasePlayer("A"), choice: 2}, accept: true}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	factory := waitingCombatFactory(t, `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([MayCost(`+anyChargeCost+`),If(CostWasPaid,[GainLife(TheNumberOfCountersRemovedThisWay)])]))]))`)
	cards := []mage.Card{factory(), factory()}
	for _, card := range cards {
		p := g.PutOnBattlefield(card, a.PlayerID())
		g.MutablePermanent(p.ID()).AddCounter(core.Charge, 3)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	if len(g.WaitingDelayedPermanentSources()) != 2 {
		t.Fatal("factory instances lost distinct references")
	}
	clone := g.Clone()
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 24 {
		t.Fatal("chosen payment results were not independent")
	}
	for _, card := range cards {
		if g.FindPermanent(card.ID()).Counters[core.Charge] != 1 || clone.FindPermanent(card.ID()).Counters[core.Charge] != 3 {
			t.Fatal("counter payment shared clone state")
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 2 {
		t.Fatal("clone shared delayed lifetime")
	}
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if len(clone.WaitingDelayedPermanentSources()) != 0 || a.Life() != 24 {
		t.Fatal("clone changed original result")
	}
}
