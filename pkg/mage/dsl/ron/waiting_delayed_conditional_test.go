package ron

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedConditionalProtection(t *testing.T) {
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
		for _, branch := range []string{"if", "unless", "else"} {
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
					chooser := &ronSourceChoicePlayer{BasePlayer: a}
					g := mage.NewGame(chooser, b)
					body := `If(PermanentPassesFilter(ThisPermanent,IsUntapped),[PermanentDealsDamage(ThisPermanent,Integer(2),Player(You))])`
					if branch == "unless" {
						body = `Unless(PermanentPassesFilter(ThisPermanent,IsTapped),[PermanentDealsDamage(ThisPermanent,Integer(2),Player(You))])`
					}
					if branch == "else" {
						body = `IfElse(PermanentPassesFilter(ThisPermanent,IsTapped),[GainLife(Integer(1))],[PermanentDealsDamage(ThisPermanent,Integer(2),Player(You))])`
					}
					factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))]))`)
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
						t.Fatal("conditional delayed body lost source choice")
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
					if _, err := Compile(record); err != nil {
						t.Fatal("complete source-choice record rejected", err)
					}
				})
			}
		}
	}

}

func TestRONWaitingDelayedConditionalInstances(t *testing.T) {
	g, a, b := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPermanentBecomesTapped(SinglePermanent(ThisPermanent)),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([If(PermanentPassesFilter(Trigger_ThatPermanent,IsUntapped),[PermanentDealsDamage(Trigger_ThatPermanent,Integer(2),Player(You))])]))]))`)
	first, second := factory(), factory()
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(first, a.PlayerID())
	q := g.PutOnBattlefield(second, b.PlayerID())
	g.TapPermanent(p)
	g.TapPermanent(q)
	g.PutTriggersOnStack()
	g.ResolveStack()
	savedP, savedQ := g.CaptureDamageSource(p.ID()), g.CaptureDamageSource(q.ID())
	g.UntapPermanent(q)
	got := g.WaitingDelayedPermanentSources()
	if len(got) != 2 {
		t.Fatal("conditional instances lost references")
	}
	foundP, foundQ := false, false
	for _, source := range got {
		foundP = foundP || source.SameObject(savedP)
		foundQ = foundQ || source.SameObject(savedQ)
	}
	if !foundP || !foundQ {
		t.Fatal("conditional instances shared source identity")
	}
	clone := g.Clone()
	clone.UntapPermanent(clone.FindPermanent(p.ID()))
	for _, game := range []*mage.Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
	}
	if a.Life() != 20 || b.Life() != 18 || clone.GetPlayer(a.PlayerID()).Life() != 18 || clone.GetPlayer(b.PlayerID()).Life() != 18 {
		t.Fatal("condition did not use independent resolution-time state or saved controller")
	}
}

func TestRONWaitingDelayedConditionalDeclarations(t *testing.T) {
	for _, body := range []string{
		`If(PermanentPassesFilter(ThisPermanent(),IsUntapped),[GainLife(Integer(1))])`,
		`If(PermanentPassesFilter(Ref_TargetPermanent,IsUntapped),[GainLife(Integer(1))])`,
		`Unless(PermanentPassesFilter(HostPermanent,IsUntapped),[GainLife(Integer(1))])`,
		`IfElse(PermanentPassesFilter(ThisPermanent,IsTapped),[GainLife(Integer(1))],[])`,
		`If(PermanentPassesFilter(ThisPermanent,IsTapped),[])`,
		`If(PermanentPassesFilter(ThisPermanent,IsTapped),[PermanentDealsDamage(Ref_TargetPermanent,Integer(2),Player(You))])`,
		`Unless(PermanentPassesFilter(ThisPermanent,IsAttacking),[GainLife(Integer(1))])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid delayed conditional accepted", body)
		}
	}
}
