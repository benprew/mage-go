package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestCombatPairDelayIdentity(t *testing.T) {
	for _, change := range []string{"none", "control", "source-return", "source-phase", "phase-cycle", "opponent-return", "opponent-phase", "copy", "remove-abilities", "source-left-after", "opponent-control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			sourceCard := NewCreature("Source", "", 2, 8)
			p := g.PutOnBattlefield(sourceCard, a.PlayerID())
			otherCard := NewCreature("Other", "", 1, 8)
			q := g.PutOnBattlefield(otherCard, b.PlayerID())
			saved := BindPermanent("opponent")
			body := Sequence(Until(EndOfTurn, GrantKeywordModification(From(saved), FirstStrike)), GainLifeFor(You, Fixed(1)))
			effect := WhenCombatPairUntilEndOfCombat(This, IsCreature, saved, body)
			evt := GameEvent{Type: EvtCombatPair, SourceID: p.ID(), TargetID: q.ID(), PlayerID: b.PlayerID()}
			g.FireEvent(evt)
			if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("event before creation was captured")
			}
			switch change {
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			case "source-return":
				g.RemoveFromBattlefield(p)
				p = g.PutOnBattlefield(sourceCard, a.PlayerID())
			case "source-phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "remove-abilities":
				g.AddContinuousEffect(RemoveAllAbilities(p.ID(), Indefinite))
				g.ApplyContinuousEffects()
			}
			g.FireEvent(evt)
			g.PutTriggersOnStack()
			want := 1
			if change == "source-return" || change == "source-phase" {
				want = 0
			}
			if g.StackSize() != want {
				t.Fatalf("stack=%d want %d", g.StackSize(), want)
			}
			if change == "source-left-after" {
				g.RemoveFromBattlefield(p)
			}
			if change == "opponent-control" {
				g.AddControlEffect(ControlEffectSpec{SourceID: q.ID(), TargetID: q.ID(), ControllerID: a.PlayerID(), Duration: Indefinite})
			}
			if change == "opponent-return" {
				g.RemoveFromBattlefield(q)
				q = g.PutOnBattlefield(otherCard, b.PlayerID())
			}
			if change == "opponent-phase" {
				g.PhaseOut(q.ID())
			}
			if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			g.ResolveStack()
			wantB := 20
			if change == "copy" {
				wantB++
			}
			if a.Life() != 20+want || b.Life() != wantB {
				t.Fatal("creator controller was lost")
			}
			keyword := want == 1 && change != "opponent-return" && change != "opponent-phase"
			if g.FindPermanentIncludingPhased(q.ID()).HasKeyword(FirstStrike) != keyword {
				t.Fatal("opponent visit was lost")
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("clone shared resolution")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != a.Life() {
				t.Fatal("clone lost frame")
			}
		})
	}
}

func TestCombatPairDelayDurationAndEvents(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 2, 8), a.PlayerID())
	q := g.PutOnBattlefield(NewCreature("Other", "", 1, 8), b.PlayerID())
	saved := BindPermanent("opponent")
	effect := WhenCombatPairUntilEndOfCombat(This, IsCreature, saved, Sequence(Until(EndOfTurn, GrantKeywordModification(From(saved), FirstStrike)), GainLifeFor(You, Fixed(1))))
	if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for _, event := range []GameEvent{
		{Type: EvtCombatPair, SourceID: p.ID(), TargetID: q.ID(), TargetIncarnation: uuid.New()},
		{Type: EvtCombatPair, SourceID: p.ID(), TargetID: q.ID(), SourceIncarnation: uuid.New()},
		{Type: EvtCombatPair, SourceID: q.ID(), TargetID: p.ID()},
	} {
		g.FireEvent(event)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("invalid identities matched")
	}
	for range 2 {
		g.FireEvent(GameEvent{Type: EvtCombatPair, SourceID: p.ID(), TargetID: q.ID()})
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 || len(g.DelayedTriggers()) != 1 {
		t.Fatal("duration consumed on first event")
	}
	clone := g.Clone()
	g.RunStepWithPriority(EndCombat)
	if len(g.DelayedTriggers()) != 0 || a.Life() != 22 || !g.FindPermanent(q.ID()).HasKeyword(FirstStrike) {
		t.Fatal("combat expiry lost pending body or turn grant")
	}
	g.FireEvent(GameEvent{Type: EvtCombatPair, SourceID: p.ID(), TargetID: q.ID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 || len(clone.DelayedTriggers()) != 1 {
		t.Fatal("expiry or clone isolation")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if g.FindPermanent(q.ID()).HasKeyword(FirstStrike) {
		t.Fatal("turn grant did not expire")
	}
}

func TestCombatPairDelayDefinition(t *testing.T) {
	saved := BindPermanent("opponent")
	for _, tc := range []struct {
		subject PermanentRef
		binding *PermanentBinding
		body    ResolvingEffect
		bad     bool
	}{
		{This, saved, TapSubject(From(saved)), false},
		{nil, saved, TapSubject(From(saved)), true},
		{From(BindPermanent("missing")), saved, TapSubject(From(saved)), true},
		{This, nil, GainLifeFor(You, Fixed(1)), true},
		{This, saved, nil, true},
		{This, saved, TapSubject(From(BindPermanent("missing"))), true},
	} {
		card := NewCreature("Test", "", 2, 8, WithAbility(Activate(Costs(), WhenCombatPairUntilEndOfCombat(tc.subject, IsCreature, tc.binding, tc.body))))
		if err := ValidateCardDefinition(card.Name(), card); (err != nil) != tc.bad {
			t.Fatalf("validation=%v bad=%v", err, tc.bad)
		}
	}
}

func TestCombatPairDelayEndPriority(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Subject", "", 2, 8), b.PlayerID())
	q := g.PutOnBattlefield(NewCreature("Other", "", 1, 8), a.PlayerID())
	binding := BindPermanent("opponent")
	target := DeclarePermanentTarget("subject", IsCreature)
	effect := WhenCombatPairUntilEndOfCombat(target, IsCreature, binding, Sequence(TapSubject(From(binding)), GainLifeFor(You, Fixed(1))))
	card := NewInstant("Create subscription", "", WithAbility(Spell(effect, WithTargets(target))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	watcher := NewArtifact("End observer", "", WithAbility(At(EachEndOfCombat, &funcEffect{fn: func(game *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
		if len(game.DelayedTriggers()) != 1 {
			t.Fatal("subscription expired before end-combat priority")
		}
		game.FireEvent(GameEvent{Type: EvtCombatPair, SourceID: p.ID(), TargetID: q.ID()})
		return nil
	}})))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.RunStepWithPriority(EndCombat)
	if a.Life() != 21 || b.Life() != 20 || !q.Tapped || len(g.DelayedTriggers()) != 0 {
		t.Fatal("creator or phase-end timing was lost")
	}
}

func TestCombatPairDelayIndependentCaptures(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Subject", "", 2, 8), a.PlayerID())
	q := g.PutOnBattlefield(NewCreature("First", "", 1, 8), b.PlayerID())
	r := g.PutOnBattlefield(NewCreature("Second", "", 1, 8), b.PlayerID())
	binding := BindPermanent("opponent")
	effect := WhenCombatPairUntilEndOfCombat(This, IsCreature, binding, TapSubject(From(binding)))
	for range 2 {
		if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []uuid.UUID{q.ID(), r.ID()} {
		g.FireEvent(GameEvent{Type: EvtCombatPair, SourceID: p.ID(), TargetID: id})
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 4 {
		t.Fatal("independent installations or opponents merged")
	}
	g.triggers.ClearEndOfCombat()
	g.RemoveFromBattlefield(q)
	clone := g.Clone()
	g.ResolveStack()
	if !g.FindPermanent(r.ID()).Tapped || clone.FindPermanent(r.ID()).Tapped {
		t.Fatalf("pending frame: original tapped=%t clone tapped=%t", g.FindPermanent(r.ID()).Tapped, clone.FindPermanent(r.ID()).Tapped)
	}
	clone.ResolveStack()
	if !clone.FindPermanent(r.ID()).Tapped {
		t.Fatal("clone lost second opponent")
	}
}

func TestCombatPairDelayCleanupAndMissing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 2, 8), a.PlayerID())
	q := g.PutOnBattlefield(NewCreature("Other", "", 1, 8), b.PlayerID())
	binding := BindPermanent("opponent")
	effect := WhenCombatPairUntilEndOfCombat(This, IsCreature, binding, TapSubject(From(binding)))
	if err := ApplyEffect(g, effect, uuid.New(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.DelayedTriggers()) != 0 {
		t.Fatal("missing source installed subscription")
	}
	if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.RunStepWithPriority(Cleanup)
	g.FireEvent(GameEvent{Type: EvtCombatPair, SourceID: p.ID(), TargetID: q.ID()})
	g.PutTriggersOnStack()
	if len(g.DelayedTriggers()) != 0 || g.StackSize() != 0 {
		t.Fatal("unused subscription survived cleanup")
	}
}
