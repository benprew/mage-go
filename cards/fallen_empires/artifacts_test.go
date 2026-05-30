package fallen_empires

import (
	"testing"

	_ "github.com/benprew/mage-go/cards/limited" // register base cards
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestDraconianCylix_Regenerates(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Draconian Cylix")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 3) // mana for {2}
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest")          // card to discard
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Draconian Cylix", "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Grizzly Bears")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Forest", 1)
}

func TestRingOfRenewal_DiscardThenDraw(t *testing.T) {
	// Ring of Renewal: {5}, {T}: Discard a card at random, then draw two cards.
	// Verify the ability activates (Ring taps) and at least 1 card is drawn.
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ring of Renewal")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Swamp")    // turn 1 draw
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Island")   // ability draw 1
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain") // ability draw 2
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Ring of Renewal")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertTapped(gametest.PlayerA, "Ring of Renewal", true)
	g.AssertHandCount(gametest.PlayerA, "Island", 1)
}

func TestRingOfRenewal_EmptyHand(t *testing.T) {
	// Ring of Renewal: {5}, {T}: Discard a card at random, then draw two cards.
	// Player starts with an empty hand. The discard is part of the effect (not a cost),
	// so with an empty hand nothing is discarded, and 2 cards are drawn.
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ring of Renewal")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Gray Ogre")  // ability draw 1
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant") // ability draw 2
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Ring of Renewal")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertTapped(gametest.PlayerA, "Ring of Renewal", true)
	g.AssertHandCount(gametest.PlayerA, "Gray Ogre", 1)
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 1)
}

func TestConchHorn_DrawThenPutBack(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Conch Horn")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest") // mana for {1}
	// Library order: index 0 drawn first. Draw step takes Plains, ability draws Island then Mountain.
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")   // draw step
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Island")   // ability draw 1
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain") // ability draw 2
	g.ChooseDiscard(gametest.PlayerA, "Mountain")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Conch Horn")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Conch Horn", 0)
	g.AssertHandCount(gametest.PlayerA, "Island", 1)
	g.AssertHandCount(gametest.PlayerA, "Mountain", 0)
	g.AssertLibraryTop(gametest.PlayerA, "Mountain")
}

func TestImplementsOfSacrifice_AddMana(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Implements of Sacrifice")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest") // mana for {1}
	g.ChooseManaColor(gametest.PlayerA, core.Green)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Implements of Sacrifice")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Implements of Sacrifice", 0)
}

func TestBalmOfRestoration_GainLife(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Balm of Restoration")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest") // mana for {1}
	g.ChooseMode(gametest.PlayerA, 0)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Balm of Restoration")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 22)
	g.AssertPermanentCount(gametest.PlayerA, "Balm of Restoration", 0)
}

func TestAeolipile_DealsDamage(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Aeolipile")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Aeolipile", "PlayerB")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 18)
	g.AssertPermanentCount(gametest.PlayerA, "Aeolipile", 0)
}

func TestAeolipile_DealsDamageToCreature(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Aeolipile")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Aeolipile", "Grizzly Bears")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Aeolipile", 0)
}

func TestElvenLyre_BoostsCreature(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Elven Lyre")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Elven Lyre", "Grizzly Bears")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 4)
	g.AssertPermanentCount(gametest.PlayerA, "Elven Lyre", 0)
}
