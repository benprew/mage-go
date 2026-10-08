package ron

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedDamageBodyProtection(t *testing.T) {
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
		for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "copy", "clone"} {
			t.Run(tc.name+"/"+state, func(t *testing.T) {
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
				factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PermanentDealsDamage(ThisPermanent,Integer(2),Player(You))]))]))`)
				sourceCard := factory()
				sourceCard.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(sourceCard, a.PlayerID())
				g.ChangeObjectColor(source.ID(), tc.color)
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
				if state == "clone" {
					clone := g.Clone()
					clone.ResolveStack()
					clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					clone.PutTriggersOnStack()
					clone.ResolveStack()
					if clone.GetPlayer(a.PlayerID()).Life() != 20 || chooser.calls != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
						t.Fatal("clone lost source or shared lifetime")
					}
				}
				g.ResolveStack()
				if chooser.calls != 1 || !chooser.source.SameObject(original) {
					t.Fatal("damage-only delayed body lost source choice")
				}
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				if len(g.WaitingDelayedPermanentSources()) != 0 {
					t.Fatal("fired ability remained waiting")
				}
				g.PutTriggersOnStack()
				g.ResolveStack()
				want := 20
				if state == "copy" {
					want = 18
				}
				if a.Life() != want || b.Life() != 20 {
					t.Fatal("delayed damage or shield consumption failed")
				}
				g.DealDamageToPlayerFromSource(chooser, 1, original)
				if a.Life() != want-1 {
					t.Fatal("shield was not consumed")
				}
				if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "ChooseADamageSource") {
					t.Fatal("incomplete general aggregation accepted", err)
				}
			})
		}
	}
}

func TestRONWaitingDelayedDamageBodyInvalid(t *testing.T) {
	plan := &chosenDamageSourcePlan{}
	binding := mage.BindDamageSource("source")
	bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
	for _, body := range []mage.ResolvingEffect{
		plan.waitingDelayedPermanentChoice(instanceBindings{}),
		mage.Sequence(mage.DealDamageFrom(mage.FromDamageSource(binding), mage.You, mage.Fixed(1)), plan.waitingDelayedPermanentChoice(bindings)),
	} {
		if mage.ValidateCardDefinition("Invalid", mage.NewArtifact("Invalid", "", mage.WithAbility(mage.Activate(mage.Costs(), body)))) == nil {
			t.Fatal("invalid instance or read-before-choice accepted")
		}
	}
	for _, text := range []string{`ChooseADamageSource()`, `ChooseADamageSource(ThisPermanent)`, `ChooseADamageSource(IsColor(Colorless))`, `ChooseADamageSource(And([]))`} {
		nodes, err := Parse(strings.NewReader(text), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compileChosenDamageSourceChoice(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", text)
		}
	}
}
