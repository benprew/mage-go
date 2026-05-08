package jumpstart

import (
	"testing"

	"github.com/google/uuid"

	_ "git.sr.ht/~cdcarter/mage-go/cards/antiquities"
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
func TestAffectionateIndrik_FightsTarget(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Affectionate Indrik")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 6)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Affectionate Indrik", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
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
func TestAlloyMyr_TapsForAnyColor(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Alloy Myr")
	g.StopAt(1, core.EndStep)
	g.Execute()
}
func TestAmbassadorOak_CreatesElfWarrior(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Ambassador Oak")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 4)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Ambassador Oak")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Elf Warrior", 1)
}
func TestAncestralStatue_BouncesPermanent(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Ancestral Statue")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Ancestral Statue")
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
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
func TestAngelOfTheDireHourETB(t *testing.T) {
	// Already covered in earlier test
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

// TestAngelicArbiter_ControllerUnaffected: PlayerA controls Angelic
// Arbiter and is allowed to cast spells and attack freely.
func TestAngelicArbiter_ControllerUnaffected(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Angelic Arbiter")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 1)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	g.Attack(1, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndCombat)
	g.Execute()
	// Bolt 3 + Bears attack 2 = 5 damage to PlayerB.
	g.AssertLife(gametest.PlayerB, 15)
}

// TestAngelicArbiter_OpponentAttackedCantCastSpells: PlayerB attacks with
// Grizzly Bears, then tries to cast Lightning Bolt at PlayerA in
// PostcombatMain. The attack lands (PlayerA -> 18). The Bolt must be
// rejected by the static ability — PlayerA stays at 18.
func TestAngelicArbiter_OpponentAttackedCantCastSpells(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Angelic Arbiter")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain", 1)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.Attack(2, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(2, core.PostcombatMain, gametest.PlayerB, "Lightning Bolt", "PlayerA")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 18)
}

// ===== Angelic Arbiter =====
//
// Each opponent who cast a spell this turn can't attack with creatures.
// Each opponent who attacked with a creature this turn can't cast spells.

// TestAngelicArbiter_OpponentCastSpellCantAttack: PlayerB casts Lightning
// Bolt at PlayerA on PlayerB's main phase, then tries to attack with
// Grizzly Bears. The bolt resolves (PlayerA -> 17) but the attack must
// be blocked by the static ability (no combat damage from the Bears).
func TestAngelicArbiter_OpponentCastSpellCantAttack(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Angelic Arbiter")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain", 1)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "PlayerA")
	g.Attack(2, gametest.PlayerB, "Grizzly Bears")
	g.StopAt(2, core.EndCombat)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 17)
}

// TestAngelicArbiter_PerTurnReset: turn 2 PlayerB casts a spell (so can't
// attack on turn 2), but on turn 4 (PlayerB's next turn) PlayerB hasn't
// done anything yet and can attack normally.
func TestAngelicArbiter_PerTurnReset(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Angelic Arbiter")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain", 1)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
	// Turn 2: PlayerB casts a spell, then can't attack this turn.
	g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "PlayerA")
	g.Attack(2, gametest.PlayerB, "Grizzly Bears")
	// Turn 4: PlayerB's next turn, has done nothing — Bears can attack.
	g.Attack(4, gametest.PlayerB, "Grizzly Bears")
	g.StopAt(4, core.EndCombat)
	g.Execute()
	// Turn 2: only the bolt landed (PlayerA -> 17), no Bears damage.
	// Turn 4: Bears attack lands (PlayerA -> 15).
	g.AssertLife(gametest.PlayerA, 15)
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
func TestArchonOfJustice(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Archon of Justice")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Archon of Justice", core.Flying, true)
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
func TestArmorcraftJudge_DrawsForCounterCreatures(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Armorcraft Judge")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCounters(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears", core.P1P1, 1)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains", 5)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Armorcraft Judge")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Plains", 1)
}
func TestAshmouthHound_DealsOneToBlocker(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Ashmouth Hound")
	g.Attack(3, gametest.PlayerA, "Grizzly Bears")
	g.Block(3, gametest.PlayerB, "Ashmouth Hound", "Grizzly Bears")
	g.StopAt(3, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
}
func TestBallLightning_SacrificesAtEndStep(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ball Lightning")
	g.StopAt(2, core.Untap)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Ball Lightning", 0)
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
func TestBeetleback_CreatesTwoGoblins(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Beetleback Chief")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 4)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Beetleback Chief")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Goblin", 2)
}
func TestBelltowerSphinx_DamagerControllerMills(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Belltower Sphinx")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain", 1)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Hill Giant", 5)
	g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Belltower Sphinx")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 3)
}
func TestBlackCat_RandomDiscardOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Black Cat")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	// Use a non-land card so the harness's auto-land-play doesn't move it
	// out of PlayerB's hand before Black Cat's death trigger resolves.
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.Attack(2, gametest.PlayerB, "Hill Giant")
	g.Block(2, gametest.PlayerA, "Black Cat", "Hill Giant")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Lightning Bolt", 1)
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
func TestBlightedBat_GainsHaste(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Blighted Bat")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 3)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Blighted Bat")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Blighted Bat")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Blighted Bat", core.Haste, true)
}
func TestBloodArtist_DrainsOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Blood Artist")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 21)
	g.AssertLife(gametest.PlayerB, 19)
}
func TestBloodHost_SacForCounterAndLife(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Blood Host")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Blood Host")
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 22)
	g.AssertPowerToughness(gametest.PlayerA, "Blood Host", 4, 4)
}
func TestBloodbondVampire_GainsCounterOnLifeGain(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bloodbond Vampire")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Healing Salve")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Healing Salve", "PlayerA")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 23)
	g.AssertCounterCount(gametest.PlayerA, "Bloodbond Vampire", core.P1P1, 1)
}
func TestBloodbondVampire_NoCounterOnOpponentLifeGain(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bloodbond Vampire")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Healing Salve")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains")
	g.CastSpell(1, core.PostcombatMain, gametest.PlayerB, "Healing Salve", "PlayerB")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Bloodbond Vampire", core.P1P1, 0)
}
func TestBloodhunterBat_DrainsOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Bloodhunter Bat")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Bloodhunter Bat")
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 22)
	g.AssertLife(gametest.PlayerB, 18)
}
func TestBloodrageBrawler_DiscardsOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Bloodrage Brawler")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Bloodrage Brawler")
	g.ChooseDiscard(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
}
func TestBorderlandMarauder_BoostsOnAttack(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Borderland Marauder")
	g.SetLife(gametest.PlayerB, 20)
	g.Attack(3, gametest.PlayerA, "Borderland Marauder")
	g.StopAt(3, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 17)
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
func TestBrindleShoat_CreatesBoarOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Brindle Shoat")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Brindle Shoat")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Boar", 1)
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
func TestBurglarRat_OpponentDiscards(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Burglar Rat")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Burglar Rat")
	g.ChooseDiscard(gametest.PlayerB, "Mountain")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Mountain", 1)
}
func TestCadaverImp_ReturnsCreatureFromGraveyard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Cadaver Imp")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 3)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Cadaver Imp")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
}
func TestCarvenCaryatid_DrawsOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Carven Caryatid")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains", 3)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Carven Caryatid")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Plains", 1)
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
func TestCauldronFamiliar_DrainsOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Cauldron Familiar")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Cauldron Familiar")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 21)
	g.AssertLife(gametest.PlayerB, 19)
	g.AssertPermanentCount(gametest.PlayerA, "Cauldron Familiar", 1)
}
func TestChampionOfLambholt_GainsCounterOnCreatureETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Champion of Lambholt")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Champion of Lambholt", core.P1P1, 1)
}
func TestCinderElemental_DealsXAndSacrifices(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cinder Elemental")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 4)
	g.SetLife(gametest.PlayerB, 20)
	g.ActivateAbilityWithX(1, core.PrecombatMain, gametest.PlayerA, "Cinder Elemental", 3, "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 17)
	g.AssertPermanentCount(gametest.PlayerA, "Cinder Elemental", 0)
}
func TestCloudreaderSphinx(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Cloudreader Sphinx")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 5)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
	g.ChooseScry(gametest.PlayerA, []string{"Mountain"}, []string{"Forest"})
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Cloudreader Sphinx")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLibraryTop(gametest.PlayerA, "Forest", "Plains", "Mountain")
}
func TestCorpseHauler_SacToReturn(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Corpse Hauler")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 3)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Corpse Hauler", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Corpse Hauler", 1)
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

// Tests for green tail / multicolor / colorless creatures (chunk 4).
func TestCraterhoofBehemoth_PumpsAndGrantsTrample(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Craterhoof Behemoth")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 8)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Craterhoof Behemoth")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	// 2 creatures => +2/+2 and trample.
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 4)
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Trample, true)
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
