package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDeathPowerOrToughnessFilter(t *testing.T) {
	for _, tc := range []struct {
		name      string
		power     int
		toughness int
		change    bool
		opponent  bool
		phase     bool
		want      int
	}{
		{"low power", 1, 4, false, false, false, 1},
		{"low toughness", 4, 1, false, false, false, 1},
		{"both high", 2, 2, false, false, false, 0},
		{"changed before death", 3, 4, true, false, false, 1},
		{"opposing low creature", 1, 4, false, true, false, 0},
		{"phased source", 1, 4, false, false, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(
				CreatureYouControlDies().PowerOrToughnessAtMost(1), GainLifeFor(You, Fixed(1)))))
			watcher.SetOwner(a.PlayerID())
			g.PutOnBattlefield(watcher, a.PlayerID())
			victim := NewCreature("Victim", "", tc.power, tc.toughness)
			victim.SetOwner(a.PlayerID())
			controller := a.PlayerID()
			if tc.opponent {
				controller = b.PlayerID()
			}
			g.PutOnBattlefield(victim, controller)
			if tc.change {
				if err := ApplyEffect(g, Until(core.EndOfTurn, ModifyPT(This, Fixed(-2), Fixed(0))), victim.ID(), controller, nil); err != nil {
					t.Fatal(err)
				}
			}
			if tc.phase {
				g.PhaseOut(watcher.ID())
			}
			g.DestroyPermanent(g.FindPermanent(victim.ID()))
			g.PutTriggersOnStack()
			if got := g.StackSize(); got != tc.want {
				t.Fatalf("triggers = %d, want %d", got, tc.want)
			}
			g.zones.ClearLKI()
			g.PutOnBattlefield(victim, controller)
			g.ResolveStack()
			if got := a.Life(); got != 20+tc.want {
				t.Fatalf("life = %d, want %d", got, 20+tc.want)
			}
		})
	}
}

func TestDeathPowerOrToughnessFilterBatchAndReplacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(
		CreatureYouControlDies().PowerOrToughnessAtMost(1), GainLifeFor(You, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	low := NewCreature("Low", "", 1, 3)
	low.SetOwner(a.PlayerID())
	high := NewCreature("High", "", 3, 3)
	high.SetOwner(a.PlayerID())
	replaced := NewCreature("Replaced", "", 1, 3)
	replaced.SetOwner(a.PlayerID())
	for _, c := range []Card{watcher, low, high, replaced} {
		g.PutOnBattlefield(c, a.PlayerID())
	}
	g.AddExileIfWouldGoToGraveyardThisTurn(replaced.ID(), watcher.ID())
	g.DestroyPermanents([]*Permanent{g.FindPermanent(watcher.ID()), g.FindPermanent(low.ID()), g.FindPermanent(high.ID()), g.FindPermanent(replaced.ID())})
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 1 {
		t.Fatalf("triggers = %d, want 1", got)
	}
	clone := g.Clone()
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	clone.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatalf("life = (%d, %d), want (21, 21)", a.Life(), b.Life())
	}
}

func TestDeathPowerOrToughnessFilterDeclaration(t *testing.T) {
	good := NewCreature("Good", "", 2, 2, WithAbility(WhenEvent(
		CreatureYouControlDies().PowerOrToughnessAtMost(1), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	if text := InspectDefinition(good.Abilities()[0]).Children[0].Text; !strings.Contains(text, "power or toughness 1 or less") {
		t.Fatalf("inspection = %q", text)
	}
	for _, pattern := range []EventPattern{
		CreatureDies().PowerOrToughnessAtMost(1).PowerOrToughnessAtMost(2),
		PermanentDies().PowerOrToughnessAtMost(1),
	} {
		bad := NewCreature("Bad", "", 2, 2, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("invalid characteristic filter accepted")
		}
	}
}
