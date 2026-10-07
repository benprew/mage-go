package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestNextUntapsCountAndOverlap(t *testing.T) {
	for _, counts := range [][]int{{2}, {2, 1}, {1, 2}, {2, 2}, {0}, {-1}} {
		g, a, _, p := controllerTestGame(t)
		ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID()}
		for _, n := range counts {
			if err := SkipNextUntaps(This, n).Apply(ctx); err != nil {
				t.Fatal(err)
			}
		}
		clone := g.Clone()
		want := 0
		for _, n := range counts {
			want = max(want, n)
		}
		for step := range 3 {
			g.TapPermanent(g.FindPermanent(p.ID()))
			g.RunStepWithPriority(Untap)
			if g.FindPermanent(p.ID()).Tapped != (step < want) {
				t.Fatalf("counts=%v step=%d", counts, step)
			}
		}
		if clone.turns.HasSkipNextUntap(p.ID()) != (want > 0) {
			t.Fatal("clone count changed")
		}
		for step := range 3 {
			clone.TapPermanent(clone.FindPermanent(p.ID()))
			clone.RunStepWithPriority(Untap)
			if clone.FindPermanent(p.ID()).Tapped != (step < want) {
				t.Fatal("clone lost remaining count")
			}
		}
	}
}

func TestNextUntapsIdentityAndTiming(t *testing.T) {
	for _, change := range []string{"control", "return", "phase", "other-untap", "already-untapped", "skip-step", "captured-overlap"} {
		t.Run(change, func(t *testing.T) {
			g, a, b, p := controllerTestGame(t)
			p.Card.SetOwner(a.PlayerID())
			ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID()}
			if err := SkipNextUntaps(This, 2).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			g.TapPermanent(g.FindPermanent(p.ID()))
			switch change {
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
				g.RunStepWithPriority(Untap)
				if !g.turns.HasSkipNextUntap(p.ID()) {
					t.Fatal("inactive controller consumed count")
				}
				g.SetActivePlayerIndex(1)
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				p = g.PutOnBattlefield(p.Card, a.PlayerID())
				g.TapPermanent(p)
			case "phase":
				g.PhaseOut(p.ID())
			case "other-untap", "already-untapped":
				g.UntapPermanent(g.FindPermanent(p.ID()))
				if g.FindPermanent(p.ID()).Tapped {
					t.Fatal("restriction blocked other untap")
				}
				if change == "other-untap" {
					g.TapPermanent(g.FindPermanent(p.ID()))
				}
			case "skip-step":
				g.turns.BuildNextTurn()
				g.SkipNextOccurrenceOfStep(Untap)
				step, ok := g.turns.PopNextStep()
				if !ok || step != Upkeep {
					t.Fatal("untap step was not skipped")
				}
				g.doUpkeepActions()
				if !g.turns.HasSkipNextUntap(p.ID()) {
					t.Fatal("skipped step consumed count")
				}
			case "captured-overlap":
				if err := SkipNextUntapOf(This, You).Apply(ctx); err != nil {
					t.Fatal(err)
				}
			}
			g.RunStepWithPriority(Untap)
			if g.FindPermanent(p.ID()).Tapped != (change != "return" && change != "already-untapped") {
				t.Fatal("first step")
			}
			g.TapPermanent(g.FindPermanent(p.ID()))
			g.RunStepWithPriority(Untap)
			if g.FindPermanent(p.ID()).Tapped != (change != "return") {
				t.Fatal("second step")
			}
			g.RunStepWithPriority(Untap)
			if g.FindPermanent(p.ID()).Tapped {
				t.Fatal("restriction did not expire")
			}
		})
	}
}

func TestNextUntapsCollectionAndDeclarations(t *testing.T) {
	g, a, b, p := controllerTestGame(t)
	q := g.PutOnBattlefield(NewCreature("Other", "", 1, 1), b.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID()}
	if err := SkipNextUntapEach(MatchingPermanents(IsCreature)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	late := g.PutOnBattlefield(NewCreature("Late", "", 1, 1), a.PlayerID())
	for _, id := range []PermanentRef{This} {
		if err := SkipNextUntaps(id, 0).Apply(ctx); err != nil {
			t.Fatal(err)
		}
	}
	g.TapPermanent(g.FindPermanent(p.ID()))
	g.TapPermanent(q)
	g.TapPermanent(late)
	g.RunStepWithPriority(Untap)
	if !g.FindPermanent(p.ID()).Tapped || g.FindPermanent(late.ID()).Tapped || !g.FindPermanent(q.ID()).Tapped {
		t.Fatal("collection capture")
	}
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(Untap)
	if !g.FindPermanent(q.ID()).Tapped {
		t.Fatal("other player's restriction")
	}
	g.RunStepWithPriority(Untap)
	if g.FindPermanent(q.ID()).Tapped {
		t.Fatal("collection expiry")
	}
	for _, effect := range []Effect{SkipNextUntaps(nil, 2), SkipNextUntaps(This, -1), SkipNextUntapEach(nil), SkipNextUntaps(From(BindPermanent("unbound")), 2), SkipNextUntaps(DeclarePermanentTarget("unbound", IsCreature), 2)} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), effect)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestNextUntapsSavedInformationCannotRestoreObject(t *testing.T) {
	for _, change := range []string{"live", "phase", "phase-cycle", "absent", "return", "control"} {
		t.Run(change, func(t *testing.T) {
			g, a, b, p := controllerTestGame(t)
			saved := BindPermanent("subject")
			ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID()}
			if err := Snapshot(This, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "absent", "return":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				if change == "return" {
					g.PutOnBattlefield(p.Card, a.PlayerID())
				}
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			}
			if err := SkipNextUntaps(From(saved), 2).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			want := change == "live" || change == "phase-cycle" || change == "control"
			if g.turns.HasSkipNextUntap(p.ID()) != want {
				t.Fatal("saved identity or presence")
			}
		})
	}
}
