package limited

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestLacesChangePermanentColor(t *testing.T) {
	tests := []struct {
		name  string
		color core.Color
	}{
		{"Purelace", core.White},
		{"Thoughtlace", core.Blue},
		{"Deathlace", core.Black},
		{"Chaoslace", core.Red},
		{"Lifelace", core.Green},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Sol Ring")
			g.AddCard(core.ZoneHand, gametest.PlayerA, tt.name)
			g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, tt.name, "Sol Ring")
			g.StopAt(1, core.BeginCombat)
			g.Execute()

			g.AssertHasColor(gametest.PlayerB, "Sol Ring", tt.color, true)
		})
	}
}

func TestLaceChangesSpellColor(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Thoughtlace")
	player := g.GetPlayer(gametest.PlayerA)
	g.SetStep(core.PrecombatMain)
	player.ManaPool().Add(core.Colorless, 1)
	player.ManaPool().Add(core.Green, 1)
	player.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByName(player.PlayerID(), "Grizzly Bears", nil); err != nil {
		t.Fatalf("cast Grizzly Bears: %v", err)
	}
	spellID := g.GetStack().Peek().SourceID
	if err := g.CastSpellByName(player.PlayerID(), "Thoughtlace", []uuid.UUID{spellID}); err != nil {
		t.Fatalf("cast Thoughtlace: %v", err)
	}
	g.ResolveTopOfStack()

	if got := g.EffectiveColors(spellID); !slices.Equal(got, []core.Color{core.Blue}) {
		t.Fatalf("spell colors = %v, want Blue", got)
	}
	g.ResolveTopOfStack()
	g.AssertHasColor(gametest.PlayerA, "Grizzly Bears", core.Blue, true)
}
