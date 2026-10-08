package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedRecipientProtection(t *testing.T) {
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
		for _, kind := range []string{"single", "packets"} {
			for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "copy", "clone"} {
				t.Run(tc.name+"/"+kind+"/"+state, func(t *testing.T) {
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
					body := `PermanentDealsDamage(Ref_TargetPermanent,Integer(2),Permanent(ThisPermanent))`
					if kind == "packets" {
						body = `PermanentDealsMultipleDamage(ThisPermanent,[DamageToRecipients(Integer(1),Permanent(ThisPermanent)),DamageToRecipients(Integer(1),Permanent(ThisPermanent)),DamageToRecipients(Integer(2),Player(You))])`
					}
					nodes, err := Parse(strings.NewReader(`Card(Name:"Recipient",Typeline:(Supertypes:[],Cardtypes:[Artifact,Creature],Subtypes:[]),CardPT:(Power:1,Toughness:10),Rules:[Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))])))])`), "recipient.ron")
					if err != nil {
						t.Fatal(err)
					}
					factory, err := Compile(nodes[0])
					if err != nil {
						t.Fatal(err)
					}
					sourceCard := factory()
					sourceCard.SetOwner(a.PlayerID())
					source := g.PutOnBattlefield(sourceCard, a.PlayerID())
					g.ChangeObjectColor(source.ID(), tc.color)
					attacker := g.PutOnBattlefield(mage.NewCreature("Other source", "", 1, 10), b.PlayerID())
					original := g.CaptureDamageSource(source.ID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{attacker.ID()}); err != nil {
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
					consumer := mage.NewArtifact("Consumer", "", mage.WithAbility(mage.Activate(mage.Costs(payment...), mage.Sequence(plan.waitingDelayedPermanentChoice(bindings), shield(bindings)))))
					if err := mage.ValidateCardDefinition(consumer.Name(), consumer); err != nil {
						t.Fatal(err)
					}
					p := g.PutOnBattlefield(consumer, a.PlayerID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil || chooser.calls != 0 || g.StackSize() != 0 {
						t.Fatal("unpaid choice changed state")
					}
					a.ManaPool().Add(core.White, 2)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if chooser.calls != 0 {
						t.Fatal("choice ran at announcement")
					}
					var clone *mage.Game
					if state == "clone" {
						clone = g.Clone()
						clone.ResolveStack()
						if chooser.calls != 0 {
							t.Fatal("clone changed chooser")
						}
					}
					g.ResolveStack()
					if chooser.calls != 1 || !chooser.source.SameObject(original) {
						t.Fatal("recipient-only delayed reference lost source choice")
					}
					if kind == "single" {
						g.DealDamageToPlayerFromSource(chooser, 2, original)
						if a.Life() != 20 {
							t.Fatal("recipient source shield failed")
						}
					}
					g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					if len(g.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("fired ability remained waiting")
					}
					g.PutTriggersOnStack()
					g.ResolveStack()
					wantDamage := 2
					if state == "copy" {
						wantDamage = 4
					}
					if state == "returned" || state == "phased" {
						wantDamage = 0
					}
					if p := g.FindPermanentIncludingPhased(source.ID()); p != nil && p.Damage != wantDamage {
						t.Fatalf("recipient damage = %d, want %d", p.Damage, wantDamage)
					}
					wantLife := 20
					if kind == "packets" && state == "copy" {
						wantLife = 18
					}
					if a.Life() != wantLife || b.Life() != 20 {
						t.Fatal("packet prevention or saved controller failed")
					}
					g.DealDamageToPlayerFromSource(chooser, 1, original)
					if a.Life() != wantLife-1 {
						t.Fatal("shield was not consumed")
					}
					if clone != nil {
						if kind == "single" {
							clone.DealDamageToPlayerFromSource(clone.GetPlayer(a.PlayerID()), 2, original)
						}
						if len(clone.WaitingDelayedPermanentSources()) == 0 {
							t.Fatal("clone shared waiting lifetime")
						}
						clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
						clone.PutTriggersOnStack()
						clone.ResolveStack()
						if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.FindPermanent(source.ID()).Damage != 2 {
							t.Fatal("clone lost recipient or shield")
						}
					}
					if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "ChooseADamageSource") {
						t.Fatal("incomplete general aggregation accepted", err)
					}
				})
			}
		}
	}
}

func TestRONWaitingDelayedRecipientInvalid(t *testing.T) {
	for _, body := range []string{
		`PermanentDealsDamage(ThisPermanent,Integer(1),Permanent(ThisPermanent()))`,
		`PermanentDealsDamage(ThisPermanent,Integer(1),Permanent(Ref_TargetPermanent))`,
		`PermanentDealsDamage(ThisPermanent,Integer(1),Permanent(HostPermanent))`,
		`PermanentDealsMultipleDamage(ThisPermanent,[DamageToRecipients(Integer(1),Permanent(Trigger_ThatPermanent))])`,
		`PermanentDealsMultipleDamage(ThisPermanent,[DamageToRecipients(Integer(1),Permanent())])`,
		`PermanentDealsMultipleDamage(ThisPermanent,[])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid recipient accepted", body)
		}
	}
}
