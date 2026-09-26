package limited

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestComposableElementalBlastDestroy(t *testing.T) {
	for _, tc := range []struct{ spell, victim string }{
		{"Blue Elemental Blast", "Shivan Dragon"},
		{"Red Elemental Blast", "Air Elemental"},
	} {
		t.Run(tc.spell, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			g.AddCard(core.ZoneHand, gametest.PlayerA, tc.spell)
			g.AddCard(core.ZoneBattlefield, gametest.PlayerB, tc.victim)
			g.ChooseMode(gametest.PlayerA, 1)
			g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, tc.spell, tc.victim)
			g.StopAt(1, core.BeginCombat)
			g.Execute()
			g.AssertGraveyardCount(gametest.PlayerB, tc.victim, 1)
		})
	}
}

func TestComposableAuraControllerActivates(t *testing.T) {
	for _, tc := range []struct {
		aura             string
		power, toughness int
	}{
		{"Holy Armor", 2, 5}, {"Blessing", 3, 3}, {"Firebreathing", 3, 2},
	} {
		t.Run(tc.aura, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			g.AddCard(core.ZoneHand, gametest.PlayerA, tc.aura)
			g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
			g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, tc.aura, "Grizzly Bears")
			g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, tc.aura)
			g.StopAt(1, core.BeginCombat)
			g.Execute()
			g.AssertPowerToughness(gametest.PlayerB, "Grizzly Bears", tc.power, tc.toughness)
		})
	}
}

func TestComposableCountersInResponse(t *testing.T) {
	for _, tc := range []struct {
		card, spell string
		activation  bool
	}{
		{"Counterspell", "Serra Angel", false},
		{"Blue Elemental Blast", "Shivan Dragon", false},
		{"Red Elemental Blast", "Air Elemental", false},
		{"Deathgrip", "Grizzly Bears", true},
		{"Lifeforce", "Black Knight", true},
	} {
		t.Run(tc.card, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			g.AddCard(core.ZoneHand, gametest.PlayerA, tc.spell)
			zone := core.ZoneHand
			if tc.activation {
				zone = core.ZoneBattlefield
			}
			g.AddCard(zone, gametest.PlayerB, tc.card)
			g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, tc.spell)
			if tc.activation {
				g.ActivateInResponseTo(gametest.PlayerB, tc.card)
			} else {
				g.ChooseMode(gametest.PlayerB, 0)
				g.CastInResponseTo(gametest.PlayerB, tc.card)
			}
			g.StopAt(1, core.BeginCombat)
			g.Execute()
			g.AssertPermanentCount(gametest.PlayerA, tc.spell, 0)
			g.AssertGraveyardCount(gametest.PlayerA, tc.spell, 1)
		})
	}
}
