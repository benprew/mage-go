package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedCombatPreventionProtection(t *testing.T) {
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
		for _, shape := range []string{"single", "to-and-by", "blockers", "blocked", "combat-duration"} {
			for _, state := range []string{"live", "departed", "returned", "phased", "control", "copy", "clone"} {
				t.Run(tc.name+"/"+shape+"/"+state, func(t *testing.T) {
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
					a := &ronDelayedPaymentPlayer{ronSourceChoicePlayer: &ronSourceChoicePlayer{BasePlayer: mage.NewBasePlayer("A")}, accept: true}
					b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
					g := mage.NewGame(a, b)
					bodies := map[string]string{
						"single":          `CreatePreventDamageUntil(CombatDamageWouldBeDealtByCreature(ThisPermanent),[PreventThatDamage],UntilEndOfTurn)`,
						"to-and-by":       `CreatePreventDamageUntil(Or([CombatDamageWouldBeDealtToRecipient(Permanent(ThisPermanent)),CombatDamageWouldBeDealtByCreature(ThisPermanent)]),[PreventThatDamage],UntilEndOfTurn)`,
						"blockers":        `CreatePreventDamageUntil(CombatDamageWouldBeDealtByACreature(Or([SinglePermanent(ThisPermanent),And([IsCardtype(Creature),IsBlockingAttacker(ThisPermanent)])])),[PreventThatDamage],UntilEndOfTurn)`,
						"blocked":         `CreatePreventDamageUntil(CombatDamageWouldBeDealtByACreature(Or([SinglePermanent(ThisPermanent),And([IsCardtype(Creature),IsBlockedByDefender(ThisPermanent)])])),[PreventThatDamage],UntilEndOfTurn)`,
						"combat-duration": `CreatePreventDamageUntil(CombatDamageWouldBeDealtByACreature(Or([SinglePermanent(ThisPermanent),And([IsCardtype(Creature),IsBlockingAttacker(ThisPermanent)])])),[PreventThatDamage],UntilEndOfCombat)`,
					}
					body := bodies[shape]
					factory := ronDelayedCombatFactory(t, `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))])))`)
					for _, name := range []string{"B1", "B2"} {
						creature := mage.NewCreature(name, "", 1, 4)
						creature.SetOwner(b.PlayerID())
						g.PutOnBattlefield(creature, b.PlayerID())
					}
					donorCard := mage.NewCreature("Donor", "", 1, 3)
					donorCard.SetOwner(a.PlayerID())
					donor := g.PutOnBattlefield(donorCard, a.PlayerID())
					card := factory()
					card.SetOwner(a.PlayerID())
					source := g.PutOnBattlefield(card, a.PlayerID())
					g.ChangeObjectColor(source.ID(), tc.color)
					original := g.CaptureDamageSource(source.ID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{donor.ID()}); err != nil {
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
						g.BouncePermanentToHand(source)
						if state == "returned" {
							a.RemoveFromHand(card.ID())
							g.PutOnBattlefield(card, b.PlayerID())
						}
					case "phased":
						g.PhaseOut(source.ID())
					case "control":
						g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					}
					binding := mage.BindDamageSource("waiting combat prevention source")
					bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
					payment := make([]mage.Cost, len(costs))
					for i, cost := range costs {
						payment[i] = cost()
					}
					consumer := mage.NewArtifact("Consumer", "", mage.WithAbility(mage.Activate(mage.Costs(payment...), mage.Sequence(plan.waitingDelayedPermanentChoice(bindings), shield(bindings)))))
					p := g.PutOnBattlefield(consumer, a.PlayerID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil || g.StackSize() != 0 || a.calls != 0 {
						t.Fatal("failed payment changed state")
					}
					a.ManaPool().Add(core.White, 2)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if a.calls != 0 || a.offers != 0 {
						t.Fatal("choice ran at announcement")
					}
					a.ManaPool().Clear()

					clone := g.Clone()
					if state == "clone" {
						clone.ResolveStack()
						clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
						clone.PutTriggersOnStack()
						clone.ResolveStack()
						if len(g.WaitingDelayedPermanentSources()) != 2 || a.calls != 0 || a.offers != 0 || a.Life() != 20 {
							t.Fatal("clone changed original")
						}
						wantClone := 20
						if clone.GetPlayer(a.PlayerID()).Life() != wantClone || len(clone.WaitingDelayedPermanentSources()) != 0 {
							t.Fatal("clone lost waiting lifetime")
						}
					}
					for range 3 {
						got := g.WaitingDelayedPermanentSources()
						found := false
						for _, candidate := range got {
							if candidate.SameObject(original) {
								found = true
							}
						}
						if len(got) != 2 || !found || a.offers != 0 || b.offers != 0 {
							t.Fatal("query lost source or offered payment")
						}
					}
					g.ResolveStack()
					if a.calls != 1 || !a.source.SameObject(original) || a.offers != 0 || b.offers != 0 {
						t.Fatal("combat prevention source read did not supply source choice")
					}
					g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					if len(g.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("detected delay still waits")
					}
					g.PutTriggersOnStack()
					g.ResolveStack()
					if a.Life() != 20 || b.Life() != 20 || a.offers != 0 || b.offers != 0 {
						t.Fatal("prevention query changed life or offered payment")
					}
					wantA := 20
					g.DealDamageToPlayerFromSource(a, 2, original)
					if a.Life() != wantA {
						t.Fatal("selected source damage not prevented")
					}
					g.DealDamageToPlayerFromSource(a, 1, original)
					if a.Life() != wantA-1 {
						t.Fatal("shield not consumed")
					}
					combatPrevented := func() bool {
						probe := g.Clone()
						if probe.FindPermanentIncludingPhased(source.ID()) == nil {
							return false
						}
						probe.PhaseIn([]uuid.UUID{source.ID()})
						return sourceCombatPrevented(probe, source.ID(), b.PlayerID())
					}
					prevented := combatPrevented()
					presentOriginal := state == "live" || state == "control" || state == "copy" || state == "clone"
					if prevented != presentOriginal {
						t.Fatal("prevention installed on absent phased or returned subject")
					}
					if g.GetEffects().ApplyReplacements(mage.NewDamageToPlayerAction(source.ID(), b.PlayerID(), 2, false), g) == nil {
						t.Fatal("combat prevention stopped noncombat damage")
					}
					g.GetEffects().RemoveEndOfCombat()
					if combatPrevented() != (presentOriginal && shape != "combat-duration") {
						t.Fatal("wrong combat expiry")
					}

					g.RunStepWithPriority(core.Cleanup)
					if combatPrevented() {
						t.Fatal("prevention survived cleanup")
					}
					if _, err := Compile(record); err != nil {
						t.Fatal("complete source-choice record rejected", err)
					}
				})
			}
		}
	}
}

func ronDelayedCombatFactory(t *testing.T, rule string) mage.CardFactory {
	t.Helper()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Delayed combat",Typeline:(Supertypes:[],Cardtypes:[Artifact,Creature],Subtypes:[]),CardPT:(Power:2,Toughness:10),Rules:[`+rule+`])`), "delayed-combat.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	return factory
}

func TestRONWaitingDelayedCombatPreventionInvalid(t *testing.T) {
	for _, action := range []string{
		`CreatePreventDamageUntil(CombatDamageWouldBeDealtByCreature(You),[PreventThatDamage],UntilEndOfTurn)`,
		`CreatePreventDamageUntil(CombatDamageWouldBeDealtByCreature(Ref_TargetPermanent),[PreventThatDamage],UntilEndOfTurn)`,
		`CreatePreventDamageUntil(CombatDamageWouldBeDealtByCreature(Trigger_ThatPermanent),[PreventThatDamage],UntilEndOfTurn)`,
		`CreatePreventDamageUntil(CombatDamageWouldBeDealtByCreature(ThisPermanent),[PreventThatDamage],UntilEndOfCombat)`,
		`CreatePreventDamageUntil(CombatDamageWouldBeDealt,[PreventThatDamage],UntilEndOfCombat)`,
		`CreatePreventDamageUntil(Or([CombatDamageWouldBeDealtToRecipient(Permanent(ThisPermanent)),CombatDamageWouldBeDealtByCreature(HostPermanent)]),[PreventThatDamage],UntilEndOfTurn)`,
		`CreatePreventDamageUntil(CombatDamageWouldBeDealtByACreature(Or([SinglePermanent(ThisPermanent)])),[PreventThatDamage],UntilEndOfTurn)`,
		`CreatePreventDamageUntil(CombatDamageWouldBeDealtByACreature(Or([SinglePermanent(ThisPermanent),And([IsCardtype(Land),IsBlockingAttacker(ThisPermanent)])])),[PreventThatDamage],UntilEndOfTurn)`,
		`CreatePreventDamageUntil(CombatDamageWouldBeDealtByCreature(ThisPermanent),[],UntilEndOfTurn)`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+action+`]))]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid delayed prevention accepted", action)
		}
	}
}

func TestRONWaitingDelayedCombatPreventionRelationInstances(t *testing.T) {
	for _, relation := range []string{"IsBlockingAttacker", "IsBlockedByDefender"} {
		t.Run(relation, func(t *testing.T) {
			g, a, b := ronGame()
			body := `CreatePreventDamageUntil(CombatDamageWouldBeDealtByACreature(Or([SinglePermanent(Ref_TargetPermanent),And([IsCardtype(Creature),` + relation + `(ThisPermanent)])])),[PreventThatDamage],UntilEndOfTurn)`
			factory := ronDelayedCombatFactory(t, `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))])))`)
			sources := make([]*mage.Permanent, 0, 2)
			targets := make([]*mage.Permanent, 0, 2)
			for _, player := range []*mage.BasePlayer{a, b} {
				source := g.PutOnBattlefield(factory(), player.PlayerID())
				target := g.PutOnBattlefield(mage.NewCreature("Target", "", 1, 10), player.PlayerID())
				if err := g.ActivateAbilityByIndex(player.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				sources = append(sources, source)
				targets = append(targets, target)
			}
			if len(g.WaitingDelayedPermanentSources()) != 4 {
				t.Fatal("empty relation lost source operand or factory capture")
			}
			clone := g.Clone()
			late := g.PutOnBattlefield(mage.NewCreature("Late opponent", "", 2, 10), b.PlayerID())
			if relation == "IsBlockingAttacker" {
				sources[0].RevokeBaseAttr(core.AttrSummonSick)
				g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{sources[0].ID()})
				g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: late.ID(), AttackerID: sources[0].ID()}})
			} else {
				late.RevokeBaseAttr(core.AttrSummonSick)
				g.SetActivePlayerIndex(1)
				g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{late.ID()})
				g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: sources[0].ID(), AttackerID: late.ID()}})
			}
			if len(g.WaitingDelayedPermanentSources()) != 4 {
				t.Fatal("query selected relation members")
			}
			for _, game := range []*mage.Game{g, clone} {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				if len(game.WaitingDelayedPermanentSources()) != 0 {
					t.Fatal("detected ability still waits")
				}
				game.PutTriggersOnStack()
				game.ResolveStack()
				for _, target := range targets {
					if !sourceCombatPrevented(game, target.ID(), a.PlayerID()) {
						t.Fatal("factory lost target")
					}
				}
				for _, source := range sources {
					if sourceCombatPrevented(game, source.ID(), a.PlayerID()) {
						t.Fatal("relation operand became a prevention subject")
					}
				}
			}
			g.RemoveFromCombat(late.ID())
			if !sourceCombatPrevented(g, late.ID(), a.PlayerID()) {
				t.Fatal("late relation member not captured at resolution")
			}
			clone.PutOnBattlefield(late.Card, b.PlayerID())
			if sourceCombatPrevented(clone, late.ID(), a.PlayerID()) {
				t.Fatal("clone inherited later relation")
			}
		})
	}
}
