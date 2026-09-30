package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestRandomTargetPublicInstanceIsolation(t *testing.T) {
	a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
	g := mage.NewGame(a, b)
	targets := dsl.TargetRandom(dsl.TargetCreature())
	source := dsl.NewArtifact("Synthetic random source", "", dsl.WithActivatedAbility(
		dsl.GainLife(dsl.You, dsl.Fixed(1)), dsl.GenericCost(0), dsl.WithTarget(targets)))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	for _, name := range []string{"First", "Second"} {
		card := dsl.NewCreature(name, "", 1, 1)
		card.SetOwner(b.PlayerID())
		g.PutOnBattlefield(card, b.PlayerID())
	}
	g.SetRandomResults([]int{0, 1})
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	first := g.Stack().Peek()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if first.Targets[0] == g.Stack().Peek().Targets[0] || targets.IsChosen() {
		t.Fatal("announcements did not keep independent choices")
	}
	g.ResolveTopOfStack()
	g.ResolveTopOfStack()
	if a.Life() != 22 {
		t.Fatal("both random-target instances must resolve")
	}
}
