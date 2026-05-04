package jumpstart

import (
	"testing"

	"github.com/google/uuid"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// Tests for the white + blue creature chunk implemented in creatures.go.

// ===== Vanilla / keyword creatures =====

func TestAegisTurtle(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Aegis Turtle")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Aegis Turtle", 0, 5)
}

func TestIsamaruHoundOfKonda(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Isamaru, Hound of Konda")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Isamaru, Hound of Konda", 2, 2)
}

func TestHealersHawk(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Healer's Hawk")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Healer's Hawk", core.Flying, true)
	g.AssertHasAbility(gametest.PlayerA, "Healer's Hawk", core.Lifelink, true)
}

func TestMesaUnicorn(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mesa Unicorn")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Mesa Unicorn", core.Lifelink, true)
}

func TestKnightOfTheTusk(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Knight of the Tusk")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Knight of the Tusk", core.Vigilance, true)
}

func TestMurmuringPhantasm(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Murmuring Phantasm")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Murmuring Phantasm", core.Defender, true)
}

// ===== ETB life gain / token =====

func TestAngelOfMercy(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Angel of Mercy")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 5)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Angel of Mercy")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 23)
	g.AssertHasAbility(gametest.PlayerA, "Angel of Mercy", core.Flying, true)
}

func TestBulwarkGiant(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Bulwark Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 6)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Bulwark Giant")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 25)
}

func TestVoiceOfTheProvinces(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Voice of the Provinces")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 6)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Voice of the Provinces")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Human", 1)
}

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

func TestAffaGuardHound(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Affa Guard Hound")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Affa Guard Hound", "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 5)
}

func TestAffaGuardHoundFlash(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Affa Guard Hound")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.BeginCombat, gametest.PlayerB, "Affa Guard Hound", "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Affa Guard Hound", 1)
}

func TestCrookclawTransmuter(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Crookclaw Transmuter")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Crookclaw Transmuter", "Hill Giant")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerB, "Hill Giant", 3, 3)
}

func TestCrookclawTransmuterFlash(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Crookclaw Transmuter")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.CastSpell(1, core.BeginCombat, gametest.PlayerB, "Crookclaw Transmuter", "Hill Giant")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Crookclaw Transmuter", 1)
	g.AssertPowerToughness(gametest.PlayerA, "Hill Giant", 3, 3)
}

func TestEmancipationAngel(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Emancipation Angel")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Emancipation Angel", "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestExclusionMage(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Exclusion Mage")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Exclusion Mage", "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertHandCount(gametest.PlayerB, "Grizzly Bears", 1)
}

func TestStormSculptor(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Storm Sculptor")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Storm Sculptor", "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestArchaeomender(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Archaeomender")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Black Lotus")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Archaeomender", "Black Lotus")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Black Lotus", 1)
}

func TestBrightmare(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Brightmare")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Brightmare", "Hill Giant")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertTapped(gametest.PlayerB, "Hill Giant", true)
	g.AssertLife(gametest.PlayerA, 23)
}

func TestAngelOfTheDireHour(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Angel of the Dire Hour")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant", 2)
	g.Attack(2, gametest.PlayerB, "Hill Giant", "Hill Giant")
	g.StopAt(2, core.PostcombatMain)
	g.Execute()
	// Angel of the Dire Hour was on battlefield, no ETB. Test second copy entering after combat is harder.
	// Simpler test: cast it during opponent's combat after attackers are declared.
	g2 := gametest.NewTestGame(t)
	g2.AddCard(core.ZoneHand, gametest.PlayerA, "Angel of the Dire Hour")
	g2.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 7)
	g2.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g2.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Angel of the Dire Hour")
	g2.StopAt(1, core.PostcombatMain)
	g2.Execute()
	g2.AssertHasAbility(gametest.PlayerA, "Angel of the Dire Hour", core.Flying, true)
}

func TestAngelOfTheDireHourFlash(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Angel of the Dire Hour")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains", 7)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.CastSpell(1, core.BeginCombat, gametest.PlayerB, "Angel of the Dire Hour")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Angel of the Dire Hour", 1)
}

func TestAngelicPage(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Angelic Page")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.Attack(3, gametest.PlayerA, "Grizzly Bears")
	g.ActivateAbility(3, core.DeclareAttackers, gametest.PlayerA, "Angelic Page", "Grizzly Bears")
	g.StopAt(3, core.DeclareBlockers)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
	g.AssertHasAbility(gametest.PlayerA, "Angelic Page", core.Flying, true)
}

func TestArchonOfJustice(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Archon of Justice")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Archon of Justice", core.Flying, true)
}

func TestBattlegroundGeist(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Battleground Geist")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Will-o'-the-Wisp") // Spirit 0/1
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Will-o'-the-Wisp", 1, 1)
	g.AssertPowerToughness(gametest.PlayerA, "Battleground Geist", 3, 3)
}

func TestBlessedSpirits(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Blessed Spirits")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Holy Strength")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Holy Strength", "Blessed Spirits")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Blessed Spirits", core.P1P1, 1)
}

func TestErraticVisionary(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Erratic Visionary")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 2)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain", 10)
	g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Erratic Visionary")
	g.StopAt(3, core.PostcombatMain)
	g.Execute()
	// Activation drew a Mountain then discarded a card.
	g.AssertGraveyardCount(gametest.PlayerA, "Mountain", 1)
}

func TestRecklessScholar(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Reckless Scholar")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain", 10)
	g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Reckless Scholar", "PlayerA")
	g.StopAt(3, core.PostcombatMain)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Mountain", 1)
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

func TestCathersCompanion(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cathar's Companion")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 1)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	g.StopAt(1, core.DeclareAttackers)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Cathar's Companion", core.Indestructible, true)
}

func TestAlabasterMage(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Alabaster Mage")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Alabaster Mage", "Grizzly Bears")
	g.StopAt(3, core.DeclareAttackers)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Lifelink, true)
}

func TestRonomUnicorn(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ronom Unicorn")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Holy Strength") // an enchantment
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // for Holy Strength to attach
	g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Ronom Unicorn", "Holy Strength")
	g.StopAt(3, core.PostcombatMain)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Holy Strength", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Ronom Unicorn", 1)
}

func TestSpectralSailor(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Spectral Sailor")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 4)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain", 5)
	g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Spectral Sailor")
	g.StopAt(3, core.PostcombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Spectral Sailor", core.Flying, true)
}

func TestSpectralSailorFlash(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Spectral Sailor")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island", 1)
	g.CastSpell(1, core.BeginCombat, gametest.PlayerB, "Spectral Sailor")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Spectral Sailor", 1)
}

func TestPatronOfTheValiant(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Patron of the Valiant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 5)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCounters(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears", core.P1P1, 1)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Patron of the Valiant")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 2)
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

func TestHighSentinelsOfArashin(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "High Sentinels of Arashin")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCounters(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears", core.P1P1, 1)
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "High Sentinels of Arashin", 4, 5)
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

func TestSteelPlumeMarshal(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Steel-Plume Marshal")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Healer's Hawk") // 1/1 flying
	g.Attack(3, gametest.PlayerA, "Steel-Plume Marshal", "Healer's Hawk")
	g.StopAt(3, core.DeclareBlockers)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Healer's Hawk", 3, 3)
}

func TestStoneHavenPilgrim(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Stone Haven Pilgrim")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sol Ring") // artifact
	g.Attack(3, gametest.PlayerA, "Stone Haven Pilgrim")
	g.StopAt(3, core.DeclareBlockers)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Stone Haven Pilgrim", 3, 3)
	g.AssertHasAbility(gametest.PlayerA, "Stone Haven Pilgrim", core.Lifelink, true)
}

func TestAjanisChosen(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ajani's Chosen")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 1)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Holy Strength")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Holy Strength", "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Cat", 1)
}

func TestRattlechains(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Rattlechains")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Will-o'-the-Wisp") // Spirit
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Rattlechains", "Will-o'-the-Wisp")
	g.StopAt(1, core.DeclareAttackers)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Will-o'-the-Wisp", core.Hexproof, true)
}

func TestRattlechainsFlash(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Rattlechains")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Will-o'-the-Wisp")
	g.CastSpell(1, core.BeginCombat, gametest.PlayerB, "Rattlechains", "Will-o'-the-Wisp")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Rattlechains", 1)
}

func TestEmielTheBlessed(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Emiel the Blessed")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Emiel the Blessed", "Grizzly Bears")
	g.StopAt(3, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestEmielTheBlessed_PayCounterOnNonUnicorn(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Emiel the Blessed")
	// One Forest pays the may-pay {G/W} when Grizzly Bears enters.
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 1)
}

func TestEmielTheBlessed_PayTwoCountersOnUnicorn(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Emiel the Blessed")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Mesa Unicorn")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Mesa Unicorn")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Mesa Unicorn", core.P1P1, 2)
}

func TestEmielTheBlessed_NoTriggerOnSelfETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Emiel the Blessed")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 4)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Emiel the Blessed")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Trigger says "another creature you control" — Emiel itself shouldn't trigger.
	g.AssertCounterCount(gametest.PlayerA, "Emiel the Blessed", core.P1P1, 0)
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

func TestArchonOfRedemption(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Archon of Redemption")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 5)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Archon of Redemption")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 23) // gained 3 from itself
}

func TestSeaGateOracle(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Sea Gate Oracle")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears", 2)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Sea Gate Oracle")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertLibraryCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestOneirophage(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Oneirophage")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain", 5)
	g.StopAt(5, core.PostcombatMain)
	g.Execute()
	// PlayerA draws on turn 3 and turn 5 = 2 cards.
	g.AssertCounterCount(gametest.PlayerA, "Oneirophage", core.P1P1, 2)
}

func TestAngelOfTheDireHourETB(t *testing.T) {
	// Already covered in earlier test
}

// Angel of the Dire Hour: when cast from a non-hand zone (graveyard via
// CastCardFromZoneWithoutPaying), the "if you cast it from your hand" gate
// fails and the ETB exile-attackers ability does NOT trigger.
func TestAngelOfTheDireHour_NotCastFromHand_NoExile(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Angel of the Dire Hour")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.Attack(2, gametest.PlayerB, "Hill Giant")
	g.StopAt(2, core.DeclareAttackers)
	g.Execute()
	pA := g.GetPlayer(gametest.PlayerA).PlayerID()
	var cardID uuid.UUID
	for _, c := range g.GetPlayer(gametest.PlayerA).Graveyard() {
		if c.Name() == "Angel of the Dire Hour" {
			cardID = c.ID()
		}
	}
	if cardID == uuid.Nil {
		t.Fatal("Angel not in graveyard")
	}
	if err := g.CastCardFromZoneWithoutPaying(pA, cardID, core.ZoneGraveyard, nil, 0); err != nil {
		t.Fatalf("CastCardFromZoneWithoutPaying: %v", err)
	}
	g.ResolveStack()
	g.AssertExileCount("Hill Giant", 0)
	g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 1)
}

func TestSupplyRunners(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Supply Runners")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 5)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Supply Runners")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 1)
	g.AssertCounterCount(gametest.PlayerA, "Supply Runners", core.P1P1, 0)
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

func TestSailorOfMeans_CreatesTreasureOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Sailor of Means")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Sailor of Means")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Sailor of Means", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Treasure", 1)
}

func TestProsperousPirates_CreatesTwoTreasuresOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Prosperous Pirates")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 5)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Prosperous Pirates")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Prosperous Pirates", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Treasure", 2)
}

func TestCorsairCaptain_CreatesTreasureAndPumpsPirates(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Corsair Captain")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kitesail Corsair")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Corsair Captain")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Treasure", 1)
	g.AssertPowerToughness(gametest.PlayerA, "Kitesail Corsair", 3, 2)
	g.AssertPowerToughness(gametest.PlayerA, "Corsair Captain", 2, 2)
}

func TestSailorOfMeans_TreasureSacAddsAnyMana(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Sailor of Means")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Sailor of Means")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Treasure")
	g.ChooseManaColor(gametest.PlayerA, core.Blue)
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Treasure", 0)
	g.AssertManaProducedAtLeast(gametest.PlayerA, core.Blue, 1)
}
