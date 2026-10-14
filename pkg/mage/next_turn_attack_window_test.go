package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestNextTurnAttackWindowLifetime(t *testing.T) {
	for _, change := range []string{"none", "source-left", "source-phase", "subject-return", "subject-phase", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.RunStepWithPriority(core.Untap)
			card := NewCreature("Subject", "", 2, 5)
			p := g.PutOnBattlefield(card, b.PlayerID())
			target := DeclarePermanentTarget("subject", IsCreature)
			source := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(Activate(Costs(), CantAttackDuringNextTurn(target, ControllerOf(target)), WithTargets(target)))), a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.restrictedFromAttacking(p) {
				t.Fatal("window started during installation turn")
			}
			switch change {
			case "source-left":
				g.RemoveFromBattlefield(source)
			case "source-phase":
				g.PhaseOut(source.ID())
			case "subject-return":
				g.RemoveFromBattlefield(p)
				p = g.PutOnBattlefield(card, b.PlayerID())
			case "subject-phase":
				g.PhaseOut(p.ID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			}
			clone := g.Clone()
			g.RunStepWithPriority(core.Cleanup)
			g.AdvanceToNextTurn()
			if g.restrictedFromAttacking(g.FindPermanentIncludingPhased(p.ID())) {
				t.Fatal("turn selection started window")
			}
			g.SkipNextOccurrenceOfStep(core.Untap)
			g.RunStepWithPriority(core.PrecombatMain)
			if change == "subject-phase" {
				g.PhaseIn([]uuid.UUID{p.ID()})
			}
			want := change != "subject-return"
			if g.restrictedFromAttacking(g.FindPermanent(p.ID())) != want {
				t.Fatal("wrong saved subject or player")
			}
			if clone.restrictedFromAttacking(clone.FindPermanentIncludingPhased(p.ID())) {
				t.Fatal("start changed clone")
			}
			g.RunStepWithPriority(core.EndStep)
			if g.restrictedFromAttacking(g.FindPermanent(p.ID())) != want {
				t.Fatal("window ended before cleanup")
			}
			g.RunStepWithPriority(core.Cleanup)
			if g.restrictedFromAttacking(g.FindPermanent(p.ID())) {
				t.Fatal("cleanup kept window")
			}
			clone.AdvanceToNextTurn()
			clone.RunStepWithPriority(core.Untap)
			if change == "subject-phase" {
				clone.PhaseIn([]uuid.UUID{p.ID()})
			}
			if clone.restrictedFromAttacking(clone.FindPermanent(p.ID())) != want {
				t.Fatal("clone lost window")
			}
		})
	}
}

func TestNextTurnAttackWindowDeclarations(t *testing.T) {
	for _, body := range []ResolvingEffect{CantAttackDuringNextTurn(nil, You), CantAttackDuringNextTurn(This, nil), CantAttackDuringNextTurn(DeclarePermanentTarget("missing", IsCreature), You), CantAttackDuringNextTurn(This, DeclarePlayerTarget("missing"))} {
		if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), body)))); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Subject", "", 1, 4), a.PlayerID())
	if err := ApplyEffect(g, CantAttackDuringNextTurn(This, You), p.ID(), uuid.New(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.effects.effects) != 0 {
		t.Fatal("absent player installed window")
	}
}

func TestNextTurnAttackWindowSkippedAndExtraTurns(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for range 12 {
		a.AddToLibrary(NewLand("A"))
		b.AddToLibrary(NewLand("B"))
	}
	g.RunStepWithPriority(core.Untap)
	p := g.PutOnBattlefield(NewCreature("Subject", "", 2, 5), a.PlayerID())
	if err := ApplyEffect(g, CantAttackDuringNextTurn(This, You), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.SkipNextTurnFor(a.PlayerID())
	g.AdvanceToNextTurn()
	g.Run(0, core.Untap, 4)
	if g.restrictedFromAttacking(g.FindPermanent(p.ID())) {
		t.Fatal("native skipped turn started window")
	}
	g.GrantExtraTurn(a.PlayerID())
	g.AdvanceToNextTurn()
	g.SkipNextOccurrenceOfStep(core.Untap)
	g.RunStepWithPriority(core.PrecombatMain)
	if !g.restrictedFromAttacking(g.FindPermanent(p.ID())) {
		t.Fatal("extra turn with skipped untap did not start window")
	}
	g.BeginActiveTurn()
	g.RunStepWithPriority(core.Upkeep)
	if !g.restrictedFromAttacking(g.FindPermanent(p.ID())) {
		t.Fatal("repeated turn entry or upkeep ended window")
	}
	g.RunStepWithPriority(core.Cleanup)
	g.RunStepWithPriority(core.Cleanup)
	if g.restrictedFromAttacking(g.FindPermanent(p.ID())) {
		t.Fatal("repeated cleanup retained window")
	}
	g.GrantExtraTurn(a.PlayerID())
	g.AdvanceToNextTurn()
	g.RunStepWithPriority(core.PrecombatMain)
	if g.restrictedFromAttacking(g.FindPermanent(p.ID())) {
		t.Fatal("later turn restarted window")
	}
}

func TestNextTurnAttackWindowIndependentPlayers(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.RunStepWithPriority(core.Untap)
	p := g.PutOnBattlefield(NewCreature("A", "", 2, 5), a.PlayerID())
	q := g.PutOnBattlefield(NewCreature("B", "", 2, 5), b.PlayerID())
	for _, subject := range []*Permanent{p, q} {
		if err := ApplyEffect(g, CantAttackDuringNextTurn(This, You), subject.ID(), subject.ControllerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	g.AdvanceToNextTurn()
	g.RunStepWithPriority(core.PrecombatMain)
	if g.restrictedFromAttacking(p) || !g.restrictedFromAttacking(q) {
		t.Fatal("player windows crossed")
	}
	clone := g.Clone()
	g.RunStepWithPriority(core.Cleanup)
	if !clone.restrictedFromAttacking(clone.FindPermanent(q.ID())) {
		t.Fatal("expiry changed active clone")
	}
	g.AdvanceToNextTurn()
	g.RunStepWithPriority(core.PrecombatMain)
	if !g.restrictedFromAttacking(p) || g.restrictedFromAttacking(q) {
		t.Fatal("second player window lost")
	}
}

func TestNextTurnAttackWindowWaitingSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	body := AtNext(EachEndStep, CantAttackDuringNextTurn(This, ControllerOf(This)))
	source := g.PutOnBattlefield(NewCreature("Source", "", 2, 5, WithAbility(Activate(Costs(), body))), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if got := g.WaitingDelayedPermanentSources(); len(got) != 1 || got[0].ID() != source.ID() {
		t.Fatal("waiting query lost source")
	}
	g.RemoveFromBattlefield(source)
	if got := g.WaitingDelayedPermanentSources(); len(got) != 1 || got[0].ID() != source.ID() {
		t.Fatal("waiting query lost departure facts")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if len(g.effects.effects) != 0 {
		t.Fatal("LKI supplied an actionable subject")
	}
}
