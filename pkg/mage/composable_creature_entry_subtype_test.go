package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCreatureEntrySubtypeFilter(t *testing.T) {
	for _, tc := range []struct {
		name, subtype                string
		token, opponent, phaseSource bool
		want                         int
	}{
		{"dragon", "Dragon", false, false, false, 1},
		{"other subtype", "Goblin", false, false, false, 0},
		{"token dragon", "Dragon", true, false, false, 0},
		{"opposing dragon", "Dragon", false, true, false, 0},
		{"phased source", "Dragon", false, false, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureYouControlEnters().Nontoken().OfSubtype("Dragon"), GainLifeFor(You, Fixed(1)))))
			source.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(source, a.PlayerID())
			var other *BaseCard
			if tc.token {
				other = NewToken("Other", 1, 1, []core.CardType{core.TypeCreature}, []string{tc.subtype})
			} else {
				other = NewCreature("Other", "", 1, 1, WithSubTypes(tc.subtype))
			}
			other.SetOwner(a.PlayerID())
			if tc.phaseSource {
				g.PhaseOut(source.ID())
			}
			controller := a.PlayerID()
			if tc.opponent {
				controller = b.PlayerID()
			}
			g.PutOnBattlefield(other, controller)
			g.PutTriggersOnStack()
			if got := g.StackSize(); got != tc.want {
				t.Fatalf("stack=%d, want %d", got, tc.want)
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 {
				t.Fatal("clone changed original life")
			}
			g.ResolveStack()
			if a.Life() != 20+tc.want {
				t.Fatalf("life=%d, want %d", a.Life(), 20+tc.want)
			}
		})
	}
}

func TestCreatureEntrySubtypeDeclaration(t *testing.T) {
	good := NewCreature("Good", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureYouControlEnters().OfSubtype("Dragon"), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	for _, subtype := range []string{"", "Forest"} {
		bad := NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(
			AnotherCreatureYouControlEnters().OfSubtype(subtype), GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatalf("accepted invalid subtype %q", subtype)
		}
	}
}
