package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedRedirectOperands(t *testing.T) {
	for name, body := range map[string]string{
		"source":      `CreateReplaceWouldDealDamageUntil(DamageWouldBeDealtByPermanentToRecipient(ThisPermanent,Player(You)),[DealToCreatureOrPlaneswalkerInstead(ThisPermanent)],UntilEndOfTurn)`,
		"destination": `CreateReplaceWouldDealDamageUntil(DamageWouldBeDealtByPermanentToRecipient(Ref_TargetPermanent,Player(You)),[DealToCreatureOrPlaneswalkerInstead(ThisPermanent)],UntilEndOfTurn)`,
		"player":      `CreateReplaceWouldDealDamageUntil(DamageWouldBeDealtByPermanentToRecipient(Ref_TargetPermanent,Player(ControllerOfPermanent(ThisPermanent))),[DealToCreatureOrPlaneswalkerInstead(Ref_TargetPermanent)],UntilEndOfTurn)`,
	} {
		for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "control", "copy", "clone"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				f := ronDelayedCombatFactory(t, `Activated(PayMana([ManaCostGeneric(1)]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))])))`)
				card := f()
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				q := g.PutOnBattlefield(mage.NewCreature("Target", "", 1, 20), a.PlayerID())
				saved := g.CaptureDamageSource(p.ID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err == nil || g.StackSize() != 0 {
					t.Fatal("failed payment installed delay")
				}
				a.ManaPool().Add(core.White, 1)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err != nil {
					t.Fatal(err)
				}
				if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				if len(g.WaitingDelayedPermanentSources()) != 0 {
					t.Fatal("delay installed at announcement")
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
				case "cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				refs := g.WaitingDelayedPermanentSources()
				found := false
				for _, ref := range refs {
					found = found || ref.SameObject(saved)
				}
				if !found || len(g.WaitingRedirectionPermanentSources()) != 0 {
					t.Fatal("waiting redirection lost source or installed early", refs)
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
				source := game.CaptureDamageSource(q.ID())
				if name == "source" {
					source = saved
				}
				recipient := game.GetPlayer(a.PlayerID())
				if name == "player" && state == "control" {
					recipient = game.GetPlayer(b.PlayerID())
				}
				game.DealDamageToPlayerFromSource(recipient, 2, source)
				works := name == "player" || state != "departed" && state != "returned"
				want := 18
				if works {
					want = 20
				}
				if recipient.Life() != want {
					t.Fatal("wrong saved source, player or destination", recipient.Life(), want)
				}
				if state == "copy" && name != "player" && works {
					game.DealDamageToPlayerFromSource(game.GetPlayer(b.PlayerID()), 2, source)
					if game.GetPlayer(b.PlayerID()).Life() != 20 {
						t.Fatal("copy lost its controller")
					}
				}
				game.GetEffects().ClearReplacementsEndOfTurn()
				game.DealDamageToPlayerFromSource(recipient, 1, source)
				if recipient.Life() != want-1 {
					t.Fatal("cleanup kept redirection")
				}
				if state == "clone" && (a.Life() != 20 || len(g.WaitingDelayedPermanentSources()) == 0) {
					t.Fatal("clone changed original")
				}
			})
		}
	}
}

func TestRONWaitingDelayedRedirectInvalid(t *testing.T) {
	for _, body := range []string{
		`CreateReplaceWouldDealDamageUntil(DamageWouldBeDealtByPermanentToRecipient(ThisPermanent,Player(You)),[DealToCreatureOrPlaneswalkerInstead(ThisPermanent)],UntilEndOfCombat)`,
		`CreateReplaceWouldDealDamageUntil(DamageWouldBeDealtByPermanentToRecipient(ThisPermanent,Player(Trigger_ThatPlayer)),[DealToCreatureOrPlaneswalkerInstead(ThisPermanent)],UntilEndOfTurn)`,
		`CreateReplaceWouldDealDamageUntil(DamageWouldBeDealtByPermanentToRecipient(ThisPermanent,Player(You)),[DealToCreatureOrPlaneswalkerInstead(Ref_TargetPermanent)],UntilEndOfTurn)`,
		`CreateFutureReplaceWouldDealDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Permanent(ThisPermanent)),[DealToPlayerInstead(You)])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))]))])`), "invalid-delay.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid delayed declaration accepted", body)
		}
	}
}

func TestRONWaitingDelayedRedirectProtection(t *testing.T) {
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
			factory := ronDelayedCombatFactory(t, `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([CreateReplaceWouldDealDamageUntil(DamageWouldBeDealtByPermanentToRecipient(ThisPermanent,Player(You)),[DealToCreatureOrPlaneswalkerInstead(ThisPermanent)],UntilEndOfTurn)]))]))`)
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
			binding := mage.BindDamageSource("waiting redirection")
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
				t.Fatal("waiting redirection body omitted departed source")
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
				t.Fatal("redirection delay remained waiting")
			}
			if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "ChooseADamageSource") {
				t.Fatal("general aggregation was accepted", err)
			}
		})
	}
}

func TestRONWaitingDelayedSelectedRedirect(t *testing.T) {
	for _, operand := range []string{"subject", "player"} {
		for _, state := range []string{"live", "departed", "returned", "phased", "copy", "clone"} {
			t.Run(operand+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				record := staticBaseRecord(t, "LEA", "Jade Monolith")
				actions := record.Fields["Rules"].Args[0].Args[1].Args[1].Args[0].Args
				choice, err := compileChosenDamageSourceChoice(actions[0])
				if err != nil {
					t.Fatal(err)
				}
				subject, player := "ThisPermanent", "You"
				if operand == "player" {
					subject = "Ref_TargetPermanent"
					player = "ControllerOfPermanent(ThisPermanent)"
				}
				nodes, err := Parse(strings.NewReader(`Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureReplaceWouldDealDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Permanent(`+subject+`)),[DealToPlayerInstead(`+player+`)])]))`), "selected-delay.ron")
				if err != nil {
					t.Fatal(err)
				}
				delay, err := compileActions(nodes[0], scope{permanent: true, chosenDamageSource: choice})
				if err != nil {
					t.Fatal(err)
				}
				q := g.PutOnBattlefield(mage.NewCreature("Damage source", "", 1, 20), a.PlayerID())
				factory := func() mage.Card {
					binding := mage.BindDamageSource("chosen source")
					bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{choice: binding}}
					body := delay.buildWithBindings(bindings)
					body.effects = []mage.Effect{mage.AtNext(mage.EachEndStep, mage.Sequence(append([]mage.Effect{mage.ChooseDamageSourceFrom(mage.You, []mage.DamageSourceRef{mage.This}, binding)}, body.effects...)...))}
					return mage.NewCreature("Selected delay", "", 2, 20, mage.WithAbility(Activated(nil, body)))
				}
				card := factory()
				card.SetOwner(a.PlayerID())
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				p := g.PutOnBattlefield(card, a.PlayerID())
				saved := g.CaptureDamageSource(p.ID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err != nil {
					t.Fatal(err)
				}
				if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
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
				got := g.WaitingDelayedPermanentSources()
				found := false
				for _, ref := range got {
					found = found || ref.SameObject(saved)
				}
				if len(got) != 2 || !found {
					t.Fatal("selected redirect source declaration missing", got)
				}
				game := g
				if state == "clone" {
					game = g.Clone()
				}
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
				if state == "phased" {
					game.PhaseIn([]uuid.UUID{p.ID()})
				}
				target := game.FindPermanent(p.ID())
				if operand == "player" {
					target = game.FindPermanent(q.ID())
				}
				if target != nil {
					game.DealDamageToPermanentFromSource(target, 2, saved)
					want := 20
					if operand == "player" || state == "live" || state == "clone" || state == "copy" || state == "phased" {
						want = 18
					}
					if operand == "subject" && state == "copy" {
						if game.GetPlayer(a.PlayerID()).Life()+game.GetPlayer(b.PlayerID()).Life() != 38 {
							t.Fatal("copy lost saved controller")
						}
					} else if game.GetPlayer(a.PlayerID()).Life() != want {
						t.Fatal("selected redirection lost identity or player", game.GetPlayer(a.PlayerID()).Life(), want)
					}
					if state == "copy" {
						game.DealDamageToPermanentFromSource(target, 2, saved)
						if operand == "subject" && (game.GetPlayer(a.PlayerID()).Life() != 18 || game.GetPlayer(b.PlayerID()).Life() != 18) {
							t.Fatal("copied redirects lost separate controllers")
						}
						if operand == "player" && game.GetPlayer(a.PlayerID()).Life() != 16 {
							t.Fatal("copied controller references lost source player")
						}
					}
					game.DealDamageToPermanentFromSource(target, 1, saved)
					wantDamage := 1
					if operand == "subject" && state == "returned" {
						wantDamage = 3
					}
					if game.FindPermanent(target.ID()).Damage != wantDamage {
						t.Fatal("wrong one-shot damage", game.FindPermanent(target.ID()).Damage, wantDamage)
					}
				}
				if len(game.WaitingDelayedPermanentSources()) != 0 {
					t.Fatal("detected delay remains waiting")
				}
				if state == "clone" && len(g.WaitingDelayedPermanentSources()) != 2 {
					t.Fatal("clone changed original lifetime")
				}
			})
		}
	}
}
