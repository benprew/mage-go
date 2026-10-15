package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const sourceLeaveUntap = `Or([WhenPermanentLeavesTheBattlefield(ThisPermanent),WhenPermanentBecomesUntapped(ThisPermanent)])`

func TestRONDelayedLeaveUntap(t *testing.T) {
	for _, event := range []string{sourceLeaveUntap, `Or([WhenPermanentBecomesUntapped(ThisPermanent),WhenPermanentLeavesTheBattlefield(ThisPermanent)])`, `WhenPermanentLeavesTheBattlefield(ThisPermanent)`, `WhenPermanentBecomesUntapped(ThisPermanent)`} {
		for _, change := range []string{"normal", "before-resolution", "phase-cycle", "control", "copy"} {
			t.Run(event+"/"+change, func(t *testing.T) {
				factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(`+event+`,ActionList([GainLife(Integer(2))]))]))`)
				g, a, b := ronGame()
				card := factory()
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				g.TapPermanent(p)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				leave := event == `WhenPermanentLeavesTheBattlefield(ThisPermanent)`
				if change == "before-resolution" {
					if leave {
						g.ExilePermanent(g.FindPermanent(p.ID()))
					} else {
						g.UntapPermanent(g.FindPermanent(p.ID()))
					}
				}
				if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				g.ResolveStack()
				if change == "before-resolution" && leave {
					if len(g.DelayedTriggers()) != 0 {
						t.Fatal("departed source subscribed")
					}
					return
				}
				want := 1
				if change == "copy" {
					want = 2
				}
				if len(g.DelayedTriggers()) != want {
					t.Fatal("subscription count")
				}
				if change == "phase-cycle" {
					g.PhaseIn(g.PhaseOut(p.ID()))
				}
				if change == "control" {
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: card.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				if leave {
					g.ExilePermanent(g.FindPermanent(p.ID()))
				} else {
					if change == "before-resolution" {
						g.TapPermanent(g.FindPermanent(p.ID()))
					}
					g.UntapPermanent(g.FindPermanent(p.ID()))
				}
				g.PutTriggersOnStack()
				if g.StackSize() != want {
					t.Fatalf("stack=%d want=%d", g.StackSize(), want)
				}
				clone := g.Clone()
				g.ResolveStack()
				clone.ResolveStack()
				if a.Life() != 20+2*want || b.Life() != 20 || clone.GetPlayer(a.PlayerID()).Life() != a.Life() {
					t.Fatal("captured controller or clone")
				}
			})
		}
	}
}

func TestRONDelayedLeaveUntapCardClauses(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"ATQ", "Tawnos's Coffin"}, {"LEA", "Animate Dead"}} {
		record := repositoryRecord(t, tc.set, tc.name, "source.ron")
		var event Node
		var find func(Node)
		find = func(n Node) {
			if n.Name == "CreateFutureTrigger" {
				event = n.Args[0]
			}
			for _, child := range n.Args {
				find(child)
			}
		}
		find(record.Fields["Rules"])
		if event.Name == "" {
			t.Fatal("missing event")
		}
		probe := nameScopeNode(t, `Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(`+sourceLeaveUntap+`,ActionList([GainLife(Integer(1))]))]))])`)
		rules := probe.Fields["Rules"]
		rules.Args[0].Args[1].Args[0].Args[0].Args[0] = event
		probe.Fields["Rules"] = rules
		factory, err := Compile(probe)
		if err != nil {
			t.Fatal(err)
		}
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(factory(), a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		g.ExilePermanent(g.FindPermanent(p.ID()))
		g.ResolveStack()
		if a.Life() != 21 {
			t.Fatal("unchanged card event did not execute")
		}
	}
}

func TestRONDelayedLeaveUntapInvalid(t *testing.T) {
	for _, event := range []string{
		`Or([])`, `Or([WhenPermanentLeavesTheBattlefield(ThisPermanent)])`,
		`Or([WhenPermanentLeavesTheBattlefield(ThisPermanent),WhenPermanentLeavesTheBattlefield(ThisPermanent)])`,
		`Or([WhenPermanentLeavesTheBattlefield(ThisPermanent),WhenPermanentBecomesUntapped(HostPermanent)])`,
		`WhenPermanentLeavesTheBattlefield`, `WhenPermanentBecomesUntapped(ThisPermanent,You)`,
		`WhenPermanentLeavesTheBattlefield(You)`, `WhenPermanentLeavesTheBattlefield(Ref_TargetPermanent)`,
	} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(`+event+`,ActionList([GainLife(Integer(1))]))]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatalf("accepted %s", event)
		}
	}
	for _, body := range []string{`ActionList([])`, `ActionList([PlayerAction(ControllerOfLeavingPermanent,GainLife(Integer(1)))])`, `Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([TapPermanent(Ref_TargetPermanent)]))`} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(`+sourceLeaveUntap+`,`+body+`)]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([CreateFutureTrigger(`+sourceLeaveUntap+`,ActionList([GainLife(Integer(1))]))]))])`)
	if _, err := Compile(n); err == nil {
		t.Fatal("spell supplied permanent source")
	}
}

func TestRONDelayedLeaveUntapSavedFrames(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([Exile([Permanent(Ref_TargetPermanent),EachPermanent(IsCardtype(Enchantment))],[]),NoteCountersOnPermanent(Ref_TargetPermanent),CreateFutureTrigger(`+sourceLeaveUntap+`,ActionList([PutEachExiledCardOntoTheBattlefield(TheOtherPermanentsExiledThisWay,[EntersUnderOwnersControl,EntersWithNotedCounters])]))])))`)
	g, a, b := ronGame()
	source := factory()
	source.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(source, a.PlayerID())
	target := mage.NewCreature("Target", "", 2, 4)
	target.SetOwner(a.PlayerID())
	q := g.PutOnBattlefield(target, a.PlayerID())
	q.AddCounter(core.Charge, 7)
	other := mage.NewEnchantment("Other", "")
	other.SetOwner(b.PlayerID())
	g.PutOnBattlefield(other, b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(g.DelayedTriggers()) != 1 {
		t.Fatal("frame did not subscribe")
	}
	for range 2 {
		if len(g.WaitingDelayedPermanentSources()) == 0 {
			t.Fatal("waiting frame lost source")
		}
		if g.FindPermanent(other.ID()) != nil {
			t.Fatal("query executed entry")
		}
	}
	clone := g.Clone()
	for _, game := range []*mage.Game{g, clone} {
		game.ExilePermanent(game.FindPermanent(p.ID()))
		game.PutTriggersOnStack()
		if game.StackSize() != 1 {
			t.Fatal("departure lost saved frame")
		}
		game.ResolveStack()
		entered := game.FindPermanent(other.ID())
		if entered == nil || entered.ControllerID() != b.PlayerID() || entered.Counters[core.Charge] != 7 {
			t.Fatal("delayed union lost exile, player or counter bindings")
		}
		if game.FindPermanent(target.ID()) != nil {
			t.Fatal("plural return changed selected exile group")
		}
	}
}

func TestRONDelayedLeaveUntapContexts(t *testing.T) {
	action := `CreateFutureTrigger(` + sourceLeaveUntap + `,ActionList([GainLife(Integer(1))]))`
	for _, rule := range []string{
		`TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([` + action + `]))`,
		`Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + action + `]))]))`,
	} {
		factory := permanentScopeFactory(t, "Artifact", rule)
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(factory(), a.PlayerID())
		if strings.HasPrefix(rule, "Activated") {
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			for range 2 {
				if len(g.WaitingDelayedPermanentSources()) != 1 || len(g.DelayedTriggers()) != 1 {
					t.Fatal("waiting nested installer lost source or ran during query")
				}
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		}
		g.ResolveStack()
		if len(g.DelayedTriggers()) != 1 {
			t.Fatal("resolving context did not install source event")
		}
		g.ExilePermanent(g.FindPermanent(p.ID()))
		g.ResolveStack()
		if a.Life() != 21 {
			t.Fatal("resolving context lost delayed body")
		}
	}
	for _, rule := range []string{
		`AsPermanentEnters(ThisPermanent,[` + action + `])`,
		`ReplaceWouldDraw(PlayerWouldDraw(You),[` + action + `])`,
	} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`)
		if _, err := Compile(n); err == nil {
			t.Fatal("replacement accepted source subscription")
		}
	}
}

func TestRONDelayedLeaveUntapInventory(t *testing.T) {
	n := nameScopeNode(t, `CreateFutureTrigger(`+sourceLeaveUntap+`,ActionList([GainLife(Integer(1))]))`)
	if got := inventoryArgumentRole(n, "action", 0); got != "delayed_source_event" {
		t.Fatalf("event role=%s", got)
	}
	union := n.Args[0]
	if got := inventoryArgumentRole(union, "delayed_source_event", 0); got != "delayed_source_event" {
		t.Fatalf("union role=%s", got)
	}
	leaf := union.Args[0].Args[0]
	if got := inventoryArgumentRole(leaf, "delayed_source_event", 0); got != "permanent_reference" {
		t.Fatalf("subject role=%s", got)
	}
}

func TestRONDelayedLeaveUntapContextRejection(t *testing.T) {
	n := nameScopeNode(t, `CreateFutureTrigger(`+sourceLeaveUntap+`,ActionList([GainLife(Integer(1))]))`)
	for _, ctx := range []scope{
		{}, {permanent: true}, {permanent: true, resolving: true, drawReplacementBody: true},
		{permanent: true, resolving: true, sourceZone: &sourceZonePlan{}},
	} {
		if _, err := compileDelayedStep(n, ctx); err == nil {
			t.Fatal("invalid source context accepted")
		}
	}
}

func TestRONDelayedLeaveUntapInstances(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(`+sourceLeaveUntap+`,ActionList([DestroyPermanent(Ref_TargetPermanent)]))])))`)
	g, a, b := ronGame()
	sources := make([]uuid.UUID, 0, 2)
	targets := make([]uuid.UUID, 0, 2)
	for _, player := range []mage.Player{a, b} {
		card := factory()
		card.SetOwner(player.PlayerID())
		source := g.PutOnBattlefield(card, player.PlayerID())
		target := g.PutOnBattlefield(mage.NewCreature("Target", "", 2, 4), player.PlayerID())
		sources = append(sources, source.ID())
		targets = append(targets, target.ID())
		g.TapPermanent(source)
		if err := g.ActivateAbilityByIndex(player.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	clone := g.Clone()
	g.UntapPermanent(g.FindPermanent(sources[0]))
	g.ResolveStack()
	if g.FindPermanent(targets[0]) != nil || g.FindPermanent(targets[1]) == nil || len(g.DelayedTriggers()) != 1 {
		t.Fatal("factory or pending frames shared bindings")
	}
	clone.ExilePermanent(clone.FindPermanent(sources[1]))
	clone.ResolveStack()
	if clone.FindPermanent(targets[1]) != nil || clone.FindPermanent(targets[0]) == nil || len(clone.DelayedTriggers()) != 1 {
		t.Fatal("clone or source subscriptions shared identity")
	}
}
