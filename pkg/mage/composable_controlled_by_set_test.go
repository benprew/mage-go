package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentsControlledByCurrentControllerAndLifecycle(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	foreign := NewArtifact("Foreign", "")
	foreign.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(foreign, b.PlayerID())
	owned := NewArtifact("Owned", "")
	owned.SetOwner(b.PlayerID())
	g.PutOnBattlefield(owned, a.PlayerID())
	g.PutOnBattlefield(NewCreature("Creature", "", 1, 1), b.PlayerID())
	set := PermanentsControlledBy(You, IsArtifact)
	ctx := &EffectContext{Game: g, Controller: b.PlayerID()}
	count := CountPermanents(set)
	if got := count.resolveValue(ctx); got != 1 {
		t.Fatalf("controlled artifacts = %d, want 1", got)
	}
	g.PhaseOut(p.ID())
	if got := count.resolveValue(ctx); got != 0 {
		t.Fatalf("phased count = %d, want 0", got)
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(p.ID()))
	if got := count.resolveValue(&EffectContext{Game: clone, Controller: b.PlayerID()}); got != 0 {
		t.Fatalf("clone count = %d, want 0", got)
	}
	if got := count.resolveValue(ctx); got != 1 {
		t.Fatalf("original count = %d, want 1", got)
	}
	g.AddControlEffect(ControlEffectSpec{SourceID: owned.ID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: Indefinite})
	g.ApplyContinuousEffects()
	if got := count.resolveValue(ctx); got != 0 {
		t.Fatalf("changed control count = %d, want 0", got)
	}
	if got := CountPermanents(PermanentsControlledBy(You, IsArtifact)).resolveValue(&EffectContext{Game: g, Controller: a.PlayerID()}); got != 2 {
		t.Fatalf("new controller artifact count = %d, want 2", got)
	}
	g.RemoveFromBattlefield(p)
	if got := count.resolveValue(ctx); got != 0 {
		t.Fatalf("departed count = %d, want 0", got)
	}
	g.PutOnBattlefield(foreign, a.PlayerID())
	if got := count.resolveValue(ctx); got != 0 {
		t.Fatalf("new controller count = %d, want 0", got)
	}
}

func TestPermanentsControlledByTargetAndDeclaration(t *testing.T) {
	target := DeclarePlayerTarget("player")
	body := GainLifeFor(You, CountPermanents(PermanentsControlledBy(target, IsArtifact)))
	spell := Spell(body, WithTargets(target))
	card := NewInstant("Count controlled", "", WithAbility(spell))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	for _, set := range []PermanentSet{PermanentsControlledBy(nil, IsArtifact), PermanentsControlledBy(target, IsArtifact)} {
		bad := NewInstant("Bad", "", WithAbility(Spell(GainLifeFor(You, CountPermanents(set)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatalf("invalid set accepted: %v", InspectDefinition(set))
		}
	}
	missing := PermanentsControlledBy(FromPlayer(BindPlayer("missing")), IsArtifact)
	if got := CountPermanents(missing).resolveValue(&EffectContext{Game: NewGame(NewBasePlayer("C"), NewBasePlayer("D"))}); got != 0 {
		t.Fatalf("absent player count = %d, want 0", got)
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	artifact := NewArtifact("Artifact", "")
	artifact.SetOwner(a.PlayerID())
	g.PutOnBattlefield(artifact, b.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{b.PlayerID()}, TargetSpecs: spell.Targets(), Effects: spell.Effects()}
	g.pushStack(obj)
	g.CopyStackObjectDirect(obj, b.PlayerID(), false)
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatalf("copy lives = %d, %d", a.Life(), b.Life())
	}
}
