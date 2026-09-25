package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCountPermanentsResolution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	value := CountPermanents(ControlledPermanents(And(IsCreature, HasKeywordFilter(core.Flying))))
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if value.resolveValue(ctx) != 0 {
		t.Fatal("empty set must count as zero")
	}
	flyer := NewCreature("Flyer", "", 1, 1, WithKeyword(core.Flying))
	flyer.SetOwner(a.PlayerID())
	g.PutOnBattlefield(flyer, a.PlayerID())
	g.PutOnBattlefield(NewCreature("Other flyer", "", 1, 1, WithKeyword(core.Flying)), b.PlayerID())
	g.PutOnBattlefield(NewCreature("Ground creature", "", 1, 1), a.PlayerID())
	g.PutOnBattlefield(NewArtifact("Artifact", "", WithKeyword(core.Flying)), a.PlayerID())
	if value.resolveValue(ctx) != 1 {
		t.Fatal("count must filter type, keyword, and controller")
	}
	permanent := g.MutablePermanent(flyer.ID())
	permanent.PhasedOut = true
	if value.resolveValue(ctx) != 0 {
		t.Fatal("phased-out members must be absent")
	}
	permanent.PhasedOut = false
	clone := g.Clone()
	clone.DestroyPermanent(clone.FindPermanent(flyer.ID()))
	if value.resolveValue(&EffectContext{Game: clone, Controller: a.PlayerID()}) != 0 || value.resolveValue(ctx) != 1 {
		t.Fatal("count must read its own game")
	}
	saved := BindNumber("before")
	body := Sequence(SnapshotNumber(value, saved), Destroy(This), GainLifeFor(You, value), GainLifeFor(You, FromNumber(saved)))
	if err := ApplyEffect(g, body, flyer.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 21 {
		t.Fatalf("life = %d; count must update after destruction while snapshot stays fixed", a.Life())
	}
	g.PutOnBattlefield(flyer, a.PlayerID())
	if value.resolveValue(ctx) != 1 {
		t.Fatal("dynamic collection must include returned objects")
	}
}

func TestCountPermanentsValidation(t *testing.T) {
	for _, valid := range []bool{false, true} {
		member := BindPermanent("member")
		value := CountPermanents(WithoutPermanent(MatchingPermanents(IsCreature), From(member)))
		var body = GainLifeFor(You, value)
		if valid {
			body = Sequence(Snapshot(This, member), body)
		}
		card := NewCreature("Count", "", 1, 1, WithAbility(Activate(Costs(), body)))
		if err := ValidateCardDefinition(card.Name(), card); (err == nil) != valid {
			t.Fatalf("valid=%v: %v", valid, err)
		}
	}
	card := NewInstant("Nil count", "", WithAbility(Spell(GainLifeFor(You, CountPermanents(nil)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("nil collection accepted")
	}
}

func TestCountPermanentsCopiedController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 1, 1, WithAbility(At(EachEndStep,
		GainLifeFor(You, CountPermanents(ControlledPermanents(IsCreature))))))
	g.PutOnBattlefield(source, a.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("expected one trigger")
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	g.PutOnBattlefield(NewCreature("Later creature", "", 1, 1), a.PlayerID())
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 20 {
		t.Fatal("count must use resolution-time membership and the copy controller")
	}
}
