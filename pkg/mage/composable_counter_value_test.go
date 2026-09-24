package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCounterValueResolutionAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Source", "", 2, 3)
	g.PutOnBattlefield(c, a.PlayerID())
	g.MutablePermanent(c.ID()).AddCounter(core.Charge, 3)
	g.MutablePermanent(c.ID()).AddCounter(core.Tide, 7)
	saved := BindPermanent("saved")
	body := Sequence(Snapshot(This, saved), AddCountersTo(This, core.Charge, Fixed(2)), GainLifeFor(You, CountersOn(From(saved), core.Charge)), RemoveCountersFrom(This, core.Charge, CountersOn(This, core.Charge)))
	clone := g.Clone()
	if err := ApplyEffect(clone, body, c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 25 || a.Life() != 20 {
		t.Fatal("counter value did not read resolution state or changed original")
	}
	if clone.FindPermanent(c.ID()).Counters[core.Charge] != 0 || g.FindPermanent(c.ID()).Counters[core.Charge] != 3 || clone.FindPermanent(c.ID()).Counters[core.Tide] != 7 {
		t.Fatal("counter kind or clone isolation changed")
	}
	ctx := &EffectContext{Game: g, SourceID: c.ID(), Controller: a.PlayerID()}
	if got := CountersOn(This, core.Spore).resolveValue(ctx); got != 0 {
		t.Fatalf("missing kind = %d", got)
	}
}

func TestCounterValueIdentity(t *testing.T) {
	for _, scenario := range []string{"present", "absent", "returned", "phased", "phase cycle", "control"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Subject", "", 2, 3)
			g.PutOnBattlefield(c, a.PlayerID())
			g.MutablePermanent(c.ID()).AddCounter(core.Charge, 4)
			saved := BindPermanent("saved")
			ctx := &EffectContext{Game: g, SourceID: c.ID(), Controller: a.PlayerID()}
			if err := Snapshot(This, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			want := 4
			switch scenario {
			case "absent", "returned":
				g.ExilePermanent(g.FindPermanent(c.ID()))
				want = 0
				if scenario == "returned" {
					g.RemoveFromExile(c.ID())
					g.PutOnBattlefield(c, a.PlayerID())
					g.MutablePermanent(c.ID()).AddCounter(core.Charge, 9)
				}
			case "phased":
				g.PhaseOut(c.ID())
				want = 0
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(c.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: c.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				g.ApplyContinuousEffects()
			}
			if got := CountersOn(From(saved), core.Charge).resolveValue(ctx); got != want {
				t.Fatalf("count = %d, want %d", got, want)
			}
		})
	}
}

func TestCounterValueCopyAndTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Subject", "", 2, 3)
	g.PutOnBattlefield(c, a.PlayerID())
	g.MutablePermanent(c.ID()).AddCounter(core.Charge, 3)
	target := DeclarePermanentTarget("subject", IsCreature)
	ability := Spell(GainLifeFor(You, CountersOn(target, core.Charge)), WithTargets(target))
	card := NewInstant("Read counters", "", WithAbility(ability))
	original := &StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{c.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(original)
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy missing")
	}
	g.MutablePermanent(c.ID()).AddCounter(core.Charge, 2)
	g.ResolveStackObject(copied)
	if b.Life() != 25 || a.Life() != 20 {
		t.Fatal("copy used wrong amount or controller")
	}
	g.ExilePermanent(g.FindPermanent(c.ID()))
	g.RemoveFromExile(c.ID())
	g.PutOnBattlefield(c, a.PlayerID())
	g.MutablePermanent(c.ID()).AddCounter(core.Charge, 9)
	g.ResolveStackObject(original)
	if a.Life() != 20 {
		t.Fatal("read returned target")
	}
}

func TestCounterValueValidation(t *testing.T) {
	for _, subject := range []PermanentRef{nil, From(BindPermanent("missing")), DeclarePermanentTarget("undeclared", IsCreature)} {
		c := NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), GainLifeFor(You, CountersOn(subject, core.Charge)))))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("invalid subject accepted")
		}
	}
	for _, kind := range []core.CounterType{-1, core.NumCounters} {
		t.Run(kind.String(), func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("invalid counter accepted")
				}
			}()
			CountersOn(This, kind)
		})
	}
	info := InspectDefinition(CountersOn(This, core.Charge))
	if info.Role != "counter-value" || info.Text != "Charge counters on subject" || len(info.Children) != 1 {
		t.Fatalf("metadata = %+v", info)
	}
}

func TestCounterValueContinuousTiming(t *testing.T) {
	for _, dynamic := range []bool{false, true} {
		name := "temporary"
		if dynamic {
			name = "static"
		}
		t.Run(name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			modification := ModifyPT(This, CountersOn(This, core.Charge), Fixed(0))
			var opts []CardOption
			if dynamic {
				opts = append(opts, WithAbility(Static(While(Controls(IsCreature), modification))))
			}
			c := NewCreature("Subject", "", 1, 1, opts...)
			g.PutOnBattlefield(c, a.PlayerID())
			g.MutablePermanent(c.ID()).AddCounter(core.Charge, 2)
			if !dynamic {
				if err := ApplyEffect(g, Until(core.EndOfTurn, modification), c.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			g.ApplyContinuousEffects()
			if got := g.FindPermanent(c.ID()).CurrentPower(g); got != 3 {
				t.Fatalf("initial power = %d, want 3", got)
			}
			g.MutablePermanent(c.ID()).AddCounter(core.Charge, 4)
			for range 3 {
				g.ApplyContinuousEffects()
			}
			want := 3
			if dynamic {
				want = 7
			}
			if got := g.FindPermanent(c.ID()).CurrentPower(g); got != want {
				t.Fatalf("later power = %d, want %d", got, want)
			}
		})
	}
}
