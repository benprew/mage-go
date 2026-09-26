package mage

import (
	"strings"
	"testing"
)

func TestOpponentLandBecomesTapped(t *testing.T) {
	for _, tc := range []struct {
		name, subtype                               string
		opp, phaseSource, phaseLand, retap, nonland bool
		want                                        int
	}{
		{"opposing Forest", "Forest", true, false, false, false, false, 1},
		{"own Forest", "Forest", false, false, false, false, false, 0},
		{"other subtype", "Mountain", true, false, false, false, false, 0},
		{"phased source", "Forest", true, true, false, false, false, 0},
		{"phased land", "Forest", true, false, true, false, false, 0},
		{"already tapped", "Forest", true, false, false, true, false, 1},
		{"nonland Forest", "Forest", true, false, false, false, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
				OpponentLandBecomesTapped().OfSubtype("Forest"), GainLifeFor(You, Fixed(1)))))
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(source, a.PlayerID())
			land := NewLand("Land", WithSubTypes(tc.subtype))
			if tc.nonland {
				land = NewCreature("Creature", "", 1, 1, WithSubTypes(tc.subtype))
			}
			controller := a.PlayerID()
			if tc.opp {
				controller = b.PlayerID()
			}
			g.PutOnBattlefield(land, controller)
			if tc.phaseSource {
				g.PhaseOut(source.ID())
			}
			if tc.phaseLand {
				g.PhaseOut(land.ID())
			}
			g.TapPermanent(g.FindPermanent(land.ID()))
			if tc.retap {
				g.TapPermanent(g.FindPermanent(land.ID()))
			}
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
			if got := a.Life(); got != 20+tc.want {
				t.Fatalf("life=%d, want %d", got, 20+tc.want)
			}
		})
	}
}

func TestOpponentLandBecomesTappedPendingIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OpponentLandBecomesTapped().OfSubtype("Forest"), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(source, a.PlayerID())
	land := NewLand("Land", WithSubTypes("Forest"))
	g.PutOnBattlefield(land, b.PlayerID())
	g.TapPermanent(g.FindPermanent(land.ID()))
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
	g.PutOnBattlefield(land, a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("pending stack=%d", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatalf("life A=%d B=%d", a.Life(), b.Life())
	}
}

func TestOpponentLandBecomesTappedDeclaration(t *testing.T) {
	valid := NewEnchantment("Valid", "", WithAbility(WhenEvent(
		OpponentLandBecomesTapped(), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(OpponentLandBecomesTapped().OfSubtype("Forest")); got.Role != "event-pattern" || got.Text != "opponent Forest becomes tapped" {
		t.Fatalf("inspection = %+v", got)
	}
	for _, subtype := range []string{"", "Goblin"} {
		card := NewEnchantment("Invalid", "", WithAbility(WhenEvent(
			OpponentLandBecomesTapped().OfSubtype(subtype), GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "subtype") {
			t.Fatalf("subtype %q: validation=%v", subtype, err)
		}
	}
	duplicate := NewEnchantment("Duplicate", "", WithAbility(WhenEvent(
		OpponentLandBecomesTapped().OfSubtype("Forest").OfSubtype("Mountain"), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil || !strings.Contains(err.Error(), "subtype") {
		t.Fatalf("duplicate subtype validation=%v", err)
	}
	spell := NewInstant("Invalid source", "", WithAbility(WhenEvent(
		OpponentLandBecomesTapped(), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil || !strings.Contains(err.Error(), "battlefield") {
		t.Fatalf("spell validation=%v", err)
	}
}
