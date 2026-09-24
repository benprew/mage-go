package mage

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDelayedTriggersWaitForPriorityAndAPNAP(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for _, controller := range []Player{b, a} {
		g.RegisterDelayedTrigger(&DelayedTrigger{EventType: EvtEndStep, Controller: controller.PlayerID(), Effects: []Effect{GainLifeFor(You, Fixed(1))}})
	}
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
	if g.GetStack().Size() != 0 || len(g.triggers.Pending()) != 2 {
		t.Fatal("delayed triggers bypassed pending placement")
	}
	clone := g.Clone()
	clone.PutTriggersOnStack()
	if clone.GetStack().Peek().Controller != b.PlayerID() {
		t.Fatal("non-active player's delayed trigger is not on top")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || clone.GetPlayer(b.PlayerID()).Life() != 21 || a.Life() != 20 {
		t.Fatal("pending delayed resolution leaked across clone")
	}
	g.PutTriggersOnStack()
	g.ResolveStack()
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
	if len(g.triggers.Pending()) != 0 {
		t.Fatal("one-shot delayed trigger fired again")
	}
}

func TestComposableDelayedCaptureAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source, host := NewCreature("Source", "", 1, 1), NewCreature("Host", "", 2, 2)
	source.SetOwner(a.PlayerID())
	host.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(host, a.PlayerID())
	g.MutablePermanent(source.ID()).AttachedTo = host.ID()
	saved := BindPermanent("host")
	effect := Sequence(Snapshot(AttachedPermanent, saved),
		AtNext(EachEndStep, Sequence(Destroy(From(saved)), GainLifeFor(You, Fixed(1)))),
		Snapshot(This, saved),
	)
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	clone := g.Clone()
	clone.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if clone.FindPermanent(host.ID()) != nil || g.FindPermanent(host.ID()) == nil || clone.GetPlayer(a.PlayerID()).Life() != 21 || a.Life() != 20 {
		t.Fatal("delayed binding, controller, or clone isolation failed")
	}
	g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
	g.PutOnBattlefield(host, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindPermanent(host.ID()) == nil || a.Life() != 21 {
		t.Fatal("delayed reference followed a returned host or stopped the sequence")
	}
}

func TestComposableDelayedStepAndCreationTime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
	if err := ApplyEffect(g, AtNext(YourUpkeep, GainLifeFor(You, Fixed(2))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.GetStack().Size() != 0 {
		t.Fatal("delayed trigger saw an earlier event")
	}
	g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: b.PlayerID()})
	if len(g.triggers.Pending()) != 0 || len(g.DelayedTriggers()) != 1 {
		t.Fatal("opponent upkeep consumed your-upkeep trigger")
	}
	g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 20 || len(g.DelayedTriggers()) != 0 {
		t.Fatal("delayed trigger lost its controller or fired more than once")
	}
}

func TestComposableDelayedBindingScope(t *testing.T) {
	saved := BindPermanent("saved")
	cases := []struct {
		effect ResolvingEffect
		valid  bool
	}{
		{Sequence(Snapshot(This, saved), AtNext(EachEndStep, Destroy(From(saved)))), true},
		{AtNext(EachEndStep, Sequence(Snapshot(This, saved), Destroy(From(saved)))), true},
		{AtNext(EachEndStep, Destroy(From(saved))), false},
		{Sequence(AtNext(EachEndStep, Snapshot(This, saved)), Destroy(From(saved))), false},
		{Sequence(AtNext(EachEndStep, Destroy(From(saved))), Snapshot(This, saved)), false},
		{AtNext(nil, Destroy(This)), false},
		{AtNext(EachEndStep, nil), false},
	}
	for i, tt := range cases {
		card := NewCreature("Delayed scope", "", 1, 1, WithAbility(Activate(Costs(), tt.effect)))
		if err := ValidateCardDefinition("Delayed scope", card); (err == nil) != tt.valid {
			t.Fatalf("case %d: validation = %v", i, err)
		}
	}
}

func TestComposableDelayedTargetFrames(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePermanentTarget("creature", IsCreature)
	effect := AtNext(EachEndStep, Sequence(Destroy(target), GainLifeFor(You, ValueAtResolution(XValue()))))
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	for i, card := range []Card{first, second} {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
		g.resolution.SetX(i + 1)
		g.resolution.targetSpecs = []Target{target.targetDeclaration}
		if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), []uuid.UUID{card.ID()}); err != nil {
			t.Fatal(err)
		}
	}
	g.resolution.SetX(0)
	g.resolution.targetSpecs = nil
	phased := g.PhaseOut(first.ID())
	g.PhaseIn(phased)
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	g.ResolveStack()
	if g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) != nil || a.Life() != 23 {
		t.Fatal("repeated schedules shared targets or lost captured X")
	}
}

func TestComposableDelayedSourceAlreadyReturned(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 2, 2)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: card.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
	g.RemoveFromBattlefield(p)
	g.PutOnBattlefield(card, a.PlayerID())
	if err := AtNext(EachEndStep, Destroy(This)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.DelayedTriggers()[0].sourceIncarnation != ctx.sourceIncarnation {
		t.Fatal("registration replaced the captured source incarnation")
	}
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
	g.ResolveStack()
	if g.FindPermanent(card.ID()) == nil {
		t.Fatal("delayed source action followed a returned card")
	}
}

func TestComposableDelayedDirectContextCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 2, 2)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: card.ID(), Controller: a.PlayerID()}
	if err := AtNext(EachEndStep, Destroy(This)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(p)
	g.PutOnBattlefield(card, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
	g.ResolveStack()
	if g.FindPermanent(card.ID()) == nil {
		t.Fatal("direct-context delayed action followed a returned source")
	}
}

func TestComposableDelayedRestoresFrameAfterError(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	saved := BindPermanent("saved")
	original := capturedPermanent{id: uuid.New(), incarnation: uuid.New()}
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{saved: original}
	g.resolution.SetX(4)
	effect := AtNext(EachEndStep, Sequence(Snapshot(This, saved),
		FuncEffect("fail", EffectProperties{}, func(*Game, uuid.UUID, uuid.UUID, []uuid.UUID) error { return ErrPlayerNotFound }),
	))
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.resolution.SetX(9)
	body := g.DelayedTriggers()[0].Effects[0]
	if err := body.Apply(ctx); !errors.Is(err, ErrPlayerNotFound) {
		t.Fatalf("error = %v", err)
	}
	if g.resolution.X() != 9 || g.resolution.permanentBindings[saved] != original {
		t.Fatal("failed delayed body leaked resolution state")
	}
	if body.(*capturedDelayedBody).permanents[saved] != original {
		t.Fatal("failed delayed body changed its immutable capture")
	}
}
