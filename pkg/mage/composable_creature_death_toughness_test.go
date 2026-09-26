package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCreatureDiesCapturesToughnessAtDetection(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	toughness := BindNumber("former toughness")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithToughness(toughness), GainLifeFor(You, FromNumber(toughness)))))
	watcher.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 2, 3)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.DestroyPermanent(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack=%d, want 1", g.StackSize())
	}
	g.zones.ClearLKI()
	g.PutOnBattlefield(victim, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 23 {
		t.Fatalf("life=%d, want 23", a.Life())
	}
}

func TestCreatureDiesToughnessValidation(t *testing.T) {
	for _, tc := range []struct {
		name string
		card Card
		want string
	}{
		{"nil", NewCreature("Nil", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithToughness(nil), GainLifeFor(You, Fixed(1))))), "binding"},
		{"missing producer", func() Card {
			b := BindNumber("toughness")
			return NewCreature("Missing", "", 1, 1, WithAbility(WhenEvent(CreatureDies(), GainLifeFor(You, FromNumber(b)))))
		}(), "producer"},
		{"duplicate name", func() Card {
			a, b := BindNumber("same"), BindNumber("same")
			return NewCreature("Duplicate", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithToughness(a), Sequence(SnapshotNumber(Fixed(1), b), GainLifeFor(You, FromNumber(a))))))
		}(), "duplicate binding name"},
		{"non dominating", func() Card {
			b := BindNumber("toughness")
			return NewCreature("Non Dominating", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithToughness(b), GainLifeFor(You, FromNumber(b)))), WithAbility(Activate(Costs(), GainLifeFor(You, FromNumber(b)))))
		}(), "producer"},
		{"valid", func() Card {
			b := BindNumber("toughness")
			return NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithToughness(b), GainLifeFor(You, FromNumber(b)))))
		}(), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCardDefinition(tc.card.Name(), tc.card)
			if tc.want == "" && err == nil {
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validation=%v, want %q", err, tc.want)
			}
		})
	}
}

func TestCreatureDiesCapturesOwnToughness(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	n := BindNumber("former toughness")
	creature := NewCreature("Self", "", 2, 6, WithAbility(WhenEvent(CreatureDies().WithToughness(n), GainLifeFor(You, FromNumber(n)))))
	creature.SetOwner(a.PlayerID())
	g.PutOnBattlefield(creature, a.PlayerID())
	g.DestroyPermanent(g.FindPermanent(creature.ID()))
	g.PutTriggersOnStack()
	g.zones.ClearLKI()
	g.ResolveStack()
	if a.Life() != 26 {
		t.Fatalf("life=%d, want 26", a.Life())
	}
}

func TestCreatureDiesToughnessIgnoresNonDeaths(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	n := BindNumber("toughness")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithToughness(n), GainLifeFor(You, FromNumber(n)))))
	watcher.SetOwner(a.PlayerID())
	land := NewLand("Land")
	land.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(land, b.PlayerID())
	g.DestroyPermanent(g.FindPermanent(land.ID()))
	g.PhaseOut(watcher.ID())
	g.FireEvent(core.GameEvent{Type: core.EvtZoneChange, SourceID: land.ID(), FromZone: core.ZoneBattlefield, ToZone: core.ZoneGraveyard})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatalf("stack=%d, want 0", g.StackSize())
	}
}

func TestCreatureDiesToughnessSimultaneousAndCopies(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	n := BindNumber("former toughness")
	former := BindPlayer("former controller")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureYouControlDies().WithController(former).WithToughness(n),
		GainLifeFor(FromPlayer(former), FromNumber(n)))))
	watcher.SetOwner(a.PlayerID())
	first := NewCreature("First", "", 2, 2)
	first.SetOwner(a.PlayerID())
	second := NewCreature("Second", "", 3, 4)
	second.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	if err := ApplyEffect(g, Until(core.EndOfTurn, ModifyPT(This, Fixed(0), Fixed(3))), first.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.DestroyPermanents([]*Permanent{g.FindPermanent(watcher.ID()), g.FindPermanent(first.ID()), g.FindPermanent(second.ID())})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack=%d, want 2", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 29 || a.Life() != 20 {
		t.Fatalf("clone life=%d, original life=%d; want 29 and 20", clone.GetPlayer(a.PlayerID()).Life(), a.Life())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.zones.ClearLKI()
	g.ResolveStack()
	if a.Life() != 34 || b.Life() != 20 {
		t.Fatalf("life=(%d,%d), want (34,20)", a.Life(), b.Life())
	}
}
