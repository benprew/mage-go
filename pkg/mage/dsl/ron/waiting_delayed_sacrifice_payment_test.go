package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedSacrificePaymentProtection(t *testing.T) {
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
			for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "control", "copy", "clone", "replaced", "stale"} {
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
					chooser := &ronWaitingSacrificePlayer{ronSourceChoicePlayer: &ronSourceChoicePlayer{BasePlayer: a}, accept: branch != "unpaid"}
					g := mage.NewGame(chooser, b)
					condition := "CostWasPaid"
					if branch == "unpaid" {
						condition = "CostWasntPaid"
					}
					body := `MayCost(SacrificePermanent(ThisPermanent)),If(` + condition + `,[GainLife(Integer(1))])`
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
					if state == "control" {
						g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					}
					if state == "replaced" {
						g.AddExileIfWouldGoToGraveyardThisTurn(source.ID(), uuid.New())
					}
					if state == "stale" {
						chooser.before = func() {
							g.BouncePermanentToHand(g.FindPermanent(sourceCard.ID()))
							a.RemoveFromHand(sourceCard.ID())
							g.PutOnBattlefield(sourceCard, a.PlayerID())
						}
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
					a.ManaPool().Add(core.White, 2)
					if state == "clone" {
						clone := g.Clone()
						clone.ResolveStack()
						clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
						clone.PutTriggersOnStack()
						clone.ResolveStack()
						cloneLife := 21
						if branch == "paid" {
							cloneLife = 20
						}
						if clone.GetPlayer(a.PlayerID()).Life() != cloneLife || chooser.calls != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
							t.Fatal("clone lost source or shared lifetime")
						}
					}
					g.ResolveStack()
					if chooser.offers != 0 {
						t.Fatal("query paid a waiting cost")
					}
					if chooser.calls != 1 || !chooser.source.SameObject(original) {
						t.Fatal("sacrifice payment lost source choice")
					}
					g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					if len(g.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("fired ability remained waiting")
					}
					g.PutTriggersOnStack()
					g.ResolveStack()
					paid := branch == "paid" && state != "departed" && state != "returned" && state != "phased" && state != "control" && state != "stale"
					want := 20
					if branch == "unpaid" || paid {
						want++
						if state == "copy" && branch == "unpaid" {
							want++
						}
					}
					if a.Life() != want || b.Life() != 20 {
						t.Fatalf("sacrifice branch life=%d want=%d", a.Life(), want)
					}
					if paid && g.FindPermanentIncludingPhased(sourceCard.ID()) != nil {
						t.Fatal("paid subject remains on battlefield")
					}
					if state == "replaced" && paid && len(g.GetExile()) != 1 {
						t.Fatal("sacrifice replacement lost")
					}
					if state == "stale" && branch == "paid" && g.FindPermanent(sourceCard.ID()) == nil {
						t.Fatal("payment consumed returned object")
					}
					g.DealDamageToPlayerFromSource(chooser, 1, original)
					if a.Life() != want {
						t.Fatal("shield did not prevent damage")
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

type ronWaitingSacrificePlayer struct {
	*ronSourceChoicePlayer
	accept bool
	offers int
	before func()
}

func (p *ronWaitingSacrificePlayer) ChooseMayAbility(string) bool {
	p.offers++
	if p.before != nil {
		p.before()
	}
	return p.accept
}

func TestRONWaitingDelayedSacrificePaymentInvalid(t *testing.T) {
	for _, body := range []string{
		`MayCost(SacrificePermanent(ThisPermanent())),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(SacrificePermanent(Ref_TargetPermanent)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(SacrificePermanent(Trigger_ThatPermanent)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(SacrificePermanent(Player(You))),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(SacrificePermanent(ThisPermanent,ThisPermanent)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(SacrificePermanent(ThisPermanent)),If(CostWasPaid,[])`,
		`MayCost(SacrificePermanent(ThisPermanent)),If(CostWasPaid(),[GainLife(Integer(1))])`,
		`MayCost(SacrificePermanent(ThisPermanent)),GainLife(Integer(1))`,
		`PlayerMayCost(UnknownPlayer,SacrificePermanent(ThisPermanent)),If(CostWasPaid,[GainLife(Integer(1))])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", body)
		}
	}
}

func TestRONWaitingDelayedSacrificePaymentInstances(t *testing.T) {
	for _, continuation := range []string{"If(CostWasPaid", "Unless(CostWasntPaid", "If(CostWasntPaid", "Unless(CostWasPaid"} {
		t.Run(continuation, func(t *testing.T) {
			a := &ronWaitingSacrificePlayer{ronSourceChoicePlayer: &ronSourceChoicePlayer{BasePlayer: mage.NewBasePlayer("A")}, accept: true}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			factory := waitingCombatFactory(t, `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PlayerMayCost(You,SacrificePermanent(ThisPermanent)),`+continuation+`,[GainLife(Integer(1))])]))]))`)
			cards := []mage.Card{factory(), factory()}
			for _, card := range cards {
				g.PutOnBattlefield(card, a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
			}
			if len(g.WaitingDelayedPermanentSources()) != 2 {
				t.Fatal("factory instances lost distinct references")
			}
			clone := g.Clone()
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			want := 20
			paidBranch := continuation == "If(CostWasPaid" || continuation == "Unless(CostWasntPaid"
			if paidBranch {
				want = 22
			}
			if a.Life() != want || a.offers != 2 || len(a.Graveyard()) != 2 {
				t.Fatal("independent sacrifice payments chose wrong branch")
			}
			for _, card := range cards {
				if g.FindPermanent(card.ID()) != nil || clone.FindPermanent(card.ID()) == nil {
					t.Fatal("sacrifice shared clone state")
				}
			}
			if len(g.WaitingDelayedPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 2 {
				t.Fatal("clone shared delayed lifetime")
			}
			clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			clone.PutTriggersOnStack()
			clone.ResolveStack()
			cloneLife := 22
			if paidBranch {
				cloneLife = 20
			}
			if clone.GetPlayer(a.PlayerID()).Life() != cloneLife || len(clone.WaitingDelayedPermanentSources()) != 0 || a.Life() != want {
				t.Fatal("clone payment changed original result")
			}
		})
	}
}
