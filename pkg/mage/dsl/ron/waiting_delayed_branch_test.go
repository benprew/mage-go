package ron

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedBranchProtection(t *testing.T) {
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
		for _, branch := range []string{"win", "loss"} {
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
					body := `FlipACoin_OnWinAndLose([PermanentDealsDamage(ThisPermanent,Integer(2),Player(You))],[GainLife(Integer(0))])`
					if branch == "loss" {
						body = `FlipACoin_OnLose([PermanentDealsDamage(ThisPermanent,Integer(2),Player(You))])`
					}
					g.SetCoinFlipResults([]bool{branch == "win", branch == "win"})
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

func TestRONWaitingDelayedBranchScopedCoin(t *testing.T) {
	for _, action := range []string{
		`PlayerActions(ControllerOfPermanent(ThisPermanent),[FlipACoin_OnWinAndLose([GainLife(Integer(2))],[LoseLife(Integer(3))])])`,
		`PlayerAction(ControllerOfPermanent(ThisPermanent),FlipACoin_OnWinAndLose([GainLife(Integer(2))],[LoseLife(Integer(3))]))`,
	} {
		for _, win := range []bool{false, true} {
			g, a, b := ronGame()
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+action+`]))]))`)
			p := g.PutOnBattlefield(factory(), a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if got := g.WaitingDelayedPermanentSources(); len(got) != 1 || !got[0].SameObject(g.CaptureDamageSource(p.ID())) {
				t.Fatal("scoped coin lost its player producer reference")
			}
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			g.SetCoinFlipResults([]bool{win, !win})
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.ResolveStack()
				want := 17
				if win {
					want = 22
				}
				if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != want {
					t.Fatal("coin branches lost resolution player scope")
				}
				if game.FlipCoin(a.PlayerID()) != !win {
					t.Fatal("scoped branch did not consume exactly one coin")
				}
			}
		}
	}
}

func TestRONWaitingDelayedBranchInvalid(t *testing.T) {
	for _, body := range []string{
		`PlayerActions(ActionPlayer,[FlipACoin_OnLose([GainLife(Integer(1))])])`,
		`PlayerActions(You,[FlipACoin_OnLose([])])`,
		`PlayerActions(You,[FlipACoin_OnWinAndLose([GainLife(Integer(1))],[])])`,
		`PlayerActions(You,[FlipACoin_OnLose([PermanentDealsDamage(Ref_TargetPermanent,Integer(1),Player(You))])])`,
		`FlipACoin_OnLose([GainLife(TheNumberOfCardsInPlayersHand(ActionPlayer))])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))]))])`), "invalid-branch.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid scoped coin accepted", body)
		}
	}
}

func TestRONWaitingDelayedBranchInstances(t *testing.T) {
	g, a, b := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PlayerActions(ControllerOfPermanent(ThisPermanent),[FlipACoin_OnLose([GainLife(Integer(2))])])]))]))`)
	first, second := factory(), factory()
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(first, a.PlayerID())
	q := g.PutOnBattlefield(second, b.PlayerID())
	saved := g.CaptureDamageSource(p.ID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), q.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.BouncePermanentToHand(p)
	a.RemoveFromHand(first.ID())
	g.PutOnBattlefield(first, b.PlayerID())
	got := g.WaitingDelayedPermanentSources()
	if len(got) != 2 || (!got[0].SameObject(saved) && !got[1].SameObject(saved)) {
		t.Fatal("instances lost their player producer incarnations")
	}
	g.SetCoinFlipResults([]bool{false, false, false})
	clone := g.Clone()
	for _, game := range []*mage.Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 24 || game.GetPlayer(b.PlayerID()).Life() != 22 {
			t.Fatal("instances or copied delay followed returned source controller")
		}
	}
}
