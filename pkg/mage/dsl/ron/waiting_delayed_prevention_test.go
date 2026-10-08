package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedPreventionOperands(t *testing.T) {
	for name, body := range map[string]string{
		"finite-recipient":    `CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Integer(2),Permanent(ThisPermanent)),[PreventThatDamage])`,
		"finite-player":       `CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Integer(2),Player(ControllerOfPermanent(ThisPermanent))),[PreventThatDamage])`,
		"finite-amount":       `CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Plus(Integer(2),ManaValueOfPermanent(ThisPermanent)),Player(You)),[PreventThatDamage])`,
		"unlimited-recipient": `CreatePreventDamageUntil(DamageWouldBeDealtToRecipient(Permanent(ThisPermanent)),[PreventThatDamage],UntilEndOfTurn)`,
		"unlimited-player":    `CreatePreventDamageUntil(DamageWouldBeDealtToRecipient(Player(ControllerOfPermanent(ThisPermanent))),[PreventThatDamage],UntilEndOfTurn)`,
	} {
		for _, state := range []string{"live", "phased", "returned", "copy", "clone"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))]))`)
				card := factory()
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				saved := g.CaptureDamageSource(p.ID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				if len(g.WaitingDelayedPermanentSources()) != 0 {
					t.Fatal("delay installed before resolution")
				}
				g.ResolveStack()
				if state == "phased" {
					g.PhaseOut(p.ID())
				}
				if state == "returned" {
					g.BouncePermanentToHand(p)
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				}
				for range 3 {
					got := g.WaitingDelayedPermanentSources()
					if len(got) != 1 || !got[0].SameObject(saved) || len(g.WaitingPreventionPermanentSources()) != 0 {
						t.Fatal("waiting prevention source missing or query installed rule")
					}
				}
				game := g
				if state == "clone" {
					game = g.Clone()
				}
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				if len(game.WaitingDelayedPermanentSources()) != 0 {
					t.Fatal("detected delay still waiting")
				}
				game.PutTriggersOnStack()
				game.ResolveStack()
				if state == "phased" {
					game.PhaseIn([]uuid.UUID{p.ID()})
				}
				if strings.Contains(name, "player") || name == "finite-amount" {
					recipient := game.GetPlayer(a.PlayerID())
					game.DealDamageToPlayer(recipient, 3, saved.ID())
					want := 19
					if name == "unlimited-player" || name == "finite-player" && state == "copy" {
						want = 20
					}
					if recipient.Life() != want {
						t.Fatal("wrong shield amount or recipient", recipient.Life(), want)
					}
					game.DealDamageToPlayer(recipient, 1, saved.ID())
					if name == "finite-amount" && recipient.Life() != 18 {
						t.Fatal("finite shield did not consume")
					}
				}
				if strings.Contains(name, "recipient") {
					refs := game.WaitingPreventionPermanentSources()
					want := 1
					if state == "returned" || state == "phased" {
						want = 0
					}
					if len(refs) != want || want == 1 && !refs[0].SameObject(saved) {
						t.Fatal("shield followed an absent or returned recipient", refs)
					}
				}
				if state == "clone" && (a.Life() != 20 || len(g.WaitingDelayedPermanentSources()) != 1) {
					t.Fatal("clone changed original")
				}
			})
		}
	}
}

func TestRONWaitingDelayedPreventionCreatedAmount(t *testing.T) {
	for _, state := range []string{"normal", "copy", "clone"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			body := `CreateValueX(` + counterInfoValue + `),RemoveCounters([AllCountersOfTypeFromPermanent(ChargeCounter,ThisPermanent)]),CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(ValueX,Player(You)),[PreventThatDamage])`
			f := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))]))`)
			p := g.PutOnBattlefield(f(), a.PlayerID())
			g.MutablePermanent(p.ID()).Counters[core.Charge] = 3
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			g.ResolveStack()
			game := g
			if state == "clone" {
				game = g.Clone()
			}
			game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			game.PutTriggersOnStack()
			game.ResolveStack()
			for _, id := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
				player := game.GetPlayer(id)
				game.DealDamageToPlayer(player, 4, p.ID())
				want := 16
				if state == "copy" && id == b.PlayerID() || state != "copy" && id == a.PlayerID() {
					want = 19
				}
				if player.Life() != want {
					t.Fatal("saved amount or instance scope lost", player.Life(), want)
				}
			}
			if state == "clone" && (a.Life() != 20 || g.FindPermanent(p.ID()).Counters[core.Charge] != 3) {
				t.Fatal("clone shared counters or shield")
			}
		})
	}
}

func TestRONWaitingDelayedPreventionInvalid(t *testing.T) {
	for _, amount := range []string{`Integer(-1)`, `ValueX`, `PowerOfPermanent(Ref_TargetPermanent)`, `Plus(Integer(1))`, `Integer(1,2)`, `ValueX()`} {
		source := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(` + amount + `,Player(You)),[PreventThatDamage])]))])`
		records, err := Parse(strings.NewReader(source), "invalid-shield.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(records[0]); err == nil {
			t.Fatal("invalid amount accepted", amount)
		}
	}
}

func TestRONWaitingDelayedPreventionProtection(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		color     core.Color
	}{
		{"LEB", "Circle of Protection: Black", core.Black}, {"LEA", "Circle of Protection: Blue", core.Blue}, {"LEA", "Circle of Protection: Green", core.Green}, {"LEA", "Circle of Protection: Red", core.Red}, {"LEA", "Circle of Protection: White", core.White}, {"ATQ", "Circle of Protection: Artifacts", core.Colorless}, {"LEG", "Greater Realm of Preservation", core.Black},
	} {
		t.Run(tc.name, func(t *testing.T) {
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
			a := &ronSourceChoicePlayer{BasePlayer: mage.NewBasePlayer("A")}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Integer(1),Player(ControllerOfPermanent(ThisPermanent))),[PreventThatDamage])]))]))`)
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.ChangeObjectColor(p.ID(), tc.color)
			saved := g.CaptureDamageSource(p.ID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			g.BouncePermanentToHand(p)
			binding := mage.BindDamageSource("waiting shield")
			bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
			payment := make([]mage.Cost, len(costs))
			for i, cost := range costs {
				payment[i] = cost()
			}
			consumer := mage.NewArtifact("Consumer", "", mage.WithAbility(mage.Activate(mage.Costs(payment...), mage.Sequence(plan.waitingDelayedPermanentChoice(bindings), shield(bindings)))))
			if err := mage.ValidateCardDefinition(consumer.Name(), consumer); err != nil {
				t.Fatal(err)
			}
			q := g.PutOnBattlefield(consumer, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 0, nil); err == nil || a.calls != 0 || g.StackSize() != 0 {
				t.Fatal("unpaid choice changed state")
			}
			a.ManaPool().Add(core.White, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if a.calls != 0 {
				t.Fatal("early source selection")
			}
			g.ResolveStack()
			if a.calls != 1 || !a.source.SameObject(saved) {
				t.Fatal("waiting shield body omitted departed source")
			}
			g.DealDamageToPlayerFromSource(a, 2, saved)
			g.DealDamageToPlayerFromSource(a, 1, saved)
			if a.Life() != 19 {
				t.Fatal("shield prevention or consumption failed")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			if len(g.WaitingDelayedPermanentSources()) != 0 {
				t.Fatal("shield delay remained waiting")
			}
			if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "ChooseADamageSource") {
				t.Fatal("general aggregation was accepted", err)
			}
		})
	}
}

func TestRONWaitingDelayedPreventionAmountTiming(t *testing.T) {
	for _, amount := range []string{counterInfoValue, `Minus(Integer(1),Integer(2))`, `Integer(0)`} {
		t.Run(amount, func(t *testing.T) {
			g, a, _ := ronGame()
			body := `CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(` + amount + `,Player(You)),[PreventThatDamage])`
			f := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([`+body+`]))`)
			p := g.PutOnBattlefield(f(), a.PlayerID())
			g.MutablePermanent(p.ID()).Counters[core.Charge] = 3
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.MutablePermanent(p.ID()).Counters[core.Charge] = 5
			g.ResolveStack()
			g.MutablePermanent(p.ID()).Counters[core.Charge] = 9
			clone := g.Clone()
			g.DealDamageToPlayer(a, 6, p.ID())
			want := 14
			if amount == counterInfoValue {
				want = 19
			}
			if a.Life() != want {
				t.Fatal("finite amount was not saved at installation", a.Life(), want)
			}
			clone.RunStepWithPriority(core.Cleanup)
			recipient := clone.GetPlayer(a.PlayerID())
			clone.DealDamageToPlayer(recipient, 6, p.ID())
			if recipient.Life() != 14 {
				t.Fatal("cleanup kept dynamic shield")
			}
		})
	}
}
