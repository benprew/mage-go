package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAnotherCreatureEntersAnyController(t *testing.T) {
	for _, tc := range []struct {
		name       string
		controller int
		phase      bool
		want       int
	}{
		{"same controller", 0, false, 3},
		{"other controller", 1, false, 3},
		{"phased watcher", 1, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			power := BindNumber("entered power")
			watcher := NewCreature("Watcher", "", 1, 4, WithAbility(WhenEvent(
				AnotherCreatureEnters().WithPower(power), AddCountersTo(This, core.P1P1, FromNumber(power)))))
			watcher.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(watcher, a.PlayerID())
			if tc.phase {
				g.PhaseOut(watcher.ID())
			}
			entered := NewCreature("Entered", "", 3, 3)
			entered.SetOwner(a.PlayerID())
			controller := a.PlayerID()
			if tc.controller == 1 {
				controller = b.PlayerID()
			}
			g.PutOnBattlefield(entered, controller)
			g.PutTriggersOnStack()
			g.FindPermanent(entered.ID()).AddCounter(core.P1P1, 2)
			clone := g.Clone()
			clone.ResolveStack()
			if got := g.FindPermanentIncludingPhased(watcher.ID()).Counters[core.P1P1]; got != 0 {
				t.Fatalf("clone changed original counters=%d", got)
			}
			g.ResolveStack()
			if got := g.FindPermanentIncludingPhased(watcher.ID()).Counters[core.P1P1]; int(got) != tc.want {
				t.Fatalf("counters=%d, want %d", got, tc.want)
			}
		})
	}
}

func TestAnotherCreatureEntersDeclaration(t *testing.T) {
	power := BindNumber("power")
	valid := NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureEnters().WithPower(power), GainLifeFor(You, FromNumber(power)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	invalid := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureEnters().WithPower(power).WithPower(power), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil {
		t.Fatal("accepted duplicate capture")
	}
}

func TestAnotherCreatureEntersExcludesSelfAndNoncreature(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureEnters(), GainLifeFor(You, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	artifact := NewArtifact("Artifact", "")
	artifact.SetOwner(b.PlayerID())
	g.PutOnBattlefield(artifact, b.PlayerID())
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 0 {
		t.Fatalf("unexpected triggers=%d", got)
	}
}
