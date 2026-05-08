package gametest

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// Engine-level tests for the conditional spell cost reduction system
// (CR 601.2f). Each test registers ad-hoc cards that exercise one shape
// of reducer (subtype, keyword, self by graveyard count, dynamic by total
// power, condition predicate, "if you control X", chosen subtype).
//
// Tests inspect the reduction directly via Game.ConditionalSpellCostReduction
// and end-to-end via the cast pipeline (mana pool drain after a successful cast).

// findHandCard returns the first card in playerID's hand matching name.
func findHandCard(tg *TestGame, p PlayerRef, name string) mage.Card {
	pl := tg.Game.GetPlayer(tg.getPlayerID(p))
	for _, c := range pl.Hand() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

// stagePrecombat advances to PrecombatMain so static abilities are applied.
func stagePrecombat(tg *TestGame) {
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()
}

// ===== Subtype reduction (Dragonlord's Servant pattern) =====
//
// "Dragon spells you cast cost {1} less to cast."

func TestCostReductionBySubtype(t *testing.T) {
	reducer := "TC Subtype Reducer"
	dragon := "TC Dragon Spell"
	other := "TC Non-Dragon Spell"
	if !mage.CardRegistered(reducer) {
		mage.Register(reducer, func() mage.Card {
			return mage.NewEnchantment(reducer, "{1}{R}",
				mage.WithStaticAbility(mage.ReduceSpellCostStatic(
					mage.SpellsAnd(mage.SpellHasType(core.TypeCreature), mage.SpellHasSubType("Dragon")),
					mage.FixedAmount(1), nil,
				)),
			)
		})
	}
	if !mage.CardRegistered(dragon) {
		mage.Register(dragon, func() mage.Card {
			return mage.NewCreature(dragon, "{3}{R}", 4, 4, mage.WithSubTypes("Dragon"))
		})
	}
	if !mage.CardRegistered(other) {
		mage.Register(other, func() mage.Card {
			return mage.NewCreature(other, "{3}{R}", 4, 4, mage.WithSubTypes("Beast"))
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, reducer)
	tg.AddCard(core.ZoneHand, PlayerA, dragon)
	tg.AddCard(core.ZoneHand, PlayerA, other)
	stagePrecombat(tg)

	pid := tg.getPlayerID(PlayerA)
	if got := tg.ConditionalSpellCostReduction(pid, findHandCard(tg, PlayerA, dragon)); got != 1 {
		t.Errorf("dragon reduction: got %d, want 1", got)
	}
	if got := tg.ConditionalSpellCostReduction(pid, findHandCard(tg, PlayerA, other)); got != 0 {
		t.Errorf("non-dragon reduction: got %d, want 0", got)
	}
}

// ===== Keyword reduction (Warden of Evos Isle pattern) =====
//
// "Creature spells with flying you cast cost {1} less to cast."

func TestCostReductionByKeyword(t *testing.T) {
	warden := "TC Warden"
	flyer := "TC Flyer"
	ground := "TC Ground"
	if !mage.CardRegistered(warden) {
		mage.Register(warden, func() mage.Card {
			return mage.NewCreature(warden, "{1}{U}", 2, 2,
				mage.WithKeyword(core.Flying),
				mage.WithStaticAbility(mage.ReduceSpellCostStatic(
					mage.SpellsAnd(mage.SpellHasType(core.TypeCreature), mage.SpellHasKeyword(core.Flying)),
					mage.FixedAmount(1), nil,
				)),
			)
		})
	}
	if !mage.CardRegistered(flyer) {
		mage.Register(flyer, func() mage.Card {
			return mage.NewCreature(flyer, "{2}{U}", 2, 1,
				mage.WithSubTypes("Bird"), mage.WithKeyword(core.Flying))
		})
	}
	if !mage.CardRegistered(ground) {
		mage.Register(ground, func() mage.Card {
			return mage.NewCreature(ground, "{2}{U}", 2, 2,
				mage.WithSubTypes("Beast"))
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, warden)
	tg.AddCard(core.ZoneHand, PlayerA, flyer)
	tg.AddCard(core.ZoneHand, PlayerA, ground)
	stagePrecombat(tg)

	pid := tg.getPlayerID(PlayerA)
	if got := tg.ConditionalSpellCostReduction(pid, findHandCard(tg, PlayerA, flyer)); got != 1 {
		t.Errorf("flyer reduction: got %d, want 1", got)
	}
	if got := tg.ConditionalSpellCostReduction(pid, findHandCard(tg, PlayerA, ground)); got != 0 {
		t.Errorf("non-flyer reduction: got %d, want 0", got)
	}

	// Static reducer must not affect the opponent's spells.
	tg.AddCard(core.ZoneHand, PlayerB, flyer)
	if got := tg.ConditionalSpellCostReduction(tg.getPlayerID(PlayerB), findHandCard(tg, PlayerB, flyer)); got != 0 {
		t.Errorf("opponent flyer reduction: got %d, want 0", got)
	}
}

// ===== Self-reduction by graveyard count (Cryptic Serpent pattern) =====
//
// "This costs {1} less to cast for each instant and sorcery card in your graveyard."

func TestCostReductionSelfByGraveyardCount(t *testing.T) {
	serpent := "TC Serpent"
	jolt := "TC Tiny Instant"
	if !mage.CardRegistered(jolt) {
		mage.Register(jolt, func() mage.Card {
			return mage.NewInstant(jolt, "{U}", mage.NewSpellAbility(mage.GainLife(1)))
		})
	}
	isInstantOrSorcery := mage.NewCardFilter("instant or sorcery", func(c mage.Card) bool {
		return c.HasType(core.TypeInstant) || c.HasType(core.TypeSorcery)
	})
	if !mage.CardRegistered(serpent) {
		mage.Register(serpent, func() mage.Card {
			return mage.NewCreature(serpent, "{6}{U}", 6, 5,
				mage.WithSubTypes("Serpent"),
				mage.WithSelfCostReduction(
					mage.AmountByGraveyardCount(isInstantOrSorcery), nil,
				),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneGraveyard, PlayerA, jolt, 4)
	tg.AddCard(core.ZoneHand, PlayerA, serpent)
	stagePrecombat(tg)

	pid := tg.getPlayerID(PlayerA)
	c := findHandCard(tg, PlayerA, serpent)
	if got := tg.ConditionalSpellCostReduction(pid, c); got != 4 {
		t.Errorf("serpent reduction with 4 in yard: got %d, want 4", got)
	}

	// Empty graveyard → no reduction.
	tg2 := NewTestGame(t)
	tg2.AddCard(core.ZoneHand, PlayerA, serpent)
	stagePrecombat(tg2)
	pid2 := tg2.getPlayerID(PlayerA)
	c2 := findHandCard(tg2, PlayerA, serpent)
	if got := tg2.ConditionalSpellCostReduction(pid2, c2); got != 0 {
		t.Errorf("serpent reduction with empty yard: got %d, want 0", got)
	}
}

// ===== Dynamic X reduction (Ghalta, Primal Hunger pattern) =====
//
// "This spell costs {X} less to cast, where X is the total power of creatures
// you control."

func TestCostReductionSelfByTotalPower(t *testing.T) {
	beef := "TC Beef"
	ghalta := "TC Ghalta"
	if !mage.CardRegistered(beef) {
		mage.Register(beef, func() mage.Card {
			return mage.NewCreature(beef, "{2}{G}", 4, 4, mage.WithSubTypes("Beast"))
		})
	}
	if !mage.CardRegistered(ghalta) {
		mage.Register(ghalta, func() mage.Card {
			return mage.NewCreature(ghalta, "{10}{G}{G}{G}", 12, 12,
				mage.WithSubTypes("Dinosaur"),
				mage.WithSelfCostReduction(
					mage.AmountByTotalPower(mage.IsCreature), nil,
				),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, beef, 3) // 12 total power
	tg.AddCard(core.ZoneHand, PlayerA, ghalta)
	stagePrecombat(tg)

	pid := tg.getPlayerID(PlayerA)
	c := findHandCard(tg, PlayerA, ghalta)
	// Generic = 10; reduction returns 12 but caps to 10.
	if got := tg.ConditionalSpellCostReduction(pid, c); got != 10 {
		t.Errorf("ghalta reduction with 12 power: got %d, want 10", got)
	}

	// One Beast (4 power) → 4 reduction.
	tg2 := NewTestGame(t)
	tg2.AddCard(core.ZoneBattlefield, PlayerA, beef, 1)
	tg2.AddCard(core.ZoneHand, PlayerA, ghalta)
	stagePrecombat(tg2)
	pid2 := tg2.getPlayerID(PlayerA)
	c2 := findHandCard(tg2, PlayerA, ghalta)
	if got := tg2.ConditionalSpellCostReduction(pid2, c2); got != 4 {
		t.Errorf("ghalta reduction with 4 power: got %d, want 4", got)
	}

	// End-to-end cast verifies the cost actually shrinks.
	tg3 := NewTestGame(t)
	tg3.AddCard(core.ZoneBattlefield, PlayerA, beef, 3)
	tg3.AddCard(core.ZoneHand, PlayerA, ghalta)
	tg3.AddCard(core.ZoneBattlefield, PlayerA, "Forest", 3)
	tg3.CastSpell(1, core.PrecombatMain, PlayerA, ghalta)
	tg3.StopAt(1, core.EndCombat)
	tg3.Execute()
	tg3.AssertPermanentCount(PlayerA, ghalta, 1) // 4 beasts (3 + Ghalta) on battlefield, +Ghalta = 1 of name
}

// ===== Conditional reduction (Bone Picker pattern) =====
//
// "This costs {3} less to cast if a creature died this turn."

func TestCostReductionIfCreatureDiedThisTurn(t *testing.T) {
	picker := "TC Picker"
	if !mage.CardRegistered(picker) {
		mage.Register(picker, func() mage.Card {
			return mage.NewCreature(picker, "{4}{B}", 3, 2,
				mage.WithSubTypes("Rogue"),
				mage.WithSelfCostReduction(mage.FixedAmount(3), mage.CondCreatureDiedThisTurn()),
			)
		})
	}

	// No creature has died this turn.
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, picker)
	stagePrecombat(tg)
	pid := tg.getPlayerID(PlayerA)
	c := findHandCard(tg, PlayerA, picker)
	if got := tg.ConditionalSpellCostReduction(pid, c); got != 0 {
		t.Errorf("picker reduction with no death: got %d, want 0", got)
	}

	// Simulate a creature death by bumping the engine's death counter via the
	// engine's normal Sacrifice path: put a creature on the battlefield then
	// destroy it. Easiest: use SetDeaths directly. The engine exposes the
	// CreatureDeaths counter via Game.creatureDeathsThisTurn (private). To
	// exercise it without reaching into internals, just kill a permanent by
	// putting it onto the battlefield and using Game.Destroy.
	bear := "TC Picker Doomed Bear"
	if !mage.CardRegistered(bear) {
		mage.Register(bear, func() mage.Card {
			return mage.NewCreature(bear, "{1}{G}", 2, 1, mage.WithSubTypes("Bear"))
		})
	}
	tg2 := NewTestGame(t)
	tg2.AddCard(core.ZoneBattlefield, PlayerB, bear)
	tg2.AddCard(core.ZoneHand, PlayerA, picker)
	stagePrecombat(tg2)
	bp := tg2.FindPermanentByName(bear, tg2.GetPlayer(PlayerB).PlayerID())
	tg2.DestroyPermanent(bp)
	tg2.CheckStateBasedActions()

	pid2 := tg2.getPlayerID(PlayerA)
	c2 := findHandCard(tg2, PlayerA, picker)
	if got := tg2.ConditionalSpellCostReduction(pid2, c2); got != 3 {
		t.Errorf("picker reduction after death: got %d, want 3", got)
	}
}

func TestCostReductionIfCardLeftYourGraveyardThisTurn(t *testing.T) {
	spell := "TC Graveyard Left Discount"
	if !mage.CardRegistered(spell) {
		mage.Register(spell, func() mage.Card {
			return mage.NewInstant(spell, "{3}{R}",
				mage.NewSpellAbility(mage.DealDamage(mage.Fixed(1))),
				mage.WithSelfCostReduction(mage.FixedAmount(2), mage.CondCardLeftYourGraveyardThisTurn()),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, spell)
	stagePrecombat(tg)
	pid := tg.getPlayerID(PlayerA)
	if got := tg.ConditionalSpellCostReduction(pid, findHandCard(tg, PlayerA, spell)); got != 0 {
		t.Errorf("reduction before graveyard move: got %d, want 0", got)
	}

	tg2 := NewTestGame(t)
	tg2.AddCard(core.ZoneHand, PlayerA, spell)
	tg2.AddCard(core.ZoneGraveyard, PlayerA, "Grizzly Bears")
	stagePrecombat(tg2)
	pid2 := tg2.getPlayerID(PlayerA)
	gyCard := tg2.GetPlayer(PlayerA).Graveyard()[0]
	removed, ok := tg2.MoveFromGraveyard(pid2, gyCard.ID(), core.ZoneExile)
	if !ok {
		t.Fatalf("MoveFromGraveyard failed")
	}
	tg2.ExileCard(removed, uuid.Nil)

	if got := tg2.ConditionalSpellCostReduction(pid2, findHandCard(tg2, PlayerA, spell)); got != 2 {
		t.Errorf("reduction after graveyard move: got %d, want 2", got)
	}
}

// ===== Condition: "if you control X" (Wizard's Retort / Winged Words) =====

func TestCostReductionIfControlsMatching(t *testing.T) {
	winged := "TC Winged Words Stand-in"
	bird := "TC Flyer Bird"
	if !mage.CardRegistered(bird) {
		mage.Register(bird, func() mage.Card {
			return mage.NewCreature(bird, "{1}{U}", 1, 1,
				mage.WithSubTypes("Bird"), mage.WithKeyword(core.Flying))
		})
	}
	if !mage.CardRegistered(winged) {
		mage.Register(winged, func() mage.Card {
			return mage.NewSorcery(winged, "{2}{U}{U}",
				mage.NewSpellAbility(mage.GainLife(1)),
				mage.WithSelfCostReduction(
					mage.FixedAmount(2),
					mage.CondControlsMatching(mage.HasKeywordFilter(core.Flying)),
				),
			)
		})
	}

	// With a flyer in play.
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, bird)
	tg.AddCard(core.ZoneHand, PlayerA, winged)
	stagePrecombat(tg)
	pid := tg.getPlayerID(PlayerA)
	c := findHandCard(tg, PlayerA, winged)
	if got := tg.ConditionalSpellCostReduction(pid, c); got != 2 {
		t.Errorf("winged reduction with flyer: got %d, want 2", got)
	}

	// Without a flyer.
	tg2 := NewTestGame(t)
	tg2.AddCard(core.ZoneHand, PlayerA, winged)
	stagePrecombat(tg2)
	pid2 := tg2.getPlayerID(PlayerA)
	c2 := findHandCard(tg2, PlayerA, winged)
	if got := tg2.ConditionalSpellCostReduction(pid2, c2); got != 0 {
		t.Errorf("winged reduction without flyer: got %d, want 0", got)
	}
}

// ===== Chosen-subtype reduction (Herald's Horn pattern) =====
//
// "Creature spells you cast of the chosen type cost {1} less to cast."
// (The "as it enters, choose a creature type" replacement is exercised in
// the card test; here we set ChosenSubtype directly to isolate the reducer.)

func TestCostReductionByChosenSubtype(t *testing.T) {
	horn := "TC Horn"
	dragon := "TC Horn Dragon"
	knight := "TC Horn Knight"
	if !mage.CardRegistered(horn) {
		mage.Register(horn, func() mage.Card {
			return mage.NewArtifact(horn, "{3}",
				mage.WithStaticAbility(mage.ReduceSpellCostStatic(
					mage.SpellsAnd(mage.SpellHasType(core.TypeCreature), mage.SpellSubTypeMatchesChosen()),
					mage.FixedAmount(1), nil,
				)),
			)
		})
	}
	if !mage.CardRegistered(dragon) {
		mage.Register(dragon, func() mage.Card {
			return mage.NewCreature(dragon, "{3}{R}", 4, 4, mage.WithSubTypes("Dragon"))
		})
	}
	if !mage.CardRegistered(knight) {
		mage.Register(knight, func() mage.Card {
			return mage.NewCreature(knight, "{3}{W}", 3, 3, mage.WithSubTypes("Knight"))
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, horn)
	tg.AddCard(core.ZoneHand, PlayerA, dragon)
	tg.AddCard(core.ZoneHand, PlayerA, knight)
	stagePrecombat(tg)
	hornPerm := tg.FindPermanentByName(horn, tg.GetPlayer(PlayerA).PlayerID())
	hornPerm.ChosenSubtype = "Dragon"

	pid := tg.getPlayerID(PlayerA)
	if got := tg.ConditionalSpellCostReduction(pid, findHandCard(tg, PlayerA, dragon)); got != 1 {
		t.Errorf("dragon (chosen Dragon) reduction: got %d, want 1", got)
	}
	if got := tg.ConditionalSpellCostReduction(pid, findHandCard(tg, PlayerA, knight)); got != 0 {
		t.Errorf("knight (chosen Dragon) reduction: got %d, want 0", got)
	}
}

// ===== Reduction never affects colored mana =====

func TestCostReductionDoesNotAffectColored(t *testing.T) {
	mega := "TC Big Reducer"
	huge := "TC Huge Spell"
	if !mage.CardRegistered(mega) {
		mage.Register(mega, func() mage.Card {
			return mage.NewEnchantment(mega, "{1}{U}",
				mage.WithStaticAbility(mage.ReduceSpellCostStatic(
					mage.SpellHasType(core.TypeCreature),
					mage.FixedAmount(99), nil,
				)),
			)
		})
	}
	if !mage.CardRegistered(huge) {
		mage.Register(huge, func() mage.Card {
			return mage.NewCreature(huge, "{4}{U}{U}", 5, 5, mage.WithSubTypes("Whale"))
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, mega)
	tg.AddCard(core.ZoneHand, PlayerA, huge)
	stagePrecombat(tg)

	pid := tg.getPlayerID(PlayerA)
	c := findHandCard(tg, PlayerA, huge)
	// Reduction returns 99 but is capped to printed generic (4).
	if got := tg.ConditionalSpellCostReduction(pid, c); got != 4 {
		t.Errorf("huge spell capped reduction: got %d, want 4", got)
	}
	// Colored portion is unchanged (the cap is on generic only).
	if c.ManaCost().Blue != 2 {
		t.Errorf("blue requirement unchanged: got %d, want 2", c.ManaCost().Blue)
	}
}

// ===== Multiple reducers stack additively =====
//
// Two Dragonlord's-Servant-style enchantments → {2} less on Dragon spells.

func TestCostReductionStacksAdditively(t *testing.T) {
	stacker := "TC Stack Reducer"
	dragon := "TC Stack Dragon"
	if !mage.CardRegistered(stacker) {
		mage.Register(stacker, func() mage.Card {
			return mage.NewEnchantment(stacker, "{R}",
				mage.WithStaticAbility(mage.ReduceSpellCostStatic(
					mage.SpellHasSubType("Dragon"),
					mage.FixedAmount(1), nil,
				)),
			)
		})
	}
	if !mage.CardRegistered(dragon) {
		mage.Register(dragon, func() mage.Card {
			return mage.NewCreature(dragon, "{4}{R}", 5, 5, mage.WithSubTypes("Dragon"))
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, stacker, 2)
	tg.AddCard(core.ZoneHand, PlayerA, dragon)
	stagePrecombat(tg)

	pid := tg.getPlayerID(PlayerA)
	if got := tg.ConditionalSpellCostReduction(pid, findHandCard(tg, PlayerA, dragon)); got != 2 {
		t.Errorf("two stackers reduction: got %d, want 2", got)
	}
}
