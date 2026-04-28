package jumpstart

import (
	"testing"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// Tests for cards wired to the type-granting / color-changing continuous
// effect APIs (BecomesSubType, BecomesColor/BecomesColors,
// GrantSubTypeToTarget, SetBasePT) in cards/jumpstart.

func TestWishfulMerfolk_BecomesHumanLosesDefender(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Wishful Merfolk")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Island", 2)

	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Wishful Merfolk")
	g.StopAt(1, EndStep)
	g.Execute()

	g.AssertHasAbility(gametest.PlayerA, "Wishful Merfolk", Defender, false)
}

func TestWishfulMerfolk_RevertsAtEOT(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Wishful Merfolk")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Island", 2)

	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Wishful Merfolk")
	g.StopAt(2, Untap)
	g.Execute()

	g.AssertHasAbility(gametest.PlayerA, "Wishful Merfolk", Defender, true)
}

func TestScuttlemutt_BecomesChosenColorUntilEOT(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Scuttlemutt")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")

	g.ChooseManaColor(gametest.PlayerA, Blue)
	g.GetPlayer(gametest.PlayerA).QueueMayAbilityChoices(false)
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Scuttlemutt", "Grizzly Bears")
	g.StopAt(1, EndStep)
	g.Execute()

	bears := g.FindPermanentByName("Grizzly Bears", g.GetPlayer(gametest.PlayerA).PlayerID())
	if bears == nil {
		t.Fatal("Grizzly Bears not found")
	}
	cs := bears.Colors()
	if len(cs) != 1 || cs[0] != Blue {
		t.Errorf("expected colors == [Blue]; got %v", cs)
	}
}

func TestAllosaurusShepherd_PumpsElvesAndGrantsDinosaur(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Allosaurus Shepherd")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Llanowar Elves")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 6)

	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Allosaurus Shepherd")
	g.StopAt(1, PostcombatMain)
	g.Execute()

	g.AssertPowerToughness(gametest.PlayerA, "Llanowar Elves", 5, 5)
	g.AssertPowerToughness(gametest.PlayerA, "Allosaurus Shepherd", 5, 5)

	elf := g.FindPermanentByName("Llanowar Elves", g.GetPlayer(gametest.PlayerA).PlayerID())
	if elf == nil {
		t.Fatal("Llanowar Elves not found")
	}
	if !elf.HasSubType("Dinosaur") {
		t.Errorf("expected Llanowar Elves to gain Dinosaur subtype")
	}
	if !elf.HasSubType("Elf") {
		t.Errorf("expected Llanowar Elves to keep Elf subtype")
	}
}
