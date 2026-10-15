package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCounterPlacementHistoryLifecycle(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Source", "")
	source := g.PutOnBattlefield(card, a.PlayerID())
	recipientCard := NewLand("Land")
	recipient := g.PutOnBattlefield(recipientCard, b.PlayerID())
	other := g.PutOnBattlefield(NewArtifact("Other", ""), b.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	set := PermanentsGivenCountersBy(This, core.Mire, MatchingPermanents(AnyPermanent))
	count := func(game *Game, want int) {
		t.Helper()
		local := *ctx
		local.Game = game
		for range 3 {
			if got := len(set.resolvePermanents(&local)); got != want {
				t.Fatalf("history count=%d want=%d", got, want)
			}
		}
	}
	g.AddCountersWithReplacement(recipient, core.Mire, 0, source.ID(), false)
	g.AddCountersWithReplacement(recipient, core.Mire, 1, other.ID(), false)
	g.AddCountersWithReplacement(recipient, core.Charge, 1, source.ID(), false)
	count(g, 0)
	g.AddCountersWithReplacement(recipient, core.Mire, 1, source.ID(), false)
	count(g, 1)
	clone := g.Clone()
	g.MutablePermanent(recipient.ID()).RemoveCounter(core.Mire, 2)
	count(g, 1)
	g.PhaseOut(recipient.ID())
	count(g, 0)
	count(clone, 1)
	g.PhaseIn([]uuid.UUID{recipient.ID()})
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	returnedSource := g.PutOnBattlefield(card, b.PlayerID())
	g.RunStepWithPriority(core.Cleanup)
	count(g, 1)
	fresh := *ctx
	fresh.sourceIncarnation = returnedSource.incarnationID
	if len(set.resolvePermanents(&fresh)) != 0 {
		t.Fatal("returned source inherited history")
	}
	g.RemoveFromBattlefield(g.FindPermanent(recipient.ID()))
	g.PutOnBattlefield(recipientCard, b.PlayerID())
	count(g, 0)
	if err := AddCountersEach(MatchingPermanents(IsLand), core.Mire, Fixed(1)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	count(g, 1)
	count(clone, 1)
	if len(set.resolvePermanents(&fresh)) != 0 {
		t.Fatal("old resolving source used new visit")
	}
}

func TestCounterPlacementHistoryDeclarations(t *testing.T) {
	for _, set := range []PermanentSet{
		PermanentsGivenCountersBy(nil, core.Mire, MatchingPermanents(AnyPermanent)),
		PermanentsGivenCountersBy(This, core.Mire, nil),
		PermanentsGivenCountersBy(From(BindPermanent("missing")), core.Mire, MatchingPermanents(AnyPermanent)),
		PermanentsGivenCountersBy(This, core.CounterType(-1), MatchingPermanents(AnyPermanent)),
		PermanentsGivenCountersBy(This, core.NumCounters, MatchingPermanents(AnyPermanent)),
	} {
		card := NewArtifact("Bad", "", WithAbility(Activate(Costs(), TapEach(set))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid history declaration accepted")
		}
	}
}

func TestCounterPlacementHistoryReplacements(t *testing.T) {
	for _, mode := range []string{"double", "prevent", "zero", "redirect", "kind", "return-source", "return-target", "phase-target", "cap"} {
		for _, batch := range []bool{false, true} {
			t.Run(mode+map[bool]string{false: "/single", true: "/batch"}[batch], func(t *testing.T) {
				g, a, b := newSpellCopyGame()
				card := NewArtifact("Source", "")
				source := g.PutOnBattlefield(card, a.PlayerID())
				targetCard := NewArtifact("Target", "")
				target := g.PutOnBattlefield(targetCard, b.PlayerID())
				redirect := g.PutOnBattlefield(NewArtifact("Redirect", ""), b.PlayerID())
				ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
				if mode == "cap" {
					target = g.PutOnBattlefield(NewArtifact("Capped", "", WithStaticAbility(SourceCounterCap(core.Mire, 0))), b.PlayerID())
				}
				g.AddReplacementEffect(&testPrioritizedReplacement{
					replacementBase: replacementBase{duration: core.Indefinite},
					matchFunc:       func(action Action) bool { _, ok := action.(*AddCountersAction); return ok },
					replaceFunc: func(action Action, game *Game) Action {
						add := action.(*AddCountersAction)
						switch mode {
						case "double":
							return add.WithAmount(2)
						case "prevent":
							return nil
						case "zero":
							return add.WithAmount(0)
						case "redirect":
							return NewAddCountersAction(add.ActionSource(), redirect.ID(), core.Mire, 1, false)
						case "kind":
							return NewAddCountersAction(add.ActionSource(), target.ID(), core.Charge, 1, false)
						case "return-source":
							game.RemoveFromBattlefield(game.FindPermanent(source.ID()))
							game.PutOnBattlefield(card, a.PlayerID())
						case "return-target":
							game.RemoveFromBattlefield(game.FindPermanent(target.ID()))
							game.PutOnBattlefield(targetCard, b.PlayerID())
						case "phase-target":
							game.PhaseOut(target.ID())
						}
						return action
					},
				})
				bound := BindPermanent("target")
				g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{bound: {target.ID(), target.incarnationID}}
				var effect = AddCountersTo(From(bound), core.Mire, Fixed(1))
				if batch {
					effect = AddCountersEach(SingletonPermanent(From(bound)), core.Mire, Fixed(1))
				}
				if err := effect.Apply(ctx); err != nil {
					t.Fatal(err)
				}
				set := PermanentsGivenCountersBy(This, core.Mire, MatchingPermanents(AnyPermanent))
				got := set.resolvePermanents(ctx)
				want := mode == "double" || mode == "redirect" || mode == "return-source"
				if (len(got) == 1) != want {
					t.Fatalf("wrong recorded placement: %v", got)
				}
				if mode == "redirect" && got[0].ID() != redirect.ID() {
					t.Fatal("original target recorded")
				}
				if mode == "kind" && len(PermanentsGivenCountersBy(This, core.Charge, MatchingPermanents(AnyPermanent)).resolvePermanents(ctx)) != 1 {
					t.Fatal("final counter kind missing")
				}
			})
		}
	}
}

func TestCounterPlacementHistoryWaiting(t *testing.T) {
	g, a, b := newSpellCopyGame()
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	target := g.PutOnBattlefield(NewLand("Target"), b.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	set := PermanentsGivenCountersBy(This, core.Mire, MatchingPermanents(IsLand))
	body := TapEach(set)
	if err := AtEach(YourUpkeep, body).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(source)
	g.RunStepWithPriority(core.Cleanup)
	for range 3 {
		got := g.WaitingDelayedPermanentSources()
		if len(got) != 1 || got[0].Incarnation() != ctx.sourceIncarnation {
			t.Fatal("waiting history lost source")
		}
	}
	if g.FindPermanent(target.ID()).Tapped || len(g.zones.counterPlacements) != 0 {
		t.Fatal("waiting query changed state")
	}
}

func TestCounterPlacementHistoryEntryAndTransaction(t *testing.T) {
	a := &counterPlacementPayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	source := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(EntersWithNCounters(core.Mire, 2))), a.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	set := PermanentsGivenCountersBy(This, core.Mire, MatchingPermanents(AnyPermanent))
	if got := set.resolvePermanents(ctx); len(got) != 1 || got[0].ID() != source.ID() {
		t.Fatal("entry history missing")
	}
	recipient := g.PutOnBattlefield(NewCreature("Recipient", "", 2, 8), a.PlayerID())
	clone := g.Clone()
	cost := PutCounterOnMatchingCost(core.Mire, IsCreature)
	for range 3 {
		if !cost.CanPay(source.ID(), a.PlayerID(), g) {
			t.Fatal("cost unavailable")
		}
	}
	if len(g.zones.counterPlacements) != 1 {
		t.Fatal("availability changed history")
	}
	tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{cost}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.zones.counterPlacements) != 1 {
		t.Fatal("preparation changed history")
	}
	g.PhaseOut(recipient.ID())
	if err := tx.Commit(); err == nil {
		t.Fatal("stale payment accepted")
	}
	if len(g.zones.counterPlacements) != 1 {
		t.Fatal("failed commit changed history")
	}
	g.PhaseIn([]uuid.UUID{recipient.ID()})
	tx, err = g.prepareCostPaymentTransaction(costPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{cost}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if len(set.resolvePermanents(ctx)) != 2 {
		t.Fatal("payment placement not recorded")
	}
	local := *ctx
	local.Game = clone
	if len(set.resolvePermanents(&local)) != 1 {
		t.Fatal("transaction history shared with clone")
	}
	binding := BindPermanent("source")
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{binding: {source.ID(), source.incarnationID}}
	if len(PermanentsGivenCountersBy(From(binding), core.Mire, MatchingPermanents(IsCreature)).resolvePermanents(ctx)) != 1 {
		t.Fatal("saved source or candidate filter failed")
	}
}

func TestCounterPlacementHistoryBatch(t *testing.T) {
	g, a, b := newSpellCopyGame()
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	first := g.PutOnBattlefield(NewLand("First"), a.PlayerID())
	second := g.PutOnBattlefield(NewLand("Second"), b.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	set := PermanentsGivenCountersBy(This, core.Mire, MatchingPermanents(IsLand))
	calls := 0
	g.AddReplacementEffect(&testPrioritizedReplacement{
		replacementBase: replacementBase{duration: core.Indefinite},
		matchFunc:       func(action Action) bool { _, ok := action.(*AddCountersAction); return ok },
		replaceFunc: func(action Action, game *Game) Action {
			calls++
			local := *ctx
			local.Game = game
			if len(set.resolvePermanents(&local)) != 0 || game.FindPermanent(first.ID()).Counters[core.Mire] != 0 || game.FindPermanent(second.ID()).Counters[core.Mire] != 0 {
				t.Fatal("batch exposed an earlier placement during preparation")
			}
			return action
		},
	})
	g.AddCountersTogether([]*Permanent{first, second, first}, core.Mire, 1, source.ID())
	if calls != 2 || len(set.resolvePermanents(ctx)) != 2 {
		t.Fatal("batch history missing or duplicate placement")
	}
	g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: first.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.PhaseOut(source.ID())
	if len(set.resolvePermanents(ctx)) != 2 {
		t.Fatal("control or source phasing changed history")
	}
}
