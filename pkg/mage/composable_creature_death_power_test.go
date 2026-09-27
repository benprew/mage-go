package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCreatureDeathPowerCapturedBeforeDeparture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	power := BindNumber("former power")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureYouControlDies().WithPower(power), GainLifeFor(You, FromNumber(power)))))
	watcher.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 2, 3)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(victim, a.PlayerID())
	if err := ApplyEffect(g, Until(core.EndOfTurn, ModifyPT(This, Fixed(3), Fixed(0))), victim.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.DestroyPermanent(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack=%d, want 1", g.StackSize())
	}
	clone := g.Clone()
	g.zones.ClearLKI()
	g.PutOnBattlefield(victim, b.PlayerID())
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	clone.ResolveStack()
	if a.Life() != 25 || b.Life() != 25 || clone.GetPlayer(a.PlayerID()).Life() != 25 {
		t.Fatalf("life original=(%d,%d) clone=%d, want (25,25) and 25", a.Life(), b.Life(), clone.GetPlayer(a.PlayerID()).Life())
	}
}

func TestCreatureDeathPowerPhasedSourceDoesNotObserve(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	power := BindNumber("power")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		CreatureDies().WithPower(power), GainLifeFor(You, FromNumber(power)))))
	watcher.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 3, 3)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.PhaseOut(watcher.ID())
	g.DestroyPermanent(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 || a.Life() != 20 {
		t.Fatalf("stack=%d life=%d, want 0 and 20", g.StackSize(), a.Life())
	}
}

func TestCreatureDeathPowerSimultaneousAndAbsentSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	power := BindNumber("power")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureYouControlDies().WithPower(power), GainLifeFor(You, FromNumber(power)))))
	watcher.SetOwner(a.PlayerID())
	one := NewCreature("One", "", 2, 2)
	two := NewCreature("Two", "", 4, 4)
	one.SetOwner(a.PlayerID())
	two.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(one, a.PlayerID())
	g.PutOnBattlefield(two, a.PlayerID())
	g.DestroyPermanents([]*Permanent{g.FindPermanent(watcher.ID()), g.FindPermanent(one.ID()), g.FindPermanent(two.ID())})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack=%d, want 2", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 26 || b.Life() != 20 {
		t.Fatalf("life=(%d,%d), want (26,20)", a.Life(), b.Life())
	}
}

func TestCreatureDeathPowerDeclaration(t *testing.T) {
	power := BindNumber("power")
	for _, tc := range []struct {
		name string
		card Card
		want string
	}{
		{"valid", NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithPower(power), GainLifeFor(You, FromNumber(power))))), ""},
		{"nil", NewCreature("Nil", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithPower(nil), GainLifeFor(You, Fixed(1))))), "power binding"},
		{"empty", NewCreature("Empty", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithPower(BindNumber("")), GainLifeFor(You, Fixed(1))))), "binding name"},
		{"wrong event", NewCreature("Wrong", "", 1, 1, WithAbility(WhenEvent(PermanentDies().WithPower(BindNumber("power")), GainLifeFor(You, Fixed(1))))), "creature"},
		{"unbound", NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(CreatureDies(), GainLifeFor(You, FromNumber(BindNumber("power")))))), "producer"},
		{"duplicate", NewCreature("Duplicate", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithPower(BindNumber("a")).WithPower(BindNumber("b")), GainLifeFor(You, Fixed(1))))), "one power"},
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
