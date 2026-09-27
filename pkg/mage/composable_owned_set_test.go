package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestPermanentsOwnedByOwnerAndLifecycle(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	owned := NewArtifact("Owned", "")
	owned.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(owned, b.PlayerID())
	other := NewArtifact("Other", "")
	other.SetOwner(b.PlayerID())
	g.PutOnBattlefield(other, a.PlayerID())
	creature := NewCreature("Creature", "", 1, 1)
	creature.SetOwner(a.PlayerID())
	g.PutOnBattlefield(creature, b.PlayerID())
	set := PermanentsOwnedBy(You, IsArtifact)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	count := CountPermanents(set)
	if got := count.resolveValue(ctx); got != 1 {
		t.Fatalf("owned artifacts = %d, want 1", got)
	}
	g.PhaseOut(p.ID())
	if got := count.resolveValue(ctx); got != 0 {
		t.Fatalf("phased count = %d, want 0", got)
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(p.ID()))
	if got := count.resolveValue(&EffectContext{Game: clone, Controller: a.PlayerID()}); got != 0 {
		t.Fatalf("clone count = %d, want 0", got)
	}
	if got := count.resolveValue(ctx); got != 1 {
		t.Fatalf("original count = %d, want 1", got)
	}
	g.RemoveFromBattlefield(p)
	if got := count.resolveValue(ctx); got != 0 {
		t.Fatalf("departed count = %d, want 0", got)
	}
	g.PutOnBattlefield(owned, b.PlayerID())
	if got := count.resolveValue(ctx); got != 1 {
		t.Fatalf("returned count = %d, want 1", got)
	}
}

func TestPermanentsOwnedByTargetAndDeclaration(t *testing.T) {
	target := DeclarePlayerTarget("player")
	body := GainLifeFor(You, CountPermanents(PermanentsOwnedBy(target, IsArtifact)))
	spell := Spell(body, WithTargets(target))
	card := NewInstant("Count owned", "", WithAbility(spell))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	for _, set := range []PermanentSet{PermanentsOwnedBy(nil, IsArtifact), PermanentsOwnedBy(target, IsArtifact)} {
		bad := NewInstant("Bad", "", WithAbility(Spell(GainLifeFor(You, CountPermanents(set)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatalf("invalid set accepted: %v", InspectDefinition(set))
		}
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	artifact := NewArtifact("Artifact", "")
	artifact.SetOwner(b.PlayerID())
	g.PutOnBattlefield(artifact, a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{b.PlayerID()}, TargetSpecs: spell.Targets(), Effects: spell.Effects()}
	g.pushStack(obj)
	g.CopyStackObjectDirect(obj, b.PlayerID(), false)
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatalf("copy lives = %d, %d", a.Life(), b.Life())
	}
}
