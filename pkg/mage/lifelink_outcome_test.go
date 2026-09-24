package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestLifelinkOutcomeEvents(t *testing.T) {
	for _, recipient := range []string{"player", "creature"} {
		for _, scenario := range []struct {
			name                           string
			prevent, wantGain, replacement int
			double, replace, lich, combat  bool
		}{
			{name: "normal", wantGain: 3},
			{name: "combat", wantGain: 3, combat: true},
			{name: "double", double: true, wantGain: 6},
			{name: "partial-prevention", prevent: 2, double: true, wantGain: 2},
			{name: "full-prevention", prevent: 3},
			{name: "replacement", replace: true, replacement: 7, wantGain: 7},
			{name: "no-gain", replace: true},
			{name: "lich", lich: true},
		} {
			t.Run(recipient+"/"+scenario.name, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				source := NewCreature("Lifelink source", "", 3, 4, WithKeyword(Lifelink))
				g.PutOnBattlefield(source, b.PlayerID())
				g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: a.PlayerID(), Duration: Indefinite})
				watcher := NewEnchantment("Watcher", "", WithAbility(WheneverYouGainLifeTrigger(
					LoseLifeFor(You, ValueAtResolution(EventAmountValue())), false)))
				g.PutOnBattlefield(watcher, a.PlayerID())
				target := NewCreature("Recipient", "", 0, 10)
				g.PutOnBattlefield(target, b.PlayerID())
				if scenario.double {
					g.AddLifeGainModifier(watcher.ID(), DoubleLifeGainForController(watcher.ID(), a.PlayerID()))
				}
				if scenario.replace {
					g.effects.AddReplacement(&lifeOutcomeReplacement{amount: scenario.replacement})
				}
				if scenario.lich {
					g.SetLichActive(a.PlayerID(), watcher.ID())
					for range 3 {
						a.AddToLibrary(NewLand("Card"))
					}
				}
				protected := b.PlayerID()
				if recipient == "creature" {
					protected = target.ID()
				}
				if scenario.prevent > 0 {
					g.AddPreventionShield(protected, scenario.prevent)
				}
				clone := g.Clone()
				clone.damage.SetResolvingCombatDamage(scenario.combat)
				if recipient == "player" {
					clone.DealDamageToPlayer(clone.GetPlayer(b.PlayerID()), 3, source.ID())
					if got := clone.GetPlayer(b.PlayerID()).Life(); got != 17+scenario.prevent {
						t.Errorf("recipient life = %d, want %d", got, 17+scenario.prevent)
					}
				} else {
					clone.DealDamageToPermanent(clone.FindPermanent(target.ID()), 3, source.ID())
					if got := clone.FindPermanent(target.ID()).Damage; got != 3-scenario.prevent {
						t.Errorf("marked damage = %d, want %d", got, 3-scenario.prevent)
					}
				}
				if got := clone.GetPlayer(a.PlayerID()).Life(); got != 20+scenario.wantGain {
					t.Errorf("controller life = %d, want %d", got, 20+scenario.wantGain)
				}
				if got := clone.PlayerLifeGainedThisTurn(a.PlayerID()); got != scenario.wantGain {
					t.Errorf("recorded gain = %d, want %d", got, scenario.wantGain)
				}
				clone.PutTriggersOnStack()
				wantTriggers := 0
				if scenario.wantGain > 0 {
					wantTriggers = 1
				}
				if got := clone.StackSize(); got != wantTriggers {
					t.Errorf("triggers = %d, want %d", got, wantTriggers)
				}
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Error("life trigger did not receive the actual gain amount")
				}
				if scenario.lich && len(clone.GetPlayer(a.PlayerID()).Hand()) != 3 {
					t.Error("replaced life gain did not draw three cards")
				}
				g.PutTriggersOnStack()
				if a.Life() != 20 || b.Life() != 20 || len(a.Hand()) != 0 || g.StackSize() != 0 ||
					g.PlayerLifeGainedThisTurn(a.PlayerID()) != 0 || g.FindPermanent(target.ID()).Damage != 0 {
					t.Error("clone damage changed the original game")
				}
			})
		}
	}
}
