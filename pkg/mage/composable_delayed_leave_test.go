package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestWhenLeavesThisTurnDestinationsAndIncarnation(t *testing.T) {
	for _, destination := range []string{"graveyard", "hand", "exile"} {
		t.Run(destination, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := g.PutOnBattlefield(NewArtifact("Watcher", ""), a.PlayerID())
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 2), b.PlayerID())
			bound := BindPermanent("victim")
			target := DeclarePermanentTarget("target", IsCreature)
			g.resolution.targetSpecs = []Target{target.targetDeclaration}
			effect := Sequence(Snapshot(target, bound), WhenLeavesThisTurn(This, DestroyWithoutRegeneration(From(bound))))
			if err := ApplyEffect(g, effect, watcher.ID(), a.PlayerID(), []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			phased := g.PhaseOut(watcher.ID())
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("phasing queued the delayed trigger")
			}
			g.PhaseIn(phased)
			clone := g.Clone()
			switch destination {
			case "graveyard":
				g.PutPermanentIntoGraveyard(g.FindPermanent(watcher.ID()))
			case "hand":
				g.BouncePermanentToHand(g.FindPermanent(watcher.ID()))
			case "exile":
				g.ExilePermanent(g.FindPermanent(watcher.ID()))
			}
			g.PutOnBattlefield(watcher.Card, b.PlayerID())
			g.PutTriggersOnStack()
			g.ResolveStack()
			if g.FindPermanent(victim.ID()) != nil {
				t.Fatal("delayed body did not act on the saved creature")
			}
			if len(g.DelayedTriggers()) != 1 {
				t.Fatal("turn-scoped subscription was removed before cleanup")
			}
			g.PutPermanentIntoGraveyard(g.FindPermanent(watcher.ID()))
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("subscription followed the returned artifact")
			}
			clone.PutPermanentIntoGraveyard(clone.FindPermanent(watcher.ID()))
			clone.PutTriggersOnStack()
			if clone.StackSize() != 1 || clone.FindPermanent(victim.ID()) == nil {
				t.Fatal("clone did not retain an independent pending trigger")
			}
			clone.ResolveStack()
			if clone.FindPermanent(victim.ID()) != nil {
				t.Fatal("cloned delayed body lost its binding")
			}
		})
	}
}

func TestWhenLeavesThisTurnValidationAndExpiry(t *testing.T) {
	saved := BindPermanent("saved")
	valid := NewArtifact("Valid", "", WithAbility(Activate(Costs(), Sequence(Snapshot(This, saved), WhenLeavesThisTurn(From(saved), GainLifeFor(You, Fixed(1)))))))
	if err := ValidateCardDefinition("Valid", valid); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{WhenLeavesThisTurn(From(saved), GainLifeFor(You, Fixed(1))), WhenLeavesThisTurn(nil, GainLifeFor(You, Fixed(1))), WhenLeavesThisTurn(This, nil)} {
		if err := ValidateCardDefinition("Invalid", NewInstant("Invalid", "", Spell(effect))); err == nil || !strings.Contains(err.Error(), "producer") && !strings.Contains(err.Error(), "delayed") {
			t.Fatalf("unexpected validation result: %v", err)
		}
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := g.PutOnBattlefield(NewArtifact("Watcher", ""), a.PlayerID())
	if err := ApplyEffect(g, WhenLeavesThisTurn(This, GainLifeFor(You, Fixed(1))), watcher.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.triggers.ClearEndOfTurn()
	g.ExilePermanent(g.FindPermanent(watcher.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("expired trigger observed a later departure")
	}
}

func TestWhenLeavesThisTurnSimultaneousAndAbsent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	watcher := g.PutOnBattlefield(NewCreature("Watcher", "", 2, 2), b.PlayerID())
	target := DeclarePermanentTarget("watcher", IsCreature)
	g.resolution.targetSpecs = []Target{target.targetDeclaration}
	if err := ApplyEffect(g, WhenLeavesThisTurn(target, GainLifeFor(You, Fixed(2))), source.ID(), a.PlayerID(), []uuid.UUID{watcher.ID()}); err != nil {
		t.Fatal(err)
	}
	g.DestroyPermanents([]*Permanent{source, watcher})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("simultaneous source and watched departure lost the delayed trigger")
	}
	returned := g.PutOnBattlefield(watcher.Card, b.PlayerID())
	g.PhaseOut(returned.ID())
	if err := ApplyEffect(g, WhenLeavesThisTurn(target, GainLifeFor(You, Fixed(2))), source.ID(), a.PlayerID(), []uuid.UUID{returned.ID()}); err != nil {
		t.Fatal(err)
	}
	if len(g.DelayedTriggers()) != 1 {
		t.Fatal("absent target created a subscription")
	}
}

func TestWhenLeavesThisTurnDestinationReplacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := g.PutOnBattlefield(NewArtifact("Watcher", ""), a.PlayerID())
	if err := ApplyEffect(g, WhenLeavesThisTurn(This, GainLifeFor(You, Fixed(2))), watcher.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.AddExileIfWouldGoToGraveyardThisTurn(watcher.ID(), watcher.ID())
	g.PutPermanentIntoGraveyard(g.FindPermanent(watcher.ID()))
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 22 || g.FindPermanent(watcher.ID()) != nil {
		t.Fatal("replaced departure did not trigger")
	}
}
