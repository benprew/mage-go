package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSourceRelativeTargetQuery(t *testing.T) {
	for _, toughness := range []bool{false, true} {
		for _, state := range []string{"boundary", "source counter", "target counter", "depart", "return", "phase", "phase cycle", "target return", "target phase", "control", "copy", "clone"} {
			t.Run(sourceRelativeCaseName(toughness, state), func(t *testing.T) {
				g, a, b := newSpellCopyGame()
				option := TargetPowerLessThanOrEqualToSourcePower()
				power, defense := 3, 8
				if toughness {
					option = TargetToughnessLessThanSourcePower()
					power, defense = 8, 2
				}
				binding := DeclarePermanentTarget("subject", IsCreature, option)
				card := NewCreature("Source", "", 3, 8, WithAbility(Activate(Costs(), Destroy(binding), WithTargets(binding))))
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				victimCard := NewCreature("Subject", "", power, defense)
				victimCard.SetOwner(b.PlayerID())
				victim := g.PutOnBattlefield(victimCard, b.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
					t.Fatal(err)
				}
				obj := g.StackPeek()
				plan, ok := InspectPermanentTarget(binding)
				if !ok {
					t.Fatal("source-relative target is opaque")
				}
				legal := true
				switch state {
				case "source counter":
					g.FindPermanent(source.ID()).AddCounter(core.M1M1, 1)
					legal = false
				case "target counter":
					g.FindPermanent(victim.ID()).AddCounter(core.P1P1, 1)
					legal = false
				case "depart", "return":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					if state == "return" {
						g.PutOnBattlefield(card, a.PlayerID()).AddCounter(core.M1M1, 2)
					}
				case "phase":
					g.PhaseOut(source.ID())
				case "phase cycle":
					g.PhaseIn(g.PhaseOut(source.ID()))
				case "target return":
					g.RemoveFromBattlefield(victim)
					victim = g.PutOnBattlefield(victimCard, b.PlayerID())
					legal = false
				case "target phase":
					g.PhaseOut(victim.ID())
					legal = false
				case "control":
					g.AddControlEffect(ControlEffectSpec{TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "copy":
					obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
				case "clone":
					clone := g.Clone()
					clone.MutablePermanent(source.ID()).AddCounter(core.M1M1, 1)
					if got, known := plan.Select(clone, clone.StackPeek()); !known || len(got) != 0 {
						t.Fatal("clone source power ignored")
					}
				}
				g.ApplyContinuousEffects()
				got, known := plan.Select(g, obj)
				if !known || (len(got) == 1) != legal {
					t.Fatalf("selection %v/%v, want legal %v", got, known, legal)
				}
				native := binding.Possible(obj.Controller, g.resolutionTargetSource(obj), g)
				nativeLegal := slices.Contains(native, victim.ID()) && state != "target return"
				if nativeLegal != legal {
					t.Fatal("native candidates differ from exact target query")
				}
				g.ResolveStack()
				if (g.FindPermanentIncludingPhased(victim.ID()) == nil) != legal {
					t.Fatal("resolution differs from query")
				}
			})
		}
	}
}

func sourceRelativeCaseName(toughness bool, state string) string {
	if toughness {
		return "toughness/" + state
	}
	return "power/" + state
}

func TestSourceRelativeTargetQueryUnknown(t *testing.T) {
	for _, option := range []PermanentTargetOption{TargetPowerLessThanOrEqualToSourcePower(), TargetToughnessLessThanSourcePower()} {
		trap := NewPermanentFilter("opaque", func(*Permanent, *Game) bool { panic("query called filter") })
		if _, ok := InspectPermanentTarget(DeclarePermanentTarget("opaque", Or(AnyPermanent, trap), option)); ok {
			t.Fatal("partial target accepted")
		}
		g, a, _ := newSpellCopyGame()
		binding := DeclarePermanentTarget("subject", IsCreature, option)
		source := g.PutOnBattlefield(NewCreature("Source", "", 3, 4), a.PlayerID())
		victim := g.PutOnBattlefield(NewCreature("Subject", "", 1, 1), a.PlayerID())
		obj := &StackObject{ID: uuid.New(), SourceID: source.ID(), TargetSource: source.Card, IsAbility: true, Controller: a.PlayerID(), Targets: []uuid.UUID{victim.ID()}, TargetSpecs: []Target{binding.targetSpec()}, TargetZones: map[uuid.UUID]core.Zone{victim.ID(): core.ZoneBattlefield}, targetIncarnations: map[uuid.UUID]uuid.UUID{victim.ID(): victim.incarnationID}}
		plan, ok := InspectPermanentTarget(binding)
		if !ok {
			t.Fatal("source-relative declaration rejected")
		}
		if _, known := plan.Select(g, obj); known {
			t.Fatal("missing source visit guessed")
		}
		obj.sourceIncarnation = uuid.New()
		if slices.Contains(binding.Possible(a.PlayerID(), g.resolutionTargetSource(obj), g), victim.ID()) {
			t.Fatal("native selection used an unrelated source visit")
		}
		if _, known := plan.Select(g, obj); known {
			t.Fatal("unrelated source visit supplied power")
		}
	}
}

func TestSourceRelativeTargetQueryComparisons(t *testing.T) {
	for _, sourcePower := range []int{-2, 0, 3} {
		for _, delta := range []int{-1, 0, 1} {
			for _, toughness := range []bool{false, true} {
				g, a, _ := newSpellCopyGame()
				source := g.PutOnBattlefield(NewCreature("Source", "", sourcePower, 8), a.PlayerID())
				victim := g.PutOnBattlefield(NewCreature("Subject", "", sourcePower+delta, sourcePower+delta), a.PlayerID())
				option := TargetPowerLessThanOrEqualToSourcePower()
				want := delta <= 0
				if toughness {
					option = TargetToughnessLessThanSourcePower()
					want = delta < 0
				}
				binding := DeclarePermanentTarget("comparison", IsCreature, option)
				plan, ok := InspectPermanentTarget(binding)
				if !ok {
					t.Fatal("comparison is opaque")
				}
				obj := &StackObject{ID: uuid.New(), SourceID: source.ID(), sourceIncarnation: source.incarnationID, TargetSource: source.Card, IsAbility: true, Controller: a.PlayerID(), Targets: []uuid.UUID{victim.ID()}, TargetSpecs: []Target{binding.targetSpec()}, TargetZones: map[uuid.UUID]core.Zone{victim.ID(): core.ZoneBattlefield}, targetIncarnations: map[uuid.UUID]uuid.UUID{victim.ID(): victim.incarnationID}}
				got, known := plan.Select(g, obj)
				native := slices.Contains(binding.Possible(a.PlayerID(), g.resolutionTargetSource(obj), g), victim.ID())
				if !known || (len(got) == 1) != want || native != want {
					t.Fatalf("power %d delta %d toughness %v: %v/%v native %v", sourcePower, delta, toughness, got, known, native)
				}
			}
		}
	}
}

func TestSourceRelativeTargetQueryConjunction(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	binding := DeclarePermanentTarget("both", IsCreature, TargetPowerLessThanOrEqualToSourcePower(), TargetToughnessLessThanSourcePower())
	source := g.PutOnBattlefield(NewCreature("Source", "", 3, 8, WithAbility(Activate(Costs(), Destroy(binding), WithTargets(binding)))), a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Subject", "", 3, 2), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
		t.Fatal(err)
	}
	plan, ok := InspectPermanentTarget(binding)
	if !ok {
		t.Fatal("combined comparison rejected")
	}
	if got, known := plan.Select(g, g.StackPeek()); !known || len(got) != 1 {
		t.Fatal("combined legal target lost")
	}
	g.MutablePermanent(victim.ID()).AddCounter(core.P1P1, 1)
	if got, known := plan.Select(g, g.StackPeek()); !known || len(got) != 0 {
		t.Fatal("combined restriction lost")
	}
}
