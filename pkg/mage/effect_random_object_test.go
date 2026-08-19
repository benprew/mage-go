package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestApplyToRandomPermanentUsesFilter(t *testing.T) {
	g, a, _ := randomTestGame()
	first := addRandomTargetCreature(g, a, "First")
	artifactCard := NewArtifact("Artifact", "{1}")
	artifactCard.SetOwner(a.PlayerID())
	artifact := g.PutOnBattlefield(artifactCard, a.PlayerID())
	second := addRandomTargetCreature(g, a, "Second")
	g.SetRandomResults([]int{1})

	if err := ApplyEffect(g, ApplyToRandomPermanent(Tap(), IsCreature), first.ID(), a.PlayerID(), nil); err != nil {
		t.Fatalf("apply random permanent effect: %v", err)
	}
	if first.Tapped || artifact.Tapped || !second.Tapped {
		t.Fatalf("tapped states = first %v, artifact %v, second %v", first.Tapped, artifact.Tapped, second.Tapped)
	}
}

func TestApplyToRandomSpellOrPermanent(t *testing.T) {
	g, a, _ := randomTestGame()
	permanent := addRandomTargetCreature(g, a, "Permanent")
	g.SetRandomResults([]int{0})

	if err := ApplyEffect(g, ApplyToRandomSpellOrPermanent(ChangeColorEffect(Blue)), permanent.ID(), a.PlayerID(), nil); err != nil {
		t.Fatalf("apply random object effect: %v", err)
	}
	if got := permanent.Colors(); len(got) != 1 || got[0] != Blue {
		t.Fatalf("permanent colors = %v, want [blue]", got)
	}
}

func TestApplyToRandomPlayer(t *testing.T) {
	g, a, b := randomTestGame()
	g.SetRandomResults([]int{1})

	if err := ApplyEffect(g, ApplyToRandomPlayer(GainLifeTarget(Fixed(2))), a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatalf("apply random player effect: %v", err)
	}
	if a.Life() != a.StartingLife() || b.Life() != b.StartingLife()+2 {
		t.Fatalf("life totals = A %d, B %d", a.Life(), b.Life())
	}
}

func TestApplyToRandomDamageTarget(t *testing.T) {
	g, a, _ := randomTestGame()
	creature := addRandomTargetCreature(g, a, "Creature")
	g.SetRandomResults([]int{0})

	if err := ApplyEffect(g, ApplyToRandomDamageTarget(DealDamage(Fixed(1))), creature.ID(), a.PlayerID(), nil); err != nil {
		t.Fatalf("apply random damage-target effect: %v", err)
	}
	if creature.Damage != 1 {
		t.Fatalf("creature damage = %d, want 1", creature.Damage)
	}
}

func TestRandomSourceColorEffects(t *testing.T) {
	g, a, _ := randomTestGame()
	sourceCard := NewCreature("Source", "{G}", 1, 1)
	sourceCard.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(sourceCard, a.PlayerID())
	g.SetRandomResults([]int{1, 3})

	if err := ApplyEffect(g, SetSourceChosenColorAtRandom(), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatalf("set chosen color: %v", err)
	}
	if source.ChosenColor != Blue {
		t.Fatalf("chosen color = %v, want blue", source.ChosenColor)
	}
	if err := ApplyEffect(g, ChangeSourceToRandomColor(), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatalf("change source color: %v", err)
	}
	if got := source.Colors(); len(got) != 1 || got[0] != Red {
		t.Fatalf("source colors = %v, want [red]", got)
	}
}
