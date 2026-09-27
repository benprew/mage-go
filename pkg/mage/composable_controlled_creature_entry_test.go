package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestControlledCreatureEntryIncludesSourceAndOtherCreatures(t *testing.T) {
	for _, tc := range []struct {
		name     string
		flying   bool
		other    bool
		opponent bool
		phased   bool
		want     int
	}{
		{"source", true, false, false, false, 3},
		{"source without keyword", false, false, false, false, 0},
		{"other controlled", true, true, false, false, 2},
		{"opposing", true, true, true, false, 0},
		{"phased source", true, true, false, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			power := BindNumber("entry power")
			watcher := NewCreature("Watcher", "", 3, 3, WithAbility(WhenEvent(
				CreatureYouControlEnters().HasKeyword(core.Flying).WithPower(power),
				GainLifeFor(You, FromNumber(power)))))
			if tc.flying {
				watcher = NewCreature("Watcher", "", 3, 3, WithKeyword(core.Flying), WithAbility(WhenEvent(
					CreatureYouControlEnters().HasKeyword(core.Flying).WithPower(power),
					GainLifeFor(You, FromNumber(power)))))
			}
			g.PutOnBattlefield(watcher, a.PlayerID())
			if tc.other {
				g.PutTriggersOnStack()
				g.ResolveStack()
				if tc.phased {
					g.PhaseOut(watcher.ID())
				}
				controller := a.PlayerID()
				if tc.opponent {
					controller = b.PlayerID()
				}
				g.PutOnBattlefield(NewCreature("Other", "", 2, 2, WithKeyword(core.Flying)), controller)
			}
			baseline := a.Life()
			g.PutTriggersOnStack()
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != baseline || b.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			if got := a.Life() - baseline; got != tc.want {
				t.Fatalf("life gained=%d, want %d", got, tc.want)
			}
		})
	}
}

func TestControlledCreatureEntryDeclaration(t *testing.T) {
	power := BindNumber("power")
	valid := NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(
		CreatureYouControlEnters().WithPower(power), GainLifeFor(You, FromNumber(power)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(valid.Abilities()[0]); got.Role != "event-trigger" {
		t.Fatalf("inspection: %+v", got)
	}
	invalid := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(
		CreatureYouControlEnters().WithPower(nil), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil {
		t.Fatal("accepted missing capture")
	}
}

func TestControlledCreatureEntryKeepsQueuedSourceIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		CreatureYouControlEnters(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutTriggersOnStack()
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatalf("queued trigger lost original controller: %d, %d", a.Life(), b.Life())
	}
}
