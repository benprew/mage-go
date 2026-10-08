package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedMillGameplay(t *testing.T) {
	for _, operand := range []string{"player", "amount", "zero"} {
		for _, state := range []string{"live", "departed", "returned", "phased", "control", "copy", "clone"} {
			t.Run(operand+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				action := `PlayerAction(ControllerOfPermanent(ThisPermanent),MillNumberCards(Integer(2)))`
				if operand == "amount" {
					action = `MillNumberCards(PowerOfPermanent(ThisPermanent))`
				}
				if operand == "zero" {
					action = `MillNumberCards(Minus(PowerOfPermanent(ThisPermanent),PowerOfPermanent(ThisPermanent)))`
				}
				factory := ronDelayedCombatFactory(t, `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+action+`]))]))`)
				card := factory()
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				saved := g.CaptureDamageSource(p.ID())
				for range 6 {
					a.AddToLibrary(repositoryCard(t, "LEB", "Island"))
					b.AddToLibrary(repositoryCard(t, "LEB", "Island"))
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				if len(g.WaitingDelayedPermanentSources()) != 0 {
					t.Fatal("delay created at announcement")
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
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				for range 3 {
					got := g.WaitingDelayedPermanentSources()
					if len(got) != 1 || !got[0].SameObject(saved) || len(a.Library()) != 6 || len(b.Library()) != 6 {
						t.Fatal("query lost mill source or ran body")
					}
				}
				clone := g.Clone()
				for _, game := range []*mage.Game{clone, g} {
					game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					game.PutTriggersOnStack()
					game.ResolveStack()
					wantA, wantB := 2, 0
					if operand == "zero" {
						wantA = 0
					}
					if state == "copy" {
						wantA *= 2
					}
					if operand == "player" && state == "control" {
						wantB, wantA = wantA, 0
					}
					if len(game.GetPlayer(a.PlayerID()).Graveyard()) != wantA || len(game.GetPlayer(b.PlayerID()).Graveyard()) != wantB {
						t.Fatal("wrong mill player or amount", len(game.GetPlayer(a.PlayerID()).Graveyard()), len(game.GetPlayer(b.PlayerID()).Graveyard()))
					}
					if len(game.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("resolved mill still waits")
					}
					if game == clone && (len(a.Library()) != 6 || len(g.WaitingDelayedPermanentSources()) != 1) {
						t.Fatal("clone changed original")
					}
				}
			})
		}
	}
}

func TestRONWaitingDelayedMillInvalid(t *testing.T) {
	for _, action := range []string{
		`MillNumberCards()`, `MillNumberCards(Integer(1),Integer(2))`, `MillNumberCards(Integer(-1))`,
		`MillNumberCards(PowerOfPermanent(Ref_TargetPermanent))`, `MillNumberCards(PowerOfPermanent(Trigger_ThatPermanent))`,
		`MillNumberCards(PowerOfPermanent(ActionPermanent))`, `MillNumberCards(PowerOfPermanent(You))`,
		`MillNumberCards(TheNumberOfCardsInPlayersHand(ActionPlayer))`, `MillNumberCards(ValueX)`,
		`PlayerAction(Ref_TargetPlayer,MillNumberCards(Integer(1)))`,
		`MayCost(MillNumberCards(PowerOfPermanent(ThisPermanent))),Unless(CostWasPaid,[GainLife(Integer(1))])`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + action + `]))]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid-mill.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid delayed mill accepted", action)
		}
	}
}

func TestRONWaitingDelayedMillProtection(t *testing.T) {
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
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PlayerAction(ControllerOfPermanent(ThisPermanent),MillNumberCards(Minus(ManaValueOfPermanent(ThisPermanent),ManaValueOfPermanent(ThisPermanent))))]))]))`)
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
			if _, err := Compile(record); err != nil {
				t.Fatal("complete source-choice record rejected", err)
			}
		})
	}
}

func TestRONWaitingDelayedMillInstances(t *testing.T) {
	g, a, b := ronGame()
	for range 12 {
		a.AddToLibrary(repositoryCard(t, "LEB", "Island"))
		b.AddToLibrary(repositoryCard(t, "LEB", "Island"))
	}
	factory := ronDelayedCombatFactory(t, `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([MillNumberCards(PowerOfPermanent(Ref_TargetPermanent))]))])))`)
	first, second := factory(), factory()
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	p, q := g.PutOnBattlefield(first, a.PlayerID()), g.PutOnBattlefield(second, b.PlayerID())
	for _, subject := range []*mage.Permanent{p, q} {
		if err := g.ActivateAbilityByIndex(subject.ControllerID(), subject.ID(), 0, []uuid.UUID{subject.ID()}); err != nil {
			t.Fatal(err)
		}
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ModifyPT(mage.This, mage.Fixed(1), mage.Fixed(0))), q.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(p)
	a.RemoveFromHand(first.ID())
	g.PutOnBattlefield(first, b.PlayerID())
	g.MutablePermanent(q.ID()).GrantBaseAttr(core.Shroud)
	clone := g.Clone()
	for _, game := range []*mage.Game{clone, g} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if len(game.GetPlayer(a.PlayerID()).Graveyard()) != 2 || len(game.GetPlayer(b.PlayerID()).Graveyard()) != 6 {
			t.Fatal("shared target or amount read before resolution")
		}
		if game == clone && (len(a.Graveyard()) != 0 || len(g.WaitingDelayedPermanentSources()) != 2) {
			t.Fatal("clone shared mill bindings")
		}
	}
}

func TestRONWaitingDelayedMillModifiers(t *testing.T) {
	for _, amount := range []string{`PowerOfPermanent(ThisPermanent)`, `Minus(Integer(0),PowerOfPermanent(ThisPermanent))`} {
		t.Run(amount, func(t *testing.T) {
			g, a, b := ronGame()
			for range 6 {
				a.AddToLibrary(repositoryCard(t, "LEB", "Island"))
			}
			factory := ronDelayedCombatFactory(t, `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([MillNumberCards(`+amount+`)]))]))`)
			p := g.PutOnBattlefield(factory(), a.PlayerID())
			calls := 0
			modifier := g.PutOnBattlefield(mage.NewArtifact("Modifier", ""), b.PlayerID())
			g.AddMillReplacement(modifier.ID(), func(_ *mage.Game, _ uuid.UUID, n int) int { calls++; return n * 2 })
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			for range 3 {
				if len(g.WaitingDelayedPermanentSources()) != 1 || calls != 0 {
					t.Fatal("query called modifier")
				}
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			want := 4
			if strings.HasPrefix(amount, "Minus") {
				want = 0
			}
			if len(a.Graveyard()) != want || calls > 1 || (want > 0 && calls != 1) {
				t.Fatal("wrong modified mill count", len(a.Graveyard()), calls)
			}
		})
	}
}
