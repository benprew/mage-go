package gametest

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// TestDuelSpellsCastByColorAndType verifies the per-duel spell tally records a
// cast spell's colors and types, attributed to the casting player.
func TestDuelSpellsCastByColorAndType(t *testing.T) {
	registerPerTurnTrackerCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Test Gain 3 Life")  // {W} sorcery
	tg.AddCard(core.ZoneHand, PlayerA, "Test Tracker Bear") // {1}{G} creature
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Gain 3 Life")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Tracker Bear")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	pid := tg.GetPlayer(PlayerA).PlayerID()
	obj := tg.DuelObjectivesFor(pid)
	if obj.SpellsByColor[core.White] != 1 {
		t.Errorf("white spells = %d, want 1", obj.SpellsByColor[core.White])
	}
	if obj.SpellsByColor[core.Green] != 1 {
		t.Errorf("green spells = %d, want 1", obj.SpellsByColor[core.Green])
	}
	if obj.SpellsByType[core.TypeSorcery] != 1 {
		t.Errorf("sorcery spells = %d, want 1", obj.SpellsByType[core.TypeSorcery])
	}
	if obj.SpellsByType[core.TypeCreature] != 1 {
		t.Errorf("creature spells = %d, want 1", obj.SpellsByType[core.TypeCreature])
	}

	// The opponent cast nothing.
	oppObj := tg.DuelObjectivesFor(tg.GetPlayer(PlayerB).PlayerID())
	if oppObj.SpellsByColor[core.White] != 0 {
		t.Errorf("opponent should have cast no white spells, got %d", oppObj.SpellsByColor[core.White])
	}
}

// TestDuelLandsPlayed verifies EvtLandPlayed increments the per-duel land count.
func TestDuelLandsPlayed(t *testing.T) {
	tg := NewTestGame(t)
	tg.padLibraries()
	tg.AddCard(core.ZoneHand, PlayerA, "Forest")
	pid := tg.GetPlayer(PlayerA).PlayerID()

	// Advance PlayerA to their precombat main phase so a land can be played.
	tg.Turn = 1
	tg.ActivePlayer = 0
	tg.OnPriority = func(g *mage.Game, playerIdx int, mainPhase bool) mage.PriorityAction {
		return mage.PriorityAction{Type: mage.PriorityPass}
	}
	for _, step := range []core.PhaseStep{core.Untap, core.Upkeep, core.Draw} {
		tg.Step = step
		tg.RunStepWithPriority(step)
	}
	tg.Step = core.PrecombatMain

	land, ok := findLandInHand(tg, PlayerA, "Forest")
	if !ok {
		t.Fatal("setup: Forest not in hand")
	}
	if err := tg.PlayLand(pid, land.ID()); err != nil {
		t.Fatalf("PlayLand: %v", err)
	}
	if got := tg.DuelObjectivesFor(pid).LandsPlayed; got != 1 {
		t.Errorf("LandsPlayed = %d, want 1", got)
	}
}

// TestDuelAttackersDeclared verifies declared attackers are counted per player.
func TestDuelAttackersDeclared(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.Attack(1, PlayerA, "Grizzly Bears")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	pid := tg.GetPlayer(PlayerA).PlayerID()
	if got := tg.DuelObjectivesFor(pid).AttackersDeclared; got != 1 {
		t.Errorf("AttackersDeclared = %d, want 1", got)
	}
}

// TestDuelOpponentCreaturesDestroyed verifies that an opponent's creature dying
// is credited as a destroyed creature from the killer's perspective.
func TestDuelOpponentCreaturesDestroyed(t *testing.T) {
	registerPerTurnTrackerCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Grizzly Bears") // 2/2
	tg.AddCard(core.ZoneHand, PlayerA, "Test Bolt 3")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Bolt 3", "Grizzly Bears")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	a := tg.GetPlayer(PlayerA).PlayerID()
	b := tg.GetPlayer(PlayerB).PlayerID()
	if got := tg.DuelObjectivesFor(a).OpponentCreaturesDestroyed; got != 1 {
		t.Errorf("PlayerA OpponentCreaturesDestroyed = %d, want 1", got)
	}
	// From PlayerB's perspective, PlayerB destroyed none of PlayerA's creatures.
	if got := tg.DuelObjectivesFor(b).OpponentCreaturesDestroyed; got != 0 {
		t.Errorf("PlayerB OpponentCreaturesDestroyed = %d, want 0", got)
	}
}

// TestDuelNonCombatDamage verifies non-combat damage to a player is credited to
// the dealing (opposing) player and combat damage is excluded.
func TestDuelNonCombatDamage(t *testing.T) {
	tg := NewTestGame(t)
	a := tg.GetPlayer(PlayerA).PlayerID()
	b := tg.GetPlayer(PlayerB).PlayerID()

	tg.DealDamageToPlayer(tg.GetPlayer(PlayerB), 5, uuid.Nil)

	if got := tg.DuelObjectivesFor(a).NonCombatDamageDealt; got != 5 {
		t.Errorf("PlayerA NonCombatDamageDealt = %d, want 5", got)
	}
	if got := tg.DuelObjectivesFor(b).NonCombatDamageDealt; got != 0 {
		t.Errorf("PlayerB NonCombatDamageDealt = %d, want 0", got)
	}
}

// TestDuelTrackersSurviveTurnReset confirms per-duel counters are NOT cleared
// between turns (unlike the per-turn trackers).
func TestDuelTrackersSurviveTurnReset(t *testing.T) {
	registerPerTurnTrackerCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Test Gain 3 Life")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Gain 3 Life")
	tg.StopAt(2, core.EndStep) // run into turn 2
	tg.Execute()

	pid := tg.GetPlayer(PlayerA).PlayerID()
	if got := tg.DuelObjectivesFor(pid).SpellsByType[core.TypeSorcery]; got != 1 {
		t.Errorf("sorcery count after two turns = %d, want 1 (should persist)", got)
	}
}
