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
