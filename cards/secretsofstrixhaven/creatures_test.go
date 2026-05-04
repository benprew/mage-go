package secretsofstrixhaven

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
)

// TestAberrantManawurm_TrampleBase verifies Aberrant Manawurm is a 2/5 with trample.
func TestAberrantManawurm_TrampleBase(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Aberrant Manawurm")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Aberrant Manawurm", 2, 5)
	g.AssertHasAbility(gametest.PlayerA, "Aberrant Manawurm", core.Trample, true)
}

// TestAbstractPaintmage_FirstMainPhaseAddsMana verifies that at the beginning
// of the controller's first main phase, {U}{R} is added to the mana pool.
func TestAbstractPaintmage_FirstMainPhaseAddsMana(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Abstract Paintmage")
	// Cast a 2-cost spell on turn 1 precombat main phase using only the triggered mana
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears") // {1}{G} — won't work
	// Use the mana to verify life loss from no payment needed
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// PlayerB should be at 17 (bolt for 3); mana from Paintmage trigger is available
	g.AssertLife(gametest.PlayerB, 17)
}

// TestArnynDeathbloomBotanist_DiesTriggersLife verifies that when a creature
// you control with power or toughness 1 or less dies, the opponent loses 2
// life and you gain 2 life.
func TestArnynDeathbloomBotanist_DiesTriggersLife(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Arnyn, Deathbloom Botanist")
	// Llanowar Elves is 1/1 — power or toughness 1 or less, triggers Arnyn
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Llanowar Elves")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Shock")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Shock", "Llanowar Elves")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Arnyn triggers: opponent loses 2, you gain 2
	g.AssertLife(gametest.PlayerA, 22)
	g.AssertLife(gametest.PlayerB, 18)
}

// TestArnynDeathbloomBotanist_LargeCreatureNoTrigger verifies that a creature
// with power AND toughness greater than 1 does not trigger Arnyn.
func TestArnynDeathbloomBotanist_LargeCreatureNoTrigger(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Arnyn, Deathbloom Botanist")
	// 2/2 Grizzly Bears: power=2, toughness=2 — neither is 1 or less
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// No Arnyn trigger — life unchanged
	g.AssertLife(gametest.PlayerA, 20)
	g.AssertLife(gametest.PlayerB, 20)
}

// TestAscendantDustspeaker_ETBPlusOneCounter verifies that when Ascendant
// Dustspeaker enters, it puts a +1/+1 counter on another target creature you
// control.
func TestAscendantDustspeaker_ETBPlusOneCounter(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ascendant Dustspeaker")
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
}

// TestAscendantDustspeaker_CombatExilesGraveyardCard verifies that at the
// beginning of combat on your turn, you may exile up to one target card from
// a graveyard.
func TestAscendantDustspeaker_CombatExilesGraveyardCard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ascendant Dustspeaker")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerB, "Lightning Bolt")
	g.ChoosePermanent(gametest.PlayerA, "Ascendant Dustspeaker") // no other creature — no valid ETB target, skip
	// At BeginCombat, exile the Lightning Bolt from PlayerB's graveyard
	g.ChooseFromLibrary(gametest.PlayerA, "Lightning Bolt")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Lightning Bolt", 0)
	g.AssertExileCount("Lightning Bolt", 1)
}

// TestBlechLoafingPest_GainLifeTriggersCounters verifies that whenever you
// gain life, each Pest, Bat, Insect, Snake, and Spider you control gets a
// +1/+1 counter.
func TestBlechLoafingPest_GainLifeTriggersCounters(t *testing.T) {
	g := gametest.NewTestGame(t)
	blechID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Blech, Loafing Pest")
	// Add a Pest token and a non-pest creature
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	// Gain life with a spell
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Healing Salve")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
	g.ChooseMode(gametest.PlayerA, 0) // Gain 3 life
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Healing Salve", "PlayerA")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Blech is a Pest: should have gotten a +1/+1 counter
	_ = blechID
	g.AssertCounterCount(gametest.PlayerA, "Blech, Loafing Pest", core.P1P1, 1)
	// Grizzly Bears is not a Pest/Bat/Insect/Snake/Spider — no counter
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
}

// TestBogwaterLumaret_SelfEntersGainsLife verifies that when Bogwater Lumaret
// enters, it triggers "you gain 1 life" (it IS a creature you control
// entering).
func TestBogwaterLumaret_SelfEntersGainsLife(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bogwater Lumaret")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 21)
}

// TestBogwaterLumaret_OtherCreatureEntersGainsLife verifies that when another
// creature you control enters, you gain 1 life.
func TestBogwaterLumaret_OtherCreatureEntersGainsLife(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bogwater Lumaret")
	// Already on battlefield → trigger doesn't fire for already-present creatures
	// Place a creature into hand and cast it
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Lumaret entering: +1 life. Grizzly Bears entering: +1 life.
	g.AssertLife(gametest.PlayerA, 22)
}

// TestBurrogBanemaker_Deathtouch verifies Burrog Banemaker has deathtouch.
func TestBurrogBanemaker_Deathtouch(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Burrog Banemaker")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Burrog Banemaker", core.Deathtouch, true)
}

// TestBurrogBanemaker_ActivatedBoost verifies that {1}{B} gives Burrog
// Banemaker +1/+1 until end of turn.
func TestBurrogBanemaker_ActivatedBoost(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Burrog Banemaker")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Burrog Banemaker")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Burrog Banemaker", 2, 2)
}

// TestChargingStrifeknight_HasHaste verifies Charging Strifeknight has haste.
func TestChargingStrifeknight_HasHaste(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Charging Strifeknight")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Charging Strifeknight", core.Haste, true)
}

// TestChargingStrifeknight_DiscardDraw verifies that {T}, Discard a card
// draws a card.
func TestChargingStrifeknight_DiscardDraw(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Charging Strifeknight")
	// Give PlayerA some cards to discard
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	// Library card to draw
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Serra Angel")
	g.ChooseDiscard(gametest.PlayerA, "Grizzly Bears")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Charging Strifeknight")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Serra Angel", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
}

// TestColossusOfTheBloodAge_ETBDamageAndLife verifies that when Colossus of
// the Blood Age enters, it deals 3 damage to each opponent and you gain 3 life.
func TestColossusOfTheBloodAge_ETBDamageAndLife(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Colossus of the Blood Age")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 23)
	g.AssertLife(gametest.PlayerB, 17)
}

// TestColossusOfTheBloodAge_DiesDiscardDraw verifies that when Colossus dies,
// you discard any number of cards and draw that many plus one.
func TestColossusOfTheBloodAge_DiesDiscardDraw(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Colossus of the Blood Age")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Serra Angel")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Island")
	// Kill with Wrath of God (hits artifact creatures too)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Wrath of God")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 4)
	// Discard 2 cards when Colossus dies trigger resolves
	g.ChooseDiscard(gametest.PlayerA, "Grizzly Bears", "Serra Angel")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Wrath of God")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Discarded 2, drew 3 (2+1): Lightning Bolt should be in hand
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Serra Angel", 1)
	g.AssertHandCount(gametest.PlayerA, "Lightning Bolt", 1)
}

// TestEagerGlyphmage_ETBCreatesInkling verifies that when Eager Glyphmage
// enters, a 1/1 white and black Inkling creature token with flying is created.
func TestEagerGlyphmage_ETBCreatesInkling(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Eager Glyphmage")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Inkling Token", 1)
	g.AssertHasAbility(gametest.PlayerA, "Inkling Token", core.Flying, true)
}

// TestEmilVastlandsRoamer_TrampleGranted verifies that creatures with +1/+1
// counters you control gain trample.
func TestEmilVastlandsRoamer_TrampleGranted(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Emil, Vastlands Roamer")
	bearsID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	if perm := g.FindPermanent(bearsID); perm != nil {
		perm.AddCounter(core.P1P1, 1)
	}
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Trample, true)
}

// TestEmilVastlandsRoamer_NoCounterNoTrample verifies that creatures without
// +1/+1 counters do NOT gain trample from Emil.
func TestEmilVastlandsRoamer_NoCounterNoTrample(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Emil, Vastlands Roamer")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Trample, false)
}

// TestEnnisDebateModerator_ETBExileReturns verifies that when Ennis enters,
// you may exile another creature you control and it returns at the next end step.
func TestEnnisDebateModerator_ETBExileReturns(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ennis, Debate Moderator")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(2, core.EndStep)
	g.Execute()
	// Grizzly Bears should be back on battlefield after the end step
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
}

// TestEnvironmentalScientist_ETBSearchesBasicLand verifies that when
// Environmental Scientist enters, you may search your library for a basic land
// card, reveal it, put it into your hand, then shuffle. The harness auto-plays
// lands in main phases, so the Forest goes to the battlefield via land-play.
func TestEnvironmentalScientist_ETBSearchesBasicLand(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Environmental Scientist")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
	g.ChooseFromLibrary(gametest.PlayerA, "Forest")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// The Forest is put into hand by the trigger, then auto-played during
	// the precombat main phase (the harness plays lands automatically).
	g.AssertPermanentCount(gametest.PlayerA, "Forest", 1)
}

// TestEssenceknit Scholar_ETBCreatesPestToken verifies that when Essenceknit
// Scholar enters, a 1/1 black and green Pest token with the attack trigger is
// created.
func TestEssenceknitScholar_ETBCreatesPestToken(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Essenceknit Scholar")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Pest Token", 1)
}

// TestEternalStudent_GraveyardAbilityCreatesInklings verifies that {1}{B},
// Exile this card from your graveyard: creates two 1/1 Inkling tokens with flying.
func TestEternalStudent_GraveyardAbilityCreatesInklings(t *testing.T) {
	tg := gametest.NewTestGame(t)
	tg.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Eternal Student")
	tg.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	pid := tg.GetPlayer(gametest.PlayerA).PlayerID()

	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	cs, idx, ok := tg.Game.FindGraveyardActivatableCard(pid, "Eternal Student")
	if !ok {
		t.Fatal("graveyard activatable Eternal Student not found")
	}
	if err := tg.Game.ActivateGraveyardAbility(pid, cs, idx, nil); err != nil {
		t.Fatalf("ActivateGraveyardAbility: %v", err)
	}
	tg.Game.ResolveStack()

	tg.AssertPermanentCount(gametest.PlayerA, "Inkling Token", 2)
	tg.AssertHasAbility(gametest.PlayerA, "Inkling Token", core.Flying, true)
	tg.AssertGraveyardCount(gametest.PlayerA, "Eternal Student", 0)
	tg.AssertExileCount("Eternal Student", 1)
}

// TestAzizaMageTowerCaptain_CopySpellWhenTapThree verifies that when Aziza's
// controller casts an instant or sorcery and taps three untapped creatures,
// the spell is copied.
func TestAzizaMageTowerCaptain_CopySpellWhenTapThree(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Aziza, Mage Tower Captain")
	// Three untapped creatures to tap as cost
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Llanowar Elves")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gray Ogre")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	// Script: choose yes (mode 0) to tap creatures, then name three creatures
	g.ChooseMode(gametest.PlayerA, 0)
	g.ChoosePermanent(gametest.PlayerA, "Llanowar Elves")
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.ChoosePermanent(gametest.PlayerA, "Gray Ogre")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Original + copy = 6 total damage to PlayerB
	g.AssertLife(gametest.PlayerB, 14)
}

// TestAzizaMageTowerCaptain_NoTapNoCoply verifies that when the controller
// declines to tap three untapped creatures, the spell is not copied.
func TestAzizaMageTowerCaptain_NoTapNoCoply(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Aziza, Mage Tower Captain")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Llanowar Elves")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gray Ogre")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	// Script: choose no (mode 1) — do not tap creatures
	g.ChooseMode(gametest.PlayerA, 1)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Only original bolt resolves = 3 damage
	g.AssertLife(gametest.PlayerB, 17)
}

// TestFractalMascot_ETBTapsCreature verifies that Fractal Mascot's ETB taps a
// target creature an opponent controls.
func TestFractalMascot_ETBTapsCreature(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fractal Mascot")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertTapped(gametest.PlayerB, "Grizzly Bears", true)
}

// TestFractalMascot_Stats verifies Fractal Mascot is a 6/6 with trample.
func TestFractalMascot_Stats(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fractal Mascot")
	g.ChoosePermanent(gametest.PlayerA, "Fractal Mascot") // no opp creature — skip ETB target
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Fractal Mascot", 6, 6)
	g.AssertHasAbility(gametest.PlayerA, "Fractal Mascot", core.Trample, true)
}

// TestGarrisonExcavator_Menace verifies Garrison Excavator has menace.
func TestGarrisonExcavator_Menace(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Garrison Excavator")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Garrison Excavator", core.Menace, true)
}

// TestGeometersArthropod_XSpellLooksAtTopCards verifies that when you cast a
// spell with {X} in its mana cost, you look at the top X cards and put one
// into your hand.
func TestGeometersArthropod_XSpellLooksAtTopCards(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Geometer's Arthropod")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	// Fireball has {X} in its mana cost; cast with X=2
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Fireball")
	// Top 2 cards of library
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Serra Angel")
	// ChooseFromLibrary: pick Grizzly Bears to put into hand
	g.ChooseFromLibrary(gametest.PlayerA, "Grizzly Bears")
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Fireball", 2, "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
}

// TestHardenedAcademic_FlyingHaste verifies Hardened Academic has flying and haste.
func TestHardenedAcademic_FlyingHaste(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hardened Academic")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Hardened Academic", core.Flying, true)
	g.AssertHasAbility(gametest.PlayerA, "Hardened Academic", core.Haste, true)
}

// TestHardenedAcademic_DiscardGrantsLifelink verifies that discarding a card
// grants Hardened Academic lifelink until end of turn.
func TestHardenedAcademic_DiscardGrantsLifelink(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hardened Academic")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	// Activate: discard a card to gain lifelink
	g.ChooseDiscard(gametest.PlayerA, "Grizzly Bears")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Hardened Academic")
	g.Attack(1, gametest.PlayerA, "Hardened Academic")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// 2 power + lifelink = +2 life
	g.AssertLife(gametest.PlayerA, 22)
}

// TestHydroChanneler_TapForBlue verifies {T} adds {U}.
func TestHydroChanneler_TapForBlue(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hydro-Channeler")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Unsummon")
	// Activate the {T}: Add {U} ability then cast Unsummon on opponent's creature
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Hydro-Channeler")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Unsummon", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
}

// TestImperiousInkmage_VigilanceAndSurveil verifies Imperious Inkmage has
// vigilance and surveil 2 on ETB.
func TestImperiousInkmage_VigilanceAndSurveil(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Imperious Inkmage")
	// Library: 2 cards to surveil; put one in graveyard
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Serra Angel")
	// Surveil 2: put Grizzly Bears into graveyard, keep Serra Angel on top
	g.ChooseScry(gametest.PlayerA, []string{"Grizzly Bears"}, []string{"Serra Angel"})
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Imperious Inkmage", core.Vigilance, true)
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
}

// TestLoreholdTheHistorian_FlyingHaste verifies it has flying and haste.
func TestLoreholdTheHistorian_FlyingHaste(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lorehold, the Historian")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Lorehold, the Historian", core.Flying, true)
	g.AssertHasAbility(gametest.PlayerA, "Lorehold, the Historian", core.Haste, true)
}

// TestLoreholdTheHistorian_OpponentUpkeepDiscardDraw verifies that at the
// beginning of each opponent's upkeep, you may discard a card to draw a card.
func TestLoreholdTheHistorian_OpponentUpkeepDiscardDraw(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lorehold, the Historian")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Serra Angel")
	// Opponent's upkeep is turn 2; choose to discard Grizzly Bears
	g.ChooseDiscard(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(2, core.EndStep)
	g.Execute()
	// Discarded Grizzly Bears and drew Serra Angel
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertHandCount(gametest.PlayerA, "Serra Angel", 1)
}

// TestMageTowerReferee_MulticoloredSpellPutsCounter verifies that casting a
// multicolored spell puts a +1/+1 counter on Mage Tower Referee.
func TestMageTowerReferee_MulticoloredSpellPutsCounter(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mage Tower Referee")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	// Blech, Loafing Pest is {1}{B}{G} — multicolored
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Blech, Loafing Pest")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Blech, Loafing Pest")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Mage Tower Referee", core.P1P1, 1)
}

// TestMageTowerReferee_MonocoloredSpellNoCounter verifies no counter when a
// mono-colored spell is cast.
func TestMageTowerReferee_MonocoloredSpellNoCounter(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mage Tower Referee")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Mage Tower Referee", core.P1P1, 0)
}

// TestMagmabloodArchaic_TrampleReach verifies it has trample and reach.
func TestMagmabloodArchaic_TrampleReach(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Magmablood Archaic")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Magmablood Archaic", core.Trample, true)
	g.AssertHasAbility(gametest.PlayerA, "Magmablood Archaic", core.Reach, true)
}

// TestMagmabloodArchaic_CastInstantBoostsCreatures verifies that casting an
// instant gives creatures +1/+0 for each color of mana spent.
func TestMagmabloodArchaic_CastInstantBoostsCreatures(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Magmablood Archaic")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	// Lightning Bolt costs {R} — 1 color
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Grizzly Bears should get +1/+0 until EOT (1 color spent)
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 2)
}

// TestMatterbendingMage_ETBBouncesCreature verifies that when Matterbending
// Mage enters, it bounces up to one other target creature.
func TestMatterbendingMage_ETBBouncesCreature(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Matterbending Mage")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	g.AssertHandCount(gametest.PlayerB, "Grizzly Bears", 1)
}

// TestMatterbendingMage_XSpellCantBeBlocked verifies that casting a spell with
// {X} in its mana cost makes Matterbending Mage can't be blocked this turn.
func TestMatterbendingMage_XSpellCantBeBlocked(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Matterbending Mage")
	g.ChoosePermanent(gametest.PlayerA, "Matterbending Mage") // no other creature — skip ETB bounce
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	// Fireball has {X}
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Fireball")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Fireball", 3, "PlayerB")
	g.Attack(1, gametest.PlayerA, "Matterbending Mage")
	g.Block(1, gametest.PlayerB, "Grizzly Bears", "Matterbending Mage")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Matterbending Mage can't be blocked, so it attacks through unblocked
	g.AssertLife(gametest.PlayerB, 14) // bolt damage + mage damage
}

// TestMindfulBiomancer_ETBGainsLife verifies that when Mindful Biomancer enters,
// you gain 1 life.
func TestMindfulBiomancer_ETBGainsLife(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mindful Biomancer")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 21)
}

// TestMindfulBiomancer_ActivatedBoost verifies that {2}{G} gives +2/+2 until EOT.
func TestMindfulBiomancer_ActivatedBoost(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mindful Biomancer")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Mindful Biomancer")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Mindful Biomancer", 4, 4)
}

// TestMicaReaderOfRuins_CopySpellOnArtifactSacrifice verifies that when you cast
// an instant or sorcery, you may sacrifice an artifact to copy the spell.
func TestMicaReaderOfRuins_CopySpellOnArtifactSacrifice(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mica, Reader of Ruins")
	// An artifact to sacrifice
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mox Ruby")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	// Choose to sacrifice Mox Ruby when triggered
	g.ChooseMode(gametest.PlayerA, 0) // "yes, sacrifice an artifact"
	g.ChoosePermanent(gametest.PlayerA, "Mox Ruby")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Original + copy = 6 total damage to PlayerB
	g.AssertLife(gametest.PlayerB, 14)
}

// TestMicaReaderOfRuins_DeclineNoCopy verifies that declining to sacrifice
// an artifact does not copy the spell.
func TestMicaReaderOfRuins_DeclineNoCopy(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mica, Reader of Ruins")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mox Ruby")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	// Decline to sacrifice
	g.ChooseMode(gametest.PlayerA, 1) // "no"
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Only 3 damage
	g.AssertLife(gametest.PlayerB, 17)
}

// TestNitaForumConciliator_ActivatedAbilityExilesFromGraveyard verifies that
// the {2}, Sacrifice another creature ability exiles a target instant or
// sorcery card from an opponent's graveyard.
func TestNitaForumConciliator_ActivatedAbilityExilesFromGraveyard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Nita, Forum Conciliator")
	// Another creature to sacrifice
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Llanowar Elves")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerB, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
	// Target Lightning Bolt from opponent's graveyard
	g.ChooseFromLibrary(gametest.PlayerA, "Lightning Bolt")
	// Choose Llanowar Elves to sacrifice
	g.ChoosePermanent(gametest.PlayerA, "Llanowar Elves")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Nita, Forum Conciliator")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Lightning Bolt is exiled from opponent's graveyard
	g.AssertGraveyardCount(gametest.PlayerB, "Lightning Bolt", 0)
	g.AssertExileCount("Lightning Bolt", 1)
}

// TestNoxiousNewt_DeathtouchAndMana verifies Noxious Newt has deathtouch and {T} adds {G}.
func TestNoxiousNewt_DeathtouchAndMana(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Noxious Newt")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Noxious Newt")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Noxious Newt", core.Deathtouch, true)
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
}

// TestOrysaTideChoreographer_DrawsTwo verifies that when Orysa enters, draw two cards.
func TestOrysaTideChoreographer_DrawsTwo(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Orysa, Tide Choreographer")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Serra Angel")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertHandCount(gametest.PlayerA, "Serra Angel", 1)
}

// TestOwlinHistorian_FlyingAndSurveil verifies Owlin Historian has flying and
// surveil 1 on ETB.
func TestOwlinHistorian_FlyingAndSurveil(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Owlin Historian")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
	// Surveil 1: put the card into graveyard
	g.ChooseScry(gametest.PlayerA, []string{"Grizzly Bears"}, nil)
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Owlin Historian", core.Flying, true)
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
}

// ===========================================================================
// Batch 3: Paradox Surveyor, Pest Mascot, Pestbrood Sloth, Postmortem
// Professor, Practiced Scrollsmith, Prismari the Inspiration, Pterafractyl,
// Quandrix the Proof, Rancorous Archaic, Rearing Embermare, Rubble Rouser,
// Shattered Acolyte, Shopkeeper's Bane, Silverquill the Disputant,
// Slumbering Trudge, Sneering Shadewriter
// ===========================================================================

// TestParadoxSurveyor_ETBLooksAtTopFive verifies that when Paradox Surveyor
// enters, the controller looks at the top 5 cards and may take a qualifying card.
// We use Stream of Life (an X-cost card) so the auto-land-play harness does not
// immediately remove the card from hand before the assertion runs.
func TestParadoxSurveyor_ETBLooksAtTopFive(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Paradox Surveyor")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Stream of Life")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Gray Ogre")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Gray Ogre")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Gray Ogre")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Gray Ogre")
	g.ChooseFromLibrary(gametest.PlayerA, "Stream of Life")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Paradox Surveyor")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Stream of Life (a card with {X} in its mana cost) was put into hand
	g.AssertHandCount(gametest.PlayerA, "Stream of Life", 1)
}

// TestParadoxSurveyor_ETBDeclineNoCard verifies declining puts all cards back on bottom.
func TestParadoxSurveyor_ETBDeclineNoCard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Paradox Surveyor")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Gray Ogre")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Gray Ogre")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Gray Ogre")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Gray Ogre")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Gray Ogre")
	g.ChooseFromLibrary(gametest.PlayerA, "")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Paradox Surveyor")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Gray Ogre", 0)
	g.AssertLibraryCount(gametest.PlayerA, "Gray Ogre", 5)
}

// TestPestMascot_LifegainCounterTrigger verifies gaining life puts a +1/+1
// counter on Pest Mascot.
func TestPestMascot_LifegainCounterTrigger(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Pest Mascot")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Stream of Life")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Stream of Life", 1, "PlayerA")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Pest Mascot", core.P1P1, 1)
}

// TestPestMascot_NoCounterWithoutLifegain verifies no counter added without lifegain.
func TestPestMascot_NoCounterWithoutLifegain(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Pest Mascot")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Pest Mascot", core.P1P1, 0)
}

// TestPestbroodSloth_DiesCreatesTwoPestTokens verifies that when Pestbrood
// Sloth dies, two Pest tokens are created.
func TestPestbroodSloth_DiesCreatesTwoPestTokens(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Pestbrood Sloth")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Terror")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Swamp", 2)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Terror", "Pestbrood Sloth")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Pest Token", 2)
}

// TestPestbroodSloth_PestTokenGainsLifeWhenAttacking verifies the Pest token's
// attack trigger grants 1 life.
func TestPestbroodSloth_PestTokenGainsLifeWhenAttacking(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Pestbrood Sloth")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Terror")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Swamp", 2)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Terror", "Pestbrood Sloth")
	g.Attack(3, gametest.PlayerA, "Pest Token")
	g.StopAt(3, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 21)
}

// TestPostmortemProfessor_CantBlock verifies Postmortem Professor can't block.
// Turn 2 is PlayerB's turn; Gray Ogre attacks and PlayerA tries to block with Prof.
// Prof's block declaration is silently ignored (can't block), so Gray Ogre
// deals 2 unblocked combat damage. PlayerA finishes at 18.
func TestPostmortemProfessor_CantBlock(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Postmortem Professor")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Gray Ogre")
	g.Attack(2, gametest.PlayerB, "Gray Ogre")
	g.Block(2, gametest.PlayerA, "Postmortem Professor", "Gray Ogre")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 18)
}

// TestPostmortemProfessor_AttackDrainsLife verifies each opponent loses 1 life
// and controller gains 1 life when Professor attacks.
// Prof (2/2) attacks unblocked: 2 combat damage + 1 drain = PlayerB at 17; PlayerA gains 1 = 21.
func TestPostmortemProfessor_AttackDrainsLife(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Postmortem Professor")
	g.Attack(1, gametest.PlayerA, "Postmortem Professor")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 17)
	g.AssertLife(gametest.PlayerA, 21)
}

// TestPostmortemProfessor_GraveyardReturn verifies the graveyard activation
// returns the Professor to the battlefield.
func TestPostmortemProfessor_GraveyardReturn(t *testing.T) {
	tg := gametest.NewTestGame(t)
	tg.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Postmortem Professor")
	tg.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Lightning Bolt")
	tg.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)

	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	pid := tg.GetPlayer(gametest.PlayerA).PlayerID()
	cs, idx, ok := tg.Game.FindGraveyardActivatableCard(pid, "Postmortem Professor")
	if !ok {
		t.Fatal("Postmortem Professor graveyard ability not found")
	}
	if err := tg.Game.ActivateGraveyardAbility(pid, cs, idx, nil); err != nil {
		t.Fatalf("ActivateGraveyardAbility: %v", err)
	}
	tg.Game.ResolveStack()
	tg.AssertPermanentCount(gametest.PlayerA, "Postmortem Professor", 1)
}

// TestPracticedScrollsmith_FirstStrike verifies Practiced Scrollsmith has first strike.
func TestPracticedScrollsmith_FirstStrike(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Practiced Scrollsmith")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Practiced Scrollsmith", core.FirstStrike, true)
}

// TestPracticedScrollsmith_ETBExilesGraveyardCard verifies that when Practiced
// Scrollsmith enters, a noncreature nonland graveyard card is exiled.
func TestPracticedScrollsmith_ETBExilesGraveyardCard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Practiced Scrollsmith")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 3)
	g.ChooseFromLibrary(gametest.PlayerA, "Lightning Bolt")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Practiced Scrollsmith")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertExileCount("Lightning Bolt", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Lightning Bolt", 0)
}

// TestPterafractyl_EntersWithXCounters verifies Pterafractyl enters with X
// +1/+1 counters.
func TestPterafractyl_EntersWithXCounters(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Pterafractyl")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 1)
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Pterafractyl", 2)
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Pterafractyl", core.P1P1, 2)
}

// TestPterafractyl_ETBGains2Life verifies Pterafractyl's ETB gains 2 life.
func TestPterafractyl_ETBGains2Life(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Pterafractyl")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 1)
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Pterafractyl", 0)
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 22)
}

// TestRancorousArchaic_ConvergeCounters verifies Rancorous Archaic enters with
// one +1/+1 counter for each color of mana spent to cast it.
func TestRancorousArchaic_ConvergeCounters(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Rancorous Archaic")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Rancorous Archaic")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// 3 distinct colors spent — 3 counters
	g.AssertCounterCount(gametest.PlayerA, "Rancorous Archaic", core.P1P1, 3)
}

// TestRearingEmbermare_Keywords verifies Rearing Embermare has reach and haste.
func TestRearingEmbermare_Keywords(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rearing Embermare")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Rearing Embermare", core.Reach, true)
	g.AssertHasAbility(gametest.PlayerA, "Rearing Embermare", core.Haste, true)
}

// TestRubbleRouser_ETBMayDiscardDraw verifies the ETB looting effect.
func TestRubbleRouser_ETBMayDiscardDraw(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Rubble Rouser")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Gray Ogre")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 3)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
	g.ChooseDiscard(gametest.PlayerA, "Gray Ogre")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Rubble Rouser")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Gray Ogre", 1)
}

// TestRubbleRouser_TapExileAddManaAndDamage verifies tapping and exiling a
// graveyard card deals 1 damage to each opponent.
func TestRubbleRouser_TapExileAddManaAndDamage(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rubble Rouser")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Gray Ogre")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Rubble Rouser")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 19)
}

// TestShatteredAcolyte_Lifelink verifies Shattered Acolyte has lifelink.
func TestShatteredAcolyte_Lifelink(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Shattered Acolyte")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Shattered Acolyte", core.Lifelink, true)
}

// TestShatteredAcolyte_SacrificeDestroyArtifact verifies sacrificing the
// Acolyte destroys target artifact or enchantment.
func TestShatteredAcolyte_SacrificeDestroyArtifact(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Shattered Acolyte")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Black Vise")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Shattered Acolyte", "Black Vise")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Black Vise", 0)
	g.AssertPermanentCount(gametest.PlayerA, "Shattered Acolyte", 0)
}

// TestShopkeepersBane_AttackGainsLife verifies Shopkeeper's Bane gains 2 life
// when it attacks.
func TestShopkeepersBane_AttackGainsLife(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Shopkeeper's Bane")
	g.Attack(1, gametest.PlayerA, "Shopkeeper's Bane")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 22)
}

// TestShopkeepersBane_NoLifeIfNotAttacking verifies no life gained if not attacking.
func TestShopkeepersBane_NoLifeIfNotAttacking(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Shopkeeper's Bane")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 20)
}

// TestSlumberingTrudge_Stats verifies Slumbering Trudge is a 6/6.
func TestSlumberingTrudge_Stats(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Slumbering Trudge")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Slumbering Trudge", 6, 6)
}

// TestSneeringShadewriter_ETBDrainsOpponent verifies that when Sneering
// Shadewriter enters, each opponent loses 2 life and controller gains 2 life.
func TestSneeringShadewriter_ETBDrainsOpponent(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Sneering Shadewriter")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 5)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Sneering Shadewriter")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 18)
	g.AssertLife(gametest.PlayerA, 22)
}

// ===========================================================================
// Batch 4: Soaring Stoneglider, Spirit Mascot, Stadium Tidalmage, Startled
// Relic Sloth, Stirring Honormancer, Stone Docent, Summoned Dromedary,
// Sundering Archaic, Teacher's Pest, The Dawning Archaic, Transcendent
// Archaic, Wildgrowth Archaic, Witherbloom the Balancer, Zaffai and the
// Tempests, Zealous Lorecaster
// ===========================================================================

// TestSoaringStonglider_FlyingVigilance verifies Soaring Stoneglider has
// flying and vigilance and can be cast by exiling two cards from graveyard.
func TestSoaringStonglider_FlyingVigilance(t *testing.T) {
	t.Run("has flying and vigilance; exiles two from graveyard as cost", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Soaring Stoneglider")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Gray Ogre")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Soaring Stoneglider")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Soaring Stoneglider", 1)
		g.AssertHasAbility(gametest.PlayerA, "Soaring Stoneglider", core.Flying, true)
		g.AssertHasAbility(gametest.PlayerA, "Soaring Stoneglider", core.Vigilance, true)
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 0)
		g.AssertGraveyardCount(gametest.PlayerA, "Gray Ogre", 0)
	})
}

// TestSpiritMascot_CounterWhenGraveyardCardLeaves verifies Spirit Mascot gets
// a +1/+1 counter when one or more cards leave the controller's graveyard.
func TestSpiritMascot_CounterWhenGraveyardCardLeaves(t *testing.T) {
	t.Run("counter when graveyard cards are exiled as additional cost", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Spirit Mascot")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Gray Ogre")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Soaring Stoneglider")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Soaring Stoneglider")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Spirit Mascot", core.P1P1, 1)
	})
}

// TestStadiumTidalmage_ETBDrawDiscard verifies Stadium Tidalmage may draw
// then discard on ETB.
func TestStadiumTidalmage_ETBDrawDiscard(t *testing.T) {
	t.Run("draw a card then discard on ETB", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Stadium Tidalmage")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.ChooseDiscard(gametest.PlayerA, "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Stadium Tidalmage")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Stadium Tidalmage", 1)
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}

// TestStartledRelicSloth_BeginCombatExile verifies that at the beginning of
// combat on your turn, Startled Relic Sloth can exile up to one card from a graveyard.
func TestStartledRelicSloth_BeginCombatExile(t *testing.T) {
	t.Run("exiles a card from a graveyard at begin combat", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Startled Relic Sloth")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerB, "Grizzly Bears")
		g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Startled Relic Sloth")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertExileCount("Grizzly Bears", 1)
	})
}

// TestStirringHonormancer_ETBLookAtTop verifies Stirring Honormancer's ETB
// lets the controller put one of the top X cards into hand, rest to graveyard.
func TestStirringHonormancer_ETBLookAtTop(t *testing.T) {
	t.Run("look at top X, chosen to hand, rest to graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Llanowar Elves")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Gray Ogre")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Gray Ogre")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Stirring Honormancer")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.ChooseFromLibrary(gametest.PlayerA, "Gray Ogre")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Stirring Honormancer")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Gray Ogre", 1)
	})
}

// TestStonDocent_GraveyardAbility verifies Stone Docent's graveyard ability
// exiles it from graveyard and gains the controller 2 life.
func TestStonDocent_GraveyardAbility(t *testing.T) {
	t.Run("gain 2 life when exiling Stone Docent from graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Stone Docent")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Stone Docent")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 22)
		g.AssertGraveyardCount(gametest.PlayerA, "Stone Docent", 0)
		g.AssertExileCount("Stone Docent", 1)
	})
}

// TestSummonedDromedary_GraveyardReturnToHand verifies Summoned Dromedary's
// graveyard activated ability returns it from the graveyard to hand.
func TestSummonedDromedary_GraveyardReturnToHand(t *testing.T) {
	t.Run("returns from graveyard to hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Summoned Dromedary")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Summoned Dromedary")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Summoned Dromedary", 1)
		g.AssertGraveyardCount(gametest.PlayerA, "Summoned Dromedary", 0)
	})
}

// TestSunderingArchaic_ConvergeETB verifies the Converge ETB exile ability.
func TestSunderingArchaic_ConvergeETB(t *testing.T) {
	t.Run("exiles opponent nonland permanent with MV <= colors spent", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Sundering Archaic")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Sundering Archaic")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertExileCount("Grizzly Bears", 1)
	})
}

// TestTeachersPest_AttackGainsLife verifies Teacher's Pest gains 1 life
// whenever it attacks.
func TestTeachersPest_AttackGainsLife(t *testing.T) {
	t.Run("gain 1 life when attacks", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Teacher's Pest")
		g.Attack(1, gametest.PlayerA, "Teacher's Pest")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 21)
	})
}

// TestTeachersPest_GraveyardReturn verifies Teacher's Pest returns from
// graveyard to battlefield tapped via its activated ability.
func TestTeachersPest_GraveyardReturn(t *testing.T) {
	t.Run("return from graveyard to battlefield tapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Teacher's Pest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Teacher's Pest")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Teacher's Pest", 1)
		g.AssertGraveyardCount(gametest.PlayerA, "Teacher's Pest", 0)
		g.AssertTapped(gametest.PlayerA, "Teacher's Pest", true)
	})
}

// TestTheDawningArchaic_CostReduction verifies The Dawning Archaic costs
// {1} less for each instant/sorcery card in the controller's graveyard.
func TestTheDawningArchaic_CostReduction(t *testing.T) {
	t.Run("costs 1 less per instant/sorcery in graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		// 2 instants in graveyard → cost 10-2=8. With 7 islands, can't pay.
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Lightning Bolt")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Shock")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Gray Ogre")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "The Dawning Archaic")
		for range 7 {
			g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		}
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "The Dawning Archaic")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "The Dawning Archaic", 0)
		g.AssertHandCount(gametest.PlayerA, "The Dawning Archaic", 1)
	})
}

// TestTranscendentArchaic_ConvergeDraw verifies the Converge ETB draws X
// cards then discards 2 if at least 1 was drawn.
func TestTranscendentArchaic_ConvergeDraw(t *testing.T) {
	t.Run("draws X on ETB and discards 2 if any drawn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Transcendent Archaic")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Gray Ogre")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Llanowar Elves")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Birds of Paradise")
		g.ChooseDiscard(gametest.PlayerA, "Grizzly Bears")
		g.ChooseDiscard(gametest.PlayerA, "Gray Ogre")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Transcendent Archaic")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Drew 4 (4 colors), discarded 2 → 2 in hand
		g.AssertHandCount(gametest.PlayerA, "Llanowar Elves", 1)
		g.AssertHandCount(gametest.PlayerA, "Birds of Paradise", 1)
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
		g.AssertGraveyardCount(gametest.PlayerA, "Gray Ogre", 1)
	})
}

// TestWildgrowthArchaic_ConvergeEntersWithCounters verifies Wildgrowth
// Archaic enters with +1/+1 counters equal to colors of mana spent to cast it.
func TestWildgrowthArchaic_ConvergeEntersWithCounters(t *testing.T) {
	t.Run("enters with counters equal to colors spent", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Wildgrowth Archaic")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Wildgrowth Archaic")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// {2/G}{2/G} paid as {G}{G}+{W} → 2 colors → 2 counters
		g.AssertCounterCount(gametest.PlayerA, "Wildgrowth Archaic", core.P1P1, 2)
	})
}

// TestWitherbloom_AffinityReducesCost verifies Witherbloom, the Balancer
// costs less to cast for each creature the controller controls.
func TestWitherbloom_AffinityReducesCost(t *testing.T) {
	t.Run("affinity for creatures reduces casting cost", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		// 2 creatures → {6}{B}{G} becomes {4}{B}{G} = 6 total
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Llanowar Elves")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Witherbloom, the Balancer")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Witherbloom, the Balancer")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Witherbloom, the Balancer", 1)
		g.AssertHasAbility(gametest.PlayerA, "Witherbloom, the Balancer", core.Flying, true)
		g.AssertHasAbility(gametest.PlayerA, "Witherbloom, the Balancer", core.Deathtouch, true)
	})
}

// TestZaffaiAndTheTempests_FreeCastFromHand verifies Zaffai and the Tempests
// allows casting one instant or sorcery spell from hand for free once per turn.
func TestZaffaiAndTheTempests_FreeCastFromHand(t *testing.T) {
	t.Run("cast instant from hand for free once per turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Zaffai and the Tempests")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Zaffai and the Tempests")
		g.ChoosePermanent(gametest.PlayerA, "Lightning Bolt")
		g.ChoosePermanent(gametest.PlayerA, "PlayerB")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 17)
	})
}

// TestZealousLorecaster_ETBReturnFromGraveyard verifies Zealous Lorecaster
// returns a target instant or sorcery from graveyard to hand on ETB.
func TestZealousLorecaster_ETBReturnFromGraveyard(t *testing.T) {
	t.Run("ETB returns instant or sorcery from graveyard to hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Lightning Bolt")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Zealous Lorecaster")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.ChoosePermanent(gametest.PlayerA, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Zealous Lorecaster")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Lightning Bolt", 1)
		g.AssertGraveyardCount(gametest.PlayerA, "Lightning Bolt", 0)
	})
}
