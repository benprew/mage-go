package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingCounterPreventionReferences(t *testing.T) {
	for _, state := range []string{"live", "phase", "cycle", "depart", "return", "control", "remove-ability", "restore-ability"} {
		t.Run(state, func(t *testing.T) {
			a := &acceptingMayPlayer{NewBasePlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Rule", "{W}", 1, 20, WithStaticAbility(PreventDamageToSourceByRemovingCounters(core.P1P1)))
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.MutablePermanent(source.ID()).AddCounter(core.P1P1, 2)
			saved := g.CaptureDamageSource(source.ID())
			clone := g.Clone()
			var stale ReplacementEffect
			for _, r := range g.effects.cycleReplacements {
				if _, ok := r.(*counterDamagePreventionReplacement); ok {
					stale = r.Clone()
					break
				}
			}
			if stale == nil {
				t.Fatal("missing replacement")
			}
			want := 1
			switch state {
			case "phase":
				g.PhaseOut(source.ID())
				want = 0
			case "cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "depart", "return":
				g.RemoveFromBattlefield(source)
				g.zones.ClearLKI()
				want = 0
				if state == "return" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "remove-ability", "restore-ability":
				g.AddContinuousEffect(RemoveAllAbilities(source.ID(), core.EndOfTurn))
				want = 0
				if state == "restore-ability" {
					g.effects.RemoveEndOfTurn()
					g.ApplyContinuousEffects()
					want = 1
				}
			}
			g.effects.cycleReplacements = []ReplacementEffect{stale}
			action := NewDamageToCreatureAction(uuid.New(), source.ID(), 1, false)
			if stale.IsActive(g) != (want == 1) || stale.Matches(action, g) != (want == 1) {
				t.Fatal("stale rule lifetime differs from execution")
			}
			refs := g.WaitingPreventionPermanentSources()
			if len(refs) != want {
				t.Fatalf("sources=%d want=%d", len(refs), want)
			}
			if want == 1 {
				if !refs[0].SameObject(saved) {
					t.Fatal("wrong incarnation")
				}
				if state == "control" && refs[0].Controller() != b.PlayerID() {
					t.Fatal("stale controller")
				}
				refs[0].card.SetOwner(b.PlayerID())
				if g.WaitingPreventionPermanentSources()[0].card.Owner() != a.PlayerID() {
					t.Fatal("query changed card")
				}
			}
			if refs := clone.WaitingPreventionPermanentSources(); len(refs) != 1 || !refs[0].SameObject(saved) {
				t.Fatal("clone changed")
			}
			if state == "return" {
				g.ApplyContinuousEffects()
				refs = g.WaitingPreventionPermanentSources()
				if len(refs) != 1 || refs[0].SameObject(saved) {
					t.Fatal("new registration did not use returned incarnation")
				}
			}
		})
	}
}

func TestWaitingCounterPreventionBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Rule", "", 1, 20, WithStaticAbility(PreventDamageToSourceByRemovingCounters(core.P1P1)), WithStaticAbility(PreventDamageToSourceByRemovingCounters(core.Charge)))
	p := g.PutOnBattlefield(card, a.PlayerID())
	saved := g.CaptureDamageSource(p.ID())
	var rules []*counterDamagePreventionReplacement
	for _, replacement := range g.effects.cycleReplacements {
		if rule, ok := replacement.(*counterDamagePreventionReplacement); ok {
			rules = append(rules, rule)
		}
	}
	if len(rules) != 2 {
		t.Fatalf("rules=%d", len(rules))
	}
	g.effects.AddReplacement(rules[0].Clone())
	g.effects.AddReplacement(&counterDamagePreventionReplacement{replacementBase: replacementBase{sourceID: p.ID()}, ruleID: rules[0].ruleID})
	g.effects.AddReplacement(&counterDamagePreventionReplacement{replacementBase: replacementBase{sourceID: uuid.New()}, incarnation: uuid.New(), ruleID: uuid.New()})
	g.PutOnBattlefield(NewCreature("Unrelated", "", 1, 20), b.PlayerID())
	if refs := g.WaitingPreventionPermanentSources(); len(refs) != 1 || !refs[0].SameObject(saved) {
		t.Fatal("invalid or duplicate reference")
	}
	if rules[0].Matches(NewDamageToCreatureAction(uuid.New(), p.ID(), 1, false), g) {
		t.Fatal("empty counters prevented damage")
	}
	g.MutablePermanent(p.ID()).AddCounter(core.P1P1, 2)
	for _, amount := range []int{0, -1} {
		if rules[0].Matches(NewDamageToCreatureAction(uuid.New(), p.ID(), amount, false), g) {
			t.Fatal("nonpositive damage matched")
		}
	}
	clone := g.Clone()
	clone.DealDamageToPermanent(clone.FindPermanent(p.ID()), 3, uuid.New())
	if subject := clone.FindPermanent(p.ID()); subject.Damage != 1 || subject.Counters[core.P1P1] != 0 {
		t.Fatal("clone prevention outcome")
	}
	if subject := g.FindPermanent(p.ID()); subject.Damage != 0 || subject.Counters[core.P1P1] != 2 {
		t.Fatal("clone changed original")
	}
	g.MutablePermanent(p.ID()).RuntimeAbilities = g.MutablePermanent(p.ID()).RuntimeAbilities[1:]
	if rules[0].IsActive(g) || !rules[1].IsActive(g) {
		t.Fatal("one rule kept a different removed rule active")
	}
	if len(g.WaitingPreventionPermanentSources()) != 1 {
		t.Fatal("remaining rule lost subject")
	}
}
