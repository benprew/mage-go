package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCounterRemovalHistoryTomb(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			factory, err := Compile(chosenDiscardRecord(t, set, "Cyclopean Tomb"))
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			card := factory()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			lands := make([]*mage.Permanent, 0, 3)
			for _, name := range []string{"First", "Second", "Unrelated"} {
				lands = append(lands, g.PutOnBattlefield(mage.NewLand(name, mage.WithSubTypes("Forest")), b.PlayerID()))
			}
			g.SetStep(core.Upkeep)
			for _, land := range lands[:2] {
				g.MutablePermanent(source.ID()).Tapped = false
				a.ManaPool().Add(core.Colorless, 2)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{land.ID()}); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
			}
			g.AddCountersWithReplacement(g.FindPermanent(lands[0].ID()), core.Mire, 2, lands[2].ID(), false)
			g.AddCountersWithReplacement(g.FindPermanent(lands[2].ID()), core.Mire, 4, lands[2].ID(), false)
			g.DestroyPermanent(g.FindPermanent(source.ID()))
			g.ResolveStack()
			g.PutOnBattlefield(card, b.PlayerID())
			g.RunStepWithPriority(core.Cleanup)
			clone := g.Clone()
			g.SetActivePlayerIndex(0)
			g.RunStepWithPriority(core.Upkeep)
			var cleared, other *mage.Permanent
			for _, land := range lands[:2] {
				live := g.FindPermanent(land.ID())
				if live.Counters[core.Mire] == 0 {
					cleared = live
				} else {
					other = live
				}
			}
			if cleared == nil || other == nil || !cleared.HasSubType("Forest") {
				t.Fatal("must remove from exactly one land")
			}
			g.MutablePermanent(cleared.ID()).AddCounter(core.Mire, 7)
			g.RunStepWithPriority(core.Cleanup)
			g.RunStepWithPriority(core.Upkeep)
			if g.FindPermanent(cleared.ID()).Counters[core.Mire] != 7 || g.FindPermanent(other.ID()).Counters[core.Mire] != 0 || !g.FindPermanent(other.ID()).HasSubType("Forest") {
				t.Fatal("history did not select the next land")
			}
			if g.FindPermanent(lands[2].ID()).Counters[core.Mire] != 4 {
				t.Fatal("unrelated land changed")
			}
			g.RunStepWithPriority(core.Cleanup)
			g.RunStepWithPriority(core.Upkeep)
			total := 0
			for _, land := range lands[:2] {
				total += g.FindPermanent(land.ID()).Counters[core.Mire]
			}
			if total != 7 {
				t.Fatal("previously cleared land selected again", total)
			}
			for _, land := range lands[:2] {
				if clone.FindPermanent(land.ID()).Counters[core.Mire] == 0 {
					t.Fatal("history shared with clone")
				}
			}
			clone.SetActivePlayerIndex(0)
			clone.RunStepWithPriority(core.Upkeep)
			remaining := 0
			for _, land := range lands[:2] {
				if clone.FindPermanent(land.ID()).Counters[core.Mire] > 0 {
					remaining++
				}
			}
			if remaining != 1 {
				t.Fatal("clone lost saved source")
			}
		})
	}
}

const chosenMireRemoval = `RemoveCounters([AllCountersOfTypeFromAPermanent(MireCounter,IsCardtype(Land))])`

func TestRONCounterRemovalHistoryInvalid(t *testing.T) {
	for _, instruction := range []string{`AllCountersOfTypeFromAPermanent`, `AllCountersOfTypeFromAPermanent()`, `AllCountersOfTypeFromAPermanent(MireCounter)`, `AllCountersOfTypeFromAPermanent(MireCounter(),IsCardtype(Land))`, `AllCountersOfTypeFromAPermanent(UnknownCounter,IsCardtype(Land))`, `AllCountersOfTypeFromAPermanent(MireCounter,ThisPermanent)`, `AllCountersOfTypeFromAPermanent(MireCounter,IsCardtype(Land),You)`} {
		assertConditionalRejected(t, `Activated(PayMana([]),ActionList([RemoveCounters([`+instruction+`])]))`)
	}
	for _, filter := range []string{`PermanentHasNotRemovedACounterOfTypeFromIt`, `PermanentHasNotRemovedACounterOfTypeFromIt()`, `PermanentHasNotRemovedACounterOfTypeFromIt(ThisPermanent)`, `PermanentHasNotRemovedACounterOfTypeFromIt(ThisPermanent(),MireCounter)`, `PermanentHasNotRemovedACounterOfTypeFromIt(ThisPermanent,MireCounter())`, `PermanentHasNotRemovedACounterOfTypeFromIt(ThisPermanent,PoisonCounter)`, `PermanentHasNotRemovedACounterOfTypeFromIt(Ref_TargetPermanent,MireCounter)`, `PermanentHasNotRemovedACounterOfTypeFromIt(ThisPermanent,MireCounter,You)`} {
		assertConditionalRejected(t, `Activated(PayMana([]),ActionList([TapEachPermanent(`+filter+`)]))`)
	}
	for _, rule := range []string{`AsPermanentEnters(ThisPermanent,[` + chosenMireRemoval + `])`, `Activated(` + chosenMireRemoval + `,ActionList([]))`, `PermanentLayerEffect(PermanentHasNotRemovedACounterOfTypeFromIt(ThisPermanent,MireCounter),[AdjustPT(1,1)])`} {
		assertConditionalRejected(t, rule)
	}
	action := nameScopeNode(t, chosenMireRemoval)
	filter := nameScopeNode(t, `PermanentHasNotRemovedACounterOfTypeFromIt(ThisPermanent,MireCounter)`)
	for _, ctx := range []scope{{}, {resolving: true, enteringSource: true}, {resolving: true, drawReplacementBody: true}} {
		if _, err := compileChosenCounterRemoval(action.Args[0].Args[0], ctx); err == nil {
			t.Fatal("invalid removal context accepted")
		}
		if _, err := compileContextualCollection(filter, ctx); err == nil {
			t.Fatal("invalid history context accepted")
		}
	}
	if got := inventoryArgumentRole(action.Args[0].Args[0], "counter_removal_instruction", 1); got != "permanent_collection" {
		t.Fatal(got)
	}
	if got := inventoryArgumentRole(filter, "permanent_collection", 0); got != "permanent_reference" {
		t.Fatal(got)
	}
	if got := inventoryArgumentRole(filter, "permanent_collection", 1); got != "permanent_counter_type" {
		t.Fatal(got)
	}
}

func TestRONCounterRemovalHistoryResolution(t *testing.T) {
	for _, mode := range []string{"none", "nil", "invalid", "phase", "return", "copy", "clone", "empty"} {
		t.Run(mode, func(t *testing.T) {
			g, base, b := ronGame()
			a := &ronPermanentChoicePlayer{Player: base}
			g.SetPlayerAt(0, a)
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([RemoveCounters([AllCountersOfTypeFromAPermanent(MireCounter,And([IsCardtype(Land),PermanentHasNotRemovedACounterOfTypeFromIt(ThisPermanent,MireCounter)]))])]))`)
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			firstCard := mage.NewLand("First")
			first := g.PutOnBattlefield(firstCard, b.PlayerID())
			second := g.PutOnBattlefield(mage.NewLand("Second"), b.PlayerID())
			first.AddCounter(core.Mire, 3)
			second.AddCounter(core.Mire, 2)
			a.choose = func(candidates []*mage.Permanent) *mage.Permanent {
				switch mode {
				case "nil":
					return nil
				case "invalid":
					return source
				case "phase":
					g.PhaseOut(first.ID())
				case "return":
					g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
					g.PutOnBattlefield(firstCard, b.PlayerID()).AddCounter(core.Mire, 6)
				}
				return candidates[0]
			}
			if mode == "empty" {
				g.RemoveFromBattlefield(first)
				g.RemoveFromBattlefield(second)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if a.calls != 0 || (mode != "empty" && g.FindPermanent(first.ID()).Counters[core.Mire] != 3) {
				t.Fatal("removal or choice ran at announcement")
			}
			clone := g.Clone()
			if mode == "copy" {
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			g.ResolveStack()
			if mode == "empty" {
				return
			}
			live := g.FindPermanentIncludingPhased(first.ID())
			want := 0
			switch mode {
			case "phase":
				want = 3
			case "return":
				want = 6
			}
			secondWant := 2
			if mode == "copy" {
				secondWant = 0
			}
			if live.Counters[core.Mire] != want || g.FindPermanent(second.ID()).Counters[core.Mire] != secondWant {
				t.Fatal("wrong singular removal", live.Counters[core.Mire])
			}
			if clone.FindPermanent(first.ID()).Counters[core.Mire] != 3 {
				t.Fatal("clone shared removal")
			}
			if mode == "clone" {
				clone.ResolveStack()
				if clone.FindPermanent(first.ID()).Counters[core.Mire] != 0 {
					t.Fatal("pending clone lost choice")
				}
			}
		})
	}
}

func TestRONCounterRemovalHistoryPlayerScope(t *testing.T) {
	g, base, opponent := ronGame()
	a := &ronPermanentChoicePlayer{Player: base}
	b := &ronPermanentChoicePlayer{Player: opponent}
	g.SetPlayerAt(0, a)
	g.SetPlayerAt(1, b)
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerActions(Ref_TargetPlayer,[`+chosenMireRemoval+`])])))`)
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	first := g.PutOnBattlefield(mage.NewLand("First", mage.WithKeyword(core.Shroud)), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewLand("Second", mage.WithKeyword(core.Shroud)), a.PlayerID())
	first.AddCounter(core.Mire, 1)
	second.AddCounter(core.Mire, 1)
	a.choose = func(candidates []*mage.Permanent) *mage.Permanent { return candidates[0] }
	b.choose = func(candidates []*mage.Permanent) *mage.Permanent { return candidates[len(candidates)-1] }
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.calls != 0 || b.calls != 1 || g.FindPermanent(first.ID()).Counters[core.Mire] != 1 || g.FindPermanent(second.ID()).Counters[core.Mire] != 0 {
		t.Fatal("wrong scoped chooser or shroud blocked a choice")
	}
}

func TestRONCounterRemovalHistoryTombLegality(t *testing.T) {
	for _, mode := range []string{"main", "opponent-upkeep", "swamp", "nonland", "unpaid", "phased"} {
		t.Run(mode, func(t *testing.T) {
			factory, err := Compile(chosenDiscardRecord(t, "LEA", "Cyclopean Tomb"))
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			g.SetStep(core.Upkeep)
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			land := mage.NewLand("Land", mage.WithSubTypes("Forest"))
			if mode == "swamp" {
				land = mage.NewLand("Swamp", mage.WithSubTypes("Swamp"))
			}
			if mode == "nonland" {
				land = mage.NewArtifact("Artifact", "")
			}
			target := g.PutOnBattlefield(land, b.PlayerID())
			switch mode {
			case "main":
				g.SetStep(core.PrecombatMain)
			case "opponent-upkeep":
				g.SetActivePlayerIndex(1)
			case "phased":
				g.PhaseOut(target.ID())
			}
			if mode != "unpaid" {
				a.ManaPool().Add(core.Colorless, 2)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err == nil {
				t.Fatal("illegal full-record activation accepted")
			}
			if g.FindPermanent(source.ID()).Tapped || g.FindPermanentIncludingPhased(target.ID()).Counters[core.Mire] != 0 || !g.GetStack().IsEmpty() {
				t.Fatal("failed activation changed state")
			}
		})
	}
}
