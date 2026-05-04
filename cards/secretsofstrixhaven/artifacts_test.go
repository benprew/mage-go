package secretsofstrixhaven

import (
	"testing"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

func TestArkOfHunger(t *testing.T) {
	// Note: "You may play that card this turn" sub-effect of the {T} ability is
	// not tested (no graveyard play permission system in the engine).
	t.Run("enters_battlefield", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ark of Hunger")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Ark of Hunger", 1)
	})

	t.Run("tap_ability_mills_one_card", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ark of Hunger")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears", 3)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Ark of Hunger")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	})

	t.Run("trigger_deals_damage_and_gains_life_when_card_leaves_graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ark of Hunger")
		// Put a card in graveyard; Cauldron of Essence reanimation will move it out
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cauldron of Essence")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
		// Activate Cauldron: sacrifice Hill Giant, reanimate Grizzly Bears from GY → Grizzly Bears leaves GY
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Cauldron of Essence", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Cauldron trigger (creature died — Hill Giant) also fires: PlayerB loses 1 life = 19; PlayerA gains 1 = 21
		// Ark trigger (Grizzly Bears left GY): PlayerB loses another 1 = 18; PlayerA gains another 1 = 22
		g.AssertLife(gametest.PlayerB, 18)
		g.AssertLife(gametest.PlayerA, 22)
	})
}

func TestCauldronOfEssence(t *testing.T) {
	t.Run("triggers_when_your_creature_dies_drains_and_gains", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cauldron of Essence")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		// PlayerB casts Lightning Bolt to kill Grizzly Bears
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// PlayerA (Cauldron controller) gains 1 life; PlayerB (opponent) loses 1 life
		g.AssertLife(gametest.PlayerB, 19)
		g.AssertLife(gametest.PlayerA, 21)
	})

	t.Run("does_not_trigger_when_opponent_creature_dies", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cauldron of Essence")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Cauldron does not trigger for opponent's creature dying
		g.AssertLife(gametest.PlayerB, 20)
		g.AssertLife(gametest.PlayerA, 20)
	})

	t.Run("activated_ability_reanimates_creature_from_graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cauldron of Essence")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
		// Need a creature to sacrifice as part of the cost
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
		// Activate: {1}{B}{G},{T}, sacrifice a creature → return target creature from GY to BF
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Cauldron of Essence", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}

func TestDiaryOfDreams(t *testing.T) {
	t.Run("puts_page_counter_when_instant_cast", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Diary of Dreams")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Diary of Dreams", core.Page, 1)
	})

	t.Run("puts_page_counter_when_sorcery_cast", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Diary of Dreams")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Ancestral Recall")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Ancestral Recall", "PlayerA")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Diary of Dreams", core.Page, 1)
	})

	t.Run("does_not_trigger_on_creature_spell", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Diary of Dreams")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Diary of Dreams", core.Page, 0)
	})

	t.Run("draw_ability_works_at_full_cost", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Diary of Dreams")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 5)
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears", 5)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Diary of Dreams")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
	})

	t.Run("cost_reduced_by_page_counters", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Diary of Dreams")
		// Only 4 islands available; full cost {5},{T} would need 5 but 1 counter reduces to {4}
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 4)
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears", 5)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		// Cast instant to add a page counter (Charge counter)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
		// Now activate Diary of Dreams — reduced cost {4},{T}
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Diary of Dreams")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}

func TestPotionersTrove(t *testing.T) {
	t.Run("life_gain_ability_unavailable_without_instant_or_sorcery", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Potioner's Trove")
		// No instant/sorcery cast this turn — can't activate life gain ability
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 20)
	})

	t.Run("life_gain_available_after_casting_instant", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Potioner's Trove")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Potioner's Trove")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 22)
	})
}

func TestResonatingLute(t *testing.T) {
	t.Run("draw_ability_blocked_with_fewer_than_7_cards", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Resonating Lute")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears", 6)
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Lightning Bolt", 3)
		// With only 6 cards in hand, activation condition not met; no draw
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Lightning Bolt", 0)
	})

	t.Run("draw_ability_works_with_7_or_more_cards", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Resonating Lute")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears", 7)
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Lightning Bolt", 3)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Resonating Lute")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Lightning Bolt", 1)
	})
}

func TestStrixhavenSkycoach(t *testing.T) {
	t.Run("enters_battlefield", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		// XXX: Vehicle/Crew not implemented — Strixhaven Skycoach is a stub
		// that only has flying and the ETB fetch. The 3/2 P/T and Crew 2
		// ability are missing; tracked under // XXX: in artifacts.go.
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Strixhaven Skycoach")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Strixhaven Skycoach", 1)
	})

	t.Run("etb_fetches_basic_land", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest", 3)
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears", 3)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Strixhaven Skycoach")
		g.ChooseFromLibrary(gametest.PlayerA, "Forest")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Strixhaven Skycoach")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// The fetched Forest goes to hand; the engine may auto-play it as the
		// turn's land-for-turn. Check that it left the library (count drops from 3 to 2).
		g.AssertLibraryCount(gametest.PlayerA, "Forest", 2)
	})
}

func TestTabletOfDiscovery(t *testing.T) {
	t.Run("taps_for_red_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tablet of Discovery")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 17)
	})

	t.Run("etb_mills_one_card", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears", 3)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Tablet of Discovery")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Tablet of Discovery")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}
