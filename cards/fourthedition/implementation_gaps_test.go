package fourthedition

import (
	"slices"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestFourthEditionSpellAIHints(t *testing.T) {
	tests := []struct {
		name          string
		roles         []mage.AIRole
		purpose       mage.AITargetPurpose
		preferTarget  mage.AITargetPreference
		expectTapMeta bool
	}{
		{
			name:         "Ashes to Ashes",
			roles:        []mage.AIRole{mage.AIRoleRemoval},
			purpose:      mage.AITargetExile,
			preferTarget: mage.PreferLargestThreat,
		},
		{
			name:         "Volcanic Eruption",
			roles:        []mage.AIRole{mage.AIRoleRemoval},
			purpose:      mage.AITargetRemoval,
			preferTarget: mage.PreferLargestThreat,
		},
		{
			name:          "Word of Binding",
			purpose:       mage.AITargetTap,
			preferTarget:  mage.PreferLargestThreat,
			expectTapMeta: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card, err := mage.CreateCard(tt.name)
			if err != nil {
				t.Fatalf("CreateCard(%q): %v", tt.name, err)
			}
			var spell *mage.SpellAbility
			for _, ability := range card.Abilities() {
				candidate, ok := ability.(*mage.SpellAbility)
				if ok && candidate.Kind() == mage.ActionSpell {
					spell = candidate
					break
				}
			}
			if spell == nil {
				t.Fatal("card has no spell ability")
			}

			hints := spell.AIHints()
			if len(hints) != 1 {
				t.Fatalf("AI hint count = %d, want 1", len(hints))
			}
			hint := hints[0]
			if !slices.Equal(hint.Roles, tt.roles) {
				t.Errorf("AI roles = %v, want %v", hint.Roles, tt.roles)
			}
			if hint.TargetPurpose != tt.purpose {
				t.Errorf("target purpose = %v, want %v", hint.TargetPurpose, tt.purpose)
			}
			if hint.PreferTarget != tt.preferTarget {
				t.Errorf("target preference = %v, want %v", hint.PreferTarget, tt.preferTarget)
			}
			if tt.expectTapMeta {
				taps := false
				for _, effect := range spell.Effects() {
					taps = taps || effect.Properties().Taps
				}
				if !taps {
					t.Error("spell effect is missing tap metadata")
				}
			}
		})
	}
}

func TestFellwarStone(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fellwar Stone")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Fellwar Stone")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertTapped(gametest.PlayerA, "Fellwar Stone", true)
	g.AssertManaProduced(gametest.PlayerA, core.Blue, 6)
}

func TestLibraryOfLeng(t *testing.T) {
	t.Run("no maximum hand size", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Library of Leng")
		if got := g.MaximumHandSize(g.GetPlayer(gametest.PlayerA).PlayerID()); got != -1 {
			t.Fatalf("maximum hand size after Library of Leng entered = %d, want no maximum", got)
		}
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears", 8)
		g.StopAt(2, core.Upkeep)
		g.Execute()
		g.AssertHandSize(gametest.PlayerA, 8)
	})

	t.Run("effect-caused discard may go on top of library", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Library of Leng")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Mind Twist")
		g.ChooseDiscard(gametest.PlayerA, "Grizzly Bears")
		g.GetPlayer(gametest.PlayerA).QueueMayAbilityChoices(true)
		g.CastSpellWithX(2, core.PrecombatMain, gametest.PlayerB, "Mind Twist", 1, "PlayerA")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 0)
		g.AssertLibraryTop(gametest.PlayerA, "Grizzly Bears")
	})
}

func TestBrainwash(t *testing.T) {
	t.Run("creature cannot attack when controller cannot pay", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Brainwash")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Brainwash", "Grizzly Bears")
		g.Attack(3, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(3, core.PostcombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 20)
	})

	t.Run("creature attacks after controller pays three", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Brainwash")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Brainwash", "Grizzly Bears")
		g.Attack(3, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(3, core.PostcombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 18)
	})
}

func TestErosion(t *testing.T) {
	t.Run("controller may pay one life", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Erosion")
		g.ChooseMode(gametest.PlayerB, 1)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Erosion", "Island")
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Island", 1)
		g.AssertLife(gametest.PlayerB, 19)
	})

	t.Run("land is destroyed when controller declines", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Erosion")
		g.ChooseMode(gametest.PlayerB, 2)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Erosion", "Island")
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Island", 0)
	})
}

func TestPowerLeak(t *testing.T) {
	t.Run("paid mana prevents that much damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Bad Moon")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Power Leak")
		g.ChooseNumber(gametest.PlayerB, 1)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Power Leak", "Bad Moon")
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 19)
	})

	t.Run("paying nothing takes two damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Bad Moon")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Power Leak")
		g.ChooseNumber(gametest.PlayerB, 0)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Power Leak", "Bad Moon")
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 18)
	})
}

func TestAshesToAshes(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Ashes to Ashes")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Ashes to Ashes", "Grizzly Bears", "Hill Giant")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertExileCount("Grizzly Bears", 1)
	g.AssertExileCount("Hill Giant", 1)
	g.AssertLife(gametest.PlayerA, 15)
}

func TestMindBomb(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Mind Bomb")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Island")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Swamp")
	g.ChooseNumber(gametest.PlayerA, 1)
	g.ChooseNumber(gametest.PlayerB, 2)
	g.ChooseDiscard(gametest.PlayerA, "Forest")
	g.ChooseDiscard(gametest.PlayerB, "Island", "Swamp")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Mind Bomb")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 18)
	g.AssertLife(gametest.PlayerB, 19)
	g.AssertGraveyardCount(gametest.PlayerA, "Forest", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Island", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Swamp", 1)
}

func TestVolcanicEruption(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Badlands")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Volcanic Eruption")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Fissure")
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Volcanic Eruption", 2, "Mountain", "Badlands")
	g.CastInResponseTo(gametest.PlayerB, "Fissure", "Mountain")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Mountain", 0)
	g.AssertPermanentCount(gametest.PlayerB, "Badlands", 0)
	g.AssertLife(gametest.PlayerA, 19)
	g.AssertLife(gametest.PlayerB, 19)
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 1)
}

func TestWordOfBinding(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Word of Binding")
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Word of Binding", 2, "Grizzly Bears", "Hill Giant")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertTapped(gametest.PlayerB, "Grizzly Bears", true)
	g.AssertTapped(gametest.PlayerB, "Hill Giant", true)
}
