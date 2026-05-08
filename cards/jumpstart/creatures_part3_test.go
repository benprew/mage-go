package jumpstart

import (
	"testing"

	_ "git.sr.ht/~cdcarter/mage-go/cards/antiquities"
	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
	mage "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

func TestInspiringCaptain(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Inspiring Captain")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Inspiring Captain")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
	g.AssertPowerToughness(gametest.PlayerA, "Inspiring Captain", 4, 4)
}
func TestInspiringUnicorn(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Inspiring Unicorn")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.Attack(3, gametest.PlayerA, "Inspiring Unicorn", "Grizzly Bears")
	g.StopAt(3, core.DeclareBlockers)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
	g.AssertPowerToughness(gametest.PlayerA, "Inspiring Unicorn", 3, 3)
}
func TestIronrootWarlord_PowerEqualsCreatures(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ironroot Warlord")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Ironroot Warlord", 2, 5)
}
func TestIronshellBeetle_AddsCounterToTarget(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Ironshell Beetle")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Ironshell Beetle", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
}
func TestIsamaruHoundOfKonda(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Isamaru, Hound of Konda")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Isamaru, Hound of Konda", 2, 2)
}
func TestJoustingDummy_BoostsPower(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Jousting Dummy")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Jousting Dummy")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Jousting Dummy", 3, 1)
}
func TestKalastriaNightwatch_GainsFlyingOnLifeGain(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kalastria Nightwatch")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Healing Salve")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
	g.AssertHasAbility(gametest.PlayerA, "Kalastria Nightwatch", core.Flying, false)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Healing Salve", "PlayerA")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Kalastria Nightwatch", core.Flying, true)
}

// Keeper of Fables: non-Human combat damage triggers a draw.
func TestKeeperOfFables_NonHumanDrawsCard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Keeper of Fables")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain", 3)
	g.Attack(3, gametest.PlayerA, "Keeper of Fables")
	g.StopAt(3, core.EndStep)
	g.Execute()
	// Keeper itself is a Cat — non-Human — so combat damage triggers draw.
	g.AssertHandCount(gametest.PlayerA, "Mountain", 1)
}
func TestKelsFightFixer_DeclineDraw(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kels, Fight Fixer")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
	tpA := g.GetPlayer(gametest.PlayerA)
	tpA.QueueMayAbilityChoices(false)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Kels, Fight Fixer")
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 0)
}
func TestKelsFightFixer_DrawOnSacrifice(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kels, Fight Fixer")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	// One Swamp pays the {1} sacrifice cost, another pays {U/B} for the may-pay draw.
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Kels, Fight Fixer")
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 1)
}
func TestKelsFightFixer_DrawOnSelfSacrifice(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kels, Fight Fixer")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	var kels *mage.Permanent
	for _, p := range g.AllBattlefield() {
		if p.Name() == "Kels, Fight Fixer" {
			kels = p
			break
		}
	}
	if kels == nil {
		t.Fatal("Kels not on battlefield")
	}
	g.Sacrifice(kels)
	g.PutTriggersOnStack()
	g.ResolveStack()
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 1)
}
func TestKelsFightFixer_SacForIndestructible(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kels, Fight Fixer")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Kels, Fight Fixer")
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Kels, Fight Fixer", core.Indestructible, true)
}
func TestKilnFiend_BoostsOnInstant(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kiln Fiend")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Kiln Fiend", 4, 2)
}

// Kira, Great Glass-Spinner: targeting Kira herself counters the spell, since
// the granted "creatures you control" trigger now includes the source.
func TestKiraGreatGlassSpinner_CountersTargetingSelf(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kira, Great Glass-Spinner")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Kira, Great Glass-Spinner")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Kira, Great Glass-Spinner", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Lightning Bolt", 1)
}
func TestKitesailCorsair(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kitesail Corsair")
	g.StopAt(3, core.PrecombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Kitesail Corsair", core.Flying, false)

	g2 := gametest.NewTestGame(t)
	g2.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kitesail Corsair")
	g2.Attack(3, gametest.PlayerA, "Kitesail Corsair")
	g2.StopAt(3, core.DeclareBlockers)
	g2.Execute()
	g2.AssertHasAbility(gametest.PlayerA, "Kitesail Corsair", core.Flying, true)
}
func TestKnightOfTheTusk(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Knight of the Tusk")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Knight of the Tusk", core.Vigilance, true)
}
func TestKorSpiritdancer(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kor Spiritdancer")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 1)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Holy Strength")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Holy Strength", "Kor Spiritdancer")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	// Holy Strength is +1/+2; with base 0/2 + 2/2 (one aura) + 1/2 aura = 3/6.
	g.AssertPowerToughness(gametest.PlayerA, "Kor Spiritdancer", 3, 6)
}
func TestKrenkoMobBoss_CreatesGoblinsEqualToCount(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Krenko, Mob Boss")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mons's Goblin Raiders")
	g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Krenko, Mob Boss")
	g.StopAt(3, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Goblin", 2)
}

// Lathliss: another nontoken Dragon you control entering creates a 5/5 Dragon token.
// Lathliss herself entering does not trigger (own ETB is filtered by EventSourceNotSelf),
// and the 5/5 Dragon token she creates does not re-trigger her ability (nontoken filter).
func TestLathlissDragonQueen_CreatesDragonOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lathliss, Dragon Queen")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 6)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Shrieker")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Shrieker")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	// Exactly one 5/5 Dragon token from Lightning Shrieker's nontoken ETB.
	// Lathliss is named "Lathliss, Dragon Queen"; the token is named "Dragon".
	g.AssertPermanentCount(gametest.PlayerA, "Dragon", 1)
}
func TestLawlessBroker_CounterOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lawless Broker")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Lawless Broker")
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
}
func TestLeafGilder_TapsForGreen(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Leaf Gilder")
	g.StopAt(1, core.EndStep)
	g.Execute()
}
func TestLenaSelflessChampion(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lena, Selfless Champion")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 6)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears", 2)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lena, Selfless Champion")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Soldier", 3)
}
func TestLightningCoreExcavator_DealsThreeOnSac(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lightning-Core Excavator")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 5)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Lightning-Core Excavator", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Lightning-Core Excavator", 1)
}
func TestLightningShrieker_ShufflesIntoLibrary(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lightning Shrieker")
	g.StopAt(2, core.Untap)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Lightning Shrieker", 0)
}
func TestLightningVisionary_ProwessOnNoncreature(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lightning Visionary")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Lightning Visionary", 3, 2)
}
func TestLightwalker(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lightwalker")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Lightwalker", core.Flying, false)

	g2 := gametest.NewTestGame(t)
	g2.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lightwalker")
	g2.AddCounters(1, core.PrecombatMain, gametest.PlayerA, "Lightwalker", core.P1P1, 1)
	g2.StopAt(1, core.PostcombatMain)
	g2.Execute()
	g2.AssertHasAbility(gametest.PlayerA, "Lightwalker", core.Flying, true)
}
func TestLilianasElite_PTByGraveyardCreatures(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Liliana's Elite")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears", 3)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Plains")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Liliana's Elite", 4, 4)
}
func TestLilianasReaver_DiscardAndZombieToken(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Liliana's Reaver")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Mountain")
	g.Attack(3, gametest.PlayerA, "Liliana's Reaver")
	g.StopAt(3, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 16)
	g.AssertHandCount(gametest.PlayerB, "Mountain", 0)
	g.AssertPermanentCount(gametest.PlayerA, "Zombie", 1)
}
func TestLinvalaKeeperOfSilence_BlocksOpponentManaAbility(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Linvala, Keeper of Silence")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Llanowar Elves")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerB, "Llanowar Elves")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertTapped(gametest.PlayerB, "Llanowar Elves", false)
}

// Linvala, Keeper of Silence: Activated abilities of creatures your opponents
// control can't be activated.
func TestLinvalaKeeperOfSilence_BlocksOpponentNonManaActivation(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Linvala, Keeper of Silence")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Prodigal Sorcerer")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerB, "Prodigal Sorcerer", "PlayerA")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 20)
	g.AssertTapped(gametest.PlayerB, "Prodigal Sorcerer", false)
}
func TestLinvalaKeeperOfSilence_OpponentNonCreatureUnaffected(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Linvala, Keeper of Silence")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Jayemdae Tome")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island", 5)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerB, "Jayemdae Tome")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertTapped(gametest.PlayerB, "Jayemdae Tome", true)
}
func TestLinvalaKeeperOfSilence_OwnCreaturesUnaffected(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Linvala, Keeper of Silence")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Prodigal Sorcerer")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Prodigal Sorcerer", "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 19)
	g.AssertTapped(gametest.PlayerA, "Prodigal Sorcerer", true)
}
func TestLivingLightning_ReturnsInstantOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Living Lightning")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")
	g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Living Lightning")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Lightning Bolt", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Living Lightning", 1)
}
func TestMalakirFamiliar_PumpsOnLifeGain(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Malakir Familiar")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Healing Salve")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Healing Salve", "PlayerA")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Malakir Familiar", 3, 2)
}
func TestMausoleumTurnkey_OpponentChoosesReturn(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Mausoleum Turnkey")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
	g.ChooseTarget(gametest.PlayerB, "Hill Giant")
	g.ChooseTarget(gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Mausoleum Turnkey")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Hill Giant", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
}
func TestMentorOfTheMeek(t *testing.T) {
	// Baseline: cast Bears with no Mentor — count Mountains drawn.
	g0 := gametest.NewTestGame(t)
	g0.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 5)
	g0.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g0.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain", 10)
	g0.CastSpell(3, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
	g0.StopAt(3, core.PostcombatMain)
	g0.Execute()
	baseline := 0
	for _, c := range g0.GetPlayer(gametest.PlayerA).Hand() {
		if c.Name() == "Mountain" {
			baseline++
		}
	}

	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mentor of the Meek")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 5)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain", 10)
	g.CastSpell(3, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(3, core.PostcombatMain)
	g.Execute()
	got := 0
	for _, c := range g.GetPlayer(gametest.PlayerA).Hand() {
		if c.Name() == "Mountain" {
			got++
		}
	}
	if got != baseline+1 {
		t.Errorf("expected baseline+1 (%d) Mountains but got %d", baseline+1, got)
	}
}
func TestMesaUnicorn(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mesa Unicorn")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Mesa Unicorn", core.Lifelink, true)
}
func TestMeteorGolem_DestroysNonland(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Meteor Golem")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 7)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Meteor Golem")
	g.ChooseTarget(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}
func TestMiasmicMummy_EachPlayerDiscards(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Miasmic Mummy")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Plains")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Miasmic Mummy")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Plains", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Mountain", 1)
}
func TestMikaeusTheLunarch(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Mikaeus, the Lunarch")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 4)
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Mikaeus, the Lunarch", 3)
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Mikaeus, the Lunarch", core.P1P1, 3)
	g.AssertPowerToughness(gametest.PlayerA, "Mikaeus, the Lunarch", 3, 3)
}
func TestMinotaurSkullcleaver_BoostsOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Minotaur Skullcleaver")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 3)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Minotaur Skullcleaver")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Minotaur Skullcleaver", 4, 2)
}
func TestMinotaurSureshot_BoostsForOneR(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Minotaur Sureshot")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Minotaur Sureshot")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Minotaur Sureshot", 3, 3)
}
func TestMireTriton_MillsAndGains(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Mire Triton")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mox Ruby", 4)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Mire Triton")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Mox Ruby", 2)
	g.AssertLife(gametest.PlayerA, 22)
}
func TestMoltenRavager_BoostsForR(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Molten Ravager")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Molten Ravager")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Molten Ravager", 1, 4)
}
func TestMurmuringPhantasm(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Murmuring Phantasm")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Murmuring Phantasm", core.Defender, true)
}
func TestMyrSire_CreatesTokenOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Myr Sire")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Craw Wurm")
	g.Attack(1, gametest.PlayerA, "Myr Sire")
	g.Block(1, gametest.PlayerB, "Craw Wurm", "Myr Sire")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Myr", 1)
}
func TestMysticArchaeologist(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mystic Archaeologist")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 5)
	// Baseline: no ability use, just normal draws.
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain", 10)
	g.StopAt(3, core.PostcombatMain)
	g.Execute()
	baseline := 0
	for _, c := range g.GetPlayer(gametest.PlayerA).Hand() {
		if c.Name() == "Mountain" {
			baseline++
		}
	}

	g2 := gametest.NewTestGame(t)
	g2.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mystic Archaeologist")
	g2.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 5)
	g2.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain", 10)
	g2.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Mystic Archaeologist")
	g2.StopAt(3, core.PostcombatMain)
	g2.Execute()
	got := 0
	for _, c := range g2.GetPlayer(gametest.PlayerA).Hand() {
		if c.Name() == "Mountain" {
			got++
		}
	}
	if got != baseline+2 {
		t.Errorf("expected baseline+2 (%d) Mountains but got %d", baseline+2, got)
	}
}
func TestNebelgastHerald(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Nebelgast Herald")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Nebelgast Herald", "Hill Giant")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertTapped(gametest.PlayerB, "Hill Giant", true)
	g.AssertHasAbility(gametest.PlayerA, "Nebelgast Herald", core.Flying, true)
}
func TestNebelgastHeraldFlash(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Nebelgast Herald")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.CastSpell(1, core.BeginCombat, gametest.PlayerB, "Nebelgast Herald", "Hill Giant")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Nebelgast Herald", 1)
	g.AssertTapped(gametest.PlayerA, "Hill Giant", true)
}
func TestNightshadeStinger_CantBlock(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Nightshade Stinger")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.Attack(2, gametest.PlayerB, "Hill Giant")
	g.Block(2, gametest.PlayerA, "Nightshade Stinger", "Hill Giant")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 17)
	g.AssertPermanentCount(gametest.PlayerA, "Nightshade Stinger", 1)
}
func TestNocturnalFeeder_DrainsOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Nocturnal Feeder")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Nocturnal Feeder")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 22)
	g.AssertLife(gametest.PlayerB, 18)
}
func TestNyxathid_DiesIfOpponentHasSevenCards(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Nyxathid")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Mountain", 7)
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Nyxathid", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Nyxathid", 1)
}
func TestNyxathid_EmptyOpponentHandLeavesBaseStats(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Nyxathid")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Nyxathid", 7, 7)
}
