package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCounterSnapshotEntry(t *testing.T) {
	record := repositoryRecord(t, "ATQ", "Tawnos's Coffin", "coffin.ron")
	if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "unsupported delayed step event (Or)") {
		t.Fatalf("unexpected remaining Coffin gap: %v", err)
	}
	var note, counters Node
	var find func(Node)
	find = func(n Node) {
		if n.Name == "NoteCountersOnPermanent" {
			note = n
		}
		if n.Name == "EntersWithNotedCounters" {
			counters = n
		}
		for _, child := range n.Args {
			find(child)
		}
	}
	find(record.Fields["Rules"])
	if note.Name == "" || counters.Name == "" {
		t.Fatal("missing Coffin clauses")
	}
	probe := nameScopeNode(t, `Card(Name:"Counter probe",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([Exile([Permanent(ThisPermanent),EachPermanent(IsCardtype(Creature))],[]),NoteCountersOnPermanent(Ref_TargetPermanent),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PutEachExiledCardOntoTheBattlefield(TheOtherPermanentsExiledThisWay,[EntersUnderOwnersControl,EntersWithNotedCounters])]))])))])`)
	var replace func(*Node)
	replace = func(n *Node) {
		if n.Name == note.Name {
			*n = note
			return
		}
		if n.Name == counters.Name {
			*n = counters
			return
		}
		for i := range n.Args {
			replace(&n.Args[i])
		}
	}
	rules := probe.Fields["Rules"]
	replace(&rules)
	probe.Fields["Rules"] = rules
	factory, err := Compile(probe)
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	source := factory()
	source.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(source, a.PlayerID())
	card := mage.NewCreature("Subject", "", 2, 8)
	card.SetOwner(b.PlayerID())
	target := g.PutOnBattlefield(card, a.PlayerID())
	target.AddCounter(core.Charge, 300)
	target.AddCounter(core.P1P1, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	clone := g.Clone()
	for _, game := range []*mage.Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.ResolveStack()
		got := game.FindPermanent(card.ID())
		if got == nil || got.Counters[core.Charge] != 300 || got.Counters[core.P1P1] != 2 || got.ControllerID() != b.PlayerID() {
			t.Fatal("delayed entry lost saved counters or owner")
		}
	}
}

func TestRONCounterSnapshotInvalid(t *testing.T) {
	entry := `PutEachExiledCardOntoTheBattlefield(TheOtherPermanentsExiledThisWay,[EntersUnderOwnersControl,EntersWithNotedCounters])`
	group := `Exile([Permanent(ThisPermanent),EachPermanent(IsCardtype(Creature))],[])`
	for _, body := range []string{
		group + `,` + entry,
		`NoteCountersOnPermanent`, `NoteCountersOnPermanent()`, `NoteCountersOnPermanent(You)`,
		`NoteCountersOnPermanent(Ref_TargetPermanent)`,
		group + `,If(PermanentPassesFilter(ThisPermanent,IsTapped),[NoteCountersOnPermanent(ThisPermanent)]),` + entry,
		group + `,` + entry + `,NoteCountersOnPermanent(ThisPermanent)`,
		group + `,NoteCountersOnPermanent(ThisPermanent),` + `PutEachExiledCardOntoTheBattlefield(TheOtherPermanentsExiledThisWay,[EntersUnderOwnersControl,EntersWithNotedCounters()])`,
	} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONCounterSnapshotInstances(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([Exile([Permanent(ThisPermanent),EachPermanent(And([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You))]))],[]),NoteCountersOnPermanent(Ref_TargetPermanent),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PutEachExiledCardOntoTheBattlefield(TheOtherPermanentsExiledThisWay,[EntersUnderOwnersControl,EntersWithNotedCounters])]))])))`)
	g, a, b := ronGame()
	ids := make([]uuid.UUID, 0, 2)
	for i, player := range []mage.Player{a, b} {
		source := factory()
		source.SetOwner(player.PlayerID())
		p := g.PutOnBattlefield(source, player.PlayerID())
		subject := mage.NewCreature("Subject", "", 2, 8)
		subject.SetOwner(player.PlayerID())
		q := g.PutOnBattlefield(subject, player.PlayerID())
		q.AddCounter(core.Charge, i+3)
		ids = append(ids, q.ID())
		if err := g.ActivateAbilityByIndex(player.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	for _, game := range []*mage.Game{clone, g} {
		game.ResolveStack()
		for i, id := range ids {
			if p := game.FindPermanent(id); p == nil || p.Counters[core.Charge] != i+3 {
				t.Fatal("instances, delayed copy or clone shared counters")
			}
		}
	}
}

func TestRONCounterSnapshotContexts(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`NoteCountersOnPermanent(ThisPermanent)`), "contexts.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{{permanent: true}, {resolving: true, permanent: true, enteringSource: true}, {resolving: true, permanent: true, drawReplacementBody: true}} {
		if _, err := compileCounterSnapshot(nodes[0], nil, ctx); err == nil {
			t.Fatal("invalid snapshot context accepted")
		}
	}
}
