package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestOtherSourceDamageHistory(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Named source", "", 1, 30), a.PlayerID())
	q := g.PutOnBattlefield(NewCreature("Named source", "", 1, 30), b.PlayerID())
	r := g.PutOnBattlefield(NewCreature("Other", "", 1, 30), b.PlayerID())
	value := DamageReceivedThisTurnFromOtherSources(This, DamageSourceHasName("Named source"))
	ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
	g.AddPreventionShield(p.ID(), 2)
	g.DealDamageToPermanent(p, 2, q.ID())
	g.DealDamageToPermanent(p, 0, q.ID())
	g.DealDamageToPermanent(p, -1, q.ID())
	g.DealDamageToPermanent(p, 4, p.ID())
	g.DealDamageToPermanent(p, 7, r.ID())
	if got := value.resolveValue(ctx); got != 0 {
		t.Fatalf("excluded damage: %d", got)
	}
	g.DealSimultaneousDamage(PermanentDamageAssignment(p, 2, g.CaptureDamageSource(q.ID())), PermanentDamageAssignment(p, 3, g.CaptureDamageSource(q.ID())))
	if got := value.resolveValue(ctx); got != 5 {
		t.Fatalf("actual damage: %d", got)
	}
	g.PhaseOut(p.ID())
	g.DealDamageToPermanent(p, 9, q.ID())
	if got := value.resolveValue(ctx); got != 5 {
		t.Fatalf("phased history: %d", got)
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	clone := g.Clone()
	cloneCtx := *ctx
	cloneCtx.Game = clone
	clone.DealDamageToPermanent(clone.FindPermanent(p.ID()), 1, q.ID())
	if value.resolveValue(&cloneCtx) != 6 || value.resolveValue(ctx) != 5 {
		t.Fatal("clone shares history")
	}
	g.RemoveFromBattlefield(p)
	returned := g.PutOnBattlefield(p.Card, a.PlayerID())
	if value.resolveValue(ctx) != 5 {
		t.Fatal("departed visit lost history")
	}
	if got := value.resolveValue(&EffectContext{Game: g, SourceID: returned.ID()}); got != 0 {
		t.Fatalf("return inherited history: %d", got)
	}
	g.damage.ClearEndOfTurn()
	if value.resolveValue(ctx) != 0 || value.resolveValue(&cloneCtx) != 6 {
		t.Fatal("cleanup changed clone or retained history")
	}
}

func TestOtherSourceDamageHistorySnapshots(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Recipient", "", 1, 30), a.PlayerID())
	q := g.PutOnBattlefield(NewCreature("Named source", "", 1, 30), b.PlayerID())
	source := g.CaptureDamageSource(q.ID())
	g.DealDamageToPermanentFromSource(p, 2, source)
	g.RemoveFromBattlefield(q)
	q.Card.(*BaseCard).name = "Changed name"
	g.PutOnBattlefield(q.Card, b.PlayerID())
	value := DamageReceivedThisTurnFromOtherSources(This, DamageSourceHasName("Named source"))
	ctx := &EffectContext{Game: g, SourceID: p.ID()}
	if value.resolveValue(ctx) != 2 {
		t.Fatal("later source state changed historical characteristics")
	}
	spell := NewInstant("Named source", "", Spell(DealDamageFrom(This, DeclarePermanentTarget("victim", IsCreature), Fixed(3))))
	captured := DamageSource{id: spell.ID(), stackID: uuid.New(), name: spell.Name(), card: spell}
	g.DealDamageToPermanentFromSource(p, 3, captured)
	if value.resolveValue(ctx) != 5 {
		t.Fatal("spell source not counted")
	}
	if !DamageSourceHasName("Named source").Match(captured) || DamageSourceHasName("Other").Match(captured) {
		t.Fatal("name filter")
	}
	g.DealDamageToPlayer(b, 4, q.ID())
	if value.resolveValue(ctx) != 5 {
		t.Fatal("player damage counted")
	}
}

func TestOtherSourceDamageHistoryDefinition(t *testing.T) {
	for _, value := range []ResolutionValue{
		DamageReceivedThisTurnFromOtherSources(nil, DamageSourceHasName("Named source")),
		DamageReceivedThisTurnFromOtherSources(From(BindPermanent("missing")), DamageSourceHasName("Named source")),
		DamageReceivedThisTurnFromOtherSources(This, DamageSourceHasName("")),
	} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), GainLifeFor(You, value))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid value accepted")
		}
	}
	card := NewCreature("Valid", "", 1, 1, WithAbility(Activate(Costs(), GainLifeFor(You, DamageReceivedThisTurnFromOtherSources(This, DamageSourceHasName("Named source"))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestOtherSourceDamageHistoryCleanupAndWaiting(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Recipient", "", 1, 20), a.PlayerID())
	q := g.PutOnBattlefield(NewCreature("Named source", "", 1, 20), b.PlayerID())
	value := DamageReceivedThisTurnFromOtherSources(This, DamageSourceHasName("Named source"))
	ctx := &EffectContext{Game: g, SourceID: p.ID()}
	g.DealDamageToPermanent(p, 2, q.ID())
	saved := g.CaptureDamageSource(p.ID())
	if err := ApplyEffect(g, AtNext(EachEndStep, GainLifeFor(You, value)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(p)
	for range 2 {
		refs := g.WaitingDelayedPermanentSources()
		if len(refs) != 1 || !refs[0].SameObject(saved) || a.Life() != 20 {
			t.Fatal("waiting query lost source or ran effect")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("delayed value lost exact departed visit")
	}
	returned := g.PutOnBattlefield(p.Card, a.PlayerID())
	ctx.SourceID = returned.ID()
	g.DealDamageToPermanent(returned, 1, q.ID())
	g.RegisterDelayedTrigger(&DelayedTrigger{EventType: core.EvtCleanup, Controller: b.PlayerID(), Effects: []Effect{GainLifeFor(You, Fixed(1))}})
	if !g.doCleanupActions() || returned.Damage != 0 || value.resolveValue(ctx) != 1 {
		t.Fatal("cleanup priority erased damage amounts")
	}
	g.ResolveStack()
	g.RunStepWithPriority(core.Cleanup)
	if value.resolveValue(ctx) != 0 {
		t.Fatal("final cleanup retained damage amounts")
	}
}

func TestOtherSourceDamageHistoryRedirection(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Named source", "", 1, 20), a.PlayerID())
	recipient := g.PutOnBattlefield(NewCreature("Recipient", "", 1, 20), b.PlayerID())
	value := DamageReceivedThisTurnFromOtherSources(This, DamageSourceHasName("Named source"))
	ctx := &EffectContext{Game: g, SourceID: recipient.ID()}
	g.SetAttackerDamageRedirect(source.ID(), recipient.ID())
	g.AddPreventionShield(recipient.ID(), 1)
	g.DealDamageToPlayerFromSource(b, 4, g.CaptureDamageSource(source.ID()))
	if b.Life() != 20 || value.resolveValue(ctx) != 3 {
		t.Fatal("history did not use final recipient and amount")
	}
}
