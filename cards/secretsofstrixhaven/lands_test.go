package secretsofstrixhaven

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
)

// --- slow lands (enter tapped unless 2+ other lands) ---

func TestDeathcapGlade(t *testing.T) {
	t.Run("enters_tapped_when_fewer_than_two_other_lands", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Deathcap Glade")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Deathcap Glade", true)
	})

	t.Run("enters_untapped_when_controlling_two_or_more_other_lands", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Deathcap Glade")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Deathcap Glade", false)
	})

	t.Run("taps_for_black_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Deathcap Glade")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Deathcap Glade")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.GetPlayer(gametest.PlayerA).ManaPool()
		if pool.CountProducedThisTurn(core.Black) < 1 {
			t.Errorf("expected at least 1 black mana, got %d", pool.CountProducedThisTurn(core.Black))
		}
	})

	t.Run("taps_for_green_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Deathcap Glade")
		g.ChooseManaColor(gametest.PlayerA, core.Green)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Deathcap Glade")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.GetPlayer(gametest.PlayerA).ManaPool()
		if pool.CountProducedThisTurn(core.Green) < 1 {
			t.Errorf("expected at least 1 green mana, got %d", pool.CountProducedThisTurn(core.Green))
		}
	})
}

func TestDreamrootCascade(t *testing.T) {
	t.Run("enters_tapped_with_fewer_than_two_other_lands", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Dreamroot Cascade")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Dreamroot Cascade", true)
	})

	t.Run("enters_untapped_with_two_or_more_other_lands", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Dreamroot Cascade")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Dreamroot Cascade", false)
	})

	t.Run("taps_for_green_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dreamroot Cascade")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Dreamroot Cascade")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.GetPlayer(gametest.PlayerA).ManaPool()
		if pool.CountProducedThisTurn(core.Green) < 1 {
			t.Errorf("expected at least 1 green mana, got %d", pool.CountProducedThisTurn(core.Green))
		}
	})

	t.Run("taps_for_blue_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dreamroot Cascade")
		g.ChooseManaColor(gametest.PlayerA, core.Blue)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Dreamroot Cascade")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.GetPlayer(gametest.PlayerA).ManaPool()
		if pool.CountProducedThisTurn(core.Blue) < 1 {
			t.Errorf("expected at least 1 blue mana, got %d", pool.CountProducedThisTurn(core.Blue))
		}
	})
}

func TestShatteredSanctum(t *testing.T) {
	t.Run("enters_tapped_with_fewer_than_two_other_lands", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Shattered Sanctum")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Shattered Sanctum", true)
	})

	t.Run("enters_untapped_with_two_or_more_other_lands", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Shattered Sanctum")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Shattered Sanctum", false)
	})

	t.Run("taps_for_white_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Shattered Sanctum")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Shattered Sanctum")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.GetPlayer(gametest.PlayerA).ManaPool()
		if pool.CountProducedThisTurn(core.White) < 1 {
			t.Errorf("expected at least 1 white mana, got %d", pool.CountProducedThisTurn(core.White))
		}
	})

	t.Run("taps_for_black_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Shattered Sanctum")
		g.ChooseManaColor(gametest.PlayerA, core.Black)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Shattered Sanctum")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.GetPlayer(gametest.PlayerA).ManaPool()
		if pool.CountProducedThisTurn(core.Black) < 1 {
			t.Errorf("expected at least 1 black mana, got %d", pool.CountProducedThisTurn(core.Black))
		}
	})
}

func TestStormcarvedCoast(t *testing.T) {
	t.Run("enters_tapped_with_fewer_than_two_other_lands", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Stormcarved Coast")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Stormcarved Coast", true)
	})

	t.Run("enters_untapped_with_two_or_more_other_lands", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Stormcarved Coast")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Stormcarved Coast", false)
	})

	t.Run("taps_for_blue_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Stormcarved Coast")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Stormcarved Coast")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.GetPlayer(gametest.PlayerA).ManaPool()
		if pool.CountProducedThisTurn(core.Blue) < 1 {
			t.Errorf("expected at least 1 blue mana, got %d", pool.CountProducedThisTurn(core.Blue))
		}
	})

	t.Run("taps_for_red_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Stormcarved Coast")
		g.ChooseManaColor(gametest.PlayerA, core.Red)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Stormcarved Coast")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.GetPlayer(gametest.PlayerA).ManaPool()
		if pool.CountProducedThisTurn(core.Red) < 1 {
			t.Errorf("expected at least 1 red mana, got %d", pool.CountProducedThisTurn(core.Red))
		}
	})
}

func TestSundownPass(t *testing.T) {
	t.Run("enters_tapped_with_fewer_than_two_other_lands", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Sundown Pass")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Sundown Pass", true)
	})

	t.Run("enters_untapped_with_two_or_more_other_lands", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Sundown Pass")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Sundown Pass", false)
	})

	t.Run("taps_for_red_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sundown Pass")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Sundown Pass")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.GetPlayer(gametest.PlayerA).ManaPool()
		if pool.CountProducedThisTurn(core.Red) < 1 {
			t.Errorf("expected at least 1 red mana, got %d", pool.CountProducedThisTurn(core.Red))
		}
	})

	t.Run("taps_for_white_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sundown Pass")
		g.ChooseManaColor(gametest.PlayerA, core.White)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Sundown Pass")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.GetPlayer(gametest.PlayerA).ManaPool()
		if pool.CountProducedThisTurn(core.White) < 1 {
			t.Errorf("expected at least 1 white mana, got %d", pool.CountProducedThisTurn(core.White))
		}
	})
}

// --- surveil dual lands (always enter tapped) ---

func TestFieldsOfStrife(t *testing.T) {
	t.Run("always_enters_tapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Fields of Strife")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Fields of Strife", true)
	})

	t.Run("taps_for_red_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fields of Strife")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Fields of Strife")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.GetPlayer(gametest.PlayerA).ManaPool()
		if pool.CountProducedThisTurn(core.Red) < 1 {
			t.Errorf("expected at least 1 red mana, got %d", pool.CountProducedThisTurn(core.Red))
		}
	})

	t.Run("taps_for_white_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fields of Strife")
		g.ChooseManaColor(gametest.PlayerA, core.White)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Fields of Strife")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.GetPlayer(gametest.PlayerA).ManaPool()
		if pool.CountProducedThisTurn(core.White) < 1 {
			t.Errorf("expected at least 1 white mana, got %d", pool.CountProducedThisTurn(core.White))
		}
	})

	t.Run("surveil_ability_puts_card_into_graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fields of Strife")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		// choose to put top card into graveyard (surveil: bottom = graveyard)
		g.ChooseScry(gametest.PlayerA, []string{"Grizzly Bears"}, nil)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Fields of Strife")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	})

	t.Run("surveil_ability_keeps_card_on_top", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fields of Strife")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		// choose to keep top card on top (surveil: topOrder = keep)
		g.ChooseScry(gametest.PlayerA, nil, []string{"Grizzly Bears"})
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Fields of Strife")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 0)
		g.AssertLibraryTop(gametest.PlayerA, "Grizzly Bears")
	})
}

func TestForumOfAmity(t *testing.T) {
	t.Run("always_enters_tapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Forum of Amity")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Forum of Amity", true)
	})

	t.Run("taps_for_white_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forum of Amity")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Forum of Amity")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.GetPlayer(gametest.PlayerA).ManaPool()
		if pool.CountProducedThisTurn(core.White) < 1 {
			t.Errorf("expected at least 1 white mana, got %d", pool.CountProducedThisTurn(core.White))
		}
	})

	t.Run("taps_for_black_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forum of Amity")
		g.ChooseManaColor(gametest.PlayerA, core.Black)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Forum of Amity")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.GetPlayer(gametest.PlayerA).ManaPool()
		if pool.CountProducedThisTurn(core.Black) < 1 {
			t.Errorf("expected at least 1 black mana, got %d", pool.CountProducedThisTurn(core.Black))
		}
	})

	t.Run("surveil_ability_puts_card_into_graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forum of Amity")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.ChooseScry(gametest.PlayerA, []string{"Grizzly Bears"}, nil)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Forum of Amity")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}

func TestParadoxGardens(t *testing.T) {
	t.Run("always_enters_tapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Paradox Gardens")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Paradox Gardens", true)
	})

	t.Run("taps_for_green_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Paradox Gardens")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Paradox Gardens")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.GetPlayer(gametest.PlayerA).ManaPool()
		if pool.CountProducedThisTurn(core.Green) < 1 {
			t.Errorf("expected at least 1 green mana, got %d", pool.CountProducedThisTurn(core.Green))
		}
	})

	t.Run("taps_for_blue_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Paradox Gardens")
		g.ChooseManaColor(gametest.PlayerA, core.Blue)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Paradox Gardens")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.GetPlayer(gametest.PlayerA).ManaPool()
		if pool.CountProducedThisTurn(core.Blue) < 1 {
			t.Errorf("expected at least 1 blue mana, got %d", pool.CountProducedThisTurn(core.Blue))
		}
	})
}

func TestSpectacleSummit(t *testing.T) {
	t.Run("always_enters_tapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Spectacle Summit")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Spectacle Summit", true)
	})

	t.Run("taps_for_blue_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Spectacle Summit")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Spectacle Summit")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.GetPlayer(gametest.PlayerA).ManaPool()
		if pool.CountProducedThisTurn(core.Blue) < 1 {
			t.Errorf("expected at least 1 blue mana, got %d", pool.CountProducedThisTurn(core.Blue))
		}
	})

	t.Run("taps_for_red_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Spectacle Summit")
		g.ChooseManaColor(gametest.PlayerA, core.Red)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Spectacle Summit")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.GetPlayer(gametest.PlayerA).ManaPool()
		if pool.CountProducedThisTurn(core.Red) < 1 {
			t.Errorf("expected at least 1 red mana, got %d", pool.CountProducedThisTurn(core.Red))
		}
	})
}

func TestTitansGrave(t *testing.T) {
	t.Run("always_enters_tapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Titan's Grave")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Titan's Grave", true)
	})

	t.Run("taps_for_black_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Titan's Grave")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Titan's Grave")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.GetPlayer(gametest.PlayerA).ManaPool()
		if pool.CountProducedThisTurn(core.Black) < 1 {
			t.Errorf("expected at least 1 black mana, got %d", pool.CountProducedThisTurn(core.Black))
		}
	})

	t.Run("taps_for_green_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Titan's Grave")
		g.ChooseManaColor(gametest.PlayerA, core.Green)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Titan's Grave")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.GetPlayer(gametest.PlayerA).ManaPool()
		if pool.CountProducedThisTurn(core.Green) < 1 {
			t.Errorf("expected at least 1 green mana, got %d", pool.CountProducedThisTurn(core.Green))
		}
	})

	t.Run("surveil_ability_puts_card_into_graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Titan's Grave")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.ChooseScry(gametest.PlayerA, []string{"Grizzly Bears"}, nil)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Titan's Grave")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}

// --- Great Hall of the Biblioplex ---

func TestGreatHallOfTheBiblioplex(t *testing.T) {
	t.Run("taps_for_colorless_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Great Hall of the Biblioplex")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Great Hall of the Biblioplex")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.GetPlayer(gametest.PlayerA).ManaPool()
		if pool.CountProducedThisTurn(core.Colorless) < 1 {
			t.Errorf("expected at least 1 colorless mana, got %d", pool.CountProducedThisTurn(core.Colorless))
		}
	})
}

// --- Skycoach Waypoint ---

func TestSkycoachWaypoint(t *testing.T) {
	t.Run("taps_for_colorless_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Skycoach Waypoint")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Skycoach Waypoint")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.GetPlayer(gametest.PlayerA).ManaPool()
		if pool.CountProducedThisTurn(core.Colorless) < 1 {
			t.Errorf("expected at least 1 colorless mana, got %d", pool.CountProducedThisTurn(core.Colorless))
		}
	})

	t.Run("prepared_ability_sets_prepared_on_eligible_creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Skycoach Waypoint")
		// Add mana sources for the {3} cost.
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
		// Elite Interceptor has a prepared spell, making it a valid target.
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Elite Interceptor // Rejoinder")
		g.ChoosePermanent(gametest.PlayerA, "Elite Interceptor // Rejoinder")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Skycoach Waypoint", "Elite Interceptor // Rejoinder")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Elite Interceptor // Rejoinder", core.AttrPrepared, true)
	})
}

// --- Petrified Hamlet ---

func TestPetrifiedHamlet(t *testing.T) {
	t.Run("taps_for_colorless_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.ChooseString(gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Petrified Hamlet")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Petrified Hamlet")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		pool := g.GetPlayer(gametest.PlayerA).ManaPool()
		if pool.CountProducedThisTurn(core.Colorless) < 1 {
			t.Errorf("expected at least 1 colorless mana, got %d", pool.CountProducedThisTurn(core.Colorless))
		}
	})

	t.Run("chosen_land_name_is_stored_on_entry", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.ChooseString(gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Petrified Hamlet")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		perm := g.FindPermanentByName("Petrified Hamlet", g.GetPlayer(gametest.PlayerA).PlayerID())
		if perm == nil {
			t.Fatal("Petrified Hamlet not found")
		}
		if perm.ChosenSubtype != "Island" {
			t.Errorf("expected chosen name 'Island', got '%s'", perm.ChosenSubtype)
		}
	})
}
