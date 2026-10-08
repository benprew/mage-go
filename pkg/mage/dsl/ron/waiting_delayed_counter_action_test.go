package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedCounterActionProtection(t *testing.T) {
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
		for _, action := range []string{`PutCounters([ACounterOfTypeOnPermanent(ChargeCounter,ThisPermanent)])`, `RemoveCounters([ACounterOfTypeFromPermanent(ChargeCounter,ThisPermanent)])`, `RemoveCounters([AllCountersOfTypeFromPermanent(ChargeCounter,ThisPermanent)])`, `PutCountersWithRestriction([UptoNumberCountersOfTypeOnPermanent(Integer(1),ChargeCounter,ThisPermanent)],CantCauseTheNumberOfCountersOfTypeOnPermanentToBeGreaterThan(ChargeCounter,ThisPermanent,Integer(7)))`} {
			for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "control", "copy", "clone"} {
				t.Run(tc.name+"/"+action+"/"+state, func(t *testing.T) {
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
					chooser := &ronSourceChoicePlayer{BasePlayer: a}
					g := mage.NewGame(chooser, b)
					factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+action+`]))]))`)
					card := factory()
					card.SetOwner(a.PlayerID())
					p := g.PutOnBattlefield(card, a.PlayerID())
					p.AddCounter(core.Charge, 2)
					g.ChangeObjectColor(p.ID(), tc.color)
					saved := g.CaptureDamageSource(p.ID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
					if len(g.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("delay exists at announcement")
					}
					g.ResolveStack()
					switch state {
					case "departed", "returned":
						g.BouncePermanentToHand(p)
						if state == "returned" {
							a.RemoveFromHand(card.ID())
							p = g.PutOnBattlefield(card, b.PlayerID())
							p.AddCounter(core.Charge, 2)
						}
					case "phased":
						g.PhaseOut(p.ID())
					case "cycle":
						g.PhaseIn(g.PhaseOut(p.ID()))
					case "control":
						g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					}
					before := p.Counters[core.Charge]
					binding := mage.BindDamageSource("counter source")
					bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
					payment := make([]mage.Cost, len(costs))
					for i, cost := range costs {
						payment[i] = cost()
					}
					consumer := g.PutOnBattlefield(mage.NewArtifact("Consumer", "", mage.WithAbility(mage.Activate(mage.Costs(payment...), mage.Sequence(plan.waitingDelayedPermanentChoice(bindings), shield(bindings))))), a.PlayerID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), consumer.ID(), 0, nil); err == nil || g.StackSize() != 0 || chooser.calls != 0 {
						t.Fatal("failed payment changed state")
					}
					a.ManaPool().Add(core.White, 2)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), consumer.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if chooser.calls != 0 {
						t.Fatal("choice ran at announcement")
					}
					for range 3 {
						got := g.WaitingDelayedPermanentSources()
						if len(got) != 1 || !got[0].SameObject(saved) || p.Counters[core.Charge] != before {
							t.Fatal("query lost source or executed counter action")
						}
					}
					var clone *mage.Game
					if state == "clone" {
						clone = g.Clone()
						clone.ResolveStack()
						if chooser.calls != 0 {
							t.Fatal("clone changed chooser")
						}
					}
					g.ResolveStack()
					if chooser.calls != 1 || !chooser.source.SameObject(saved) {
						t.Fatal("counter reference missing from source choice")
					}
					for _, game := range []*mage.Game{g, clone} {
						if game == nil {
							continue
						}
						game.DealDamageToPlayerFromSource(game.GetPlayer(a.PlayerID()), 2, saved)
						if game.GetPlayer(a.PlayerID()).Life() != 20 {
							t.Fatal("chosen source not prevented")
						}
						game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
						if len(game.WaitingDelayedPermanentSources()) != 0 {
							t.Fatal("detected delay still waits")
						}
						game.PutTriggersOnStack()
						game.ResolveStack()
						if live := game.FindPermanentIncludingPhased(p.ID()); live != nil {
							want := uint8(3)
							if strings.Contains(action, "ACounterOfTypeFromPermanent") {
								want = 1
							}
							if strings.Contains(action, "AllCountersOfTypeFromPermanent") {
								want = 0
							}
							if state == "copy" && strings.HasPrefix(action, "PutCounters") {
								want = 4
							}
							if state == "copy" && strings.Contains(action, "ACounterOfTypeFromPermanent") {
								want = 0
							}
							if state == "returned" || state == "phased" {
								want = before
							}
							if live.Counters[core.Charge] != want {
								t.Fatal("counter action followed wrong incarnation or phase state")
							}
						}
						game.DealDamageToPlayerFromSource(game.GetPlayer(a.PlayerID()), 1, saved)
						if game.GetPlayer(a.PlayerID()).Life() != 19 {
							t.Fatal("shield not consumed")
						}
						if game == g && clone != nil && len(clone.WaitingDelayedPermanentSources()) != 1 {
							t.Fatal("clone shares lifetime")
						}
					}
					if _, err := Compile(record); err != nil {
						t.Fatal("complete source-choice record rejected", err)
					}
				})
			}
		}
	}
}

func TestRONWaitingDelayedCounterActionInvalid(t *testing.T) {
	for _, body := range []string{
		`PutCounters([])`,
		`PutCounters([ACounterOfTypeOnPermanent(ChargeCounter,You)])`,
		`PutCounters([ACounterOfTypeOnPermanent(ChargeCounter,Ref_TargetPermanent)])`,
		`PutCounters([ACounterOfTypeOnPermanent(ChargeCounter,HostPermanent)])`,
		`PutCounters([ACounterOfTypeOnPermanent(PoisonCounter,ThisPermanent)])`,
		`PutCounters([NumberCountersOfTypeOnPermanent(ManaValueOfPermanent(Ref_TargetPermanent),ChargeCounter,ThisPermanent)])`,
		`RemoveCounters([])`,
		`RemoveCounters([AllCountersOfTypeFromPermanent(ChargeCounter,Trigger_ThatPermanent)])`,
		`RemoveCounters([ACounterOfTypeFromPermanent(ChargeCounter,ThisPermanent,ThisPermanent)])`,
		`PutCountersWithRestriction([UptoNumberCountersOfTypeOnPermanent(Integer(1),ChargeCounter,ThisPermanent)],CantCauseTheNumberOfCountersOfTypeOnPermanentToBeGreaterThan(TimeCounter,ThisPermanent,Integer(7)))`,
		`PutCountersWithRestriction([UptoNumberCountersOfTypeOnPermanent(Integer(1),ChargeCounter,ThisPermanent)],CantCauseTheNumberOfCountersOfTypeOnPermanentToBeGreaterThan(ChargeCounter,ThisPermanent,Integer(-1)))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid delayed counter action accepted", body)
		}
	}
}

func TestRONWaitingDelayedCounterActionInstances(t *testing.T) {
	for _, bounded := range []bool{false, true} {
		for _, amount := range []string{`ManaValueOfPermanent(ThisPermanent)`, `TheTotalNumberOfCountersOfTypeAmongPermanents(ChargeCounter,SinglePermanent(ThisPermanent))`, `Minus(ManaValueOfPermanent(ThisPermanent),ManaValueOfPermanent(ThisPermanent))`} {
			t.Run(amount+fmt.Sprint(bounded), func(t *testing.T) {
				g, a, b := ronGame()
				body := `PutCounters([NumberCountersOfTypeOnPermanent(` + amount + `,ChargeCounter,Ref_TargetPermanent)])`
				if bounded {
					body = `PutCountersWithRestriction([UptoNumberCountersOfTypeOnPermanent(` + amount + `,ChargeCounter,Ref_TargetPermanent)],CantCauseTheNumberOfCountersOfTypeOnPermanentToBeGreaterThan(ChargeCounter,Ref_TargetPermanent,Integer(7)))`
				}
				nodes, err := Parse(strings.NewReader(`Card(Name:"Counter Source",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),ManaCost:[ManaCostGeneric(2)],Rules:[Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))])))])`), "counter-source.ron")
				if err != nil {
					t.Fatal(err)
				}
				factory, err := Compile(nodes[0])
				if err != nil {
					t.Fatal(err)
				}
				sources := make([]*mage.Permanent, 0, 2)
				targets := make([]*mage.Permanent, 0, 2)
				for _, player := range []*mage.BasePlayer{a, b} {
					card := factory()
					card.SetOwner(player.PlayerID())
					source := g.PutOnBattlefield(card, player.PlayerID())
					source.AddCounter(core.Charge, 2)
					targetCard := mage.NewCreature("Target", "", 1, 3)
					targetCard.SetOwner(player.PlayerID())
					target := g.PutOnBattlefield(targetCard, player.PlayerID())
					if err := g.ActivateAbilityByIndex(player.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
						t.Fatal(err)
					}
					g.ResolveStack()
					sources = append(sources, source)
					targets = append(targets, target)
				}
				for range 3 {
					got := g.WaitingDelayedPermanentSources()
					if len(got) != 4 {
						t.Fatal("source amount or instance target binding lost", len(got))
					}
					for _, source := range sources {
						if source.Counters[core.Charge] != 2 {
							t.Fatal("query executes amount")
						}
					}
					for _, target := range targets {
						if target.Counters[core.Charge] != 0 {
							t.Fatal("query places counters")
						}
					}
				}
				returnedCard := targets[0].Card
				g.BouncePermanentToHand(targets[0])
				a.RemoveFromHand(returnedCard.ID())
				returned := g.PutOnBattlefield(returnedCard, b.PlayerID())
				g.MutablePermanent(targets[1].ID()).GrantBaseAttr(core.Shroud)
				g.BouncePermanentToHand(sources[1])
				clone := g.Clone()
				for _, game := range []*mage.Game{clone, g} {
					game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					game.PutTriggersOnStack()
					game.ResolveStack()
					if game.FindPermanent(returned.ID()).Counters[core.Charge] != 0 {
						t.Fatal("counters followed returned target")
					}
					want := uint8(0)
					if strings.HasPrefix(amount, "TheTotalNumber") || strings.HasPrefix(amount, "ManaValue") {
						want = 2
					}
					if game.FindPermanent(targets[1].ID()).Counters[core.Charge] != want {
						t.Fatal("saved amount or target legality failed")
					}
					if len(game.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("resolved delay supplies sources")
					}
					if game == clone && len(g.WaitingDelayedPermanentSources()) != 4 {
						t.Fatal("clone shares delay lifetime")
					}
				}
			})
		}
	}
}
