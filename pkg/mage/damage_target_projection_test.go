package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestDamageTargetProjectionDeclarations(t *testing.T) {
	target := DeclareOpponentChosenDamageTarget("union")
	ref := PermanentFromDamageTarget(target)
	card := NewInstant("Union", "", Spell(Sequence(TapSubject(ref), GainLifeFor(ControllerOfWithLKI(ref), Fixed(2))), WithTargets(target)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if !target.OpponentChoosesTarget() {
		t.Fatal("lost opponent choice")
	}
	if PermanentFromDamageTarget(nil) != nil {
		t.Fatal("nil target projection")
	}
	if ValidateCardDefinition("Unbound", NewInstant("Unbound", "", Spell(TapSubject(ref)))) == nil {
		t.Fatal("unbound projection accepted")
	}
	if ValidateCardDefinition("Nil", NewInstant("Nil", "", Spell(TapSubject(PermanentFromDamageTarget(nil))))) == nil {
		t.Fatal("nil projection accepted")
	}
}

func TestDamageTargetProjectionIdentity(t *testing.T) {
	g, a, b := newObjectColorGame()
	target := DeclareDamageTarget("union")
	ref := PermanentFromDamageTarget(target)
	p := g.PutOnBattlefield(NewCreature("Subject", "", 2, 5), b.PlayerID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{p.ID(), p.incarnationID}}}}
	if got, ok := ref.resolvePermanent(ctx); !ok || got != p {
		t.Fatal("live projection")
	}
	g.PhaseOut(p.ID())
	if _, ok := ref.resolvePermanent(ctx); ok {
		t.Fatal("phased projection permits action")
	}
	if id, ok := ControllerOfWithLKI(ref).resolvePlayer(ctx); !ok || id != b.PlayerID() {
		t.Fatal("phase information")
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	g.BouncePermanentToHand(p)
	returned := g.PutOnBattlefield(p.Card, a.PlayerID())
	if _, ok := ref.resolvePermanent(ctx); ok {
		t.Fatal("projection followed returned card")
	}
	if id, ok := ControllerOfWithLKI(ref).resolvePlayer(ctx); !ok || id != b.PlayerID() {
		t.Fatal("departure controller lost")
	}
	ctx.targetBindings[target.targetDeclaration] = []capturedPermanent{{returned.ID(), returned.incarnationID}}
	if got, ok := ref.resolvePermanent(ctx); !ok || got != returned {
		t.Fatal("fresh visit unavailable")
	}
	ctx.targetBindings[target.targetDeclaration] = []capturedPermanent{{b.PlayerID(), uuid.Nil}}
	if _, ok := ref.resolvePermanent(ctx); ok {
		t.Fatal("player became permanent")
	}
	if _, ok := ControllerOfWithLKI(ref).resolvePlayer(ctx); ok {
		t.Fatal("player supplied permanent information")
	}
}

func TestDamageTargetProjectionOpponentChoice(t *testing.T) {
	g, a, b := newTriggerTargetGame()
	mine := g.PutOnBattlefield(NewCreature("Mine", "", 2, 5), a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 2, 5), b.PlayerID())
	first, second := DeclareDamageTarget("first"), DeclareOpponentChosenDamageTarget("second")
	card := NewInstant("Union", "", Spell(Sequence(TapSubject(PermanentFromDamageTarget(first)), TapSubject(PermanentFromDamageTarget(second))), WithTargets(first, second)))
	a.AddToHand(card)
	b.chooseQueue = [][]uuid.UUID{{mine.ID()}}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{mine.ID(), other.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	if g.StackPeek().Targets[0] != mine.ID() || g.StackPeek().Targets[1] != mine.ID() {
		t.Fatal("independent target groups cannot share a recipient")
	}
	if len(a.calls) != 0 || len(b.calls) != 1 {
		t.Fatal("wrong target chooser")
	}
	clone := g.Clone()
	g.ResolveStack()
	if clone.FindPermanent(mine.ID()).Tapped {
		t.Fatal("clone state shared")
	}
	clone.ResolveStack()
	if !g.FindPermanent(mine.ID()).Tapped || !clone.FindPermanent(mine.ID()).Tapped || g.FindPermanent(other.ID()).Tapped {
		t.Fatal("projection group mismatch")
	}
}
