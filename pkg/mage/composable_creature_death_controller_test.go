package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCreatureDiesCapturesFormerController(t *testing.T) {
	for _, tc := range []struct {
		name        string
		pattern     func(*PlayerBinding) EventPattern
		ownerB      bool
		controllerB bool
		want        int
	}{
		{"any creature", func(p *PlayerBinding) EventPattern { return CreatureDies().WithController(p) }, true, false, 1},
		{"controlled creature", func(p *PlayerBinding) EventPattern { return CreatureYouControlDies().WithController(p) }, true, false, 1},
		{"another controlled creature", func(p *PlayerBinding) EventPattern { return AnotherCreatureYouControlDies().WithController(p) }, true, false, 1},
		{"opponent creature", func(p *PlayerBinding) EventPattern { return CreatureYouControlDies().WithController(p) }, false, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			former := BindPlayer("former controller")
			watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(tc.pattern(former), GainLifeFor(FromPlayer(former), Fixed(1)))))
			watcher.SetOwner(a.PlayerID())
			victim := NewCreature("Victim", "", 1, 1)
			owner := a.PlayerID()
			if tc.ownerB {
				owner = b.PlayerID()
			}
			victim.SetOwner(owner)
			if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(watcher, a.PlayerID())
			g.PutOnBattlefield(victim, owner)
			if tc.controllerB != tc.ownerB {
				controller := a.PlayerID()
				if tc.controllerB {
					controller = b.PlayerID()
				}
				g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: victim.ID(), ControllerID: controller, Duration: core.Indefinite})
			}
			g.DestroyPermanent(g.FindPermanent(victim.ID()))
			g.PutTriggersOnStack()
			if g.StackSize() != tc.want {
				t.Fatalf("stack=%d, want %d", g.StackSize(), tc.want)
			}
			if tc.want == 1 {
				clone := g.Clone()
				clone.ResolveStack()
				if a.Life() != 20 || b.Life() != 20 {
					t.Fatal("clone changed original life")
				}
			}
			g.ResolveStack()
			wantA, wantB := 20, 20
			if tc.want == 1 {
				if tc.controllerB {
					wantB++
				} else {
					wantA++
				}
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("life=(%d,%d), want (%d,%d)", a.Life(), b.Life(), wantA, wantB)
			}
		})
	}
}

func TestCreatureDiesControllerCopyAndSimultaneousDeath(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	former := BindPlayer("former controller")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(AnotherCreatureYouControlDies().WithController(former), GainLifeFor(FromPlayer(former), Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 1, 1)
	victim.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(victim, a.PlayerID())
	g.DestroyPermanents([]*Permanent{g.FindPermanent(watcher.ID()), g.FindPermanent(victim.ID())})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack=%d, want 1", g.StackSize())
	}
	copied := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 20 {
		t.Fatalf("copy changed event controller: life=(%d,%d)", a.Life(), b.Life())
	}
}

func TestCreatureDiesControllerSeparateSimultaneousEvents(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	former := BindPlayer("former controller")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithController(former), GainLifeFor(FromPlayer(former), Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	first := NewCreature("First", "", 1, 1)
	first.SetOwner(a.PlayerID())
	second := NewCreature("Second", "", 1, 1)
	second.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.DestroyPermanents([]*Permanent{g.FindPermanent(first.ID()), g.FindPermanent(second.ID())})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack=%d, want 2", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatalf("event controller leaked between triggers: life=(%d,%d)", a.Life(), b.Life())
	}
}

func TestCreatureDiesControllerPhasedSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	former := BindPlayer("former controller")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithController(former), GainLifeFor(FromPlayer(former), Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 1, 1)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.PhaseOut(watcher.ID())
	g.DestroyPermanent(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source observed a death")
	}
}

func TestCreatureDiesControllerValidation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		build func() Card
		want  string
	}{
		{"valid", func() Card {
			b := BindPlayer("controller")
			return NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithController(b), GainLifeFor(FromPlayer(b), Fixed(1)))))
		}, ""},
		{"nil", func() Card {
			return NewCreature("Nil", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithController(nil), GainLifeFor(You, Fixed(1)))))
		}, "binding"},
		{"undeclared read", func() Card {
			b := BindPlayer("controller")
			return NewCreature("Read", "", 1, 1, WithAbility(WhenEvent(CreatureDies(), GainLifeFor(FromPlayer(b), Fixed(1)))))
		}, "producer"},
		{"duplicate name", func() Card {
			a := BindPlayer("same")
			b := BindPlayer("same")
			return NewCreature("Duplicate", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithController(a), Sequence(SnapshotPlayer(You, b), GainLifeFor(FromPlayer(a), Fixed(1))))))
		}, "duplicate binding name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := tc.build()
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validation=%v, want %q", err, tc.want)
			}
		})
	}
}
