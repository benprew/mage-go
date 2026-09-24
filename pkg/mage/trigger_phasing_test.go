package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestPhasedSourceDoesNotObserveEvents(t *testing.T) {
	for _, scenario := range []string{"death", "simultaneous deaths", "end step", "phase in", "phase after trigger"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			ability := WhenEvent(CreatureDies(), GainLifeFor(You, Fixed(1)))
			if scenario == "end step" {
				ability = NewTriggered(EvtEndStep, false, GainLifeFor(You, Fixed(1)))
			}
			source := NewCreature("Watcher", "", 1, 1, WithAbility(ability))
			source.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			other := NewCreature("Other", "", 1, 1)
			other.SetOwner(b.PlayerID())
			g.PutOnBattlefield(other, b.PlayerID())
			if scenario != "phase after trigger" {
				phased := g.PhaseOut(source.ID())
				if scenario == "phase in" {
					g.PhaseIn(phased)
				}
			}
			switch scenario {
			case "end step":
				g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
			case "simultaneous deaths":
				second := NewCreature("Second", "", 1, 1)
				second.SetOwner(b.PlayerID())
				g.PutOnBattlefield(second, b.PlayerID())
				g.MutablePermanent(other.ID()).Damage = 1
				g.MutablePermanent(second.ID()).Damage = 1
				g.CheckStateBasedActions()
			default:
				g.DestroyPermanent(g.FindPermanent(other.ID()))
			}
			if scenario == "phase after trigger" {
				g.PhaseOut(source.ID())
			}
			g.ResolveStack()
			want := 20
			if scenario == "phase in" || scenario == "phase after trigger" {
				want = 21
			}
			if a.Life() != want {
				t.Fatalf("life = %d, want %d", a.Life(), want)
			}
		})
	}
}
