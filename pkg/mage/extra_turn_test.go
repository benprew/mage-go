package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestExtraTurnSchedulingOrderAndResume(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.GrantExtraTurn(a.PlayerID())
	g.GrantExtraTurn(b.PlayerID())
	g.AdvanceToNextTurn()
	if g.ActivePlayerObj().PlayerID() != b.PlayerID() || g.CurrentTurn() != 2 {
		t.Fatal("newest extra turn did not start first")
	}
	clone := g.Clone()
	g.GrantExtraTurn(b.PlayerID())
	for _, want := range []uuid.UUID{b.PlayerID(), a.PlayerID(), b.PlayerID(), a.PlayerID()} {
		g.AdvanceToNextTurn()
		if g.ActivePlayerObj().PlayerID() != want {
			t.Fatalf("wrong nested or resumed turn: want %s", want)
		}
	}
	for _, want := range []uuid.UUID{a.PlayerID(), b.PlayerID(), a.PlayerID()} {
		clone.AdvanceToNextTurn()
		if clone.ActivePlayerObj().PlayerID() != want {
			t.Fatalf("clone lost normal turn or shared extra turns: want %s", want)
		}
	}
}

func TestExtraTurnSchedulingNativeRun(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for range 10 {
		a.AddToLibrary(NewLand("A land"))
		b.AddToLibrary(NewLand("B land"))
	}
	g.GrantExtraTurn(b.PlayerID())
	for turn, want := range []uuid.UUID{b.PlayerID(), b.PlayerID(), a.PlayerID()} {
		g.Run(0, core.Untap, turn+1)
		if g.ActivePlayerObj().PlayerID() != want || g.CurrentTurn() != turn+2 {
			t.Fatalf("native turn %d: got turn %d player %s want %s, winner %s", turn+2, g.CurrentTurn(), g.ActivePlayerObj().PlayerID(), want, g.Winner())
		}
	}
}

func TestExtraTurnTypedPlayerAndDeclarations(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	bound := BindPlayer("recipient")
	g.SetActivePlayerIndex(1)
	if err := Sequence(SnapshotPlayer(ActivePlayer, bound), ExtraTurnFor(FromPlayer(bound))).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if id, ok := g.PopExtraTurn(); !ok || id != b.PlayerID() {
		t.Fatal("extra turn ignored the bound player")
	}
	if err := ExtraTurnFor(FromPlayer(BindPlayer("absent"))).Apply(ctx); err != nil || g.HasExtraTurns() {
		t.Fatal("missing player granted a turn")
	}
	if err := ExtraTurn().Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if id, ok := g.PopExtraTurn(); !ok || id != a.PlayerID() {
		t.Fatal("legacy extra turn lost controller behavior")
	}
	for i, effect := range []Effect{ExtraTurnFor(You), ExtraTurnFor(nil), ExtraTurnFor(FromPlayer(bound)), Sequence(SnapshotPlayer(ActivePlayer, bound), ExtraTurnFor(FromPlayer(bound)))} {
		err := ValidateCardDefinition("Turns", NewInstant("Turns", "", Spell(effect)))
		if (err != nil) != (i == 1 || i == 2) {
			t.Fatalf("declaration %d: %v", i, err)
		}
	}
}

func TestExtraTurnDelayedPlayerSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Source", "")
	card.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(card, b.PlayerID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: p.ID()}
	if err := AtNext(EachEndStep, ExtraTurnFor(ControllerOfWithLKI(This))).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.DestroyPermanent(p)
	g.PutOnBattlefield(p.Card, a.PlayerID())
	if len(g.WaitingDelayedPermanentSources()) != 1 || g.HasExtraTurns() {
		t.Fatal("waiting query executed the action or lost the original source")
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.RunStepWithPriority(core.EndStep)
		if id, ok := game.PopExtraTurn(); !ok || id != b.PlayerID() {
			t.Fatal("delayed turn followed a returned source or lost LKI")
		}
	}
}

func TestExtraTurnSchedulingSkippedTurn(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for range 5 {
		a.AddToLibrary(NewLand("A land"))
		b.AddToLibrary(NewLand("B land"))
	}
	g.GrantExtraTurn(b.PlayerID())
	g.SkipNextTurnFor(b.PlayerID())
	g.Run(0, core.Untap, 2)
	if g.ActivePlayerObj().PlayerID() != b.PlayerID() || g.CurrentTurn() != 3 || len(b.Hand()) != 0 {
		t.Fatal("skipping an extra turn consumed the next normal turn")
	}
	g.Run(0, core.Untap, 3)
	if g.ActivePlayerObj().PlayerID() != a.PlayerID() || len(b.Hand()) != 1 {
		t.Fatal("normal turn did not resume after a skipped extra turn")
	}
}
