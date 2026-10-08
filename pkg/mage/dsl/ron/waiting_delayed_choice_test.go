package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedChoiceAbility(t *testing.T) {
	for _, state := range []string{"live", "phased", "departed", "returned", "copy", "clone"} {
		t.Run(state, func(t *testing.T) {
			a := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("A"), choice: 1}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([ChooseAnAbility([Flying,Trample]),CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbilityVariable(TheChosenAbility)],UntilEndOfTurn)]))]))`)
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			if len(g.WaitingDelayedPermanentSources()) != 0 || a.prompts != 0 {
				t.Fatal("early choice or delay")
			}
			g.ResolveStack()
			switch state {
			case "phased":
				g.PhaseOut(p.ID())
			case "departed", "returned":
				g.BouncePermanentToHand(p)
				if state == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				}
			}
			got := g.WaitingDelayedPermanentSources()
			if len(got) != 1 || !got[0].SameObject(saved) || a.prompts != 0 {
				t.Fatal("choice lost waiting source")
			}
			game := g
			if state == "clone" {
				game = g.Clone()
			}
			game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			game.PutTriggersOnStack()
			game.ResolveStack()
			if len(game.WaitingDelayedPermanentSources()) != 0 {
				t.Fatal("fired choice still waiting")
			}
			current := game.FindPermanent(p.ID())
			want := state == "live" || state == "copy" || state == "clone"
			keyword := core.Trample
			if state == "clone" {
				keyword = core.Flying
			}
			if current != nil && current.HasKeyword(keyword) != want {
				t.Fatal("choice followed absent or returned subject")
			}
			if state == "clone" && len(g.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("clone shares waiting lifetime")
			}
		})
	}
}

func TestRONWaitingDelayedChoiceScopedCounter(t *testing.T) {
	choice := `ChooseAnAction([[PutCounters([ACounterOfTypeOnPermanent(PTCounter(0,1),Ref_TargetPermanent)])],[PutCounters([ACounterOfTypeOnPermanent(PTCounter(1,0),Ref_TargetPermanent)])]])`
	for _, action := range []string{`PlayerAction(ControllerOfPermanent(ThisPermanent),` + choice + `)`, `PlayerActions(ControllerOfPermanent(ThisPermanent),[` + choice + `])`} {
		a := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("A"), choice: 0}
		b := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("B"), choice: 1}
		g := mage.NewGame(a, b)
		factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+action+`]))])))`)
		p := g.PutOnBattlefield(factory(), a.PlayerID())
		q := g.PutOnBattlefield(mage.NewCreature("Target", "", 2, 4), a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
		if a.prompts != 0 || b.prompts != 0 {
			t.Fatal("early scoped choice")
		}
		g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		g.PutTriggersOnStack()
		g.ResolveStack()
		if a.prompts != 0 || b.prompts != 1 || g.FindPermanent(q.ID()).CurrentPower(g) != 3 || g.FindPermanent(q.ID()).CurrentToughness(g) != 4 {
			t.Fatal("choice lost lexical player")
		}
	}
}

func TestRONWaitingDelayedChoiceInvalid(t *testing.T) {
	for _, action := range []string{`PlayerAction(ActionPlayer,ChooseAnAction([]))`, `PlayerActions(You,[ChooseAnAction([[PutCounters([ACounterOfTypeOnPermanent(PTCounter(0,1),Ref_TargetPermanent)])],[PutCounters([ACounterOfTypeOnPermanent(PTCounter(1,0),Ref_TargetPermanent)])]])])`, `ChooseAnAbility([Flying,Trample]),GainLife(Integer(1))`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+action+`]))]))])`), "invalid-choice.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid delayed choice accepted")
		}
	}
}

func TestRONWaitingDelayedChoiceProtection(t *testing.T) {
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
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([ChooseAnAbility([Flying,Trample]),CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbilityVariable(TheChosenAbility)],UntilEndOfTurn)]))]))`)
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
			binding := mage.BindDamageSource("waiting choice")
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
				t.Fatal("waiting choice body omitted departed source")
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
				t.Fatal("choice remained waiting")
			}
			if _, err := Compile(record); err != nil {
				t.Fatal("complete source-choice record rejected", err)
			}
		})
	}
}

func TestRONWaitingDelayedChoiceScopedAbility(t *testing.T) {
	a := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("A"), choice: 0}
	b := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("B"), choice: 1}
	g := mage.NewGame(a, b)
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PlayerActions(ControllerOfPermanent(ThisPermanent),[FlipACoin_OnLose([ChooseAnAbility([Flying,Trample]),CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbilityVariable(TheChosenAbility)],UntilEndOfTurn)])])]))]))`)
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.SetCoinFlipResults([]bool{false})
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.prompts != 0 || b.prompts != 1 || !g.FindPermanent(p.ID()).HasKeyword(core.Trample) || g.FindPermanent(p.ID()).HasKeyword(core.Flying) {
		t.Fatal("ability choice lost lexical player through coin branch")
	}
}
