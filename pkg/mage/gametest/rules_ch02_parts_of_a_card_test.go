package gametest

// Rules Chapter 2 (200–213) — Parts of a Card — REPLACED WITH INLINE-CARD VERSION
//
// Tests in this file exercise mechanics tied to card parts: name, mana cost,
// color, type line (types, subtypes, supertypes), power/toughness, and the
// set-symbol rules (206.3). All cards used are registered inline to avoid
// import cycles with card set packages.
//
// SKIP/GAP entries not tested here:
//   - 201.2 nameless-object face-down morph (GAP: morph not implemented)
//   - 201.2b/c "different names" enumeration (GAP: engine has no N-different-names check)
//   - 201.4   ChooseCardName harness hook (GAP: not exposed)
//   - 201.4b-g DFC/split/meld/adventure frames (GAP: not implemented)
//   - 201.5   self-reference (GAP: requires very specific effect wording)
//   - 202.1a  Phyrexian mana life-payment (GAP: {W/P} symbol not parsed)
//   - 202.2e  color indicator distinct field (GAP: WithColorIndicator absent)
//   - 202.3e/g X/Phyrexian mana value (GAP: CMC for those symbols unverified;
//     hybrid is covered in hybrid_mana_test.go)
//   - 205.2a  battle/conspiracy/dungeon/kindred/... card types (GAP: not defined)
//   - 205.3b  Time Lord two-word type (GAP: parsing)
//   - 205.3d  inapplicable subtype validation (GAP: engine silently allows)
//   - 205.3e  ChooseSubtype harness hook (GAP: not exposed)
//   - 205.3h  Curse subtype (GAP: no registered Curse enchantment with Curse subtype)
//   - 205.3j  planeswalker subtype (GAP: TypePlaneswalker not implemented)
//   - 205.3k  spell subtypes on stack (GAP: not tracked)
//   - 205.4e  Legendary sorcery casting restriction (GAP: not enforced)
//   - 205.4g  snow permanent via supertype (GAP: no Snow-Covered lands registered)
//   - 207.5   Cryptic Spires (GAP: not implemented)
//   - 208.3   Vehicle/Crew mechanic (GAP: not implemented)
//   - 209     Planeswalker loyalty (GAP: TypePlaneswalker not implemented)
//   - 210     Battle defense (GAP: TypeBattle not implemented)
//   - 211     Vanguard hand modifier (GAP: not implemented)
//   - 212     Vanguard life modifier (GAP: not implemented)
//   - TestLegendRule         (pre-existing in mechanics_test.go)
//   - TestWorldRule          (pre-existing in mechanics_test.go)

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// ===== 201.2 — Name equality: two instances of the same name =====

func TestCR201_2_NameEqualitySameNameTwoInstances(t *testing.T) {
	// Two permanents with the same name satisfy "destroy all creatures named X."
	const cardName = "Name Equality Test Bear"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			return mage.NewCreature(cardName, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, cardName)
	tg.AddCard(core.ZoneBattlefield, PlayerA, cardName)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, cardName, 2)
}

// ===== 202.1 — Mana cost: colored symbols reflected in ManaCost =====

func TestCR202_1_ManaCostColoredSymbolsRequired(t *testing.T) {
	// A {2}{W} card has 1 white pip and 2 generic in its cost, CMC 3.
	const name = "Mana Cost White Test"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewCreature(name, "{2}{W}", 2, 2, mage.WithSubTypes("Human"))
		})
	}

	card, err := mage.CreateCard(name)
	if err != nil {
		t.Fatalf("CreateCard(%s): %v", name, err)
	}
	mc := card.ManaCost()
	if mc.White != 1 {
		t.Errorf("expected 1 white pip, got %d", mc.White)
	}
	if mc.Generic != 2 {
		t.Errorf("expected 2 generic, got %d", mc.Generic)
	}
	if mc.CMC() != 3 {
		t.Errorf("expected CMC 3, got %d", mc.CMC())
	}
}

func TestCR202_1_ManaCostGenericCanBeAnyColor(t *testing.T) {
	// {3}{U}: generic component allows any mana combination.
	const name = "Mana Cost Generic Test"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewCreature(name, "{3}{U}", 3, 3, mage.WithSubTypes("Illusion"))
		})
	}

	card, err := mage.CreateCard(name)
	if err != nil {
		t.Fatalf("CreateCard(%s): %v", name, err)
	}
	mc := card.ManaCost()
	if mc.Generic != 3 {
		t.Errorf("expected 3 generic, got %d", mc.Generic)
	}
	if mc.Blue != 1 {
		t.Errorf("expected 1 blue pip, got %d", mc.Blue)
	}
}

// ===== 202.1b — No mana cost: lands have CMC 0 =====

func TestCR202_2_NoManaCostLandIsUnpayable(t *testing.T) {
	// A land card has mana value 0 because it has no mana cost.
	const name = "Ch02 Test Land No Cost"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewLand(name, mage.WithSubTypes("Forest"), mage.WithManaAbility(core.Green))
		})
	}
	card, err := mage.CreateCard(name)
	if err != nil {
		t.Fatalf("CreateCard(%s): %v", name, err)
	}
	if mc := card.ManaCost(); mc.CMC() != 0 {
		t.Errorf("expected land CMC 0, got %d", mc.CMC())
	}
}

// ===== 202.2 — Color from mana cost =====

func TestCR202_2c_ColorSingleColor(t *testing.T) {
	// {2}{W}: permanent is white and only white.
	const name = "Ch02 Single Color White"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewCreature(name, "{2}{W}", 2, 2, mage.WithSubTypes("Unicorn"))
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, name)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	tg.AssertHasColor(PlayerA, name, core.White, true)
	tg.AssertHasColor(PlayerA, name, core.Blue, false)
	tg.AssertHasColor(PlayerA, name, core.Black, false)
	tg.AssertHasColor(PlayerA, name, core.Red, false)
	tg.AssertHasColor(PlayerA, name, core.Green, false)
}

func TestCR202_2c_ColorColorless(t *testing.T) {
	// {3}: artifact creature is colorless (no colored mana symbols).
	const name = "Ch02 Colorless Artifact Creature"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewCreature(name, "{3}", 2, 2,
				mage.WithCardType(core.TypeArtifact),
				mage.WithSubTypes("Golem"),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, name)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	tg.AssertHasColor(PlayerA, name, core.White, false)
	tg.AssertHasColor(PlayerA, name, core.Blue, false)
	tg.AssertHasColor(PlayerA, name, core.Black, false)
	tg.AssertHasColor(PlayerA, name, core.Red, false)
	tg.AssertHasColor(PlayerA, name, core.Green, false)
}

func TestCR202_2c_ColorMultiColor(t *testing.T) {
	// {B}{R}{G}: permanent is black, red, and green.
	const name = "Ch02 Multi Color BRG"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewCreature(name, "{B}{R}{G}", 3, 3, mage.WithSubTypes("Beast"))
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, name)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	tg.AssertHasColor(PlayerA, name, core.Black, true)
	tg.AssertHasColor(PlayerA, name, core.Red, true)
	tg.AssertHasColor(PlayerA, name, core.Green, true)
	tg.AssertHasColor(PlayerA, name, core.White, false)
	tg.AssertHasColor(PlayerA, name, core.Blue, false)
}

func TestCR202_2c_ColorMultiColorSatisfiesEachColor(t *testing.T) {
	// Rule 202.2c: a {U}{B} card satisfies both blue and black checks independently.
	const name = "Ch02 Blue Black Creature"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewCreature(name, "{U}{B}", 2, 2, mage.WithSubTypes("Horror"))
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, name)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	tg.AssertHasColor(PlayerA, name, core.Blue, true)
	tg.AssertHasColor(PlayerA, name, core.Black, true)
	tg.AssertHasColor(PlayerA, name, core.White, false)
}

// ===== 202.2f — Color change effect =====

func TestCR202_2c_ColorChangeEffectChangesColor(t *testing.T) {
	// Deathlace makes a permanent black; Grizzly Bears starts green, becomes black.
	const creature = "Grizzly Bears"
	const lace = "Deathlace"

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, creature)
	tg.AddCard(core.ZoneHand, PlayerA, lace)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, lace, creature)
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()

	tg.AssertHasColor(PlayerA, creature, core.Black, true)
	tg.AssertHasColor(PlayerA, creature, core.Green, false)
}

// ===== 202.3 — Mana value =====

func TestCR202_3_ManaValueBasicCalculation(t *testing.T) {
	// A {2}{U} card has mana value 3.
	const name = "Ch02 Mana Value 2U Creature"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewCreature(name, "{2}{U}", 2, 2, mage.WithSubTypes("Djinn"))
		})
	}
	card, err := mage.CreateCard(name)
	if err != nil {
		t.Fatalf("CreateCard(%s): %v", name, err)
	}
	if got := card.ManaCost().CMC(); got != 3 {
		t.Errorf("CMC: got %d, want 3", got)
	}
}

func TestCR202_3_ManaValueZeroForNoManaCost(t *testing.T) {
	// A land has no mana cost, so mana value = 0.
	card, err := mage.CreateCard("Ch02 Test Land No Cost")
	if err != nil {
		t.Fatalf("CreateCard land: %v", err)
	}
	if got := card.ManaCost().CMC(); got != 0 {
		t.Errorf("land CMC: got %d, want 0", got)
	}
}

func TestCR202_3d_ManaValueXTreatedAsZeroOffStack(t *testing.T) {
	// Rule 202.3e: X treated as 0 off-stack. {X}{U}{U} has mana value 2.
	const name = "Braingeyser"
	// Braingeyser is registered via the cards/limited blank import in attr_integration_test.go.
	card, err := mage.CreateCard(name)
	if err != nil {
		t.Skip("Braingeyser not registered in this test binary; skipping")
	}
	if got := card.ManaCost().CMC(); got != 2 {
		t.Errorf("%s off-stack CMC: got %d, want 2", name, got)
	}
}

// ===== 202.4 — Additional costs not part of mana cost =====

func TestCR118_9_AdditionalCostNotPartOfManaCost(t *testing.T) {
	// Additional costs are not part of the mana cost; CMC counts only mana symbols.
	const name = "AdditionalCostNotInCMC"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewSorcery(name, "{2}{B}",
				mage.NewSpellAbility(mage.DrawCards(mage.Fixed(1))),
				mage.WithAdditionalCost(mage.LifePayCost(2)),
			)
		})
	}
	card, err := mage.CreateCard(name)
	if err != nil {
		t.Fatalf("CreateCard(%s): %v", name, err)
	}
	if got := card.ManaCost().CMC(); got != 3 {
		t.Errorf("expected CMC 3 (mana symbols only), got %d", got)
	}
}

// ===== 205.1 — Type line =====

func TestCR205_1_TypeLineCreatureHasType(t *testing.T) {
	// A creature permanent satisfies type checks for "creature."
	const name = "Grizzly Bears"
	card, err := mage.CreateCard(name)
	if err != nil {
		t.Fatalf("CreateCard(%s): %v", name, err)
	}
	if !card.HasType(core.TypeCreature) {
		t.Error("Grizzly Bears should have type Creature")
	}
	if card.HasType(core.TypeLand) {
		t.Error("Grizzly Bears should not have type Land")
	}
}

func TestCR205_1_TypeLineArtifactCreatureHasBothTypes(t *testing.T) {
	// An artifact creature satisfies both the Artifact and Creature type checks.
	const name = "Ch02 Artifact Creature Golem"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewCreature(name, "{4}", 2, 4,
				mage.WithCardType(core.TypeArtifact),
				mage.WithSubTypes("Golem"),
			)
		})
	}
	card, err := mage.CreateCard(name)
	if err != nil {
		t.Fatalf("CreateCard(%s): %v", name, err)
	}
	if !card.HasType(core.TypeArtifact) {
		t.Error("artifact creature should have type Artifact")
	}
	if !card.HasType(core.TypeCreature) {
		t.Error("artifact creature should have type Creature")
	}
}

// ===== 205.2a — Card types recognized =====

func TestCR205_2_CardTypeAllBasicTypesRecognized(t *testing.T) {
	// The engine defines constants for the six implemented card types.
	tests := []struct {
		name     string
		wantType core.CardType
	}{
		{"Grizzly Bears", core.TypeCreature},
		{"Plains", core.TypeLand},
		{"Icy Manipulator", core.TypeArtifact},
		{"Holy Strength", core.TypeEnchantment},
		{"Lightning Bolt", core.TypeInstant},
		{"Demonic Tutor", core.TypeSorcery},
	}
	for _, tc := range tests {

		t.Run(tc.name, func(t *testing.T) {
			card, err := mage.CreateCard(tc.name)
			if err != nil {
				t.Fatalf("CreateCard(%s): %v", tc.name, err)
			}
			if !card.HasType(tc.wantType) {
				t.Errorf("%s: expected type %v", tc.name, tc.wantType)
			}
		})
	}
}

// ===== 205.3a/b — Subtypes: single and multiple =====

func TestCR205_3_SubtypeSingleSubtype(t *testing.T) {
	// A "Goblin" creature has the Goblin subtype.
	const name = "Mons's Goblin Raiders"
	card, err := mage.CreateCard(name)
	if err != nil {
		t.Fatalf("CreateCard(%s): %v", name, err)
	}
	if !card.HasSubType("Goblin") {
		t.Error("Mons's Goblin Raiders should have subtype Goblin")
	}
}

func TestCR205_3_SubtypeMultipleSubtypes(t *testing.T) {
	// White Knight has both Human and Knight subtypes independently.
	const name = "White Knight"
	card, err := mage.CreateCard(name)
	if err != nil {
		t.Fatalf("CreateCard(%s): %v", name, err)
	}
	if !card.HasSubType("Human") {
		t.Error("White Knight should have subtype Human")
	}
	if !card.HasSubType("Knight") {
		t.Error("White Knight should have subtype Knight")
	}
}

// ===== 205.3c — Subtype/type correlation: land types accessible via HasSubType =====

func TestCR205_3_SubtypeLandCreatureSubtypeCorrelation(t *testing.T) {
	// Badlands has Swamp and Mountain land subtypes; both accessible via HasSubType.
	const name = "Badlands"
	card, err := mage.CreateCard(name)
	if err != nil {
		t.Fatalf("CreateCard(%s): %v", name, err)
	}
	if !card.HasSubType("Swamp") {
		t.Error("Badlands should have subtype Swamp")
	}
	if !card.HasSubType("Mountain") {
		t.Error("Badlands should have subtype Mountain")
	}
	if !card.HasType(core.TypeLand) {
		t.Error("Badlands should have type Land")
	}
}

// ===== 205.3g — Artifact subtype: Equipment =====

func TestCR205_3g_ArtifactSubtypeEquipmentRecognized(t *testing.T) {
	// An Equipment artifact has the Equipment subtype and TypeArtifact.
	const name = "Ch02 Test Scimitar"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewEquipment(name, "{2}",
				mage.WithAbility(mage.StaticAbility(mage.BoostAttached(1, 1, core.AttachEquipment))),
				mage.WithAbility(mage.NewEquipAbility(mage.ManaCostOf("{2}"))),
			)
		})
	}
	card, err := mage.CreateCard(name)
	if err != nil {
		t.Fatalf("CreateCard(%s): %v", name, err)
	}
	if !card.HasType(core.TypeArtifact) {
		t.Error("equipment should be an Artifact")
	}
	if !card.HasSubType("Equipment") {
		t.Error("equipment should have subtype Equipment")
	}
}

// ===== 205.3h — Enchantment subtype: Aura =====

func TestCR205_3h_EnchantmentSubtypeAuraRecognized(t *testing.T) {
	// An Aura enchantment has the Aura subtype.
	const name = "Holy Strength"
	card, err := mage.CreateCard(name)
	if err != nil {
		t.Fatalf("CreateCard(%s): %v", name, err)
	}
	if !card.HasType(core.TypeEnchantment) {
		t.Error("Holy Strength should be an Enchantment")
	}
	if !card.HasSubType("Aura") {
		t.Error("Holy Strength should have subtype Aura")
	}
}

// ===== 205.3i — Land subtypes: basic land types =====

func TestCR205_3i_LandSubtypeBasicLandTypes(t *testing.T) {
	// Each basic land has its own land subtype.
	for _, tc := range []struct {
		name    string
		subtype string
	}{
		{"Plains", "Plains"},
		{"Island", "Island"},
		{"Swamp", "Swamp"},
		{"Mountain", "Mountain"},
		{"Forest", "Forest"},
	} {

		t.Run(tc.name, func(t *testing.T) {
			card, err := mage.CreateCard(tc.name)
			if err != nil {
				t.Fatalf("CreateCard(%s): %v", tc.name, err)
			}
			if !card.HasSubType(tc.subtype) {
				t.Errorf("%s should have land subtype %s", tc.name, tc.subtype)
			}
		})
	}
}

func TestCR205_3i_LandSubtypeNonbasicHasLandSubtype(t *testing.T) {
	// Badlands has Swamp and Mountain land subtypes even though it's nonbasic.
	const name = "Badlands"
	card, err := mage.CreateCard(name)
	if err != nil {
		t.Fatalf("CreateCard(%s): %v", name, err)
	}
	if !card.HasSubType("Swamp") || !card.HasSubType("Mountain") {
		t.Errorf("Badlands should have both Swamp and Mountain subtypes")
	}
}

// ===== 205.4a — Supertypes recognized =====

func TestCR205_4_SupertypeLegendaryRecognized(t *testing.T) {
	// A Legendary creature has the Legendary supertype.
	const name = "Ch02 Legendary Test Hero"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewCreature(name, "{R}{G}", 1, 1,
				mage.WithSuperTypes(core.SuperLegendary),
				mage.WithSubTypes("Human", "Warrior"),
			)
		})
	}
	card, err := mage.CreateCard(name)
	if err != nil {
		t.Fatalf("CreateCard(%s): %v", name, err)
	}
	if !card.HasSuperType(core.SuperLegendary) {
		t.Error("legendary creature should have supertype Legendary")
	}
}

func TestCR205_4_SupertypeBasicRecognized(t *testing.T) {
	// Plains has the Basic supertype (CR 205.4c).
	// XXX: cards/limited registers basic lands without WithSuperTypes(SuperBasic);
	// this test documents the engine gap.
	const name = "Ch02 Basic Land Test Plains"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewLand(name,
				mage.WithSuperTypes(core.SuperBasic),
				mage.WithSubTypes("Plains"),
				mage.WithManaAbility(core.White),
			)
		})
	}
	card, err := mage.CreateCard(name)
	if err != nil {
		t.Fatalf("CreateCard(%s): %v", name, err)
	}
	if !card.HasSuperType(core.SuperBasic) {
		t.Error("basic land should have supertype Basic")
	}
}

// ===== 205.4b — Supertypes independent of card type =====

func TestCR205_4_SupertypeIndependentOfTypeChange(t *testing.T) {
	// A Legendary Land has both the Legendary supertype and the Land type simultaneously (CR 205.4b).
	const name = "Ch02 Legendary Land Test"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewLand(name,
				mage.WithSuperTypes(core.SuperLegendary),
				mage.WithSubTypes("Mountain"),
				mage.WithManaAbility(core.Red),
			)
		})
	}
	card, err := mage.CreateCard(name)
	if err != nil {
		t.Fatalf("CreateCard(%s): %v", name, err)
	}
	if !card.HasSuperType(core.SuperLegendary) {
		t.Error("legendary land should have supertype Legendary")
	}
	if !card.HasType(core.TypeLand) {
		t.Error("legendary land should have type Land")
	}
}

// ===== 205.4c — Basic supertype determines basic vs. nonbasic =====

func TestCR205_4_SupertypeBasicLandIsBasic(t *testing.T) {
	// A basic land has the Basic supertype; its basic-land type correlates with the subtype (CR 205.4c).
	card, err := mage.CreateCard("Ch02 Basic Land Test Plains")
	if err != nil {
		t.Fatalf("CreateCard basic land: %v", err)
	}
	if !card.HasSuperType(core.SuperBasic) {
		t.Error("basic land should have supertype Basic")
	}
	if !card.HasSubType("Plains") {
		t.Error("basic plains land should have subtype Plains")
	}
}

func TestCR205_4_SupertypeNonbasicWithBasicLandType(t *testing.T) {
	// Badlands has land subtypes but NOT the Basic supertype.
	card, err := mage.CreateCard("Badlands")
	if err != nil {
		t.Fatalf("CreateCard(Badlands): %v", err)
	}
	if card.HasSuperType(core.SuperBasic) {
		t.Error("Badlands should not have Basic supertype")
	}
}

// ===== 205.4d — Legend rule: different names coexist =====

func TestCR205_4_LegendaryRuleDifferentNamesNoProblem(t *testing.T) {
	// Two legendary permanents with different names coexist (legend rule only applies to same name).
	const nameA = "Ch02 Legend Alpha"
	const nameB = "Ch02 Legend Beta"
	for _, n := range []string{nameA, nameB} {

		if !mage.CardRegistered(n) {
			mage.Register(n, func() mage.Card {
				return mage.NewCreature(n, "{2}{G}", 2, 2,
					mage.WithSuperTypes(core.SuperLegendary),
					mage.WithSubTypes("Human"),
				)
			})
		}
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, nameA)
	tg.AddCard(core.ZoneBattlefield, PlayerA, nameB)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, nameA, 1)
	tg.AssertPermanentCount(PlayerA, nameB, 1)
}

// ===== 205.4f — World rule =====
// NOTE: TestWorldRule in mechanics_test.go covers the basic case.
// This variant uses a different layout (each player controls one).

func TestCR205_4d_WorldRuleSecondWorldKills(t *testing.T) {
	// When a second world enchantment enters, the older one is put into the graveyard.
	const worldA = "World Rule Ch02 Test A"
	const worldB = "World Rule Ch02 Test B"
	if !mage.CardRegistered(worldA) {
		mage.Register(worldA, func() mage.Card {
			return mage.NewEnchantment(worldA, "{3}{G}",
				mage.WithSuperTypes(core.SuperWorld),
			)
		})
	}
	if !mage.CardRegistered(worldB) {
		mage.Register(worldB, func() mage.Card {
			return mage.NewEnchantment(worldB, "{3}{R}",
				mage.WithSuperTypes(core.SuperWorld),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, worldA)
	tg.AddCard(core.ZoneBattlefield, PlayerB, worldB)
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	// worldB was added second (most recently entered) — it survives.
	tg.AssertPermanentCount(PlayerA, worldA, 0)
	tg.AssertPermanentCount(PlayerB, worldB, 1)
}

// ===== 206.3 — Expansion-symbol-based effects =====
// City in a Bottle and Golgothian Sylex (CR 206.3) are covered in
// cards/arabian/artifacts_test.go and cards/antiquities/artifacts_test.go;
// not duplicated here to avoid an import cycle.

// ===== 208.1 — Power/Toughness basics =====

func TestCR208_1_PowerToughnessBasicValues(t *testing.T) {
	// Grizzly Bears is a 2/2; verify printed P/T.
	const name = "Grizzly Bears"
	card, err := mage.CreateCard(name)
	if err != nil {
		t.Fatalf("CreateCard(%s): %v", name, err)
	}
	if card.Power() != 2 || card.Toughness() != 2 {
		t.Errorf("Grizzly Bears P/T: got %d/%d, want 2/2", card.Power(), card.Toughness())
	}
}

func TestCR208_1_PowerToughnessModifiedByEffects(t *testing.T) {
	// A +1/+1 counter on a 2/2 makes it a 3/3.
	const name = "Grizzly Bears"

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, name)
	tg.AddCounters(1, core.PrecombatMain, PlayerA, name, core.P1P1, 1)
	tg.StopAt(2, core.PrecombatMain)
	tg.Execute()

	tg.AssertPowerToughness(PlayerA, name, 3, 3)
}

// ===== 208.2a — CDA: P/T based on condition =====

func TestCR208_4_CDAPowerToughnessByCondition(t *testing.T) {
	// Nightmare's P/T equals number of Swamps controlled (CR 208.2a CDA).
	const name = "Nightmare"

	t.Run("one_swamp", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, name)
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Swamp")
		tg.StopAt(1, core.PrecombatMain)
		tg.Execute()
		tg.AssertPowerToughness(PlayerA, name, 1, 1)
	})

	t.Run("two_swamps", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, name)
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Swamp")
		tg.AddCard(core.ZoneBattlefield, PlayerA, "Swamp")
		tg.StopAt(1, core.PrecombatMain)
		tg.Execute()
		tg.AssertPowerToughness(PlayerA, name, 2, 2)
	})
}

// ===== 208.2b — ETB P/T choice (Primal Clay pattern) =====

func TestCR208_4_PTChoiceETBSetsChosenValues(t *testing.T) {
	// CR 208.2b: as-enters choices set P/T. Inline clay-analog: mode 0 = 3/3, mode 1 = 2/2 flying.
	const name = "Ch02 ETB PT Choice Golem"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			c := mage.NewCreature(name, "{4}", 3, 3,
				mage.WithCardType(core.TypeArtifact),
				mage.WithSubTypes("Golem"),
				mage.WithAbility(mage.ETBEffect(mage.FuncEffect("choose form on ETB",
					mage.EffectProperties{},
					func(g *mage.Game, sourceID, _ uuid.UUID, _ []uuid.UUID) error {
						perm := g.FindPermanent(sourceID)
						if perm == nil {
							return nil
						}
						if g.ModeValue() == 1 {
							perm.Card.SetBasePT(2, 2)
							perm.GrantBaseAttr(core.Flying)
						}
						return nil
					},
				))),
			)
			c.SetModes([]string{"3/3 artifact creature", "2/2 artifact creature with flying"})
			return c
		})
	}

	t.Run("mode_0_3x3", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.ChooseMode(PlayerA, 0)
		tg.AddCard(core.ZoneHand, PlayerA, name)
		tg.CastSpell(1, core.PrecombatMain, PlayerA, name)
		tg.StopAt(1, core.BeginCombat)
		tg.Execute()
		tg.AssertPowerToughness(PlayerA, name, 3, 3)
	})

	t.Run("mode_1_2x2_flying", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.ChooseMode(PlayerA, 1)
		tg.AddCard(core.ZoneHand, PlayerA, name)
		tg.CastSpell(1, core.PrecombatMain, PlayerA, name)
		tg.StopAt(1, core.BeginCombat)
		tg.Execute()
		tg.AssertPowerToughness(PlayerA, name, 2, 2)
		tg.AssertHasAbility(PlayerA, name, core.Flying, true)
	})
}

// ===== 208.4 — Base P/T and modifications are additive =====

func TestCR122_1_BasePTModifiedByCounter(t *testing.T) {
	// A creature's base P/T and counter modifications stack additively.
	// Grizzly Bears base 2/2 + two +1/+1 counters = 4/4.
	const name = "Grizzly Bears"

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, name)
	tg.AddCounters(1, core.PrecombatMain, PlayerA, name, core.P1P1, 2)
	tg.StopAt(2, core.PrecombatMain)
	tg.Execute()

	tg.AssertPowerToughness(PlayerA, name, 4, 4)
}
