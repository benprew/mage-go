package ron

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedMultiDamageProtection(t *testing.T) {
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
		for _, shape := range []string{"mixed", "players", "packets"} {
			for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "copy", "clone"} {
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
					_, a, b := ronGame()
					chooser := &ronSourceChoicePlayer{BasePlayer: a}
					g := mage.NewGame(chooser, b)
					body := `PermanentDealsDamage(ThisPermanent,Integer(2),MultipleRecipients([EachPlayer(AnyPlayer),EachPermanent(IsCardtype(Creature))]))`
					if shape == "players" {
						body = `PermanentDealsDamage(ThisPermanent,Integer(2),EachPlayer(AnyPlayer))`
					}
					if shape == "packets" {
						body = `PermanentDealsMultipleDamage(ThisPermanent,[DamageToRecipients(Integer(1),Player(You)),DamageToRecipients(Integer(1),Player(You))])`
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
					victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 10), b.PlayerID())
					phased := g.PutOnBattlefield(mage.NewCreature("Phased", "", 1, 10), b.PlayerID())
					g.PhaseOut(phased.ID())
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
					callbacks := 0
					g.SetOnDamageDealt(func(_, _ string, _ int, _ bool) {
						callbacks++
						wantB := 18
						wantDamage := 0
						if shape == "packets" {
							wantB = 20
						}
						if shape == "mixed" {
							wantDamage = 2
						}
						if state == "copy" && shape != "packets" && b.Life() == 16 {
							wantB = 16
							wantDamage *= 2
						}
						if b.Life() != wantB || g.FindPermanent(victim.ID()).Damage != wantDamage {
							t.Error("notification saw an incomplete simultaneous batch")
						}
					})
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
					wantB, wantDamage, wantCallbacks := 18, 0, 1
					if shape == "packets" {
						wantB, wantCallbacks = 20, 0
					}
					if shape == "mixed" {
						wantDamage, wantCallbacks = 2, 2
					}
					if state == "copy" {
						if shape != "packets" {
							wantB = 16
						}
						wantDamage *= 2
						wantCallbacks = wantCallbacks*2 + 1
					}
					if a.Life() != want || b.Life() != wantB || g.FindPermanent(victim.ID()).Damage != wantDamage || g.FindPermanentIncludingPhased(phased.ID()).Damage != 0 || callbacks != wantCallbacks {
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

func TestRONWaitingDelayedMultiDamageInvalid(t *testing.T) {
	for _, text := range []string{
		`PermanentDealsMultipleDamage(ThisPermanent(),[DamageToRecipients(Integer(1),Player(You))])`,
		`PermanentDealsMultipleDamage(Ref_TargetPermanent,[DamageToRecipients(Integer(1),Player(You))])`,
		`PermanentDealsMultipleDamage(ThisPermanent,[])`,
		`PermanentDealsMultipleDamage(ThisPermanent,[DamageToRecipients(Integer(1),Player(Ref_TargetPlayer))])`,
		`PermanentDealsDamage(ThisPermanent,Integer(2),EachPlayer(You))`,
		`PermanentDealsDamage(ThisPermanent,Integer(2),MultipleRecipients([EachPlayer(AnyPlayer)]))`,
		`PermanentDealsDamage(ThisPermanent,Integer(2),EachPermanent(Ref_TargetPermanent))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+text+`]))]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", text)
		}
	}
	for _, tc := range []struct {
		text      string
		permanent bool
		ctx       scope
	}{
		{"ThisPermanent()", true, scope{permanent: true}},
		{"ThisPermanent", true, scope{}},
		{"ThisSpell", false, scope{permanent: true}},
		{"Ref_TargetPermanent", true, scope{permanent: true}},
	} {
		nodes, err := Parse(strings.NewReader(tc.text), "invalid-source.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compileSelfDamageSource(nodes[0], tc.ctx, tc.permanent); err == nil {
			t.Fatal("invalid source accepted", tc.text)
		}
	}
}

func TestRONWaitingDelayedMultiDamageInstances(t *testing.T) {
	g, a, b := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPermanentBecomesTapped(SinglePermanent(ThisPermanent)),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PermanentDealsMultipleDamage(Trigger_ThatPermanent,[DamageToRecipients(Integer(1),Player(You)),DamageToRecipients(Integer(1),Player(You))])]))]))`)
	firstCard, secondCard := factory(), factory()
	firstCard.SetOwner(a.PlayerID())
	secondCard.SetOwner(b.PlayerID())
	first := g.PutOnBattlefield(firstCard, a.PlayerID())
	second := g.PutOnBattlefield(secondCard, b.PlayerID())
	originalFirst := g.CaptureDamageSource(first.ID())
	originalSecond := g.CaptureDamageSource(second.ID())
	g.TapPermanent(first)
	g.TapPermanent(second)
	g.PutTriggersOnStack()
	g.ResolveStack()
	g.BouncePermanentToHand(first)
	a.RemoveFromHand(firstCard.ID())
	g.PutOnBattlefield(firstCard, b.PlayerID())
	got := g.WaitingDelayedPermanentSources()
	if len(got) != 2 || !got[0].SameObject(originalFirst) && !got[1].SameObject(originalFirst) || !got[0].SameObject(originalSecond) && !got[1].SameObject(originalSecond) {
		t.Fatal("factory instances lost separate saved source incarnations")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 18 || len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("instances shared a controller or delayed lifetime")
	}
}
