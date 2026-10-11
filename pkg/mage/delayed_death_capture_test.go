package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDelayedDeathCaptureFacts(t *testing.T) {
	for _, timing := range []string{"waiting-clone", "pending-clone", "stack-clone", "simultaneous"} {
		t.Run(timing, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			card := NewCreature("Watched", "", 2, 3)
			card.SetOwner(b.PlayerID())
			watched := g.PutOnBattlefield(card, a.PlayerID())
			permanent, owner, controller := BindPermanent("dead"), BindPlayer("owner"), BindPlayer("controller")
			body := Sequence(GainLifeFor(FromPlayer(owner), Fixed(3)), GainLifeFor(FromPlayer(controller), Fixed(5)))
			effect := WhenDiesThisTurnWithBindings(This, permanent, owner, controller, body)
			if err := ApplyEffect(g, effect, watched.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if err := ApplyEffect(g, Until(core.Indefinite, ChangeController(This, You)), watched.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			var clone *Game
			if timing == "waiting-clone" {
				clone = g.Clone()
			}
			if timing == "simultaneous" {
				g.DestroyPermanents([]*Permanent{g.FindPermanent(watched.ID()), source})
			} else {
				g.PutPermanentIntoGraveyard(g.FindPermanent(watched.ID()))
			}
			if g.StackSize() != 0 {
				t.Fatal("trigger placed during death")
			}
			if timing == "pending-clone" {
				clone = g.Clone()
			}
			g.PutTriggersOnStack()
			if timing == "stack-clone" {
				clone = g.Clone()
			}
			g.RunStepWithPriority(core.Cleanup)
			if len(g.DelayedTriggers()) != 0 {
				t.Fatal("cleanup retained subscription")
			}
			g.ResolveStack()
			if a.Life() != 20 || b.Life() != 28 {
				t.Fatalf("death facts: life %d/%d", a.Life(), b.Life())
			}
			if clone != nil {
				if clone.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatal("original changed clone")
				}
				if timing == "waiting-clone" {
					clone.PutPermanentIntoGraveyard(clone.FindPermanent(watched.ID()))
				}
				clone.PutTriggersOnStack()
				clone.ResolveStack()
				if clone.GetPlayer(b.PlayerID()).Life() != 28 {
					t.Fatal("clone lost facts")
				}
			}
		})
	}
}

func TestDelayedDeathCaptureDefinition(t *testing.T) {
	permanent, owner, controller := BindPermanent("dead"), BindPlayer("owner"), BindPlayer("controller")
	target := DeclarePermanentTarget("watched", IsCreature)
	body := GainLifeFor(FromPlayer(owner), Fixed(1))
	valid := WhenDiesThisTurnWithBindings(target, permanent, owner, controller, body)
	if err := ValidateCardDefinition("Valid", NewInstant("Valid", "", Spell(valid, WithTargets(target)))); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{
		WhenDiesThisTurnWithBindings(target, nil, owner, controller, body),
		WhenDiesThisTurnWithBindings(target, permanent, nil, controller, body),
		WhenDiesThisTurnWithBindings(target, permanent, owner, nil, body),
		WhenDiesThisTurnWithBindings(target, permanent, owner, owner, body),
		WhenDiesThisTurnWithBindings(nil, permanent, owner, controller, body),
		WhenDiesThisTurnWithBindings(target, permanent, owner, controller, nil),
		Sequence(valid, body),
		Sequence(valid, Destroy(From(permanent))),
	} {
		if ValidateCardDefinition("Invalid", NewInstant("Invalid", "", Spell(effect, WithTargets(target)))) == nil {
			t.Fatal("invalid death binding declaration accepted")
		}
	}
}

func TestDelayedDeathCaptureIdentityAndFrames(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	watchedCard := NewCreature("Watched", "", 2, 3)
	watchedCard.SetOwner(b.PlayerID())
	watched := g.PutOnBattlefield(watchedCard, a.PlayerID())
	exact := g.CaptureDamageSource(watched.ID())
	permanent, owner, controller := BindPermanent("dead"), BindPlayer("owner"), BindPlayer("controller")
	body := Sequence(GainLifeFor(FromPlayer(owner), ToughnessOfWithLKI(From(permanent))), Destroy(From(permanent)))
	if err := ApplyEffect(g, WhenDiesThisTurnWithBindings(This, permanent, owner, controller, body), watched.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	refs := g.WaitingDelayedPermanentSources()
	if len(refs) != 1 || !refs[0].SameObject(exact) {
		t.Fatal("waiting query lost watched visit")
	}
	g.ExilePermanent(source)
	g.PutPermanentIntoGraveyard(g.FindPermanent(watched.ID()))
	b.RemoveFromGraveyard(watchedCard.ID())
	g.PutOnBattlefield(watchedCard, a.PlayerID())
	g.PutTriggersOnStack()
	refs = g.StackBoundPermanentSources()
	if len(refs) != 1 || !refs[0].SameObject(exact) {
		t.Fatal("pending facts lost exact referred visit")
	}
	saved := g.resolution.playerBindings
	g.ResolveStack()
	if b.Life() != 23 || g.FindPermanent(watchedCard.ID()) == nil {
		t.Fatal("LKI or returned incarnation contract failed")
	}
	if len(g.resolution.playerBindings) != len(saved) {
		t.Fatal("death bindings leaked into outer frame")
	}
}
