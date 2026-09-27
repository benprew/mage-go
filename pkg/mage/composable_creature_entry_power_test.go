package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCreatureEntryPowerCapture(t *testing.T) {
	for _, tc := range []struct {
		name              string
		opponent, phased  bool
		depart, copyStack bool
		want              int
	}{
		{"controlled", false, false, false, false, 3},
		{"opponent", true, false, false, false, 0},
		{"phased source", false, true, false, false, 0},
		{"departed and returned", false, false, true, false, 3},
		{"stack copy", false, false, false, true, 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			power := BindNumber("entry power")
			watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureYouControlEnters().WithPower(power), GainLifeFor(You, FromNumber(power)))))
			watcher.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(watcher, a.PlayerID())
			if tc.phased {
				g.PhaseOut(watcher.ID())
			}
			creature := NewCreature("Entered", "", 3, 3)
			creature.SetOwner(b.PlayerID())
			controller := a.PlayerID()
			if tc.opponent {
				controller = b.PlayerID()
			}
			g.PutOnBattlefield(creature, controller)
			g.PutTriggersOnStack()
			if !tc.opponent && !tc.phased {
				g.FindPermanent(creature.ID()).AddCounter(core.P1P1, 2)
			}
			if tc.depart {
				g.RemoveFromBattlefield(g.FindPermanent(creature.ID()))
				g.PutOnBattlefield(creature, b.PlayerID())
			}
			if tc.copyStack && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("stack copy failed")
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			if got := a.Life() + b.Life() - 40; got != tc.want {
				t.Fatalf("life gained=%d, want %d", got, tc.want)
			}
		})
	}
}

func TestCreatureEntryPowerDeclaration(t *testing.T) {
	power := BindNumber("power")
	valid := NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureYouControlEnters().WithPower(power), AddCountersTo(This, core.P1P1, FromNumber(power)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(valid.Abilities()[0]); got.Role != "event-trigger" || len(got.Children) == 0 {
		t.Fatalf("inspection: %+v", got)
	}
	for _, pattern := range []creatureEntersPattern{
		AnotherCreatureYouControlEnters().WithPower(nil),
		AnotherCreatureYouControlEnters().WithPower(BindNumber("")),
		AnotherCreatureYouControlEnters().WithPower(power).WithPower(power),
	} {
		invalid := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil {
			t.Fatal("accepted invalid power capture")
		}
	}
	unbound := NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureYouControlEnters(), GainLifeFor(You, FromNumber(power)))))
	if err := ValidateCardDefinition(unbound.Name(), unbound); err == nil {
		t.Fatal("accepted unbound power")
	}
}
