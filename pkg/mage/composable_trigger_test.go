package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestComposableInterveningIf(t *testing.T) {
	for _, present := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		artifact := NewArtifact("Artifact", "")
		artifact.SetOwner(a.PlayerID())
		g.PutOnBattlefield(artifact, a.PlayerID())
		source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(Enters(This), DrawFor(You, Fixed(1)), WithInterveningIf(Controls(IsArtifact)))))
		source.SetOwner(a.PlayerID())
		a.AddToLibrary(NewCreature("Drawn", "", 1, 1))
		g.PutOnBattlefield(source, a.PlayerID())
		g.PutTriggersOnStack()
		if !present {
			g.RemoveFromBattlefield(g.FindPermanent(artifact.ID()))
		}
		g.ResolveStack()
		if (len(a.Hand()) == 1) != present {
			t.Fatalf("intervening condition ignored: present=%v hand=%d", present, len(a.Hand()))
		}
	}
}

func TestComposableTriggerTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePlayerTarget("player")
	trigger := WhenEvent(Enters(This), DrawFor(target, Fixed(1)), WithTargets(target))
	card := NewCreature("Source", "", 1, 1, WithAbility(trigger))
	card.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition("Source", card); err != nil {
		t.Fatal(err)
	}
	b.AddToLibrary(NewCreature("Drawn", "", 1, 1))
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Controller: a.PlayerID(), IsAbility: true, TargetSource: card, Targets: []uuid.UUID{b.PlayerID()}, TargetSpecs: trigger.Targets(), Effects: trigger.Effects()}
	g.pushStack(obj)
	g.ResolveStackObject(obj)
	if len(b.Hand()) != 1 {
		t.Fatal("trigger lost player binding")
	}
}
