package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCounterRemovalHistoryLifecycle(t *testing.T) {
	g, a, b := newSpellCopyGame()
	card := NewArtifact("Source", "")
	source := g.PutOnBattlefield(card, a.PlayerID())
	landCard := NewLand("Land")
	land := g.PutOnBattlefield(landCard, b.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	bound := BindPermanent("land")
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{bound: {land.ID(), land.incarnationID}}
	set := PermanentsWithoutCountersRemovedBy(This, core.Mire, MatchingPermanents(IsLand))
	check := func(game *Game, want int) {
		t.Helper()
		local := *ctx
		local.Game = game
		for range 3 {
			if got := len(set.resolvePermanents(&local)); got != want {
				t.Fatalf("history count=%d want=%d", got, want)
			}
		}
	}
	check(g, 1)
	g.PhaseOut(source.ID())
	check(g, 1)
	g.PhaseIn([]uuid.UUID{source.ID()})
	if err := RemoveAllCounters(From(bound), core.Mire).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	check(g, 1)
	land.AddCounter(core.Mire, 3)
	clone := g.Clone()
	ctx.Targets = []uuid.UUID{land.ID()}
	if err := RemoveCountersFrom(From(bound), core.Mire, Fixed(1)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	check(g, 0)
	check(clone, 1)
	g.RunStepWithPriority(core.Cleanup)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	returned := g.PutOnBattlefield(card, b.PlayerID())
	check(g, 0)
	fresh := *ctx
	fresh.sourceIncarnation = returned.incarnationID
	if len(set.resolvePermanents(&fresh)) != 1 {
		t.Fatal("returned source inherited history")
	}
	g.PhaseOut(land.ID())
	check(g, 0)
	g.PhaseIn([]uuid.UUID{land.ID()})
	check(g, 0)
	g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
	g.PutOnBattlefield(landCard, b.PlayerID())
	check(g, 1)
	if len(PermanentsWithoutCountersRemovedBy(This, core.Charge, MatchingPermanents(IsLand)).resolvePermanents(ctx)) != 1 {
		t.Fatal("counter kinds mixed")
	}
}

func TestCounterRemovalHistoryDeclarations(t *testing.T) {
	for _, set := range []PermanentSet{
		PermanentsWithoutCountersRemovedBy(nil, core.Mire, MatchingPermanents(AnyPermanent)),
		PermanentsWithoutCountersRemovedBy(This, core.Mire, nil),
		PermanentsWithoutCountersRemovedBy(From(BindPermanent("missing")), core.Mire, MatchingPermanents(AnyPermanent)),
		PermanentsWithoutCountersRemovedBy(This, core.NumCounters, MatchingPermanents(AnyPermanent)),
	} {
		card := NewArtifact("Bad", "", WithAbility(Activate(Costs(), TapEach(set))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid history accepted")
		}
	}
}

func TestCounterRemovalHistoryPayments(t *testing.T) {
	for _, kind := range []string{"fixed", "X", "chosen"} {
		for _, commit := range []bool{false, true} {
			t.Run(kind+map[bool]string{false: "/failed", true: "/paid"}[commit], func(t *testing.T) {
				a := &counterChoicePlayer{BasePlayer: NewBasePlayer("A"), choice: 2}
				g := NewGame(a, NewBasePlayer("B"))
				source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
				source.AddCounter(core.Charge, 4)
				source.AddCounter(core.Mire, 1)
				if err := ApplyEffect(g, RemoveAllCounters(This, core.Mire), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID()}
				set := PermanentsWithoutCountersRemovedBy(This, core.Charge, MatchingPermanents(AnyPermanent))
				cost := RemoveCountersCost(core.Charge, 2)
				switch kind {
				case "X":
					cost = RemoveXCountersFromSourceCost(core.Charge)
				case "chosen":
					cost = RemoveAnyCountersCost(core.Charge, BindNumber("paid"))
				}
				g.resolution.SetX(2)
				for range 3 {
					if !cost.CanPay(source.ID(), a.PlayerID(), g) {
						t.Fatal("cost unavailable")
					}
				}
				tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{cost}}, nil)
				if err != nil {
					t.Fatal(err)
				}
				if len(set.resolvePermanents(ctx)) != 1 || source.Counters[core.Charge] != 4 {
					t.Fatal("preparation published removal")
				}
				clone := g.Clone()
				if !commit {
					g.MutablePermanent(source.ID()).RemoveCounter(core.Charge, 3)
				}
				err = tx.Commit()
				if commit {
					if err != nil || len(set.resolvePermanents(ctx)) != 0 {
						t.Fatal("committed payment not recorded", err)
					}
				} else if err == nil || len(set.resolvePermanents(ctx)) != 1 {
					t.Fatal("failed payment published history", err)
				}
				local := *ctx
				local.Game = clone
				if len(set.resolvePermanents(&local)) != 1 {
					t.Fatal("payment mutated clone")
				}
			})
		}
	}
}

func TestCounterRemovalHistoryWaitingAndSavedSource(t *testing.T) {
	g, a, b := newSpellCopyGame()
	card := NewArtifact("Source", "")
	source := g.PutOnBattlefield(card, a.PlayerID())
	land := g.PutOnBattlefield(NewLand("Land"), b.PlayerID())
	land.AddCounter(core.Mire, 2)
	binding := BindPermanent("source")
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{binding: {source.ID(), source.incarnationID}}
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	set := PermanentsWithoutCountersRemovedBy(From(binding), core.Mire, MatchingPermanents(IsLand))
	choice := BindPermanent("removal")
	body := Sequence(ChoosePermanent(You, set, choice), RemoveAllCounters(From(choice), core.Mire))
	if err := AtEach(YourUpkeep, body).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(source)
	g.PutOnBattlefield(card, b.PlayerID())
	g.RunStepWithPriority(core.Cleanup)
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || g.FindPermanent(land.ID()).Counters[core.Mire] != 2 || len(g.zones.counterRemovals) != 0 {
			t.Fatal("waiting query changed state or lost source")
		}
	}
	g.SetActivePlayerIndex(0)
	g.RunStepWithPriority(core.Upkeep)
	if g.FindPermanent(land.ID()).Counters[core.Mire] != 0 {
		t.Fatal("saved source body failed")
	}
	if len(set.resolvePermanents(ctx)) != 0 {
		t.Fatal("saved source query lost history")
	}
}
