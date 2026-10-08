package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedLayerProtection(t *testing.T) {
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
		for _, shape := range []string{"keyword", "dynamic", "upkeep"} {
			for _, state := range []string{"live", "departed", "returned", "phased", "clone"} {
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
					a := &ronSourceChoicePlayer{BasePlayer: mage.NewBasePlayer("A")}
					b := mage.NewBasePlayer("B")
					g := mage.NewGame(a, b)
					mod, duration := `AddAbility([Flying])`, `UntilEndOfTurn`
					if shape == "dynamic" {
						mod = `AdjustPTX(PlusX, Zero, PowerOfPermanent(ThisPermanent))`
					}
					if shape == "upkeep" {
						duration = `UntilNextUpkeep(You)`
					}
					factory := ronDelayedCombatFactory(t, fmt.Sprintf(`Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[%s],%s)]))]))`, mod, duration))
					card := factory()
					card.SetOwner(a.PlayerID())
					source := g.PutOnBattlefield(card, a.PlayerID())
					g.ChangeObjectColor(source.ID(), tc.color)
					saved := g.CaptureDamageSource(source.ID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if len(g.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("delay created at announcement")
					}
					g.ResolveStack()
					current := source.ID()
					switch state {
					case "departed", "returned":
						g.BouncePermanentToHand(source)
						if state == "returned" {
							a.RemoveFromHand(card.ID())
							current = g.PutOnBattlefield(card, b.PlayerID()).ID()
						}
					case "phased":
						g.PhaseOut(source.ID())
					}
					if got := g.WaitingDelayedPermanentSources(); len(got) != 1 || !got[0].SameObject(saved) {
						t.Fatal("layer lost waiting source")
					}
					binding := mage.BindDamageSource("waiting layer")
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
					if a.calls != 0 {
						t.Fatal("choice ran at announcement")
					}
					g.ResolveStack()
					if a.calls != 1 || !a.source.SameObject(saved) {
						t.Fatal("choice lost exact layer reference")
					}
					if err := mage.ApplyEffect(g, mage.DealDamageFrom(mage.FromDamageSource(binding), mage.You, mage.Fixed(3)), p.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					if a.Life() != 20 {
						t.Fatal("real RON shield did not prevent damage")
					}
					clone := g.Clone()
					if state == "clone" {
						clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
						if len(g.WaitingDelayedPermanentSources()) != 1 {
							t.Fatal("clone consumed original delay")
						}
					}
					g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					if len(g.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("triggered layer still waiting")
					}
					g.PutTriggersOnStack()
					g.ResolveStack()
					g.PhaseIn([]uuid.UUID{current})
					if state != "departed" {
						subject := g.FindPermanent(current)
						installed := state == "live" || state == "clone"
						if subject.HasKeyword(core.Flying) != (installed && shape != "dynamic") {
							t.Fatal("layer followed absent or returned subject")
						}
						want := 2
						if installed && shape == "dynamic" {
							want = 4
						}
						if subject.CurrentPower(g) != want {
							t.Fatal("dynamic layer lost resolution value")
						}
						if shape != "upkeep" {
							g.GetEffects().RemoveEndOfTurn()
							g.ApplyContinuousEffects()
							if subject.CurrentPower(g) != 2 || subject.HasKeyword(core.Flying) {
								t.Fatal("layer survived cleanup")
							}
						}
					}
				})
			}
		}
	}
}

func TestRONWaitingDelayedLayerInvalid(t *testing.T) {
	for _, action := range []string{
		`CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[AddAbility([Flying])],UntilEndOfTurn)`,
		`CreatePermanentLayerEffectUntil(You,[AddAbility([Flying])],UntilEndOfTurn)`,
		`CreatePermanentLayerEffectUntil(ThisPermanent,[],UntilEndOfTurn)`,
		`CreatePermanentLayerEffectUntil(ThisPermanent,[AdjustPTX(PlusX,Zero,PowerOfPermanent(Trigger_ThatPermanent))],UntilEndOfTurn)`,
		`CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbility([Flying])],UntilEndOfCombat())`,
		`CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbility([Flying])],UntilNextUpkeep(Opponent))`,
		`CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbility([Flying])],ForAsLongAsPermanentRemainsTapped(Ref_TargetPermanent))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + action + `]))]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid-delayed-layer.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("invalid declaration accepted: %s", action)
		}
	}
}

func TestRONWaitingDelayedLayerInstanceValues(t *testing.T) {
	g, a, b := ronGame()
	targetCard := repositoryCard(t, "LEB", "Llanowar Elves")
	targetCard.SetOwner(b.PlayerID())
	target := g.PutOnBattlefield(targetCard, b.PlayerID())
	factory := ronDelayedCombatFactory(t, `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[AdjustPTX(PlusX,Zero,PowerOfPermanent(ThisPermanent))],UntilEndOfTurn)]))])))`)
	first, second := factory(), factory()
	first.SetOwner(a.PlayerID())
	second.SetOwner(a.PlayerID())
	p, q := g.PutOnBattlefield(first, a.PlayerID()), g.PutOnBattlefield(second, a.PlayerID())
	for _, source := range []*mage.Permanent{p, q} {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
			t.Fatal(err)
		}
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if len(g.WaitingDelayedPermanentSources()) != 3 {
		t.Fatal("source and target frames did not retain distinct identities")
	}
	g.BouncePermanentToHand(p)
	a.RemoveFromHand(first.ID())
	g.PutOnBattlefield(first, a.PlayerID())
	clone := g.Clone()
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if clone.FindPermanent(target.ID()).CurrentPower(clone) != 7 || g.FindPermanent(target.ID()).CurrentPower(g) != 1 {
		t.Fatal("clone or copied value frame lost isolation")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindPermanent(target.ID()).CurrentPower(g) != 7 {
		t.Fatalf("departed source LKI value: got %d, want 7", g.FindPermanent(target.ID()).CurrentPower(g))
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if g.FindPermanent(target.ID()).CurrentPower(g) != 1 || clone.FindPermanent(target.ID()).CurrentPower(clone) != 7 {
		t.Fatal("cleanup shared layer state")
	}
}
