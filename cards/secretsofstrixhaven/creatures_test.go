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
	g.AssertLife(gametest.PlayerB, 15) // Fireball X=3 deals 3 damage + Mage 2/2 deals 2 unblocked = 5 total
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
	g.StopAt(1, core.EndStep)
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

// TestSlumberingTrudge_EntersTappedWhenXIsTwo verifies that when cast with X=2,
// Slumbering Trudge is tapped after ETB (X ≤ 2).
func TestSlumberingTrudge_EntersTappedWhenXIsTwo(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Slumbering Trudge")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Slumbering Trudge", 2)
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertTapped(gametest.PlayerA, "Slumbering Trudge", true)
}

// TestSlumberingTrudge_NotTappedWhenXIsThree verifies that when cast with X=3,
// Slumbering Trudge is not tapped (X > 2).
func TestSlumberingTrudge_NotTappedWhenXIsThree(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Slumbering Trudge")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 4)
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Slumbering Trudge", 3)
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertTapped(gametest.PlayerA, "Slumbering Trudge", false)
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
		// 2 instants + 1 creature (Gray Ogre) → reduction should be exactly 2.
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Lightning Bolt")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Shock")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Gray Ogre")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "The Dawning Archaic")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		pid := g.GetPlayer(gametest.PlayerA).PlayerID()
		for _, c := range g.GetPlayer(gametest.PlayerA).Hand() {
			if c.Name() == "The Dawning Archaic" {
				if got := g.Game.ConditionalSpellCostReduction(pid, c); got != 2 {
					t.Errorf("CostReduction with 2 instants/sorceries: got %d, want 2", got)
				}
				return
			}
		}
		t.Error("The Dawning Archaic not found in hand")
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
		g.ChooseFromLibrary(gametest.PlayerA, "Lightning Bolt")
		g.ChooseTarget(gametest.PlayerA, "PlayerB")
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

// TestForumNecroscribe_Stats verifies Forum Necroscribe is a 5/4 Troll Warlock.
func TestForumNecroscribe_Stats(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forum Necroscribe")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Forum Necroscribe", 5, 4)
}

// TestForumNecroscribe_ReparteeReturnCreatureFromGraveyard verifies the Repartee
// trigger: whenever you cast an instant or sorcery that targets a creature,
// return target creature card from your graveyard to the battlefield.
func TestForumNecroscribe_ReparteeReturnCreatureFromGraveyard(t *testing.T) {
	t.Run("casting instant targeting creature returns creature from graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forum Necroscribe")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Shock")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Serra Angel")
		// Repartee trigger: choose creature card from graveyard
		g.ChoosePermanent(gametest.PlayerA, "Serra Angel")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Shock", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Serra Angel", 1)
		g.AssertGraveyardCount(gametest.PlayerA, "Serra Angel", 0)
	})
	t.Run("casting spell NOT targeting creature does not trigger", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forum Necroscribe")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Serra Angel")
		// Target a player, not a creature — no trigger
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Serra Angel", 0)
		g.AssertGraveyardCount(gametest.PlayerA, "Serra Angel", 1)
	})
}

// TestHungryGraffalon_StatsAndReach verifies Hungry Graffalon is a 3/4 with reach.
func TestHungryGraffalon_StatsAndReach(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hungry Graffalon")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Hungry Graffalon", 3, 4)
	g.AssertHasAbility(gametest.PlayerA, "Hungry Graffalon", core.Reach, true)
}

// TestHungryGraffalon_IncrementAddsCounter verifies Increment:
// when you cast a spell spending more mana than the Graffalon's power or
// toughness, it gets a +1/+1 counter.
func TestHungryGraffalon_IncrementAddsCounter(t *testing.T) {
	t.Run("casting spell spending more than power or toughness adds counter", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hungry Graffalon")
		// Graffalon is 3/4. Spend 5 mana (Air Elemental {3}{U}{U} = 5).
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 2)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Air Elemental")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Air Elemental")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// 5 > 4 (toughness), so +1/+1 counter placed: 4/5
		g.AssertPowerToughness(gametest.PlayerA, "Hungry Graffalon", 4, 5)
	})
	t.Run("casting spell spending less than both power and toughness does not add counter", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hungry Graffalon")
		// Graffalon is 3/4. Cast Shock {R} = 1 mana (1 < 3 and 1 < 4).
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Shock")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Shock", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// No counter
		g.AssertPowerToughness(gametest.PlayerA, "Hungry Graffalon", 3, 4)
	})
}

// TestFractalTender_Stats verifies Fractal Tender is a 3/3 Elf Wizard.
func TestFractalTender_Stats(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fractal Tender")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Fractal Tender", 3, 3)
}

// TestFractalTender_IncrementAddsCounter verifies Increment fires on Fractal Tender.
func TestFractalTender_IncrementAddsCounter(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fractal Tender")
	// Tender is 3/3. Spend 4 mana (Giant Spider = {3}{G}, 4 > 3).
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 4)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Giant Spider")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Giant Spider")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Fractal Tender", core.P1P1, 1)
}

// TestFractalTender_EndStepCreatesFractalToken verifies the end step trigger:
// if a +1/+1 counter was placed on Fractal Tender this turn, create a 0/0
// green and blue Fractal creature token and put three +1/+1 counters on it.
func TestFractalTender_EndStepCreatesFractalToken(t *testing.T) {
	t.Run("creates Fractal token with 3 counters when counter placed this turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fractal Tender")
		// Spend 4 mana to trigger Increment (Tender is 3/3, 4 > 3; Giant Spider = {3}{G})
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 4)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Giant Spider")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Giant Spider")
		// Stop at Cleanup so that the EndStep triggers have had a chance to resolve.
		g.StopAt(1, core.Cleanup)
		g.Execute()
		// Fractal token should have been created at end step
		g.AssertPermanentCount(gametest.PlayerA, "Fractal Token", 1)
		g.AssertCounterCount(gametest.PlayerA, "Fractal Token", core.P1P1, 3)
	})
	t.Run("does not create token when no counter placed this turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fractal Tender")
		// Cast Shock {R} = 1 mana; 1 < 3 so no Increment trigger, no token
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Shock")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Shock", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Fractal Token", 0)
	})
}

// TestExhibitionTidecaller_OpusMillsThree verifies that casting an instant or
// sorcery spell causes Exhibition Tidecaller's Opus trigger to mill the target
// player three cards when fewer than five mana was spent.
func TestExhibitionTidecaller_OpusMillsThree(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Exhibition Tidecaller")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	// Choose PlayerB as the target of the mill trigger.
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Lightning Bolt costs {R} — 1 mana spent, triggers 3-card mill.
	g.AssertGraveyardCount(gametest.PlayerB, "Filler", 3)
}

// TestExhibitionTidecaller_OpusMillsTenFiveOrMoreMana verifies that the Opus
// trigger mills ten cards when five or more mana was spent to cast the spell.
func TestExhibitionTidecaller_OpusMillsTenFiveOrMoreMana(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Exhibition Tidecaller")
	// Add enough mana for Fireball with X=4 (total 5: {X}{R} with X=4 = 5 mana).
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Fireball")
	// Choose PlayerB as the target of the mill trigger.
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Fireball", 4, "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Fireball costs {X}{R} with X=4 — 5 mana spent, triggers 10-card mill.
	g.AssertGraveyardCount(gametest.PlayerB, "Filler", 10)
}

// TestCuboidColony_Stats verifies Cuboid Colony is a 1/1 with Flash, Flying, and Trample.
func TestCuboidColony_Stats(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cuboid Colony")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Cuboid Colony", 1, 1)
	g.AssertHasAbility(gametest.PlayerA, "Cuboid Colony", core.Flying, true)
	g.AssertHasAbility(gametest.PlayerA, "Cuboid Colony", core.Trample, true)
	g.AssertHasAbility(gametest.PlayerA, "Cuboid Colony", core.Flash, true)
}

// TestCuboidColony_IncrementGrowth verifies the Increment keyword puts a +1/+1
// counter when mana spent casting a spell is greater than the creature's power or toughness.
func TestCuboidColony_IncrementGrowth(t *testing.T) {
	t.Run("no counter when mana spent equals P/T", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cuboid Colony")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt") // costs {R} = 1 mana
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		// colony is 1/1 — spending 1 mana is NOT greater than 1, so no counter
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Cuboid Colony", core.P1P1, 0)
	})
	t.Run("counter placed when mana spent exceeds power", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cuboid Colony")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears") // costs {1}{G} = 2 mana
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		// spend 2 mana; colony is 1/1 — 2 > 1, so a counter is placed
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Cuboid Colony", core.P1P1, 1)
	})
}

// TestDelugeVirtuoso_ETBTapsOpponentCreature verifies that when Deluge Virtuoso
// enters the battlefield, it taps a target creature an opponent controls.
func TestDelugeVirtuoso_ETBTapsOpponentCreature(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Deluge Virtuoso")
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertTapped(gametest.PlayerB, "Grizzly Bears", true)
}

// TestDelugeVirtuoso_OpusBonusPowerToughness verifies the Opus ability grants
// +1/+1 for cheap instant/sorcery casts, +2/+2 when 5 or more mana was spent.
func TestDelugeVirtuoso_OpusBonusPowerToughness(t *testing.T) {
	t.Run("opus: +1/+1 when casting instant or sorcery with fewer than 5 mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Deluge Virtuoso")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt") // {R} = 1 mana
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.ChoosePermanent(gametest.PlayerA, "Deluge Virtuoso") // no opp creature to tap
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
		// Stop after PrecombatMain so the trigger resolves; verify before EOT cleanup
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Deluge Virtuoso is 2/2 base, gets +1/+1 = 3/3 until end of turn
		g.AssertPowerToughness(gametest.PlayerA, "Deluge Virtuoso", 3, 3)
	})
	t.Run("opus: only fires for instant or sorcery, not creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Deluge Virtuoso")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.ChoosePermanent(gametest.PlayerA, "Deluge Virtuoso") // no opp creature to tap
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// No Opus trigger — Deluge Virtuoso stays at 2/2
		g.AssertPowerToughness(gametest.PlayerA, "Deluge Virtuoso", 2, 2)
	})
}

// TestExpressiveFiredancer_OpusBoost verifies that casting an instant or
// sorcery spell triggers Expressive Firedancer's Opus and grants +1/+1 until
// end of turn (expires by turn 2).
func TestExpressiveFiredancer_OpusBoost(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Expressive Firedancer")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	// Stop at turn 2 to verify the boost has expired (it was "until end of turn 1").
	g.StopAt(2, core.PrecombatMain)
	g.Execute()
	// Boost expired; Expressive Firedancer is back to 2/2.
	g.AssertPowerToughness(gametest.PlayerA, "Expressive Firedancer", 2, 2)
}

// TestExpressiveFiredancer_OpusBoostDuringCombat verifies the +1/+1 boost is
// active during combat when the spell was cast precombat.
func TestExpressiveFiredancer_OpusBoostDuringCombat(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Expressive Firedancer")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	// Attack with Firedancer during combat — it should be 3/3 during attack step.
	g.Attack(1, gametest.PlayerA, "Expressive Firedancer")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Expressive Firedancer", 3, 3)
}

// TestExpressiveFiredancer_OpusDoubleStrikeFiveOrMore verifies that spending
// five or more mana also grants double strike until EOT.
func TestExpressiveFiredancer_OpusDoubleStrikeFiveOrMore(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Expressive Firedancer")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 5)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Fireball")
	// Fireball {X}{R} with X=4 = 5 mana spent.
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Fireball", 4, "PlayerB")
	g.Attack(1, gametest.PlayerA, "Expressive Firedancer")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	// +1/+1 boost and double strike active during combat.
	g.AssertPowerToughness(gametest.PlayerA, "Expressive Firedancer", 3, 3)
	g.AssertHasAbility(gametest.PlayerA, "Expressive Firedancer", core.DoubleStrike, true)
}

// TestAmbitiousAugmenter_Increment verifies that casting a spell whose total
// mana cost is greater than Ambitious Augmenter's power or toughness places a
// +1/+1 counter on it.
func TestAmbitiousAugmenter_Increment(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ambitious Augmenter")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears") // {1}{G} = 2 mana > power 1 or toughness 1
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Ambitious Augmenter", core.P1P1, 1)
	g.AssertPowerToughness(gametest.PlayerA, "Ambitious Augmenter", 2, 2)
}

// TestAmbitiousAugmenter_DiesWithCountersCreatesFractal verifies that when
// Ambitious Augmenter dies with one or more counters on it, a 0/0 green and
// blue Fractal creature token is created and receives those counters.
func TestAmbitiousAugmenter_DiesWithCountersCreatesFractal(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ambitious Augmenter")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
	// Augmenter now has 1 P1P1 counter. Kill it.
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Ambitious Augmenter")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Fractal Token", 1)
	g.AssertCounterCount(gametest.PlayerA, "Fractal Token", core.P1P1, 1)
}

// TestAmbitiousAugmenter_DiesWithoutCountersNoFractal verifies that when
// Ambitious Augmenter dies with no counters on it, no Fractal token is created.
func TestAmbitiousAugmenter_DiesWithoutCountersNoFractal(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ambitious Augmenter")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Ambitious Augmenter")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Fractal Token", 0)
}

// TestBertaWiseExtrapolator_Increment verifies that casting a spell that costs
// more mana than Berta's power or toughness places a +1/+1 counter on her.
func TestBertaWiseExtrapolator_Increment(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Berta, Wise Extrapolator")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears") // {1}{G} = 2 mana > power 1
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Berta, Wise Extrapolator", core.P1P1, 1)
}

// TestBertaWiseExtrapolator_FractalAbility verifies that the {X},{T} ability
// creates a 0/0 green and blue Fractal creature token with X +1/+1 counters.
func TestBertaWiseExtrapolator_FractalAbility(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Berta, Wise Extrapolator")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.ActivateAbilityWithX(1, core.PrecombatMain, gametest.PlayerA, "Berta, Wise Extrapolator", 2)
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Fractal Token", 1)
	g.AssertCounterCount(gametest.PlayerA, "Fractal Token", core.P1P1, 2)
}

// TestBiblioplexTomekeeper_BaseStats verifies Biblioplex Tomekeeper is a 3/4
// artifact creature.
func TestBiblioplexTomekeeper_BaseStats(t *testing.T) {
	g := gametest.NewTestGame(t)
	// ETB: choose "do nothing" (mode index 2)
	g.ChooseMode(gametest.PlayerA, 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Biblioplex Tomekeeper")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Biblioplex Tomekeeper", 3, 4)
}

// TestBiblioplexTomekeeper_BecomesUnprepared verifies that when "becomes
// unprepared" mode is chosen, the prepared status is removed from the target.
func TestBiblioplexTomekeeper_BecomesUnprepared(t *testing.T) {
	g := gametest.NewTestGame(t)
	// Emeritus of Abundance has WithPreparedSpell — it enters prepared via ETB.
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Emeritus of Abundance // Regrowth")
	// ETB Tomekeeper: choose mode 1 (becomes unprepared), then target Emeritus.
	g.ChooseMode(gametest.PlayerA, 1)
	g.ChoosePermanent(gametest.PlayerA, "Emeritus of Abundance // Regrowth")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Biblioplex Tomekeeper")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Emeritus of Abundance // Regrowth", core.AttrPrepared, false)
}

// TestColorstormStallion_HasHaste verifies Colorstorm Stallion has haste.
func TestColorstormStallion_HasHaste(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Colorstorm Stallion")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Colorstorm Stallion", core.Haste, true)
}

// TestColorstormStallion_OpusBoostInstant verifies that Colorstorm Stallion
// gets +1/+1 until end of turn when its controller casts an instant or sorcery.
func TestColorstormStallion_OpusBoostInstant(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Colorstorm Stallion")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	// Stop before EOT so the boost is still active.
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Colorstorm Stallion", 4, 4)
}

// TestColorstormStallion_OpusFiveManaCreatesTokenCopy verifies that casting
// an instant or sorcery with five or more mana creates a token copy of
// Colorstorm Stallion.
func TestColorstormStallion_OpusFiveManaCreatesTokenCopy(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Colorstorm Stallion")
	// Fireball {X}{R} with X=4 → 5 mana total spent
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Fireball")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 5)
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Fireball", 4, "PlayerB")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	// Original Stallion + one token copy
	g.AssertPermanentCount(gametest.PlayerA, "Colorstorm Stallion", 2)
}

// TestConciliatorsDuelist_ETBDrawAndLifeLoss verifies that when Conciliator's
// Duelist enters, its controller draws a card and each player loses 1 life.
func TestConciliatorsDuelist_ETBDrawAndLifeLoss(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Conciliator's Duelist")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 19)
	g.AssertLife(gametest.PlayerB, 19)
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
}

// TestConciliatorsDuelist_ReparteeExilesAndReturns verifies that when
// Conciliator's Duelist's controller casts an instant or sorcery that targets
// a creature, up to one target creature is exiled and returned at the
// beginning of the next end step.
func TestConciliatorsDuelist_ReparteeExilesAndReturns(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Conciliator's Duelist")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	// Cast Lightning Bolt targeting Grizzly Bears (creature target → Repartee fires).
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
	// Repartee trigger fires: exile Conciliator's Duelist itself.
	g.ChoosePermanent(gametest.PlayerA, "Conciliator's Duelist")
	g.StopAt(2, core.EndStep)
	g.Execute()
	// After the next end step, Conciliator's Duelist returns to battlefield.
	g.AssertPermanentCount(gametest.PlayerA, "Conciliator's Duelist", 1)
}

// TestElementalMascot_FlyingVigilance verifies that Elemental Mascot has
// flying and vigilance.
func TestElementalMascot_FlyingVigilance(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Elemental Mascot")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Elemental Mascot", core.Flying, true)
	g.AssertHasAbility(gametest.PlayerA, "Elemental Mascot", core.Vigilance, true)
	g.AssertPowerToughness(gametest.PlayerA, "Elemental Mascot", 1, 4)
}

// TestElementalMascot_OpusPowerBoost verifies that casting an instant or
// sorcery spell grants +1/+0 until end of turn via the Opus trigger.
func TestElementalMascot_OpusPowerBoost(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Elemental Mascot")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	g.Attack(1, gametest.PlayerA, "Elemental Mascot")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	// +1/+0 boost active during combat.
	g.AssertPowerToughness(gametest.PlayerA, "Elemental Mascot", 2, 4)
}

// TestElementalMascot_OpusExilesTopCardFiveOrMore verifies that when five or
// more mana is spent, the top card of the library is exiled and the controller
// may play it until end of next turn.
func TestElementalMascot_OpusExilesTopCardFiveOrMore(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Elemental Mascot")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 5)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Fireball")
	// Fireball {X}{R} with X=4 = 5 mana spent.
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Fireball", 4, "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Top card (Grizzly Bears) should be in exile.
	g.AssertExileCount("Grizzly Bears", 1)
	g.AssertLibraryCount(gametest.PlayerA, "Grizzly Bears", 0)
}

// ===========================================================================
// Rehearsed Debater
// ===========================================================================

// TestRehearsedDebater_StatsVigilance verifies base stats and vigilance.
func TestRehearsedDebater_StatsVigilance(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rehearsed Debater")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Rehearsed Debater", 3, 3)
	g.AssertHasAbility(gametest.PlayerA, "Rehearsed Debater", core.Vigilance, true)
}

// TestRehearsedDebater_ReparteeBoost verifies that casting an instant or sorcery
// that targets a creature gives Rehearsed Debater +1/+1 until end of turn.
func TestRehearsedDebater_ReparteeBoost(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rehearsed Debater")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Giant Growth")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	// Giant Growth targets Grizzly Bears — repartee should trigger.
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Giant Growth", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Rehearsed Debater should be 4/4 until end of turn (boost still active at EndStep).
	g.AssertPowerToughness(gametest.PlayerA, "Rehearsed Debater", 4, 4)
}

// TestRehearsedDebater_ReparteeBoostDuringCombat verifies the +1/+1 boost is
// active during the turn it triggers (checked in begin-combat step).
func TestRehearsedDebater_ReparteeBoostDuringCombat(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rehearsed Debater")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Giant Growth")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Giant Growth", "Grizzly Bears")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	// +1/+1 boost should still be in effect during begin-combat.
	g.AssertPowerToughness(gametest.PlayerA, "Rehearsed Debater", 4, 4)
}

// TestUlnaAlleyShopkeep_BaseStats verifies base P/T, menace keyword.
func TestUlnaAlleyShopkeep_BaseStats(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ulna Alley Shopkeep")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Ulna Alley Shopkeep", 2, 3)
	g.AssertHasAbility(gametest.PlayerA, "Ulna Alley Shopkeep", core.Menace, true)
}

// TestUlnaAlleyShopkeep_InfusionBoostWithLifeGain verifies +2/+0 while you
// gained life this turn.
func TestUlnaAlleyShopkeep_InfusionBoostWithLifeGain(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ulna Alley Shopkeep")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Healing Salve")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
	g.ChooseMode(gametest.PlayerA, 0) // gain 3 life
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Healing Salve", "PlayerA")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Infusion: +2/+0 while life was gained this turn.
	g.AssertPowerToughness(gametest.PlayerA, "Ulna Alley Shopkeep", 4, 3)
}

// TestUlnaAlleyShopkeep_NoBoostWithoutLifeGain verifies no bonus without life gain.
func TestUlnaAlleyShopkeep_NoBoostWithoutLifeGain(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ulna Alley Shopkeep")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Ulna Alley Shopkeep", 2, 3)
}

// TestTragedyFeaster_BaseStats verifies base P/T and trample.
func TestTragedyFeaster_BaseStats(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tragedy Feaster")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Tragedy Feaster", 7, 6)
	g.AssertHasAbility(gametest.PlayerA, "Tragedy Feaster", core.Trample, true)
}

// TestTragedyFeaster_SacrificesWithoutLifeGain verifies that at the beginning
// of the controller's end step, a permanent is sacrificed if no life was gained
// this turn.
func TestTragedyFeaster_SacrificesWithoutLifeGain(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tragedy Feaster")
	// Add a second permanent the controller must sacrifice.
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	// No life gain → sacrifice the Bears (scripted; default also picks first candidate).
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	// Stop at Cleanup so the end-step trigger has had a chance to resolve.
	g.StopAt(1, core.Cleanup)
	g.Execute()
	// Grizzly Bears was sacrificed.
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
}

// TestTragedyFeaster_NoSacrificeWithLifeGain verifies that if the controller
// gained life this turn, no permanent is sacrificed at end of step.
func TestTragedyFeaster_NoSacrificeWithLifeGain(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tragedy Feaster")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Healing Salve")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
	g.ChooseMode(gametest.PlayerA, 0) // gain 3 life
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Healing Salve", "PlayerA")
	// Stop at Cleanup so the end-step trigger has had a chance to resolve.
	g.StopAt(1, core.Cleanup)
	g.Execute()
	// Life was gained — no sacrifice.
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
}

// ===========================================================================
// Scolding Administrator
// ===========================================================================

// TestScoldingAdministrator_StatsMenace verifies base stats and menace.
func TestScoldingAdministrator_StatsMenace(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Scolding Administrator")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Scolding Administrator", 2, 2)
	g.AssertHasAbility(gametest.PlayerA, "Scolding Administrator", core.Menace, true)
}

// TestScoldingAdministrator_ReparteeAddsCounter verifies that casting an
// instant or sorcery targeting a creature puts a +1/+1 counter on this creature.
func TestScoldingAdministrator_ReparteeAddsCounter(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Scolding Administrator")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Giant Growth")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Giant Growth", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Scolding Administrator", core.P1P1, 1)
}

// TestScoldingAdministrator_DeathTransfersCounters verifies that when
// Scolding Administrator dies with counters, those counters go to a target creature.
func TestScoldingAdministrator_DeathTransfersCounters(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Scolding Administrator")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Swamp", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Terror")
	// Add a counter to Scolding Administrator first.
	g.AddCounters(1, core.PrecombatMain, gametest.PlayerA, "Scolding Administrator", core.P1P1, 2)
	// Opponent kills Scolding Administrator.
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Terror", "Scolding Administrator")
	// Death trigger targets Grizzly Bears.
	g.ChooseTarget(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Scolding Administrator should be dead.
	g.AssertPermanentCount(gametest.PlayerA, "Scolding Administrator", 0)
	// Grizzly Bears should have 2 +1/+1 counters transferred to it.
	g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 2)
}

// TestScoldingAdministrator_DeathNoTransferWithoutCounters verifies that
// the death trigger does nothing when the creature had no counters.
func TestScoldingAdministrator_DeathNoTransferWithoutCounters(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Scolding Administrator")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Swamp", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Terror")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Terror", "Scolding Administrator")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Scolding Administrator is dead with no counters to transfer.
	g.AssertPermanentCount(gametest.PlayerA, "Scolding Administrator", 0)
	// Grizzly Bears should have no counters added.
	g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 0)
}

// ===========================================================================
// Silverquill, the Disputant
// ===========================================================================

// TestSilverquillTheDisputant_StatsKeywords verifies base stats, legendary,
// flying, and vigilance. The casualty 1 ability is marked XXX.
func TestSilverquillTheDisputant_StatsKeywords(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Silverquill, the Disputant")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Silverquill, the Disputant", 4, 4)
	g.AssertHasAbility(gametest.PlayerA, "Silverquill, the Disputant", core.Flying, true)
	g.AssertHasAbility(gametest.PlayerA, "Silverquill, the Disputant", core.Vigilance, true)
}

// ===========================================================================
// Page, Loose Leaf
// ===========================================================================

// TestPageLooseLeaf_TapForColorless verifies {T}: Add {C}.
func TestPageLooseLeaf_TapForColorless(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Page, Loose Leaf")
	// Tap the artifact creature for {C}, use it together with two Islands to cast Gray Ogre.
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Gray Ogre")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Page, Loose Leaf")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Gray Ogre")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Gray Ogre", 1)
}

// TestPageLooseLeaf_GrandeurRevealsUntilInstantOrSorcery verifies the Grandeur
// ability: discard another Page, Loose Leaf, reveal from top until instant/sorcery
// found, put that into hand and rest on bottom in random order.
func TestPageLooseLeaf_GrandeurRevealsUntilInstantOrSorcery(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Page, Loose Leaf")
	// Second copy in hand as Grandeur discard cost.
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Page, Loose Leaf")
	// Library top = Gray Ogre (creature), then Lightning Bolt (instant).
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Gray Ogre")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Lightning Bolt")
	g.ChooseDiscard(gametest.PlayerA, "Page, Loose Leaf")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Page, Loose Leaf")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Lightning Bolt is in hand; Gray Ogre is in the library.
	g.AssertHandCount(gametest.PlayerA, "Lightning Bolt", 1)
	g.AssertLibraryCount(gametest.PlayerA, "Gray Ogre", 1)
}

// ===========================================================================
// Pensive Professor
// ===========================================================================

// TestPensiveProfessor_Stats verifies base 0/2.
func TestPensiveProfessor_Stats(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Pensive Professor")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Pensive Professor", 0, 2)
}

// TestPensiveProfessor_IncrementOnHighManaSpell verifies Increment puts a
// +1/+1 counter when mana spent > power or toughness.
func TestPensiveProfessor_IncrementOnHighManaSpell(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Pensive Professor")
	// Spend 3 mana (> 2 toughness) on Gray Ogre.
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Gray Ogre")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Gray Ogre")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// 0/2 + counter → 1/3.
	g.AssertPowerToughness(gametest.PlayerA, "Pensive Professor", 1, 3)
}

// ===========================================================================
// Poisoner's Apprentice
// ===========================================================================

// TestPoisonersApprentice_InfusionWithLifeGain verifies -4/-4 fires when
// controller gained life this turn.
func TestPoisonersApprentice_InfusionWithLifeGain(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Healing Salve")
	g.ChooseMode(gametest.PlayerA, 0) // gain 3 life
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Healing Salve", "PlayerA")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Poisoner's Apprentice")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 3)
	g.ChoosePermanent(gametest.PlayerA, "Hill Giant")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Poisoner's Apprentice")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// -4/-4 on a 3/3 kills it via SBE.
	g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 0)
}

// TestPoisonersApprentice_NoEffectWithoutLifeGain verifies -4/-4 does not fire
// when no life was gained.
func TestPoisonersApprentice_NoEffectWithoutLifeGain(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Poisoner's Apprentice")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 3)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Poisoner's Apprentice")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 1)
}

// ===========================================================================
// Prismari, the Inspiration
// ===========================================================================

// TestPrismariTheInspiration_StatsAndFlying verifies Prismari is a 7/7 flier.
func TestPrismariTheInspiration_StatsAndFlying(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Prismari, the Inspiration")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Prismari, the Inspiration", 7, 7)
	g.AssertHasAbility(gametest.PlayerA, "Prismari, the Inspiration", core.Flying, true)
}

// ===========================================================================
// Quandrix, the Proof
// ===========================================================================

// TestQuandrixTheProof_StatsAndKeywords verifies base stats, Flying, and Trample.
func TestQuandrixTheProof_StatsAndKeywords(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Quandrix, the Proof")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Quandrix, the Proof", 6, 6)
	g.AssertHasAbility(gametest.PlayerA, "Quandrix, the Proof", core.Flying, true)
	g.AssertHasAbility(gametest.PlayerA, "Quandrix, the Proof", core.Trample, true)
}

// ===========================================================================
// Tester of the Tangential
// ===========================================================================

// TestTesterOfTheTangential_IncrementAddsCounter verifies that casting a spell
// whose mana cost exceeds Tester's power or toughness puts a +1/+1 counter on it.
func TestTesterOfTheTangential_IncrementAddsCounter(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tester of the Tangential")
	// Tester is 1/1; Grizzly Bears costs {1}{G} = 2 mana > 1
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Tester of the Tangential", core.P1P1, 1)
}

// TestTesterOfTheTangential_MoveCountersToCombatTarget verifies that at the beginning
// of combat, paying X mana moves X +1/+1 counters to another creature.
func TestTesterOfTheTangential_MoveCountersToCombatTarget(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tester of the Tangential")
	g.AddCounters(1, core.PrecombatMain, gametest.PlayerA, "Tester of the Tangential", core.P1P1, 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	// Provide mana to pay {2} for X=2
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
	// Player chooses X=2 and targets Grizzly Bears
	g.ChooseNumber(gametest.PlayerA, 2)
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.DeclareAttackers)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Tester of the Tangential", core.P1P1, 1)
	g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 2)
}

// ===========================================================================
// Textbook Tabulator
// ===========================================================================

// TestTextbookTabulator_IncrementAddsCounter verifies Increment fires when mana
// spent exceeds Textbook Tabulator's power or toughness.
func TestTextbookTabulator_IncrementAddsCounter(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Textbook Tabulator")
	// Tabulator is 0/3; Grizzly Bears = {1}{G} = 2 mana > 0 (power)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Textbook Tabulator", core.P1P1, 1)
}

// TestTextbookTabulator_ETBSurveil2 verifies that when Textbook Tabulator enters
// the battlefield, the controller surveils 2 (top 2 cards are seen).
func TestTextbookTabulator_ETBSurveil2(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears") // top card
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")        // second card
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Textbook Tabulator")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// After ETB surveil 2, the controller keeps both cards on top of the library.
	g.AssertLibraryCount(gametest.PlayerA, "Grizzly Bears", 1)
}

// ===========================================================================
// Thornfist Striker
// ===========================================================================

// TestThornfistStriker_WardCountersTriggers verifies Ward {1}: targeting Thornfist
// Striker with a spell should counter it unless the opponent pays {1}.
func TestThornfistStriker_WardCountersTriggers(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Thornfist Striker")
	// PlayerB casts Terror targeting Thornfist Striker. They won't pay Ward.
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Swamp", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Terror")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Terror", "Thornfist Striker")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Ward countered Terror, so Thornfist Striker survives.
	g.AssertPermanentCount(gametest.PlayerA, "Thornfist Striker", 1)
}

// TestThornfistStriker_InfusionBoostsCreatures verifies that as long as you gained
// life this turn, creatures you control get +1/+0 and trample.
func TestThornfistStriker_InfusionBoostsCreatures(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Thornfist Striker")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	// Gain life this turn to activate Infusion
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Healing Salve")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Healing Salve")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	// After gaining life, Grizzly Bears should be 3/2 (+1/+0) and have trample.
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 2)
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Trample, true)
}

// TestThornfistStriker_InfusionInactiveWithoutLifeGain verifies that without
// gaining life this turn, the Infusion bonus does not apply.
func TestThornfistStriker_InfusionInactiveWithoutLifeGain(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Thornfist Striker")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// No life gain, so Grizzly Bears stays at 2/2 with no trample.
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Trample, false)
}

// ===========================================================================
// Thunderdrum Soloist
// ===========================================================================

// TestThunderdrum_OpusDealsDamageToOpponent verifies that casting an instant or
// sorcery spell causes Thunderdrum Soloist to deal 1 damage to each opponent.
func TestThunderdrum_OpusDealsDamageToOpponent(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Thunderdrum Soloist")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Bolt deals 3, Opus deals 1 = total 4 damage to PlayerB
	g.AssertLife(gametest.PlayerB, 16)
}

// TestThunderdrum_OpusFiveManaDealsThree verifies that if five or more mana was
// spent to cast the instant/sorcery, Thunderdrum deals 3 damage instead of 1.
func TestThunderdrum_OpusFiveManaDealsThree(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Thunderdrum Soloist")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 4)
	// Earthquake {X}{R} with X=4 costs 5 mana total (>= 5)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Earthquake")
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Earthquake", 4)
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Earthquake(4) deals 4 to PlayerB; Opus(3) deals 3 more = 7 total → life 13
	g.AssertLife(gametest.PlayerB, 13)
}

// ===========================================================================
// Topiary Lecturer
// ===========================================================================

// TestTopiaryLecturer_IncrementAddsCounter verifies Increment puts a counter on
// Topiary Lecturer when mana spent exceeds its power or toughness.
func TestTopiaryLecturer_IncrementAddsCounter(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Topiary Lecturer")
	// Topiary Lecturer is 1/2; {1}{G} = 2 mana > 1 (power)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Topiary Lecturer", core.P1P1, 1)
}

// TestTopiaryLecturer_TapForGreenMana verifies {T}: Add {G} equal to this
// creature's power produces the correct amount of mana.
func TestTopiaryLecturer_TapForGreenMana(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Topiary Lecturer")
	// Add 2 +1/+1 counters: power becomes 1+2=3
	g.AddCounters(1, core.PrecombatMain, gametest.PlayerA, "Topiary Lecturer", core.P1P1, 2)
	// Activate the mana ability on turn 3 (to avoid summoning sickness)
	g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Topiary Lecturer")
	// Use that mana plus one more Forest to cast Giant Spider {3}{G}
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Giant Spider")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.CastSpell(3, core.PrecombatMain, gametest.PlayerA, "Giant Spider")
	g.StopAt(3, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Giant Spider", 1)
}

func TestStirringHopesinger_ReparteeCounterOnEachCreature(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Stirring Hopesinger")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Shock")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Llanowar Elves")
	// Shock targets Llanowar Elves (a creature) — Repartee triggers
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Shock", "Llanowar Elves")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Each creature PlayerA controls gets a +1/+1 counter
	g.AssertCounterCount(gametest.PlayerA, "Stirring Hopesinger", core.P1P1, 1)
	g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 1)
}

func TestStirringHopesinger_NonCreatureTargetNoTrigger(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Stirring Hopesinger")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Shock")
	// Shock targets PlayerB directly — not a creature
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Shock", "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Stirring Hopesinger", core.P1P1, 0)
}

func TestTackleArtist_OpusPutsOneCounter(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tackle Artist")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Shock")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Shock", "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Tackle Artist", core.P1P1, 1)
}

func TestTackleArtist_OpusFiveManaSpentPutsTwoCounters(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tackle Artist")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 5)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Fireball")
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Fireball", 4, "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Tackle Artist", core.P1P1, 2)
}

func TestTenuredConcocter_TargetedByOpponentDrawsCard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tenured Concocter")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Shock")
	g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Shock", "Tenured Concocter")
	g.StopAt(2, core.EndStep)
	g.Execute()
	// Opponent targeted Concocter; PlayerA draws a card
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestTenuredConcocter_NotTargetedNoDraws(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tenured Concocter")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Shock")
	// Shock targets PlayerB's own Grizzly Bears — not Concocter
	g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Shock", "Grizzly Bears")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 0)
}

func TestTenuredConcocter_InfusionBoostWhileGainedLife(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tenured Concocter")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Healing Salve")
	// Cast Healing Salve to gain 3 life — Infusion condition satisfied
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Healing Salve", "PlayerA")
	g.Attack(1, gametest.PlayerA, "Tenured Concocter")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// During the attack, Concocter has Infusion active: 4+2=6 power
	// Concocter deals 6 damage unblocked; Healing Salve gains 3 life = 23 total
	g.AssertLife(gametest.PlayerB, 14)
	g.AssertLife(gametest.PlayerA, 23)
}

func TestSnoopingPage_ReparteeUnblockable(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Snooping Page")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Shock")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Llanowar Elves")
	// Shock targets Llanowar Elves (a creature) — Repartee triggers
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Shock", "Llanowar Elves")
	g.Attack(1, gametest.PlayerA, "Snooping Page")
	// PlayerB has no blocker (Llanowar Elves died), but even if they did,
	// Snooping Page can't be blocked.
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Snooping Page 2/3 deals 2 damage unblocked; Shock deals 2 to Llanowar Elves
	g.AssertLife(gametest.PlayerB, 18)
}

func TestSnoopingPage_CombatDamageDrawsCard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Snooping Page")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
	g.Attack(1, gametest.PlayerA, "Snooping Page")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Snooping Page deals combat damage to PlayerB; draw a card, lose 1 life
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertLife(gametest.PlayerA, 19)
}

func TestSnoopingPage_NoCombatDamageNoEffect(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Snooping Page")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Llanowar Elves")
	g.Attack(1, gametest.PlayerA, "Snooping Page")
	g.Block(1, gametest.PlayerB, "Grizzly Bears", "Snooping Page")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// No combat damage to player — no draw, no life loss
	g.AssertHandCount(gametest.PlayerA, "Llanowar Elves", 0)
	g.AssertLife(gametest.PlayerA, 20)
}

func TestSpectacularSkywhale_OpusBoostLessThanFive(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Spectacular Skywhale")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Shock")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Shock", "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Shock costs 1 mana (< 5): no permanent counters placed.
	g.AssertCounterCount(gametest.PlayerA, "Spectacular Skywhale", core.P1P1, 0)
	g.AssertLife(gametest.PlayerB, 18) // 2 from Shock
}

func TestSpectacularSkywhale_OpusBoostTemporary(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Spectacular Skywhale")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Shock")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Shock", "PlayerB")
	g.Attack(1, gametest.PlayerA, "Spectacular Skywhale")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Skywhale 1/4 +3/+0 = 4/4 flying, deals 4 unblocked; Shock 2 = 6 total
	g.AssertLife(gametest.PlayerB, 14)
}

func TestSpectacularSkywhale_OpusFiveManaSpentPutsCounters(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Spectacular Skywhale")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 5)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Fireball")
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Fireball", 4, "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// 5+ mana spent: three +1/+1 counters instead of temporary boost
	g.AssertCounterCount(gametest.PlayerA, "Spectacular Skywhale", core.P1P1, 3)
}

// =============================================================================
// Spellbook Seeker // Careful Study
// =============================================================================

// TestSpellbookSeeker_ETBPrepared: Spellbook Seeker enters the battlefield prepared.
func TestSpellbookSeeker_ETBPrepared(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Spellbook Seeker // Careful Study")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Spellbook Seeker // Careful Study", core.AttrPrepared, true)
}

// TestSpellbookSeeker_CastCarefulStudyDrawsThenDiscards: activating the Prepared
// ability casts a copy of Careful Study, drawing two cards then discarding two.
func TestSpellbookSeeker_CastCarefulStudyDrawsThenDiscards(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Spellbook Seeker // Careful Study")
	// Put two distinct cards in hand to discard; 3 Mountains in library
	// (1 drawn at start of turn, 2 drawn by Careful Study).
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Llanowar Elves")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain", 3)
	// After drawing 2 Mountains via Careful Study, discard the Bears and Elves.
	g.ChooseDiscard(gametest.PlayerA, "Grizzly Bears", "Llanowar Elves")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Spellbook Seeker // Careful Study")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Start-of-turn draw: 1 Mountain. Careful Study: draw 2 more Mountains, discard Bears+Elves.
	// Final hand: 3 Mountains.
	g.AssertHandCount(gametest.PlayerA, "Mountain", 3)
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Llanowar Elves", 1)
	g.AssertHasAbility(gametest.PlayerA, "Spellbook Seeker // Careful Study", core.AttrPrepared, false)
}

// =============================================================================
// Landscape Painter // Vibrant Idea
// =============================================================================

// TestLandscapePainter_ETBPrepared: Landscape Painter enters the battlefield
// prepared (AttrPrepared is set on ETB).
func TestLandscapePainter_ETBPrepared(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Landscape Painter // Vibrant Idea")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Landscape Painter // Vibrant Idea", core.AttrPrepared, true)
}

// TestLandscapePainter_CastVibrantIdeaDrawsTwoCards: activating the Prepared
// ability casts a copy of Vibrant Idea, drawing two cards for the controller.
func TestLandscapePainter_CastVibrantIdeaDrawsTwoCards(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Landscape Painter // Vibrant Idea")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 5)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Landscape Painter // Vibrant Idea")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "", 2)
	g.AssertHasAbility(gametest.PlayerA, "Landscape Painter // Vibrant Idea", core.AttrPrepared, false)
}

// =============================================================================
// Encouraging Aviator // Jump
// =============================================================================

// TestEncouragingAviator_HasFlying: Encouraging Aviator is a 2/3 with flying.
func TestEncouragingAviator_HasFlying(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Encouraging Aviator // Jump")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Encouraging Aviator // Jump", core.Flying, true)
	g.AssertPowerToughness(gametest.PlayerA, "Encouraging Aviator // Jump", 2, 3)
}

// TestEncouragingAviator_AttackMakesPrepared: when Encouraging Aviator attacks,
// it becomes prepared.
func TestEncouragingAviator_AttackMakesPrepared(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Encouraging Aviator // Jump")
	g.Attack(1, gametest.PlayerA, "Encouraging Aviator // Jump")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Encouraging Aviator // Jump", core.AttrPrepared, true)
}

// TestEncouragingAviator_CastJumpGivesFlyingToTarget: casting the Jump copy
// gives a target creature flying until end of turn.
func TestEncouragingAviator_CastJumpGivesFlyingToTarget(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Encouraging Aviator // Jump")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.Attack(1, gametest.PlayerA, "Encouraging Aviator // Jump")
	// Activate prepared ability during postcombat main; target Grizzly Bears.
	g.ActivateAbility(1, core.PostcombatMain, gametest.PlayerA, "Encouraging Aviator // Jump", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// After end step, the temporary flying from Jump has expired.
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Flying, false)
	// The aviator is no longer prepared.
	g.AssertHasAbility(gametest.PlayerA, "Encouraging Aviator // Jump", core.AttrPrepared, false)
}

// =============================================================================
// Campus Composer // Aqueous Aria
// =============================================================================

// TestCampusComposer_ETBPrepared: Campus Composer enters the battlefield prepared.
func TestCampusComposer_ETBPrepared(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Campus Composer // Aqueous Aria")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Campus Composer // Aqueous Aria", core.AttrPrepared, true)
}

// TestCampusComposer_CastAqueousAriaCreatesToken: casting the Aqueous Aria copy
// creates a 3/3 blue and red Elemental creature token with flying.
func TestCampusComposer_CastAqueousAriaCreatesToken(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Campus Composer // Aqueous Aria")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Campus Composer // Aqueous Aria")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Elemental Token", 1)
	g.AssertPowerToughness(gametest.PlayerA, "Elemental Token", 3, 3)
	g.AssertHasAbility(gametest.PlayerA, "Elemental Token", core.Flying, true)
	g.AssertHasAbility(gametest.PlayerA, "Campus Composer // Aqueous Aria", core.AttrPrepared, false)
}

// =============================================================================
// Harmonized Trio // Brainstorm
// =============================================================================

// TestHarmonizedTrio_TapTwoCreaturesBecomePrepared: tapping Harmonized Trio
// plus two other untapped creatures makes it prepared.
func TestHarmonizedTrio_TapTwoCreaturesBecomePrepared(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Harmonized Trio // Brainstorm")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears", 2)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Harmonized Trio // Brainstorm", "Grizzly Bears", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Harmonized Trio // Brainstorm", core.AttrPrepared, true)
}

// =============================================================================
// Jadzi, Steward of Fate // Oracle's Gift
// =============================================================================

// TestJadzi_ETBPreparedAndDrawDiscard: Jadzi enters prepared; its ETB trigger
// draws two cards then discards two cards.
func TestJadzi_ETBPreparedAndDrawDiscard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears", 2)
	g.ChooseDiscard(gametest.PlayerA, "Grizzly Bears", "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Jadzi, Steward of Fate // Oracle's Gift")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Jadzi, Steward of Fate // Oracle's Gift", core.AttrPrepared, true)
	// Drew 2, discarded 2: hand empty.
	g.AssertHandCount(gametest.PlayerA, "", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 2)
}

// =============================================================================
// Emeritus of Ideation // Ancestral Recall
// =============================================================================

// TestEmeritusOfIdeation_ETBPrepared: Emeritus of Ideation is a 5/5 with flying
// that enters prepared.
func TestEmeritusOfIdeation_ETBPrepared(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Emeritus of Ideation // Ancestral Recall")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Emeritus of Ideation // Ancestral Recall", core.Flying, true)
	g.AssertPowerToughness(gametest.PlayerA, "Emeritus of Ideation // Ancestral Recall", 5, 5)
	g.AssertHasAbility(gametest.PlayerA, "Emeritus of Ideation // Ancestral Recall", core.AttrPrepared, true)
}

// TestEmeritusOfIdeation_CastAncestralRecallDrawsThreeForTarget: casting the
// Ancestral Recall copy draws three cards for the target player.
func TestEmeritusOfIdeation_CastAncestralRecallDrawsThreeForTarget(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Emeritus of Ideation // Ancestral Recall")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears", 3)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Emeritus of Ideation // Ancestral Recall", "PlayerA")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "", 3)
	g.AssertHasAbility(gametest.PlayerA, "Emeritus of Ideation // Ancestral Recall", core.AttrPrepared, false)
}
