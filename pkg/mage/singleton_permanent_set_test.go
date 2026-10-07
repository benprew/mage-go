package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSingletonPermanentSetSourceIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 2, 4)
	p := g.PutOnBattlefield(card, a.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: p.ID(), sourceIncarnation: p.incarnationID, sourceCaptured: true, Controller: a.PlayerID()}
	set := SingletonPermanent(This)
	if got := set.resolvePermanents(ctx); len(got) != 1 || got[0].ID() != p.ID() {
		t.Fatal("missing source")
	}
	clone := g.Clone()
	child := *ctx
	child.Game = clone
	if err := TapEach(set).Apply(&child); err != nil {
		t.Fatal(err)
	}
	if !clone.FindPermanent(p.ID()).Tapped || g.FindPermanent(p.ID()).Tapped {
		t.Fatal("clone crossed games")
	}
	g.PhaseOut(p.ID())
	if got := set.resolvePermanents(ctx); len(got) != 0 {
		t.Fatal("phased source selected")
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	g.AddControlEffect(ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	if got := set.resolvePermanents(ctx); len(got) != 1 {
		t.Fatal("control change lost identity")
	}
	g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
	if got := set.resolvePermanents(ctx); len(got) != 0 {
		t.Fatal("LKI restored source")
	}
	g.PutOnBattlefield(card, a.PlayerID())
	if got := set.resolvePermanents(ctx); len(got) != 0 {
		t.Fatal("returned source replaced original")
	}
}

func TestSingletonPermanentSetDeclaration(t *testing.T) {
	for _, ref := range []PermanentRef{nil, DeclarePermanentTarget("missing", IsCreature)} {
		card := NewInstant("Invalid", "", WithAbility(Spell(TapEach(SingletonPermanent(ref)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid reference accepted")
		}
	}
	target := DeclarePermanentTarget("valid", IsCreature)
	card := NewInstant("Valid", "", WithAbility(Spell(TapEach(SingletonPermanent(target)), WithTargets(target))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestSingletonPermanentSetBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := DeclarePermanentTarget("first", IsCreature)
	second := DeclarePermanentTarget("second", IsCreature)
	card := NewInstant("Batch", "", WithAbility(Spell(TapEach(UnionPermanents(SingletonPermanent(first), SingletonPermanent(second), SingletonPermanent(first))), WithTargets(first, second))))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	p := g.PutOnBattlefield(NewCreature("First", "", 2, 4), b.PlayerID())
	q := g.PutOnBattlefield(NewCreature("Second", "", 2, 4), b.PlayerID())
	observer := g.PutOnBattlefield(NewLand("Observer", WithAbility(NewStateTriggered(false, Sacrifice(This)).SetCondition(func(_ *core.GameEvent, game GameReader, _, _ uuid.UUID) bool {
		return game.FindPermanent(p.ID()).Tapped && !game.FindPermanent(q.ID()).Tapped
	}))), a.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID(), q.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if !g.FindPermanent(p.ID()).Tapped || !g.FindPermanent(q.ID()).Tapped || g.FindPermanent(observer.ID()) == nil {
		t.Fatal("batch exposed intermediate state")
	}
}
