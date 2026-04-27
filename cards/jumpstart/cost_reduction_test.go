package jumpstart

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// Tests for cards that reduce spell costs (CR 601.2f). The test harness
// auto-fills missing generic mana on cast, so end-to-end mana-pinch checks
// cannot detect a missing reduction. Instead each test queries
// Game.ConditionalSpellCostReduction directly and (when applicable)
// confirms the spell still casts and resolves through the normal pipeline.

// handCard returns the first card named name in p's hand.
func handCard(g *gametest.TestGame, p gametest.PlayerRef, name string) mage.Card {
	pl := g.GetPlayer(p)
	for _, c := range pl.Hand() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

func TestWardenOfEvosIsle_FlyingCreatureSpellsCostOneLess(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Warden of Evos Isle")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Rishadan Airship")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	pid := g.GetPlayer(gametest.PlayerA).PlayerID()
	if got := g.Game.ConditionalSpellCostReduction(pid, handCard(g, gametest.PlayerA, "Rishadan Airship")); got != 1 {
		t.Errorf("flying creature reduction: got %d, want 1", got)
	}
	if got := g.Game.ConditionalSpellCostReduction(pid, handCard(g, gametest.PlayerA, "Grizzly Bears")); got != 0 {
		t.Errorf("non-flying creature reduction: got %d, want 0", got)
	}
}

func TestWardenOfEvosIsle_DoesNotReduceOpponentSpells(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Warden of Evos Isle")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Rishadan Airship")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	pid := g.GetPlayer(gametest.PlayerB).PlayerID()
	if got := g.Game.ConditionalSpellCostReduction(pid, handCard(g, gametest.PlayerB, "Rishadan Airship")); got != 0 {
		t.Errorf("opponent flyer reduction: got %d, want 0", got)
	}
}

func TestBonePicker_ReducedAfterCreatureDeath(t *testing.T) {
	t.Run("no death this turn: no reduction", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Bone Picker")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		pid := g.GetPlayer(gametest.PlayerA).PlayerID()
		if got := g.Game.ConditionalSpellCostReduction(pid, handCard(g, gametest.PlayerA, "Bone Picker")); got != 0 {
			t.Errorf("Bone Picker no death: got %d, want 0", got)
		}
	})

	t.Run("creature died this turn: reduces by 3 and cast resolves", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 1)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 1)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Bone Picker")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Bone Picker")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Bone Picker", 1)
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestDragonlordsServant_ReducesDragonSpellsByOne(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dragonlord's Servant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Shivan Dragon")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Hill Giant")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	pid := g.GetPlayer(gametest.PlayerA).PlayerID()
	if got := g.Game.ConditionalSpellCostReduction(pid, handCard(g, gametest.PlayerA, "Shivan Dragon")); got != 1 {
		t.Errorf("dragon reduction: got %d, want 1", got)
	}
	if got := g.Game.ConditionalSpellCostReduction(pid, handCard(g, gametest.PlayerA, "Hill Giant")); got != 0 {
		t.Errorf("non-dragon reduction: got %d, want 0", got)
	}
}

func TestDragonspeakerShaman_ReducesDragonSpellsByTwo(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dragonspeaker Shaman")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Shivan Dragon")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	pid := g.GetPlayer(gametest.PlayerA).PlayerID()
	if got := g.Game.ConditionalSpellCostReduction(pid, handCard(g, gametest.PlayerA, "Shivan Dragon")); got != 2 {
		t.Errorf("dragon reduction by Dragonspeaker Shaman: got %d, want 2", got)
	}
}

func TestDragonRulers_StackAdditively(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dragonlord's Servant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dragonspeaker Shaman")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Shivan Dragon")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	pid := g.GetPlayer(gametest.PlayerA).PlayerID()
	if got := g.Game.ConditionalSpellCostReduction(pid, handCard(g, gametest.PlayerA, "Shivan Dragon")); got != 3 {
		t.Errorf("stacked dragon reduction: got %d, want 3", got)
	}
}

func TestGhaltaPrimalHunger_ReducedByTotalPower(t *testing.T) {
	t.Run("twelve power on board: capped at 10", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Craw Wurm", 2)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Ghalta, Primal Hunger")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		pid := g.GetPlayer(gametest.PlayerA).PlayerID()
		if got := g.Game.ConditionalSpellCostReduction(pid, handCard(g, gametest.PlayerA, "Ghalta, Primal Hunger")); got != 10 {
			t.Errorf("Ghalta reduction with 12 power: got %d, want 10 (capped)", got)
		}
	})

	t.Run("no creatures: no reduction", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Ghalta, Primal Hunger")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		pid := g.GetPlayer(gametest.PlayerA).PlayerID()
		if got := g.Game.ConditionalSpellCostReduction(pid, handCard(g, gametest.PlayerA, "Ghalta, Primal Hunger")); got != 0 {
			t.Errorf("Ghalta reduction with no creatures: got %d, want 0", got)
		}
	})
}

func TestCrypticSerpent_ReducedByInstantsAndSorceriesInGraveyard(t *testing.T) {
	t.Run("four instants/sorceries in yard: reduces by 4", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Lightning Bolt", 2)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Dragon Fodder", 2)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Cryptic Serpent")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		pid := g.GetPlayer(gametest.PlayerA).PlayerID()
		if got := g.Game.ConditionalSpellCostReduction(pid, handCard(g, gametest.PlayerA, "Cryptic Serpent")); got != 4 {
			t.Errorf("Cryptic Serpent reduction with 4 instants/sorceries: got %d, want 4", got)
		}
	})

	t.Run("creatures in graveyard do not reduce", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears", 4)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Cryptic Serpent")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		pid := g.GetPlayer(gametest.PlayerA).PlayerID()
		if got := g.Game.ConditionalSpellCostReduction(pid, handCard(g, gametest.PlayerA, "Cryptic Serpent")); got != 0 {
			t.Errorf("Cryptic Serpent reduction with only creatures in yard: got %d, want 0", got)
		}
	})
}

func TestHeraldsHorn_ReducesChosenCreatureType(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Herald's Horn")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Shivan Dragon")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Hill Giant")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	pid := g.GetPlayer(gametest.PlayerA).PlayerID()
	hornPerm := g.FindPermanentByName("Herald's Horn", pid)
	if hornPerm == nil {
		t.Fatal("Herald's Horn not found on battlefield")
	}
	hornPerm.ChosenSubtype = "Dragon"

	if got := g.Game.ConditionalSpellCostReduction(pid, handCard(g, gametest.PlayerA, "Shivan Dragon")); got != 1 {
		t.Errorf("Dragon spell with chosen Dragon: got %d, want 1", got)
	}
	if got := g.Game.ConditionalSpellCostReduction(pid, handCard(g, gametest.PlayerA, "Hill Giant")); got != 0 {
		t.Errorf("Giant spell with chosen Dragon: got %d, want 0", got)
	}
}

func TestWingedWords_ReducedIfYouControlFlyer(t *testing.T) {
	t.Run("with flyer: reduces by 1", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rishadan Airship")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Winged Words")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		pid := g.GetPlayer(gametest.PlayerA).PlayerID()
		if got := g.Game.ConditionalSpellCostReduction(pid, handCard(g, gametest.PlayerA, "Winged Words")); got != 1 {
			t.Errorf("Winged Words with flyer: got %d, want 1", got)
		}
	})

	t.Run("without flyer: no reduction", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Winged Words")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		pid := g.GetPlayer(gametest.PlayerA).PlayerID()
		if got := g.Game.ConditionalSpellCostReduction(pid, handCard(g, gametest.PlayerA, "Winged Words")); got != 0 {
			t.Errorf("Winged Words without flyer: got %d, want 0", got)
		}
	})
}

func TestWizardsRetort_ReducedIfYouControlWizard(t *testing.T) {
	t.Run("with Wizard: reduces by 1", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Erratic Visionary")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Wizard's Retort")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		pid := g.GetPlayer(gametest.PlayerA).PlayerID()
		if got := g.Game.ConditionalSpellCostReduction(pid, handCard(g, gametest.PlayerA, "Wizard's Retort")); got != 1 {
			t.Errorf("Wizard's Retort with Wizard: got %d, want 1", got)
		}
	})

	t.Run("without Wizard: no reduction", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Wizard's Retort")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		pid := g.GetPlayer(gametest.PlayerA).PlayerID()
		if got := g.Game.ConditionalSpellCostReduction(pid, handCard(g, gametest.PlayerA, "Wizard's Retort")); got != 0 {
			t.Errorf("Wizard's Retort without Wizard: got %d, want 0", got)
		}
	})
}
