package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedPlayerDraw(t *testing.T) {
	for _, state := range []string{"live", "departed", "returned", "phased", "clone"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			factory := ronDelayedCombatFactory(t, `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([EachPlayerAction(AnyPlayer,DrawNumberCards(PowerOfPermanent(ThisPermanent)))]))]))`)
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			for range 4 {
				a.AddToLibrary(repositoryCard(t, "LEB", "Island"))
				b.AddToLibrary(repositoryCard(t, "LEB", "Island"))
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if len(g.WaitingDelayedPermanentSources()) != 0 {
				t.Fatal("delay created before resolution")
			}
			g.ResolveStack()
			switch state {
			case "departed", "returned":
				g.BouncePermanentToHand(p)
				if state == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phased":
				g.PhaseOut(p.ID())
			}
			if got := g.WaitingDelayedPermanentSources(); len(got) != 1 || !got[0].SameObject(saved) {
				t.Fatal("draw lost source reference")
			}
			beforeA, beforeB := len(a.Hand()), len(b.Hand())
			if state == "clone" {
				clone := g.Clone()
				clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				clone.PutTriggersOnStack()
				clone.ResolveStack()
				if len(clone.GetPlayer(a.PlayerID()).Hand()) != beforeA+2 || len(a.Hand()) != beforeA || len(g.WaitingDelayedPermanentSources()) != 1 {
					t.Fatal("clone changed original draw or wait")
				}
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			if len(g.WaitingDelayedPermanentSources()) != 0 {
				t.Fatal("triggered draw remains waiting")
			}
			g.PutTriggersOnStack()
			g.ResolveStack()
			if len(a.Hand()) != beforeA+2 || len(b.Hand()) != beforeB+2 {
				t.Fatal("draw did not use source power or LKI for both players")
			}
		})
	}
}

func TestRONWaitingDelayedPlayerInvalid(t *testing.T) {
	for _, value := range []string{`PowerOfPermanent(Ref_TargetPermanent)`, `PowerOfPermanent(Trigger_ThatPermanent)`, `PowerOfPermanent(ActionPermanent)`, `PowerOfPermanent(You)`, `TheNumberOfCardsInPlayersHand(ActionPlayer)`, `Integer(-1)`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([EachPlayerAction(AnyPlayer,DrawNumberCards(` + value + `))]))]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid-delayed-player.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted invalid value %s", value)
		}
	}
}

func TestRONWaitingDelayedPlayerProtection(t *testing.T) {
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
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([EachPlayerAction(AnyPlayer,DrawNumberCards(Minus(PowerOfPermanent(ThisPermanent),PowerOfPermanent(ThisPermanent))))]))]))`)
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
			a.RemoveFromHand(card.ID())
			g.PutOnBattlefield(card, b.PlayerID())
			binding := mage.BindDamageSource("waiting player collection")
			bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
			payment := make([]mage.Cost, len(costs))
			for i, cost := range costs {
				payment[i] = cost()
			}
			consumer := mage.NewArtifact("Consumer", "", mage.WithAbility(mage.Activate(mage.Costs(payment...), mage.Sequence(plan.waitingDelayedPermanentChoice(bindings), shield(bindings)))))
			q := g.PutOnBattlefield(consumer, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 0, nil); err == nil || g.StackSize() != 0 || a.calls != 0 {
				t.Fatal("failed payment changed choice or stack")
			}
			a.ManaPool().Add(core.White, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if a.calls != 0 {
				t.Fatal("choice ran at announcement")
			}
			g.ResolveStack()
			if a.calls != 1 || !a.source.SameObject(saved) {
				t.Fatal("choice lost original draw operand source")
			}
			if err := mage.ApplyEffect(g, mage.DealDamageFrom(mage.FromDamageSource(binding), mage.You, mage.Fixed(3)), q.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if a.Life() != 20 {
				t.Fatal("RON shield did not prevent damage")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			if len(g.WaitingDelayedPermanentSources()) != 0 || len(a.Hand()) != 0 || len(b.Hand()) != 0 {
				t.Fatal("zero draw changed hands or retained waiting source")
			}
		})
	}
}

func TestRONWaitingDelayedPlayerInstanceValues(t *testing.T) {
	g, a, b := ronGame()
	for range 12 {
		a.AddToLibrary(repositoryCard(t, "LEB", "Island"))
		b.AddToLibrary(repositoryCard(t, "LEB", "Island"))
	}
	factory := ronDelayedCombatFactory(t, `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([EachPlayerAction(AnyPlayer,DrawNumberCards(PowerOfPermanent(Ref_TargetPermanent)))]))])))`)
	first, second := factory(), factory()
	first.SetOwner(a.PlayerID())
	second.SetOwner(a.PlayerID())
	p, q := g.PutOnBattlefield(first, a.PlayerID()), g.PutOnBattlefield(second, a.PlayerID())
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ModifyPT(mage.This, mage.Fixed(1), mage.Fixed(0))), q.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]*mage.Permanent{{p, p}, {q, q}} {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), pair[0].ID(), 0, []uuid.UUID{pair[1].ID()}); err != nil {
			t.Fatal(err)
		}
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("ability copy failed")
	}
	g.ResolveStack()
	if len(g.WaitingDelayedPermanentSources()) != 2 {
		t.Fatal("target frames lost distinct identities")
	}
	g.BouncePermanentToHand(p)
	a.RemoveFromHand(first.ID())
	g.PutOnBattlefield(first, b.PlayerID())
	clone := g.Clone()
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if len(clone.GetPlayer(a.PlayerID()).Hand()) != 8 || len(a.Hand()) != 0 {
		t.Fatal("clone lost copied target amounts or isolation")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if len(a.Hand()) != 8 || len(b.Hand()) != 8 {
		t.Fatal("factory or copied target amounts shared bindings")
	}
}

func TestRONWaitingDelayedPlayerAmountOnce(t *testing.T) {
	g, a, b := ronGame()
	for range 4 {
		a.AddToLibrary(repositoryCard(t, "LEB", "Island"))
		b.AddToLibrary(repositoryCard(t, "LEB", "Island"))
	}
	for range 2 {
		a.AddToHand(repositoryCard(t, "LEB", "Forest"))
	}
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([EachPlayerAction(AnyPlayer,DrawNumberCards(TheNumberOfCardsInPlayersHand(You)))]))]))`)
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if len(a.Hand()) != 4 || len(b.Hand()) != 2 {
		t.Fatal("draw amount was read again after the first draw")
	}
}
