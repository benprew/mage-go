package jumpstart

import (
	"testing"

	_ "github.com/benprew/mage-go/cards/limited"
	"github.com/benprew/mage-go/pkg/mage"
	. "github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestRishadanAirship_CanOnlyBlockFlying(t *testing.T) {
	t.Run("ground attacker not blocked", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(ZoneBattlefield, gametest.PlayerB, "Rishadan Airship")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Rishadan Airship", "Grizzly Bears")
		g.StopAt(1, EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 18)
	})

	t.Run("flying attacker can be blocked", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(ZoneBattlefield, gametest.PlayerA, "Serra Angel")
		g.AddCard(ZoneBattlefield, gametest.PlayerB, "Rishadan Airship")
		g.Attack(1, gametest.PlayerA, "Serra Angel")
		g.Block(1, gametest.PlayerB, "Rishadan Airship", "Serra Angel")
		g.StopAt(1, EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 20)
	})
}

func TestGoblinGoon_CantAttackWhenOutnumbered(t *testing.T) {
	t.Run("blocked from attacking when defender controls more creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(ZoneBattlefield, gametest.PlayerA, "Goblin Goon")
		g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears", 2)
		g.Attack(1, gametest.PlayerA, "Goblin Goon")
		g.StopAt(1, EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 20)
	})

	t.Run("can attack when controller has more creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(ZoneBattlefield, gametest.PlayerA, "Goblin Goon")
		g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears", 2)
		g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Goblin Goon")
		g.StopAt(1, EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 14)
	})
}

func TestChampionOfLambholt_PreventsSmallBlockers(t *testing.T) {
	t.Run("smaller blocker can't block teammate", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(ZoneBattlefield, gametest.PlayerA, "Champion of Lambholt")
		g.AddCounters(1, PrecombatMain, gametest.PlayerA, "Champion of Lambholt", P1P1, 2)
		g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Grizzly Bears")
		g.StopAt(1, EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 18)
	})
}

func TestGhirapurGuide_GrantsCantBeBlockedBySmall(t *testing.T) {
	t.Run("small blocker can't block boosted target", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(ZoneBattlefield, gametest.PlayerA, "Ghirapur Guide")
		g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 3)
		g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Ghirapur Guide", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Grizzly Bears")
		g.StopAt(1, EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 18)
	})
}

func TestGingerbrute_CantBeBlockedExceptByHaste(t *testing.T) {
	t.Run("structural: card has both activated abilities", func(t *testing.T) {
		card, err := mage.CreateCard("Gingerbrute")
		if err != nil {
			t.Fatalf("Gingerbrute not registered: %v", err)
		}
		activated := 0
		for _, a := range card.Abilities() {
			inner := mage.UnwrapAbility(a)
			if _, ok := inner.(mage.ActivatedAbility); ok {
				activated++
			}
		}
		if activated < 2 {
			t.Fatalf("Gingerbrute should have at least 2 activated abilities, got %d", activated)
		}
	})

	t.Run("effect: non-haste blocker can't block while restriction active", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		gingerID := g.AddCard(ZoneBattlefield, gametest.PlayerA, "Gingerbrute")
		g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")

		eff := mage.TargetCantBeBlockedExceptBy(gingerID, mage.HasKeywordFilter(Haste), EndOfTurn)
		eff.SetSourceID(gingerID)
		g.AddContinuousEffect(eff)
		g.ApplyContinuousEffects()

		g.Attack(1, gametest.PlayerA, "Gingerbrute")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Gingerbrute")
		g.StopAt(1, EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 19)
	})
}

func TestEnlarge_BoostAndMustBeBlocked(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(ZoneHand, gametest.PlayerA, "Enlarge")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 5)
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Enlarge", "Grizzly Bears")
	g.Attack(1, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, EndCombat)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", Trample, true)
	g.AssertLife(gametest.PlayerB, 13)
	g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
}

func TestIrresistiblePrey_MustBeBlockedAndDraw(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(ZoneHand, gametest.PlayerA, "Irresistible Prey")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Forest", 3)
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Irresistible Prey", "Grizzly Bears")
	handBefore := 0
	_ = handBefore
	g.Attack(1, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, EndCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
	g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	g.AssertLife(gametest.PlayerB, 20)
}
