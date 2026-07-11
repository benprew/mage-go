package gametest

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// "As ~ enters the battlefield, choose ___" — CR 614.12.
//
// These tests cover the four engine constructors that record an
// "as it enters" choice on a permanent (ETBChooseColor,
// ETBChooseColorOtherThan, ETBChooseOpponent, ETBChooseCreatureType).
// Each test registers a small card that uses the constructor and asserts
// that the chosen value is stored on the resulting Permanent.

// TestETBChooseColor: a permanent with ETBChooseColor stores the picked
// color, and re-resolving the same printed card with a different scripted
// choice records the new color (i.e. the choice is per-instance, made at
// ETB-replacement time).
func TestETBChooseColor(t *testing.T) {
	name := "ETB Color Picker"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewEnchantment(name, "{1}{W}",
				mage.WithAbility(mage.ETBChooseColor("choose a color")),
			)
		})
	}

	tg := NewTestGame(t)
	tg.ChooseManaColor(PlayerA, core.Red)
	tg.AddCard(core.ZoneHand, PlayerA, name)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, name)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, name, 1)
	perm := tg.FindPermanentByName(name, tg.GetPlayer(PlayerA).PlayerID())
	if perm.ChosenColor != core.Red {
		t.Errorf("ChosenColor: got %v, want Red", perm.ChosenColor)
	}
}

// TestETBChooseColor_DifferentChoiceForNewInstance verifies that two
// AddCard calls of the same printed card receive independent choices —
// the choice is recorded per-Permanent, not per-Card.
func TestETBChooseColor_DifferentChoiceForNewInstance(t *testing.T) {
	name := "ETB Color Picker"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewEnchantment(name, "{1}{W}",
				mage.WithAbility(mage.ETBChooseColor("choose a color")),
			)
		})
	}

	tg := NewTestGame(t)
	tg.ChooseManaColor(PlayerA, core.Blue)  // first instance
	tg.ChooseManaColor(PlayerA, core.Green) // second instance
	tg.AddCard(core.ZoneHand, PlayerA, name)
	tg.AddCard(core.ZoneHand, PlayerA, name)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, name)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, name)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, name, 2)
	colors := map[core.Color]int{}
	pid := tg.GetPlayer(PlayerA).PlayerID()
	for _, p := range tg.AllBattlefield() {
		if p.Name() == name && p.ControllerID() == pid {
			colors[p.ChosenColor]++
		}
	}
	if colors[core.Blue] != 1 || colors[core.Green] != 1 {
		t.Errorf("expected one Blue and one Green chosen color, got %v", colors)
	}
}

// TestETBChooseColorOtherThan: the Thriving cycle pattern — controller must
// pick a color other than the land's own intrinsic color. If the player
// scripts the excluded color, the engine must reject it and pick a legal
// alternative.
func TestETBChooseColorOtherThan(t *testing.T) {
	name := "ETB Other-Color Picker"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewLand(name,
				mage.WithAbility(mage.ETBChooseColorOtherThan(
					"choose a color other than red", core.Red)),
			)
		})
	}

	tg := NewTestGame(t)
	tg.ChooseManaColor(PlayerA, core.Black)
	tg.AddCard(core.ZoneBattlefield, PlayerA, name)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	perm := tg.FindPermanentByName(name, tg.GetPlayer(PlayerA).PlayerID())
	if perm.ChosenColor != core.Black {
		t.Errorf("ChosenColor: got %v, want Black", perm.ChosenColor)
	}
}

// TestETBChooseColorOtherThan_RejectsExcluded: when the only scripted
// choice is the excluded color, the engine must fall back to a legal one
// rather than store the illegal value.
func TestETBChooseColorOtherThan_RejectsExcluded(t *testing.T) {
	name := "ETB Other-Color Picker"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewLand(name,
				mage.WithAbility(mage.ETBChooseColorOtherThan(
					"choose a color other than red", core.Red)),
			)
		})
	}

	tg := NewTestGame(t)
	// Script the illegal color; engine should ignore it and pick a legal one.
	tg.ChooseManaColor(PlayerA, core.Red)
	tg.AddCard(core.ZoneBattlefield, PlayerA, name)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	perm := tg.FindPermanentByName(name, tg.GetPlayer(PlayerA).PlayerID())
	if perm.ChosenColor == core.Red {
		t.Errorf("ChosenColor: got Red (excluded), want any non-Red color")
	}
	if perm.ChosenColor == core.Colorless || perm.ChosenColor == core.AnyColor {
		t.Errorf("ChosenColor: got sentinel %v, want a basic color", perm.ChosenColor)
	}
}

// TestETBChooseOpponent: in 2-player the opponent reference is fixed at
// ETB-replacement time and persists for the rest of the game (used by
// Nyxathid, Black Vise, Jihad, etc.).
func TestETBChooseOpponent(t *testing.T) {
	name := "ETB Opponent Picker"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewCreature(name, "{2}{B}", 4, 4,
				mage.WithAbility(mage.ETBChooseOpponent()),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, name)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	perm := tg.FindPermanentByName(name, tg.GetPlayer(PlayerA).PlayerID())
	wantPID := tg.GetPlayer(PlayerB).PlayerID()
	if perm.ChosenPlayer != wantPID {
		t.Errorf("ChosenPlayer: got %v, want PlayerB %v", perm.ChosenPlayer, wantPID)
	}
}

// TestETBChooseOpponent_PerInstance: choice is recorded per-Permanent.
// Two copies controlled by different players each pin to that player's
// opponent.
func TestETBChooseOpponent_PerInstance(t *testing.T) {
	name := "ETB Opponent Picker"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewCreature(name, "{2}{B}", 4, 4,
				mage.WithAbility(mage.ETBChooseOpponent()),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, name)
	tg.AddCard(core.ZoneBattlefield, PlayerB, name)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	pidA := tg.GetPlayer(PlayerA).PlayerID()
	pidB := tg.GetPlayer(PlayerB).PlayerID()
	for _, p := range tg.AllBattlefield() {
		if p.Name() != name {
			continue
		}
		switch p.ControllerID() {
		case pidA:
			if p.ChosenPlayer != pidB {
				t.Errorf("PlayerA's instance: ChosenPlayer = %v, want %v", p.ChosenPlayer, pidB)
			}
		case pidB:
			if p.ChosenPlayer != pidA {
				t.Errorf("PlayerB's instance: ChosenPlayer = %v, want %v", p.ChosenPlayer, pidA)
			}
		}
	}
}

// TestETBChooseCreatureType: Herald's Horn pattern — the scripted creature
// type is recorded on the artifact for later cost-reduction / reveal logic.
func TestETBChooseCreatureType(t *testing.T) {
	name := "ETB Type Picker"
	options := []string{"Goblin", "Elf", "Wizard", "Zombie"}
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewArtifact(name, "{3}",
				mage.WithAbility(mage.ETBChooseCreatureType(options)),
			)
		})
	}

	tg := NewTestGame(t)
	tg.ChooseString(PlayerA, "Elf")
	tg.AddCard(core.ZoneHand, PlayerA, name)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, name)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	perm := tg.FindPermanentByName(name, tg.GetPlayer(PlayerA).PlayerID())
	if perm.ChosenSubtype != "Elf" {
		t.Errorf("ChosenSubtype: got %q, want %q", perm.ChosenSubtype, "Elf")
	}
}

// TestETBChooseCreatureType_FallbackWhenIllegal: scripting a value not in
// the options falls back to the first option (the BasePlayer default).
func TestETBChooseCreatureType_FallbackWhenIllegal(t *testing.T) {
	name := "ETB Type Picker"
	options := []string{"Goblin", "Elf", "Wizard", "Zombie"}
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewArtifact(name, "{3}",
				mage.WithAbility(mage.ETBChooseCreatureType(options)),
			)
		})
	}

	tg := NewTestGame(t)
	tg.ChooseString(PlayerA, "Dragon") // not in options
	tg.AddCard(core.ZoneBattlefield, PlayerA, name)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	perm := tg.FindPermanentByName(name, tg.GetPlayer(PlayerA).PlayerID())
	if perm.ChosenSubtype != "Goblin" {
		t.Errorf("ChosenSubtype: got %q, want fallback %q", perm.ChosenSubtype, "Goblin")
	}
}

// TestETBChoiceVisibleToETBTriggers: the choice is recorded BEFORE
// "when this enters" triggers fire, so a triggered ability resolving on
// the same permanent can read the chosen value (CR 614.12).
func TestETBChoiceVisibleToETBTriggers(t *testing.T) {
	name := "ETB Choice + Trigger"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewCreature(name, "{2}{R}", 2, 2,
				mage.WithAbility(mage.ETBChooseColor("choose a color")),
				mage.WithAbility(mage.EntersBattlefieldTrigger(
					mage.FuncEffect(
						"verify color was chosen",
						mage.EffectProperties{},
						func(g *mage.Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							perm := g.FindPermanent(sourceID)
							if perm == nil || perm.ChosenColor == core.Colorless {
								t.Errorf("ETB trigger fired before choice was stored")
							}
							return nil
						},
					), false,
				)),
			)
		})
	}

	tg := NewTestGame(t)
	tg.ChooseManaColor(PlayerA, core.Blue)
	tg.AddCard(core.ZoneHand, PlayerA, name)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, name)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	perm := tg.FindPermanentByName(name, tg.GetPlayer(PlayerA).PlayerID())
	if perm.ChosenColor != core.Blue {
		t.Errorf("ChosenColor: got %v, want Blue", perm.ChosenColor)
	}
}
