package mage

import (
	"testing"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestCounterEffectsUseContextX(t *testing.T) {
	for _, effect := range []struct {
		name   string
		effect Effect
	}{
		{name: "mana value", effect: CounterSpellIfXMeetsCMC()},
		{name: "payment", effect: PowerSinkEffect()},
	} {
		for _, tc := range []struct {
			name      string
			contextX  int
			gameX     int
			countered bool
		}{
			{name: "counter", contextX: 3, gameX: 0, countered: true},
			{name: "keep", contextX: 1, gameX: 4},
		} {
			t.Run(effect.name+"/"+tc.name, func(t *testing.T) {
				g := newPriorityTestGame()
				caster := g.players[0]
				defender := g.players[1]
				spell := NewInstant("Target", "{2}")
				spell.SetOwner(defender.PlayerID())
				g.pushStack(&StackObject{
					ID: uuid.New(), Card: spell, SourceID: spell.ID(), Controller: defender.PlayerID(),
				})
				defender.ManaPool().Add(Colorless, 2)
				g.resolution.SetX(tc.contextX)
				ctx := NewEffectContext(g, uuid.New(), caster.PlayerID(), []uuid.UUID{spell.ID()})
				g.resolution.SetX(tc.gameX)
				if err := effect.effect.Apply(ctx); err != nil {
					t.Fatal(err)
				}
				if countered := g.FindStackObject(spell.ID()) == nil; countered != tc.countered {
					t.Fatalf("countered = %v, want %v", countered, tc.countered)
				}
				if effect.name == "payment" && !tc.countered && defender.ManaPool().TotalMana() != 1 {
					t.Fatalf("mana after payment = %d, want 1", defender.ManaPool().TotalMana())
				}
			})
		}
	}
}
