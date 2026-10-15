package ron

import (
	"maps"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCounterPlacementHistoryTomb(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			record := chosenDiscardRecord(t, set, "Cyclopean Tomb")
			record.Fields = maps.Clone(record.Fields)
			rules := record.Fields["Rules"]
			trigger := rules.Args[1]
			creation := trigger.Args[1].Args[0].Args[0]
			filter := creation.Args[1].Args[0].Args[0].Args[0].Args[0].Args[1]
			if filter.Args[0].Args[1].Name != "PermanentPutACounterOfTypeOnIt" {
				t.Fatal("missing real history filter")
			}
			filter.Args = append([]Node(nil), filter.Args...)
			filter.Args[0].Args = append([]Node(nil), filter.Args[0].Args[:2]...)
			probe, err := Parse(strings.NewReader(`Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(WhenAPermanentIsPutIntoAPlayersGraveyard(SinglePermanent(ThisPermanent),AnyPlayer),ActionList([CreateTrigger(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([TapEachPermanent(IsCardtype(Land))]))]))])`), "probe")
			if err != nil {
				t.Fatal(err)
			}
			projected := probe[0].Fields["Rules"].Args[0]
			projected.Args[1].Args[0].Args[0].Args[1].Args[0].Args[0].Args[0] = filter
			rules.Args = []Node{rules.Args[0], projected}
			record.Fields["Rules"] = rules
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			card := factory()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			land := g.PutOnBattlefield(mage.NewLand("Forest", mage.WithSubTypes("Forest")), b.PlayerID())
			other := g.PutOnBattlefield(mage.NewLand("Other", mage.WithSubTypes("Forest")), b.PlayerID())
			other.AddCounter(core.Mire, 1)
			g.SetStep(core.Upkeep)
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{land.ID()}); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			g.ResolveStack()
			g.DestroyPermanent(g.FindPermanent(source.ID()))
			g.ResolveStack()
			returned := g.PutOnBattlefield(card, b.PlayerID())
			g.RunStepWithPriority(core.Cleanup)
			g.SetActivePlayerIndex(0)
			g.RunStepWithPriority(core.Upkeep)
			if !g.FindPermanent(land.ID()).Tapped || g.FindPermanent(other.ID()).Tapped {
				t.Fatal("wrong history recipients")
			}
			if g.FindPermanent(returned.ID()).Tapped {
				t.Fatal("history used returned source")
			}
			if clone.FindPermanent(land.ID()).Counters[core.Mire] != 0 {
				t.Fatal("pending clone shared placement")
			}
			clone.ResolveStack()
			clone.DestroyPermanent(clone.FindPermanent(source.ID()))
			clone.ResolveStack()
			clone.RunStepWithPriority(core.Cleanup)
			clone.SetActivePlayerIndex(0)
			clone.RunStepWithPriority(core.Upkeep)
			if !clone.FindPermanent(land.ID()).Tapped || clone.FindPermanent(other.ID()).Tapped {
				t.Fatal("pending clone lost history")
			}
		})
	}
}

func TestRONCounterPlacementHistoryInvalid(t *testing.T) {
	for _, filter := range []string{`PermanentPutACounterOfTypeOnIt`, `PermanentPutACounterOfTypeOnIt()`, `PermanentPutACounterOfTypeOnIt(ThisPermanent)`, `PermanentPutACounterOfTypeOnIt(ThisPermanent(),MireCounter)`, `PermanentPutACounterOfTypeOnIt(ThisPermanent,MireCounter())`, `PermanentPutACounterOfTypeOnIt(ThisPermanent,UnknownCounter)`, `PermanentPutACounterOfTypeOnIt(Ref_TargetPermanent,MireCounter)`, `PermanentPutACounterOfTypeOnIt(ThisPermanent,MireCounter,You)`} {
		assertConditionalRejected(t, `Activated(PayMana([]),ActionList([TapEachPermanent(`+filter+`)]))`)
	}
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		if _, err := Compile(chosenDiscardRecord(t, set, "Cyclopean Tomb")); err != nil {
			t.Fatal("complete Tomb rejected", err)
		}
	}
}

func TestRONCounterPlacementHistoryTiming(t *testing.T) {
	for _, mode := range []string{"paid", "unpaid", "illegal", "return-target", "phase-target", "return-source", "copy", "clone"} {
		t.Run(mode, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(2)]),Targeted([TargetPermanent(IsCardtype(Land))],ActionList([PutCounters([ACounterOfTypeOnPermanent(MireCounter,Ref_TargetPermanent)]),TapEachPermanent(PermanentPutACounterOfTypeOnIt(ThisPermanent,MireCounter))])))`)
			g, a, b := ronGame()
			card := factory()
			source := g.PutOnBattlefield(card, a.PlayerID())
			landCard := mage.NewLand("Land")
			land := g.PutOnBattlefield(landCard, b.PlayerID())
			a.ManaPool().Add(core.Colorless, 1)
			if mode != "unpaid" {
				a.ManaPool().Add(core.Colorless, 1)
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{land.ID()})
			if mode == "unpaid" {
				if err == nil || !g.GetStack().IsEmpty() || land.Counters[core.Mire] != 0 || land.Tapped {
					t.Fatal("unpaid activation changed state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if land.Tapped {
				t.Fatal("query ran during announcement")
			}
			clone := g.Clone()
			switch mode {
			case "illegal":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.GrantKeywordModification(mage.This, core.Shroud)), land.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "return-target":
				g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
				g.PutOnBattlefield(landCard, b.PlayerID())
			case "phase-target":
				g.PhaseOut(land.ID())
			case "return-source":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			g.ResolveStack()
			live := g.FindPermanentIncludingPhased(land.ID())
			active := mode != "illegal" && mode != "return-target" && mode != "phase-target"
			if live.Tapped != active {
				t.Fatal("wrong history action timing")
			}
			clone.ResolveStack()
			if !clone.FindPermanent(land.ID()).Tapped {
				t.Fatal("pending clone lost query")
			}
		})
	}
}

func TestRONCounterPlacementHistoryInventory(t *testing.T) {
	src := `Card(Name:"History",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([TapEachPermanent(PermanentPutACounterOfTypeOnIt(ThisPermanent,MireCounter))]))])`
	report, err := InspectFS(fstest.MapFS{"history.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, sig := range report.Signatures {
		if !strings.Contains(sig.Context, "PermanentPutACounterOfTypeOnIt") {
			continue
		}
		want, ok := map[string]string{"ThisPermanent": "permanent_reference", "MireCounter": "permanent_counter_type"}[sig.Name]
		if ok {
			if sig.Role != want {
				t.Fatalf("%s role=%s want=%s", sig.Name, sig.Role, want)
			}
			found[sig.Name] = true
		}
	}
	if len(found) != 2 {
		t.Fatal("missing history operands", found)
	}
}

func TestRONCounterPlacementHistoryContexts(t *testing.T) {
	filter := `PermanentPutACounterOfTypeOnIt(ThisPermanent,MireCounter)`
	for _, rule := range []string{
		`AsPermanentEnters(ThisPermanent,[TapEachPermanent(` + filter + `)])`,
		`Activated(TapEachPermanent(` + filter + `),ActionList([]))`,
		`PermanentLayerEffect(` + filter + `,[AdjustPT(1,1)])`,
	} {
		assertConditionalRejected(t, rule)
	}
	node, err := Parse(strings.NewReader(`Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([TapEachPermanent(`+filter+`)]))])`), "probe")
	if err != nil {
		t.Fatal(err)
	}
	relation := node[0].Fields["Rules"].Args[0].Args[1].Args[0].Args[0].Args[0]
	for _, ctx := range []scope{{resolving: false}, {resolving: true, enteringSource: true}, {resolving: true, drawReplacementBody: true}} {
		if _, err := compileContextualCollection(relation, ctx); err == nil {
			t.Fatal("unsupported context accepted")
		}
	}
}
