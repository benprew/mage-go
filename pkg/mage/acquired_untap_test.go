package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAcquiredUntapRule(t *testing.T) {
	for _, change := range []string{"normal", "zero", "return", "phase", "cycle", "remove-before", "remove-after", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Recipient", "", 2, 5)
			p := g.PutOnBattlefield(card, a.PlayerID())
			rule := PreventUntapDuringStep(This, SourceHasCounters(core.Glyph, 1))
			target := DeclarePermanentTarget("recipient", IsCreature)
			declaration := NewInstant("Grant", "", Spell(AcquireStaticAbility(target, rule), WithTargets(target)))
			if err := ValidateCardDefinition("Grant", declaration); err != nil {
				t.Fatal(err)
			}
			if change == "remove-before" {
				g.AddContinuousEffect(RemoveAllAbilities(p.ID(), core.Indefinite))
			}
			if err := ApplyEffect(g, AcquireStaticAbility(This, rule), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.MutablePermanent(p.ID()).AddCounter(core.Glyph, 1)
			clone := g.Clone()
			blocked := true
			switch change {
			case "zero":
				g.MutablePermanent(p.ID()).Counters[core.Glyph] = 0
				blocked = false
			case "return":
				g.RemoveFromBattlefield(p)
				p = g.PutOnBattlefield(card, a.PlayerID())
				g.MutablePermanent(p.ID()).AddCounter(core.Glyph, 1)
				blocked = false
			case "phase":
				g.PhaseOut(p.ID())
				blocked = false
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "remove-after":
				g.AddContinuousEffect(RemoveAllAbilities(p.ID(), core.Indefinite))
				blocked = false
			case "control":
				g.AddControlEffect(ControlEffectSpec{TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.ApplyContinuousEffects()
			if got := g.matchingUntapRestrictions()[p.ID()]; got != blocked {
				t.Fatalf("blocked=%v want %v", got, blocked)
			}
			clone.ApplyContinuousEffects()
			if !clone.matchingUntapRestrictions()[p.ID()] {
				t.Fatal("clone lost rule")
			}
			if change != "phase" {
				g.TapPermanent(g.FindPermanent(p.ID()))
				if !g.UntapPermanent(g.FindPermanent(p.ID())) {
					t.Fatal("other untap was prevented")
				}
			}
		})
	}
}

func TestAcquiredUntapInvalid(t *testing.T) {
	for _, rule := range []StaticRule{PreventUntapDuringStep(nil, SourceHasCounters(core.Glyph, 1)), PreventUntapDuringStep(This, nil), PreventUntapDuringStep(From(BindPermanent("outer")), SourceHasCounters(core.Glyph, 1))} {
		c := NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), AcquireStaticAbility(This, rule))))
		if err := ValidateCardDefinition("Invalid", c); err == nil {
			t.Fatal("invalid rule accepted")
		}
	}
}
