package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCreatureEntryKeywordFilter(t *testing.T) {
	for _, tc := range []struct {
		name         string
		flying       bool
		opponent     bool
		phased       bool
		sourceReturn bool
		want         int
	}{
		{"controlled flyer", true, false, false, false, 1},
		{"controlled ground creature", false, false, false, false, 0},
		{"opposing flyer", true, true, false, false, 0},
		{"phased watcher", true, false, true, false, 0},
		{"watcher leaves and returns", true, false, false, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureYouControlEnters().HasKeyword(core.Flying),
				GainLifeFor(You, Fixed(1)))))
			watcher.SetOwner(a.PlayerID())
			g.PutOnBattlefield(watcher, a.PlayerID())
			if tc.phased {
				g.PhaseOut(watcher.ID())
			}
			var options []CardOption
			if tc.flying {
				options = append(options, WithKeyword(core.Flying))
			}
			entered := NewCreature("Entered", "", 2, 2, options...)
			entered.SetOwner(b.PlayerID())
			controller := a.PlayerID()
			if tc.opponent {
				controller = b.PlayerID()
			}
			g.PutOnBattlefield(entered, controller)
			g.PutTriggersOnStack()
			if tc.sourceReturn {
				g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
				g.PutOnBattlefield(watcher, b.PlayerID())
			}
			g.RemoveFromBattlefield(g.FindPermanent(entered.ID()))
			g.PutOnBattlefield(entered, b.PlayerID())
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original life")
			}
			g.ResolveStack()
			if got := a.Life() - 20; got != tc.want {
				t.Fatalf("life gained=%d, want %d", got, tc.want)
			}
		})
	}
}

func TestCreatureEntryKeywordDeclaration(t *testing.T) {
	valid := NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureYouControlEnters().HasKeyword(core.Flying).Nontoken(),
		GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, pattern := range []creatureEntersPattern{
		AnotherCreatureYouControlEnters().HasKeyword(core.Attr(0)),
		AnotherCreatureYouControlEnters().HasKeyword(core.Flying).HasKeyword(core.Vigilance),
	} {
		invalid := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil {
			t.Fatal("accepted invalid entry keyword filter")
		}
	}
}

func TestCreatureEntryKeywordUsesEffectiveEntryState(t *testing.T) {
	for _, tc := range []struct {
		name    string
		printed bool
		remove  bool
		want    int
	}{
		{"static grant", false, false, 1},
		{"static removal", true, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureYouControlEnters().HasKeyword(core.Flying), GainLifeFor(You, Fixed(1)))))
			g.PutOnBattlefield(watcher, a.PlayerID())
			modifier := GrantEachKeyword(ControlledPermanents(IsCreature), core.Flying)
			if tc.remove {
				modifier = RemoveAbilitiesEach(MatchingPermanents(HasSubType("Bird")))
			}
			donor := NewEnchantment("Donor", "", WithAbility(Static(While(Controls(IsCreature), modifier))))
			g.PutOnBattlefield(donor, a.PlayerID())
			var options []CardOption
			if tc.printed {
				options = append(options, WithKeyword(core.Flying))
			}
			options = append(options, WithSubTypes("Bird"))
			g.PutOnBattlefield(NewCreature("Entered", "", 2, 2, options...), a.PlayerID())
			g.PutTriggersOnStack()
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original life")
			}
			g.ResolveStack()
			if got := a.Life() - 20; got != tc.want {
				t.Fatalf("life gained=%d, want %d", got, tc.want)
			}
		})
	}
}
