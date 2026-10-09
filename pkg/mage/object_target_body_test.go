package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestObjectTargetBodyDeclarations(t *testing.T) {
	target := DeclareSpellOrPermanentTarget("object")
	body := ResolveObjectTarget(target, GainLifeFor(ControllerOfSpellWithLKI(SpellFromObjectTarget(target)), Fixed(2)), TapSubject(PermanentFromObjectTarget(target)))
	card := NewInstant("Union", "", Spell(body, WithTargets(target)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{ResolveObjectTarget(nil, GainLifeFor(You, Fixed(1)), GainLifeFor(You, Fixed(1))), ResolveObjectTarget(target, nil, GainLifeFor(You, Fixed(1))), ResolveObjectTarget(target, GainLifeFor(You, Fixed(1)), nil)} {
		c := NewInstant("Invalid", "", Spell(effect, WithTargets(target)))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("accepted invalid dispatch")
		}
	}
	if ValidateCardDefinition("Unbound", NewInstant("Unbound", "", Spell(body))) == nil {
		t.Fatal("accepted unbound union")
	}
}

func TestObjectTargetBodySavedIdentity(t *testing.T) {
	for _, state := range []string{"live", "phased", "returned"} {
		g, a, b := newObjectColorGame()
		target := DeclareSpellOrPermanentTarget("object")
		ref := PermanentFromObjectTarget(target)
		effect := ResolveObjectTarget(target, GainLifeFor(You, Fixed(9)), Sequence(ReturnToHand(ref), GainLifeFor(ControllerOfWithLKI(ref), Fixed(3))))
		p := g.PutOnBattlefield(NewArtifact("subject", ""), b.PlayerID())
		card := NewInstant("Union", "", Spell(effect, WithTargets(target)))
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
			t.Fatal(err)
		}
		switch state {
		case "phased":
			g.PhaseOut(p.ID())
		case "returned":
			g.BouncePermanentToHand(p)
			g.PutOnBattlefield(p.Card, b.PlayerID())
		}
		g.ResolveTopOfStack()
		want := 20
		if state == "live" {
			want = 23
		}
		if b.Life() != want || a.Life() != 20 {
			t.Fatalf("%s: wrong identity/LKI %d/%d", state, a.Life(), b.Life())
		}
	}
}

func TestObjectTargetBodyProjectionIdentity(t *testing.T) {
	g, a, b := newObjectColorGame()
	target := DeclareSpellOrPermanentTarget("object")
	p := g.PutOnBattlefield(NewArtifact("subject", ""), b.PlayerID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{p.ID(), p.incarnationID}}}}
	if _, ok := SpellFromObjectTarget(target).resolveStackObject(ctx); ok {
		t.Fatal("permanent became spell")
	}
	g.BouncePermanentToHand(p)
	next := &StackObject{ID: uuid.New(), SourceID: p.ID(), Card: p.Card, Controller: b.PlayerID()}
	g.pushStack(next)
	if _, ok := SpellFromObjectTarget(target).resolveStackObject(ctx); ok {
		t.Fatal("projection followed physical card")
	}
	ctx.targetBindings[target.targetDeclaration] = []capturedPermanent{{next.ID, uuid.Nil}}
	if _, ok := PermanentFromObjectTarget(target).resolvePermanent(ctx); ok {
		t.Fatal("spell became permanent")
	}
	if obj, ok := SpellFromObjectTarget(target).resolveStackObject(ctx); !ok || obj != next {
		t.Fatal("lost exact stack identity")
	}
	g.CounterStackObject(next.ID)
	replacement := &StackObject{ID: uuid.New(), SourceID: next.SourceID, Card: next.Card, Controller: b.PlayerID()}
	g.pushStack(replacement)
	if _, ok := SpellFromObjectTarget(target).resolveStackObject(ctx); ok {
		t.Fatal("projection followed later cast")
	}
}

func TestObjectTargetBodyBranchBindings(t *testing.T) {
	target := DeclareSpellOrPermanentTarget("object")
	saved := BindNumber("local value")
	producer := SnapshotNumber(Fixed(2), saved)
	consumer := GainLifeFor(You, FromNumber(saved))
	for _, body := range []ResolvingEffect{
		ResolveObjectTarget(target, producer, consumer),
		Sequence(ResolveObjectTarget(target, producer, GainLifeFor(You, Fixed(1))), consumer),
	} {
		card := NewInstant("Invalid", "", Spell(body, WithTargets(target)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("branch result escaped")
		}
	}
	valid := NewInstant("Valid", "", Spell(ResolveObjectTarget(target, Sequence(producer, consumer), GainLifeFor(You, Fixed(1))), WithTargets(target)))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	if PermanentFromObjectTarget(nil) != nil || SpellFromObjectTarget(nil) != nil {
		t.Fatal("nil projection")
	}
}
