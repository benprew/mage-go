package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedManaPaymentProtection(t *testing.T) {
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
		for _, branch := range []string{"paid", "unpaid"} {
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
					chooser := &ronDelayedPaymentPlayer{ronSourceChoicePlayer: &ronSourceChoicePlayer{BasePlayer: a}, accept: branch == "paid"}
					g := mage.NewGame(chooser, b)
					damage := `PermanentDealsDamage(ThisPermanent,Integer(2),Player(You))`
					body := `MayCost(PayManaX([ManaCostX,ManaCostGeneric(1)],Integer(1))),If(CostWasPaid,[` + damage + `])`
					if branch == "unpaid" {
						body = `MayCost(PayManaX([ManaCostX,ManaCostGeneric(1)],Integer(1))),Unless(CostWasPaid,[` + damage + `])`
					}
					factory := waitingCombatFactory(t, `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))]))`)
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
					a.ManaPool().Clear()
					funding := 2
					if state == "copy" {
						funding = 4
					}
					a.ManaPool().Add(core.White, funding)
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
					if chooser.offers != 0 {
						t.Fatal("query paid a waiting cost")
					}
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
					wantMana := funding
					if branch == "paid" {
						wantMana = 0
					}
					if a.ManaPool().TotalMana() != wantMana {
						t.Fatal("dynamic payment spent the wrong amount")
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

func TestRONWaitingDelayedManaPaymentInvalid(t *testing.T) {
	for _, body := range []string{
		`MayCost(PayManaX([ManaCostX],Integer(1)))`,
		`MayCost(PayManaX([],Integer(1))),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(PayManaX([ManaCostX],Integer(-1))),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(PayManaX([ManaCostX],ValueX)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(PayManaX([ManaCostX],ManaValueOfSpell(Trigger_ThatSpell))),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(PayManaX([ManaCostX],Integer(1))),If(CostWasPaid,[])`,
		`MayCost(PayManaX([ManaCostX],Integer(1))),If(CostWasPaid(),[GainLife(Integer(1))])`,
		`PlayerMayCost(Trigger_ThatPlayer,PayManaX([ManaCostX],Integer(1))),Unless(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(PayManaX([ManaCostX],Integer(1))),IfElse(CostWasPaid,[GainLife(Integer(1))],[GainLife(Integer(2))])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid delayed mana payment accepted", body)
		}
	}
}

func TestRONWaitingDelayedManaPaymentInstances(t *testing.T) {
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: false}
	g := mage.NewGame(a, b)
	a.ManaPool().Add(core.Colorless, 3)
	b.ManaPool().Add(core.Colorless, 3)
	factory := waitingCombatFactory(t, `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([MayCost(PayManaX([ManaCostX],Integer(1))),If(CostWasPaid,[MayCost(PayManaX([ManaCostX],Integer(1))),If(CostWasPaid,[PermanentDealsDamage(ThisPermanent,Integer(2),Player(You))])]),MayCost(PayManaX([ManaCostX],Integer(1))),Unless(CostWasPaid,[PermanentDealsDamage(ThisPermanent,Integer(3),Player(You))])]))])))`)
	q := g.PutOnBattlefield(mage.NewCreature("Target", "", 1, 10), b.PlayerID())
	cards := []mage.Card{factory(), factory()}
	if cards[0].Abilities()[0].(*mage.ActionDefinition).Targets()[0] == cards[1].Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("factory shared target declaration")
	}
	saved := make([]mage.DamageSource, 0, 2)
	for i, player := range []*ronPaymentPlayer{a, b} {
		cards[i].SetOwner(player.PlayerID())
		p := g.PutOnBattlefield(cards[i], player.PlayerID())
		saved = append(saved, g.CaptureDamageSource(p.ID()))
		if err := g.ActivateAbilityByIndex(player.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	got := g.WaitingDelayedPermanentSources()
	if len(got) != 3 || a.offers != 0 || b.offers != 0 {
		t.Fatal("nested payment body lost distinct waiting sources", len(got))
	}
	for _, original := range saved {
		found := false
		for _, ref := range got {
			found = found || ref.SameObject(original)
		}
		if !found {
			t.Fatal("lost per-instance source")
		}
	}
	clone := g.Clone()
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 17 || a.offers != 3 || b.offers != 2 || a.ManaPool().TotalMana() != 0 || b.ManaPool().TotalMana() != 3 {
		t.Fatal("nested or adjacent payments shared a result or controller", a.Life(), b.Life(), a.offers, b.offers)
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 3 {
		t.Fatal("clone shared waiting lifetime")
	}
}
