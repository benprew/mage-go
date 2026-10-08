package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedManaProductionGameplay(t *testing.T) {
	for _, instruction := range []string{`Integer(0)`, `Minus(PowerOfPermanent(ThisPermanent),Integer(2))`, `Minus(PowerOfPermanent(ThisPermanent),Integer(3))`, `PowerOfPermanent(ThisPermanent)`, `Plus(Integer(1),TheTotalNumberOfCountersOfTypeAmongPermanents(ChargeCounter,SinglePermanent(ThisPermanent)))`} {
		for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "control", "copy"} {
			t.Run(instruction+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				f := waitingCombatFactory(t, `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([AddManaRepeatedWithModifiers(AnyManaColor,`+instruction+`,CanOnlySpendOnSpells(IsCardtype(Creature)))]))]))`)
				c := f()
				c.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(c, a.PlayerID())
				saved := g.CaptureDamageSource(p.ID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				if len(g.WaitingDelayedPermanentSources()) != 0 {
					t.Fatal("delay created before resolution")
				}
				g.ResolveStack()
				p.AddCounter(core.Charge, 1)
				p.AddCounter(core.P1P1, 1)
				switch state {
				case "departed", "returned":
					g.BouncePermanentToHand(p)
					if state == "returned" {
						a.RemoveFromHand(c.ID())
						g.PutOnBattlefield(c, b.PlayerID())
					}
				case "phased", "cycle":
					ids := g.PhaseOut(p.ID())
					if state == "cycle" {
						g.PhaseIn(ids)
					}
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				if instruction != "Integer(0)" {
					for range 3 {
						got := g.WaitingDelayedPermanentSources()
						if len(got) != 1 || !got[0].SameObject(saved) || a.ManaPool().TotalMana() != 0 || b.ManaPool().TotalMana() != 0 {
							t.Fatal("query lost identity or added mana", got)
						}
					}
				}
				clone := g.Clone()
				for _, game := range []*mage.Game{clone, g} {
					game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					if len(game.WaitingDelayedPermanentSources()) != 0 || game.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 || game.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != 0 {
						t.Fatal("detection executed body or kept waiting")
					}
					game.PutTriggersOnStack()
					count := 1
					if state == "copy" {
						count = 2
					}
					if game.StackSize() != count {
						t.Fatal("lost delayed body")
					}
					game.ResolveStack()
					amount := 2
					if instruction == "Integer(0)" || strings.HasPrefix(instruction, "Minus(") {
						amount = 0
					}
					wantA, wantB := count*amount, 0
					if game.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != wantA || game.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != wantB || game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 20 {
						t.Fatal("wrong saved controller or amount")
					}
					if wantA > 0 {
						pool := game.GetPlayer(a.PlayerID()).ManaPool()
						creature, artifact := core.SpellPaymentContext{IsCreature: true}, core.SpellPaymentContext{}
						cost := core.ManaCost{Generic: wantA}
						if !pool.CanPay(cost, &creature) || pool.CanPay(cost, &artifact) || pool.CanPay(cost, nil) {
							t.Fatal("delayed mana lost its spending rule")
						}
					}
					waiting := 1
					if instruction == "Integer(0)" {
						waiting = 0
					}
					if game == clone && (len(g.WaitingDelayedPermanentSources()) != waiting || a.ManaPool().TotalMana() != 0 || b.ManaPool().TotalMana() != 0) {
						t.Fatal("clone changed original")
					}
				}
			})
		}
	}
}

func TestRONWaitingDelayedManaProductionInvalid(t *testing.T) {
	for _, amount := range []string{
		`PowerOfPermanent()`, `PowerOfPermanent(ThisPermanent())`, `PowerOfPermanent(Ref_TargetPermanent)`,
		`PowerOfPermanent(Trigger_ThatPermanent)`, `PowerOfPermanent(ActionPermanent)`,
		`TheNumberOfCountersRemovedThisWay`, `ManaValueOfTheSacrificedPermanent`, `ValueX`,
		`Integer(1,2)`, `Plus(Integer(1))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([AddManaRepeatedWithModifiers(AnyManaColor,`+amount+`,CanOnlySpendOnSpells(IsCardtype(Creature)))]))]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid delayed mana accepted", amount)
		}
	}
}

func TestRONWaitingDelayedManaProductionInstances(t *testing.T) {
	g, a, b := ronGame()
	f := waitingCombatFactory(t, `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([AddManaRepeatedWithModifiers(AnyManaColor,PowerOfPermanent(Ref_TargetPermanent),CanOnlySpendOnSpells(IsCardtype(Creature)))]))])))`)
	p, q := g.PutOnBattlefield(f(), a.PlayerID()), g.PutOnBattlefield(f(), b.PlayerID())
	p.AddCounter(core.P1P1, 1)
	q.AddCounter(core.P1P1, 2)
	if p.Card.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == q.Card.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("shared target declarations")
	}
	for _, source := range []*mage.Permanent{p, q} {
		if err := g.ActivateAbilityByIndex(source.ControllerID(), source.ID(), 0, []uuid.UUID{source.ID()}); err != nil {
			t.Fatal(err)
		}
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if len(g.WaitingDelayedPermanentSources()) != 2 {
		t.Fatal("lost saved target frames")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.ManaPool().TotalMana() != 2 || b.ManaPool().TotalMana() != 6 {
		t.Fatal("mana player bindings overwritten")
	}
}

func TestRONWaitingDelayedManaProductionProtection(t *testing.T) {
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
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([AddManaRepeatedWithModifiers(AnyManaColor,Plus(Integer(1),PowerOfPermanent(ThisPermanent)),CanOnlySpendOnSpells(IsCardtype(Creature)))]))]))`)
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
			manaBeforeDelay := a.ManaPool().TotalMana()
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			if len(g.WaitingDelayedPermanentSources()) != 0 || a.ManaPool().TotalMana() != manaBeforeDelay+1 || b.ManaPool().TotalMana() != 0 {
				t.Fatal("mana delay remained waiting")
			}
			if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "ChooseADamageSource") {
				t.Fatal("general aggregation was accepted", err)
			}
		})
	}
}
